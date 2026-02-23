package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"trading/api"
)

// ============================================================
// 常量
// ============================================================

const (
	defaultFeeRate  = 0.001 // 默认手续费率 0.1%
	minPollInterval = 2 * time.Second
	extraBidStep    = 0.001 // 卖单积压超过 2× 层数时，额外追加的买单价差（0.1%）
)

// ============================================================
// 交易对精度（从 /api/v3/exchangeInfo 动态获取）
// ============================================================

// SymbolFilters 保存 tickSize / stepSize 及其推导出的小数位数
type SymbolFilters struct {
	TickSize       float64
	StepSize       float64
	PricePrecision int
	QtyPrecision   int
}

// exchangeInfoResp 仅解析 /api/v3/exchangeInfo 中需要的字段
type exchangeInfoResp struct {
	Symbols []struct {
		Symbol  string `json:"symbol"`
		Filters []struct {
			FilterType string `json:"filterType"`
			TickSize   string `json:"tickSize"`
			StepSize   string `json:"stepSize"`
		} `json:"filters"`
	} `json:"symbols"`
}

// loadSymbolFilters 调用 Binance /api/v3/exchangeInfo 获取交易对精度。
// 使用标准 http 包发起无鉴权公开请求即可。
func loadSymbolFilters(baseURL, symbol string) (*SymbolFilters, error) {
	url := baseURL + "/api/v3/exchangeInfo?symbol=" + symbol
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("请求 exchangeInfo 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 exchangeInfo 响应失败: %w", err)
	}

	var info exchangeInfoResp
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("解析 exchangeInfo 失败: %w", err)
	}

	for _, s := range info.Symbols {
		if s.Symbol != symbol {
			continue
		}
		f := &SymbolFilters{}
		for _, filter := range s.Filters {
			switch filter.FilterType {
			case "PRICE_FILTER":
				if filter.TickSize != "" {
					f.TickSize, _ = strconv.ParseFloat(filter.TickSize, 64)
					f.PricePrecision = countDecimalPlaces(filter.TickSize)
				}
			case "LOT_SIZE":
				if filter.StepSize != "" {
					f.StepSize, _ = strconv.ParseFloat(filter.StepSize, 64)
					f.QtyPrecision = countDecimalPlaces(filter.StepSize)
				}
			}
		}
		if f.TickSize == 0 {
			return nil, fmt.Errorf("未找到 %s 的 PRICE_FILTER", symbol)
		}
		if f.StepSize == 0 {
			return nil, fmt.Errorf("未找到 %s 的 LOT_SIZE", symbol)
		}
		return f, nil
	}
	return nil, fmt.Errorf("exchangeInfo 中未找到交易对 %s", symbol)
}

// countDecimalPlaces 统计 "0.0100" 这类字符串的有效小数位数 → 2
func countDecimalPlaces(s string) int {
	s = strings.TrimRight(s, "0") // "0.01000000" → "0.01"
	dot := strings.Index(s, ".")
	if dot < 0 {
		return 0
	}
	return len(s) - dot - 1
}

// floorToTick 向下对齐到 tick 整数倍（用于买入价 / 数量，避免超出限制）
func floorToTick(value, tick float64) float64 {
	if tick == 0 {
		return value
	}
	return math.Floor(value/tick+1e-9) * tick
}

// ceilToTick 向上对齐到 tick 整数倍（用于卖出价，确保不低于盈利目标）
func ceilToTick(value, tick float64) float64 {
	if tick == 0 {
		return value
	}
	return math.Ceil(value/tick-1e-9) * tick
}

// formatWithPrecision 以固定小数位格式化浮点数为字符串
func formatWithPrecision(v float64, precision int) string {
	return strconv.FormatFloat(v, 'f', precision, 64)
}

// ============================================================
// 核心数据结构
// ============================================================

// BuySlot 一个买单层的运行状态
type BuySlot struct {
	Layer    int       // 层索引（0 = 最接近市价）
	OrderID  int64     // 交易所订单 ID
	BuyPrice float64   // 实际挂单买入价
	Quantity float64   // 挂单数量
	PlacedAt time.Time // 下单时刻（用于超时检测）
}

// SellSlot 一个卖单的运行状态（由买单成交后触发）
type SellSlot struct {
	BuyOrderID int64   // 触发本卖单的买单 ID（追踪用）
	OrderID    int64   // 交易所卖单 ID
	BuyPrice   float64 // 原始买入价（用于盈利计算）
	SellPrice  float64 // 实际挂单卖出价
	Quantity   float64 // 实际卖出量（扣除买入手续费后，对齐 stepSize）
}

// WaitMarketConfig 策略运行参数
type WaitMarketConfig struct {
	Market           string        // 交易对，如 "SOLUSDT"
	BidSpread        float64       // 基础买单价差（如 0.001 = 0.1%）
	AskSpread        float64       // 目标盈利率（如 0.01 = 1%）
	OrderRefreshTime time.Duration // 轮询周期
	MaxOrderAge      time.Duration // 买单最长存活时间（超时撤单重挂）
	FilledOrderDelay time.Duration // 买单成交后，延迟多久下卖单
	OrderAmount      float64       // 每笔买单数量（基础资产）
	OrderLevels      int           // 挂单层数（最多同时挂 N 个买单）
	FeeRate          float64       // 手续费率（买卖统一）
}

// WaitMarketStrategy 挂单做市策略主体
type WaitMarketStrategy struct {
	cfg     WaitMarketConfig
	client  *api.BinanceClient
	filters *SymbolFilters // 启动时加载

	mu         sync.Mutex
	buySlots   map[int64]*BuySlot  // 活跃买单，key = orderId
	sellSlots  map[int64]*SellSlot // 活跃卖单，key = orderId
	usedLayers map[int]bool        // 已占用层位
}

// ============================================================
// 构造
// ============================================================

// NewWaitMarketStrategy 从参数 map 构造策略实例。
// 精度加载在 Run() 中进行，以便日志输出有序。
func NewWaitMarketStrategy(client *api.BinanceClient, params map[string]interface{}) (*WaitMarketStrategy, error) {
	cfg, err := parseWaitMarketConfig(params)
	if err != nil {
		return nil, fmt.Errorf("解析策略参数失败: %w", err)
	}
	return &WaitMarketStrategy{
		cfg:        cfg,
		client:     client,
		buySlots:   make(map[int64]*BuySlot),
		sellSlots:  make(map[int64]*SellSlot),
		usedLayers: make(map[int]bool),
	}, nil
}

// ============================================================
// 主循环
// ============================================================

func (s *WaitMarketStrategy) Run(ctx context.Context) error {
	// ── 加载交易对精度 ────────────────────────────────────
	fmt.Printf("📐 正在获取 %s 交易对精度...\n", s.cfg.Market)
	filters, err := loadSymbolFilters(api.BaseURL, s.cfg.Market)
	if err != nil {
		return fmt.Errorf("获取交易对精度失败: %w", err)
	}
	s.filters = filters
	fmt.Printf("✅ 价格精度: tickSize=%.8g（%d 位小数）  数量精度: stepSize=%.8g（%d 位小数）\n",
		filters.TickSize, filters.PricePrecision,
		filters.StepSize, filters.QtyPrecision)

	fmt.Println("🚀 挂单做市策略启动")
	s.printConfig()

	ticker := time.NewTicker(s.cfg.OrderRefreshTime)
	defer ticker.Stop()

	// 启动时立即执行一次
	if err := s.tick(); err != nil {
		fmt.Printf("⚠️  首次 tick 出错: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("🛑 策略收到停止信号，正在撤销所有买单...")
			s.cancelAllBuyOrders()
			fmt.Println("✅ 策略已停止（卖单保留，等待自然成交）")
			return nil
		case <-ticker.C:
			if err := s.tick(); err != nil {
				fmt.Printf("⚠️  tick 出错: %v\n", err)
			}
		}
	}
}

func (s *WaitMarketStrategy) tick() error {
	// 1. 同步所有订单状态（成交检测 / 超时撤单）
	if err := s.syncOrders(); err != nil {
		return fmt.Errorf("同步订单失败: %w", err)
	}

	// 2. 获取最新价格
	latestPrice, err := s.getLatestPrice()
	if err != nil {
		return fmt.Errorf("获取价格失败: %w", err)
	}

	// 3. 根据规则补充缺失的买单层
	s.refillBuySlots(latestPrice)

	// 4. 打印当前状态
	s.printStatus(latestPrice)
	return nil
}

// ============================================================
// 订单状态同步
// ============================================================

func (s *WaitMarketStrategy) syncOrders() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ── 买单检查 ──────────────────────────────────────────
	for oid, slot := range s.buySlots {
		status, err := s.queryOrderStatus(s.cfg.Market, oid)
		if err != nil {
			fmt.Printf("  ⚠️  查询买单 %d 状态失败: %v\n", oid, err)
			continue
		}

		switch status {
		case "FILLED":
			// ── 需求1：买单成交 → 立即计算盈利价格挂卖单 ──
			fmt.Printf("  ✅ 买单 %d（层%d, 价格%s）已成交，立即挂出卖单...\n",
				oid, slot.Layer, s.fmtPrice(slot.BuyPrice))

			sellPrice := s.calcSellPrice(slot.BuyPrice)
			// 实际持有量：扣除买入手续费，向下对齐 stepSize
			sellQty := s.floorQty(slot.Quantity * (1 - s.cfg.FeeRate))

			time.Sleep(s.cfg.FilledOrderDelay)

			sellOID, err := s.placeSellOrder(s.cfg.Market, sellQty, sellPrice)
			if err != nil {
				fmt.Printf("  ❌ 挂卖单失败: %v（持仓可能滞留，将在下次 tick 重试）\n", err)
				// 买单层位仍然释放，避免该层卡死；卖单在下次 tick 会被重新检测
			} else {
				s.sellSlots[sellOID] = &SellSlot{
					BuyOrderID: oid,
					OrderID:    sellOID,
					BuyPrice:   slot.BuyPrice,
					SellPrice:  sellPrice,
					Quantity:   sellQty,
				}
				fmt.Printf("  📤 卖单已挂出 %d，价格 %s，数量 %s\n",
					sellOID, s.fmtPrice(sellPrice), s.fmtQty(sellQty))
			}

			// 无论卖单是否成功，都释放买单层位
			delete(s.usedLayers, slot.Layer)
			delete(s.buySlots, oid)

		case "CANCELED", "REJECTED", "EXPIRED":
			fmt.Printf("  ⚠️  买单 %d 状态异常: %s，释放层位 %d\n", oid, status, slot.Layer)
			delete(s.usedLayers, slot.Layer)
			delete(s.buySlots, oid)

		case "NEW", "PARTIALLY_FILLED":
			// ── 需求2：超时则撤单，等待 refillBuySlots 以新价重挂 ──
			age := time.Since(slot.PlacedAt)
			if age > s.cfg.MaxOrderAge {
				fmt.Printf("  ⏰ 买单 %d 存活 %.0fs 已超时，撤单并按最新价重挂...\n",
					oid, age.Seconds())
				if err := s.cancelOrder(s.cfg.Market, oid); err != nil {
					fmt.Printf("  ❌ 撤单失败: %v\n", err)
				} else {
					delete(s.usedLayers, slot.Layer)
					delete(s.buySlots, oid)
					// 层位空出后，下方 refillBuySlots 会以最新价重新挂单
				}
			}
		}
	}

	// ── 卖单检查 ──────────────────────────────────────────
	for oid, slot := range s.sellSlots {
		status, err := s.queryOrderStatus(s.cfg.Market, oid)
		if err != nil {
			fmt.Printf("  ⚠️  查询卖单 %d 状态失败: %v\n", oid, err)
			continue
		}

		switch status {
		case "FILLED":
			profit := (slot.SellPrice - slot.BuyPrice) * slot.Quantity
			fmt.Printf("  💰 卖单 %d 成交！买入 %s → 卖出 %s，数量 %s，盈利 ≈%.4f USDT\n",
				oid,
				s.fmtPrice(slot.BuyPrice),
				s.fmtPrice(slot.SellPrice),
				s.fmtQty(slot.Quantity),
				profit)
			delete(s.sellSlots, oid)

		case "CANCELED", "REJECTED", "EXPIRED":
			// ── 需求1：卖单异常 → 必须重新挂出，不留库存 ──
			fmt.Printf("  ⚠️  卖单 %d 状态异常: %s，重新挂出以清除库存...\n", oid, status)
			newOID, err := s.placeSellOrder(s.cfg.Market, slot.Quantity, slot.SellPrice)
			if err != nil {
				fmt.Printf("  ❌ 重新挂卖单失败: %v（将在下次 tick 再试）\n", err)
				// 保留旧 slot 等下次 tick 重试
			} else {
				s.sellSlots[newOID] = &SellSlot{
					BuyOrderID: slot.BuyOrderID,
					OrderID:    newOID,
					BuyPrice:   slot.BuyPrice,
					SellPrice:  slot.SellPrice,
					Quantity:   slot.Quantity,
				}
				delete(s.sellSlots, oid)
			}
		}
	}

	return nil
}

// ============================================================
// 补充买单（含积压保护逻辑）
// ============================================================

func (s *WaitMarketStrategy) refillBuySlots(latestPrice float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ── 需求4：买单数量上限 = order_levels ──────────────
	if len(s.buySlots) >= s.cfg.OrderLevels {
		return
	}

	sellCount := len(s.sellSlots)
	maxSell := s.cfg.OrderLevels

	// ── 需求6：卖单积压 > 3× 层数 → 暂停所有买入 ────────
	if sellCount > maxSell*3 {
		fmt.Printf("  ⏸️  卖单积压 %d 个（> %d×3），暂停挂买单，等待卖出...\n",
			sellCount, maxSell)
		return
	}

	// ── 需求5：卖单积压 > 2× 层数 → 加深价差，更低价买入 ─
	bidSpread := s.cfg.BidSpread
	if sellCount > maxSell*2 {
		bidSpread += extraBidStep
		fmt.Printf("  📉 卖单积压 %d 个（> %d×2），价差加深 %.1f%%，当前有效价差 %.4f%%\n",
			sellCount, maxSell, extraBidStep*100, bidSpread*100)
	}

	// ── 需求3：逐层补充，直到达到 order_levels 上限 ──────
	for layer := 0; layer < s.cfg.OrderLevels; layer++ {
		if s.usedLayers[layer] {
			continue // 该层已有买单，跳过
		}
		if len(s.buySlots) >= s.cfg.OrderLevels {
			break // 总数已达上限
		}

		// 计算当层买入价：每深一层多偏移一档 bidSpread
		rawPrice := latestPrice * (1 - bidSpread*float64(layer+1))
		buyPrice := s.floorPrice(rawPrice) // 向下对齐 tickSize
		qty := s.floorQty(s.cfg.OrderAmount)

		oid, err := s.placeBuyOrder(s.cfg.Market, qty, buyPrice)
		if err != nil {
			fmt.Printf("  ❌ 层%d 挂买单失败 (价格%s): %v\n",
				layer, s.fmtPrice(buyPrice), err)
			continue
		}

		s.buySlots[oid] = &BuySlot{
			Layer:    layer,
			OrderID:  oid,
			BuyPrice: buyPrice,
			Quantity: qty,
			PlacedAt: time.Now(),
		}
		s.usedLayers[layer] = true

		fmt.Printf("  📥 层%d 买单 %d 已挂出，价格 %s，数量 %s\n",
			layer, oid, s.fmtPrice(buyPrice), s.fmtQty(qty))
	}
}

// ============================================================
// 价格 / 数量精度辅助
// ============================================================

// calcSellPrice 含手续费的盈利卖出价，向上对齐 tickSize
//
//	P_sell = P_buy × (1 + askSpread) / ((1 − feeBuy) × (1 − feeSell))
func (s *WaitMarketStrategy) calcSellPrice(buyPrice float64) float64 {
	raw := buyPrice * (1 + s.cfg.AskSpread) / ((1 - s.cfg.FeeRate) * (1 - s.cfg.FeeRate))
	return ceilToTick(raw, s.filters.TickSize)
}

func (s *WaitMarketStrategy) floorPrice(p float64) float64 {
	return floorToTick(p, s.filters.TickSize)
}

func (s *WaitMarketStrategy) floorQty(q float64) float64 {
	return floorToTick(q, s.filters.StepSize)
}

func (s *WaitMarketStrategy) fmtPrice(p float64) string {
	return formatWithPrecision(p, s.filters.PricePrecision)
}

func (s *WaitMarketStrategy) fmtQty(q float64) string {
	return formatWithPrecision(q, s.filters.QtyPrecision)
}

// ============================================================
// API 调用封装
// ============================================================

func (s *WaitMarketStrategy) getLatestPrice() (float64, error) {
	sym := s.cfg.Market
	tickers, err := s.client.GetTickerPrice(api.GetTickerPriceParams{Symbol: &sym})
	if err != nil {
		return 0, err
	}
	if len(tickers) == 0 {
		return 0, fmt.Errorf("未获取到 %s 的价格", sym)
	}
	return strconv.ParseFloat(tickers[0].Price, 64)
}

func (s *WaitMarketStrategy) placeBuyOrder(symbol string, qty, price float64) (int64, error) {
	tif := "GTC"
	priceStr := s.fmtPrice(price)
	qtyStr := s.fmtQty(qty)
	resp, err := s.client.PlaceOrder(api.PlaceOrderParams{
		Symbol:      symbol,
		Side:        "BUY",
		Type:        "LIMIT",
		TimeInForce: &tif,
		Quantity:    &qtyStr,
		Price:       &priceStr,
	})
	if err != nil {
		return 0, err
	}
	return resp.OrderId, nil
}

func (s *WaitMarketStrategy) placeSellOrder(symbol string, qty, price float64) (int64, error) {
	tif := "GTC"
	priceStr := s.fmtPrice(price)
	qtyStr := s.fmtQty(qty)
	resp, err := s.client.PlaceOrder(api.PlaceOrderParams{
		Symbol:      symbol,
		Side:        "SELL",
		Type:        "LIMIT",
		TimeInForce: &tif,
		Quantity:    &qtyStr,
		Price:       &priceStr,
	})
	if err != nil {
		return 0, err
	}
	return resp.OrderId, nil
}

func (s *WaitMarketStrategy) cancelOrder(symbol string, orderID int64) error {
	_, err := s.client.CancelOrder(api.CancelOrderParams{
		Symbol:  symbol,
		OrderId: &orderID,
	})
	return err
}

func (s *WaitMarketStrategy) queryOrderStatus(symbol string, orderID int64) (string, error) {
	resp, err := s.client.GetOrder(api.GetOrderParams{
		Symbol:  symbol,
		OrderId: &orderID,
	})
	if err != nil {
		return "", err
	}
	return resp.Status, nil
}

// cancelAllBuyOrders 退出时撤销所有买单；卖单保留，等待自然成交
func (s *WaitMarketStrategy) cancelAllBuyOrders() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for oid, slot := range s.buySlots {
		if err := s.cancelOrder(s.cfg.Market, oid); err != nil {
			fmt.Printf("  ⚠️  撤销买单 %d（层%d）失败: %v\n", oid, slot.Layer, err)
		} else {
			fmt.Printf("  🗑️  买单 %d（层%d）已撤销\n", oid, slot.Layer)
		}
	}
}

// ============================================================
// 配置解析
// ============================================================

func parseWaitMarketConfig(p map[string]interface{}) (WaitMarketConfig, error) {
	cfg := WaitMarketConfig{FeeRate: defaultFeeRate}

	if v, ok := p["market"].(string); ok && v != "" {
		cfg.Market = v
	} else {
		return cfg, fmt.Errorf("缺少必填参数 market")
	}

	cfg.BidSpread = getFloat(p, "bid_spread", 0.001)
	cfg.AskSpread = getFloat(p, "ask_spread", 0.01)
	cfg.OrderAmount = getFloat(p, "order_amount", 0.1)

	refreshSec := getInt(p, "order_refresh_time", 60)
	cfg.OrderRefreshTime = time.Duration(refreshSec) * time.Second
	if cfg.OrderRefreshTime < minPollInterval {
		cfg.OrderRefreshTime = minPollInterval
	}

	maxAgeSec := getInt(p, "max_order_age", 300)
	cfg.MaxOrderAge = time.Duration(maxAgeSec) * time.Second

	delaySec := getInt(p, "filled_order_delay", 1)
	cfg.FilledOrderDelay = time.Duration(delaySec) * time.Second

	cfg.OrderLevels = getInt(p, "order_levels", 3)
	if cfg.OrderLevels < 1 {
		cfg.OrderLevels = 1
	}

	return cfg, nil
}

func getFloat(p map[string]interface{}, key string, defaultVal float64) float64 {
	v, ok := p[key]
	if !ok {
		return defaultVal
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return defaultVal
}

func getInt(p map[string]interface{}, key string, defaultVal int) int {
	v, ok := p[key]
	if !ok {
		return defaultVal
	}
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	}
	return defaultVal
}

// ============================================================
// 状态打印
// ============================================================

func (s *WaitMarketStrategy) printConfig() {
	c := s.cfg
	f := s.filters
	maxSell := c.OrderLevels
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  交易对:           %s\n", c.Market)
	fmt.Printf("  价格精度:         tickSize=%.8g（%d 位小数）\n", f.TickSize, f.PricePrecision)
	fmt.Printf("  数量精度:         stepSize=%.8g（%d 位小数）\n", f.StepSize, f.QtyPrecision)
	fmt.Printf("  基础买单价差:     %.4f%%\n", c.BidSpread*100)
	fmt.Printf("  目标盈利率:       %.4f%%\n", c.AskSpread*100)
	fmt.Printf("  每笔买单数量:     %s\n", s.fmtQty(c.OrderAmount))
	fmt.Printf("  挂单层数:         %d（最多同时 %d 个买单）\n", c.OrderLevels, c.OrderLevels)
	fmt.Printf("  加深价差阈值:     卖单 > %d（%d×2）时，买单价差 +%.1f%%\n",
		maxSell*2, maxSell, extraBidStep*100)
	fmt.Printf("  暂停买入阈值:     卖单 > %d（%d×3）时，暂停下买单\n",
		maxSell*3, maxSell)
	fmt.Printf("  买单最长存活:     %s\n", c.MaxOrderAge)
	fmt.Printf("  刷新周期:         %s\n", c.OrderRefreshTime)
	fmt.Printf("  成交后延迟:       %s\n", c.FilledOrderDelay)
	fmt.Printf("  手续费率:         %.4f%%\n", c.FeeRate*100)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func (s *WaitMarketStrategy) printStatus(latestPrice float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sellCount := len(s.sellSlots)
	maxSell := s.cfg.OrderLevels

	// 计算当前有效价差（与 refillBuySlots 中逻辑保持一致）
	effectiveSpread := s.cfg.BidSpread
	if sellCount > maxSell*2 {
		effectiveSpread += extraBidStep
	}

	// 买入状态标签
	buyState := "正常"
	switch {
	case sellCount > maxSell*3:
		buyState = "⏸️ 暂停（卖单积压 > 3×）"
	case sellCount > maxSell*2:
		buyState = fmt.Sprintf("📉 加深价差（%.4f%%）", effectiveSpread*100)
	}

	fmt.Printf("\n[%s] 最新价: %s  买单: %d/%d  卖单: %d  买入状态: %s\n",
		time.Now().Format("15:04:05"),
		s.fmtPrice(latestPrice),
		len(s.buySlots), s.cfg.OrderLevels,
		sellCount,
		buyState,
	)

	if len(s.buySlots) > 0 {
		fmt.Println("  📥 买单列表:")
		for _, slot := range s.buySlots {
			age := time.Since(slot.PlacedAt).Round(time.Second)
			remaining := s.cfg.MaxOrderAge - time.Since(slot.PlacedAt)
			if remaining < 0 {
				remaining = 0
			}
			fmt.Printf("     层%-2d  ID:%-12d  价格:%-12s  存活:%-8s  剩余:%s\n",
				slot.Layer, slot.OrderID,
				s.fmtPrice(slot.BuyPrice),
				age, remaining.Round(time.Second))
		}
	}

	if sellCount > 0 {
		fmt.Printf("  📤 卖单列表（%d 个）:\n", sellCount)
		for _, slot := range s.sellSlots {
			fmt.Printf("     ID:%-12d  买入:%-12s  卖出:%-12s  数量:%-10s\n",
				slot.OrderID,
				s.fmtPrice(slot.BuyPrice),
				s.fmtPrice(slot.SellPrice),
				s.fmtQty(slot.Quantity))
		}

		// 积压警告
		switch {
		case sellCount > maxSell*3:
			fmt.Printf("  🚨 卖单积压严重（%d > %d×3=%d），已暂停挂买单！\n",
				sellCount, maxSell, maxSell*3)
		case sellCount > maxSell*2:
			fmt.Printf("  ⚠️  卖单积压（%d > %d×2=%d），买单价差已加深至 %.4f%%\n",
				sellCount, maxSell, maxSell*2, effectiveSpread*100)
		}
	}
}
