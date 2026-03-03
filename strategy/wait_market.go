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

	// ATR 动态价差参数
	atrKlineInterval = "15m" // 计算 ATR 的 K 线周期
	atrPeriod        = 14    // ATR 周期
	atrMultiplierMin = 0.8   // ATR 乘数下限（低波动时收窄价差）
	atrMultiplierMax = 2.5   // ATR 乘数上限（高波动时放宽价差，保护资金）
	atrFetchLimit    = 20    // 拉取 K 线数量（>= atrPeriod+1）

	// EMA 趋势过滤参数
	emaPeriod       = 20  // EMA 计算周期
	emaKlineLimit   = 25  // 拉取 K 线数量（>= emaPeriod+1）
	trendBearFactor = 1.5 // 熊市时价差放大倍数（降低买入频率）

	// 卖单超时重定价参数
	sellRepriceFactor = 0.998 // 每次重定价降低 0.2%（尽快清库存）
	minSellMargin     = 0.001 // 重定价后的最低保本利润率（0.1%，扣费后仍不亏损）
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
	s = strings.TrimRight(s, "0")
	dot := strings.Index(s, ".")
	if dot < 0 {
		return 0
	}
	return len(s) - dot - 1
}

// floorToTick 向下对齐到 tick 整数倍
func floorToTick(value, tick float64) float64 {
	if tick == 0 {
		return value
	}
	return math.Floor(value/tick+1e-9) * tick
}

// ceilToTick 向上对齐到 tick 整数倍
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
	Layer    int
	OrderID  int64
	BuyPrice float64
	Quantity float64
	PlacedAt time.Time
}

// SellSlot 一个卖单的运行状态
type SellSlot struct {
	BuyOrderID   int64
	OrderID      int64
	BuyPrice     float64 // 原始买入价（含手续费成本基准）
	SellPrice    float64 // 当前挂单卖出价
	Quantity     float64
	PlacedAt     time.Time // 卖单挂出时刻（用于超时重定价）
	RepriceCount int       // 已重定价次数
}

// WaitMarketConfig 策略运行参数
type WaitMarketConfig struct {
	Market           string
	BidSpread        float64
	AskSpread        float64
	OrderRefreshTime time.Duration
	MaxOrderAge      time.Duration // 买单最长存活时间
	MaxSellAge       time.Duration // 卖单最长存活时间（超时重定价）
	FilledOrderDelay time.Duration
	OrderAmount      float64
	OrderLevels      int
	FeeRate          float64
	UseATR           bool // 是否启用 ATR 动态价差
	UseTrend         bool // 是否启用趋势过滤
	MaxReprice       int  // 卖单最大重定价次数（0=不限）
}

// pnlStats 盈亏统计
type pnlStats struct {
	TotalProfit    float64 // 累计毛利润（USDT）
	TotalFees      float64 // 累计手续费（估算）
	TotalNetProfit float64 // 累计净利润
	FilledRounds   int     // 已完成买卖轮次
}

// WaitMarketStrategy 挂单做市策略主体
type WaitMarketStrategy struct {
	cfg     WaitMarketConfig
	client  *api.BinanceClient
	filters *SymbolFilters

	mu         sync.Mutex
	buySlots   map[int64]*BuySlot
	sellSlots  map[int64]*SellSlot
	usedLayers map[int]bool

	// ── 优化1：动态指标缓存 ────────────────────────────────
	lastATR        float64   // 最近计算的 ATR 值（绝对价格）
	lastEMA        float64   // 最近计算的 EMA 值
	lastIndicatorT time.Time // 上次指标刷新时间（避免每 tick 都拉 K 线）

	// ── 优化2：盈亏追踪 ───────────────────────────────────
	pnl pnlStats
}

// ============================================================
// 构造
// ============================================================

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
	fmt.Printf("📐 正在获取 %s 交易对精度...\n", s.cfg.Market)
	filters, err := loadSymbolFilters(api.BaseURL, s.cfg.Market)
	if err != nil {
		return fmt.Errorf("获取交易对精度失败: %w", err)
	}
	s.filters = filters
	fmt.Printf("✅ 价格精度: tickSize=%.8g（%d 位）  数量精度: stepSize=%.8g（%d 位）\n",
		filters.TickSize, filters.PricePrecision,
		filters.StepSize, filters.QtyPrecision)

	fmt.Println("🚀 优化版挂单做市策略启动")
	s.printConfig()

	ticker := time.NewTicker(s.cfg.OrderRefreshTime)
	defer ticker.Stop()

	if err := s.tick(); err != nil {
		fmt.Printf("⚠️  首次 tick 出错: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("🛑 策略收到停止信号，正在撤销所有买单...")
			s.cancelAllBuyOrders()
			fmt.Println("✅ 策略已停止（卖单保留，等待自然成交）")
			s.printPnL()
			return nil
		case <-ticker.C:
			if err := s.tick(); err != nil {
				fmt.Printf("⚠️  tick 出错: %v\n", err)
			}
		}
	}
}

func (s *WaitMarketStrategy) tick() error {
	// ── 优化3：每 5 分钟刷新一次技术指标（减少 K 线 API 频率）─
	if s.cfg.UseATR || s.cfg.UseTrend {
		if time.Since(s.lastIndicatorT) > 5*time.Minute {
			s.refreshIndicators()
		}
	}

	if err := s.syncOrders(); err != nil {
		return fmt.Errorf("同步订单失败: %w", err)
	}

	// ── 优化4：使用 BookTicker（bid 价）作为参考价，更贴近成交 ─
	bidPrice, askPrice, err := s.getBidAskPrice()
	if err != nil {
		return fmt.Errorf("获取价格失败: %w", err)
	}
	midPrice := (bidPrice + askPrice) / 2

	s.refillBuySlots(bidPrice, midPrice)
	s.printStatus(midPrice)
	return nil
}

// ============================================================
// 优化5：动态技术指标（ATR + EMA）
// ============================================================

// refreshIndicators 刷新 ATR 和 EMA，供动态价差 / 趋势过滤使用
func (s *WaitMarketStrategy) refreshIndicators() {
	limit := atrFetchLimit
	if limit < emaKlineLimit {
		limit = emaKlineLimit
	}
	klines, err := s.client.GetKlines(api.GetKlinesParams{
		Symbol:   s.cfg.Market,
		Interval: atrKlineInterval,
		Limit:    &limit,
	})
	if err != nil {
		fmt.Printf("  ⚠️  拉取 K 线失败，跳过指标刷新: %v\n", err)
		return
	}

	highs := make([]float64, 0, len(klines))
	lows := make([]float64, 0, len(klines))
	closes := make([]float64, 0, len(klines))
	for _, k := range klines {
		d, err := api.ParseKline(k)
		if err != nil {
			continue
		}
		h, _ := strconv.ParseFloat(d.High, 64)
		l, _ := strconv.ParseFloat(d.Low, 64)
		c, _ := strconv.ParseFloat(d.Close, 64)
		highs = append(highs, h)
		lows = append(lows, l)
		closes = append(closes, c)
	}

	if len(closes) >= atrPeriod+1 {
		s.lastATR = calcATR(highs, lows, closes, atrPeriod)
	}
	if len(closes) >= emaPeriod {
		s.lastEMA = calcEMA(closes, emaPeriod)
	}
	s.lastIndicatorT = time.Now()

	if s.cfg.UseATR && s.lastATR > 0 {
		fmt.Printf("  📊 指标刷新 — ATR(14)=%.4f  EMA(%d)=%.4f\n",
			s.lastATR, emaPeriod, s.lastEMA)
	}
}

// calcATR 使用 Wilder 平滑法计算 ATR
func calcATR(highs, lows, closes []float64, period int) float64 {
	n := len(closes)
	if n < period+1 {
		return 0
	}
	trValues := make([]float64, n-1)
	for i := 1; i < n; i++ {
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		trValues[i-1] = math.Max(hl, math.Max(hc, lc))
	}
	// 初始 ATR = 前 period 个 TR 的均值
	atr := 0.0
	for i := 0; i < period; i++ {
		atr += trValues[i]
	}
	atr /= float64(period)
	// Wilder 平滑
	for i := period; i < len(trValues); i++ {
		atr = (atr*float64(period-1) + trValues[i]) / float64(period)
	}
	return atr
}

// calcEMA 计算指数移动平均
func calcEMA(closes []float64, period int) float64 {
	if len(closes) < period {
		return 0
	}
	k := 2.0 / float64(period+1)
	ema := closes[len(closes)-period]
	for i := len(closes) - period + 1; i < len(closes); i++ {
		ema = closes[i]*k + ema*(1-k)
	}
	return ema
}

// effectiveBidSpread 根据 ATR 和趋势动态调整买单价差
//
//	规则：
//	  1. 基准价差 = cfg.BidSpread
//	  2. ATR 启用时：atrRatio = ATR/price，normalize 到 [min,max] 乘数区间
//	  3. 熊市（price < EMA）时：价差再乘以 trendBearFactor，降低成交概率保护资金
func (s *WaitMarketStrategy) effectiveBidSpread(midPrice float64, sellCount int) float64 {
	spread := s.cfg.BidSpread

	// ── ATR 动态调整 ──────────────────────────────────────
	if s.cfg.UseATR && s.lastATR > 0 && midPrice > 0 {
		atrRatio := s.lastATR / midPrice
		// 以配置 bidSpread 为基准：atrRatio 小则缩小，大则放大
		baseRatio := s.cfg.BidSpread
		multiplier := atrRatio / baseRatio
		multiplier = math.Max(atrMultiplierMin, math.Min(atrMultiplierMax, multiplier))
		spread = s.cfg.BidSpread * multiplier
	}

	// ── 卖单积压加深价差（原逻辑保留）────────────────────
	maxSell := s.cfg.OrderLevels
	if sellCount > maxSell*2 {
		spread += extraBidStep
	}

	// ── 趋势过滤：熊市时放宽价差（买得更低）──────────────
	if s.cfg.UseTrend && s.lastEMA > 0 && midPrice < s.lastEMA {
		spread *= trendBearFactor
		fmt.Printf("  🐻 价格低于 EMA(%.2f)，熊市模式，价差放宽至 %.4f%%\n",
			s.lastEMA, spread*100)
	}

	return spread
}

// ============================================================
// 订单状态同步
// ============================================================

func (s *WaitMarketStrategy) syncOrders() error {
	// ── 优化6：收集需要延迟处理的操作，锁外执行 ──────────
	type pendingSell struct {
		slot      *BuySlot
		sellPrice float64
		sellQty   float64
		delay     time.Duration
	}
	var pendingSells []pendingSell

	s.mu.Lock()

	// ── 买单检查 ──────────────────────────────────────────
	for oid, slot := range s.buySlots {
		status, err := s.queryOrderStatus(s.cfg.Market, oid)
		if err != nil {
			fmt.Printf("  ⚠️  查询买单 %d 状态失败: %v\n", oid, err)
			continue
		}

		switch status {
		case "FILLED":
			fmt.Printf("  ✅ 买单 %d（层%d, 价格%s）已成交\n",
				oid, slot.Layer, s.fmtPrice(slot.BuyPrice))

			sellPrice := s.calcSellPrice(slot.BuyPrice)
			sellQty := s.floorQty(slot.Quantity * (1 - s.cfg.FeeRate))

			// 收集待处理卖单（避免在锁内 sleep）
			slotCopy := *slot
			pendingSells = append(pendingSells, pendingSell{
				slot:      &slotCopy,
				sellPrice: sellPrice,
				sellQty:   sellQty,
				delay:     s.cfg.FilledOrderDelay,
			})

			delete(s.usedLayers, slot.Layer)
			delete(s.buySlots, oid)

		case "CANCELED", "REJECTED", "EXPIRED":
			fmt.Printf("  ⚠️  买单 %d 状态异常: %s，释放层位 %d\n", oid, status, slot.Layer)
			delete(s.usedLayers, slot.Layer)
			delete(s.buySlots, oid)

		case "NEW", "PARTIALLY_FILLED":
			age := time.Since(slot.PlacedAt)
			if age > s.cfg.MaxOrderAge {
				fmt.Printf("  ⏰ 买单 %d 存活 %.0fs 超时，撤单重挂...\n", oid, age.Seconds())
				if err := s.cancelOrder(s.cfg.Market, oid); err != nil {
					fmt.Printf("  ❌ 撤单失败: %v\n", err)
				} else {
					delete(s.usedLayers, slot.Layer)
					delete(s.buySlots, oid)
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
			// ── 优化7：精确计算含费净利润 ─────────────────
			grossProfit := (slot.SellPrice - slot.BuyPrice) * slot.Quantity
			buyFee := slot.BuyPrice * slot.Quantity / (1 - s.cfg.FeeRate) * s.cfg.FeeRate
			sellFee := slot.SellPrice * slot.Quantity * s.cfg.FeeRate
			netProfit := grossProfit - buyFee - sellFee

			s.pnl.TotalProfit += grossProfit
			s.pnl.TotalFees += buyFee + sellFee
			s.pnl.TotalNetProfit += netProfit
			s.pnl.FilledRounds++

			fmt.Printf("  💰 卖单 %d 成交！买 %s → 卖 %s，数量 %s，净利润 %.4f USDT（累计 %.4f USDT）\n",
				oid,
				s.fmtPrice(slot.BuyPrice),
				s.fmtPrice(slot.SellPrice),
				s.fmtQty(slot.Quantity),
				netProfit,
				s.pnl.TotalNetProfit)
			delete(s.sellSlots, oid)

		case "CANCELED", "REJECTED", "EXPIRED":
			// 异常卖单：按原价重新挂出，保证持仓不落空
			fmt.Printf("  ⚠️  卖单 %d 状态异常: %s，重新挂出...\n", oid, status)
			newOID, err := s.placeSellOrder(s.cfg.Market, slot.Quantity, slot.SellPrice)
			if err != nil {
				fmt.Printf("  ❌ 重新挂卖单失败: %v（下次 tick 重试）\n", err)
			} else {
				s.sellSlots[newOID] = &SellSlot{
					BuyOrderID:   slot.BuyOrderID,
					OrderID:      newOID,
					BuyPrice:     slot.BuyPrice,
					SellPrice:    slot.SellPrice,
					Quantity:     slot.Quantity,
					PlacedAt:     time.Now(),
					RepriceCount: slot.RepriceCount,
				}
				delete(s.sellSlots, oid)
			}

		case "NEW":
			// ── 优化8：卖单超时 → 动态重定价，逐步降低清仓 ─
			if s.cfg.MaxSellAge > 0 && time.Since(slot.PlacedAt) > s.cfg.MaxSellAge {
				if s.cfg.MaxReprice > 0 && slot.RepriceCount >= s.cfg.MaxReprice {
					fmt.Printf("  ⚠️  卖单 %d 已重定价 %d 次达上限，保留等待成交\n",
						oid, slot.RepriceCount)
					continue
				}

				newSellPrice := s.calcRepricedSell(slot)
				if newSellPrice <= 0 {
					fmt.Printf("  🔒 卖单 %d 重定价后低于保本线，保留原价\n", oid)
					continue
				}

				fmt.Printf("  📉 卖单 %d 超时（存活%.0fs），重定价 %s → %s（第%d次）\n",
					oid, time.Since(slot.PlacedAt).Seconds(),
					s.fmtPrice(slot.SellPrice), s.fmtPrice(newSellPrice),
					slot.RepriceCount+1)

				if err := s.cancelOrder(s.cfg.Market, oid); err != nil {
					fmt.Printf("  ❌ 撤卖单失败: %v\n", err)
					continue
				}
				newOID, err := s.placeSellOrder(s.cfg.Market, slot.Quantity, newSellPrice)
				if err != nil {
					fmt.Printf("  ❌ 重新挂卖单失败: %v\n", err)
					// 重新挂回原价
					origOID, _ := s.placeSellOrder(s.cfg.Market, slot.Quantity, slot.SellPrice)
					if origOID > 0 {
						s.sellSlots[origOID] = &SellSlot{
							BuyOrderID:   slot.BuyOrderID,
							OrderID:      origOID,
							BuyPrice:     slot.BuyPrice,
							SellPrice:    slot.SellPrice,
							Quantity:     slot.Quantity,
							PlacedAt:     time.Now(),
							RepriceCount: slot.RepriceCount,
						}
					}
				} else {
					s.sellSlots[newOID] = &SellSlot{
						BuyOrderID:   slot.BuyOrderID,
						OrderID:      newOID,
						BuyPrice:     slot.BuyPrice,
						SellPrice:    newSellPrice,
						Quantity:     slot.Quantity,
						PlacedAt:     time.Now(),
						RepriceCount: slot.RepriceCount + 1,
					}
				}
				delete(s.sellSlots, oid)
			}
		}
	}

	s.mu.Unlock()

	// ── 锁外执行延迟挂卖单（避免锁内 sleep 阻塞）─────────
	for _, ps := range pendingSells {
		if ps.delay > 0 {
			time.Sleep(ps.delay)
		}
		s.mu.Lock()
		sellOID, err := s.placeSellOrder(s.cfg.Market, ps.sellQty, ps.sellPrice)
		if err != nil {
			fmt.Printf("  ❌ 挂卖单失败: %v\n", err)
		} else {
			s.sellSlots[sellOID] = &SellSlot{
				BuyOrderID: ps.slot.OrderID,
				OrderID:    sellOID,
				BuyPrice:   ps.slot.BuyPrice,
				SellPrice:  ps.sellPrice,
				Quantity:   ps.sellQty,
				PlacedAt:   time.Now(),
			}
			fmt.Printf("  📤 卖单已挂出 %d，价格 %s，数量 %s\n",
				sellOID, s.fmtPrice(ps.sellPrice), s.fmtQty(ps.sellQty))
		}
		s.mu.Unlock()
	}

	return nil
}

// calcRepricedSell 计算重定价后的卖出价，保证不低于保本线
//
//	保本线 = buyPrice × (1 + 2×feeRate + minSellMargin)
func (s *WaitMarketStrategy) calcRepricedSell(slot *SellSlot) float64 {
	newPrice := ceilToTick(slot.SellPrice*sellRepriceFactor, s.filters.TickSize)
	// 保本线：至少覆盖双边手续费 + 最低利润
	breakeven := ceilToTick(
		slot.BuyPrice*(1+2*s.cfg.FeeRate+minSellMargin),
		s.filters.TickSize,
	)
	if newPrice < breakeven {
		return 0 // 触碰保本线，不再降价
	}
	return newPrice
}

// ============================================================
// 补充买单（含积压保护 + 动态价差）
// ============================================================

func (s *WaitMarketStrategy) refillBuySlots(bidPrice, midPrice float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.buySlots) >= s.cfg.OrderLevels {
		return
	}

	sellCount := len(s.sellSlots)
	maxSell := s.cfg.OrderLevels

	// ── 卖单积压 > 3× → 暂停买入 ─────────────────────────
	if sellCount > maxSell*3 {
		fmt.Printf("  ⏸️  卖单积压 %d（> %d×3），暂停挂买单\n", sellCount, maxSell)
		return
	}

	// ── 优化9：动态价差（ATR + 趋势 + 积压保护）──────────
	bidSpread := s.effectiveBidSpread(midPrice, sellCount)

	for layer := 0; layer < s.cfg.OrderLevels; layer++ {
		if s.usedLayers[layer] {
			continue
		}
		if len(s.buySlots) >= s.cfg.OrderLevels {
			break
		}

		// ── 优化10：以 bid 价为基准，避免立即被市价吃掉 ──
		// 层0 稍低于 bid，层1/2 更深，防止大幅下跌全部成交
		rawPrice := bidPrice * (1 - bidSpread*float64(layer+1))
		buyPrice := s.floorPrice(rawPrice)
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

		fmt.Printf("  📥 层%d 买单 %d 挂出，价格 %s（偏离 bid %.3f%%），数量 %s\n",
			layer, oid, s.fmtPrice(buyPrice),
			bidSpread*float64(layer+1)*100, s.fmtQty(qty))
	}
}

// ============================================================
// 价格 / 数量精度辅助
// ============================================================

// calcSellPrice 含手续费的盈利卖出价（向上对齐 tickSize）
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

// getBidAskPrice 使用 BookTicker 获取最优买卖价
func (s *WaitMarketStrategy) getBidAskPrice() (bid, ask float64, err error) {
	sym := s.cfg.Market
	tickers, err := s.client.GetBookTicker(api.GetBookTickerParams{Symbol: &sym})
	if err != nil {
		return 0, 0, err
	}
	if len(tickers) == 0 {
		return 0, 0, fmt.Errorf("未获取到 %s 的 BookTicker", sym)
	}
	bid, _ = strconv.ParseFloat(tickers[0].BidPrice, 64)
	ask, _ = strconv.ParseFloat(tickers[0].AskPrice, 64)
	return bid, ask, nil
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

	cfg.BidSpread = getFloat(p, "bid_spread", 0.003) // 默认 0.3%（原 0.1% 过小）
	cfg.AskSpread = getFloat(p, "ask_spread", 0.008) // 默认 0.8%（原 1% 可降低成交难度）
	cfg.OrderAmount = getFloat(p, "order_amount", 0.1)

	refreshSec := getInt(p, "order_refresh_time", 30) // 默认 30s（原 60s，更灵敏）
	cfg.OrderRefreshTime = time.Duration(refreshSec) * time.Second
	if cfg.OrderRefreshTime < minPollInterval {
		cfg.OrderRefreshTime = minPollInterval
	}

	maxAgeSec := getInt(p, "max_order_age", 300)
	cfg.MaxOrderAge = time.Duration(maxAgeSec) * time.Second

	// ── 新参数：卖单超时重定价 ────────────────────────────
	maxSellAgeSec := getInt(p, "max_sell_age", 1800) // 默认 30 分钟无成交则重定价
	cfg.MaxSellAge = time.Duration(maxSellAgeSec) * time.Second

	delaySec := getInt(p, "filled_order_delay", 1)
	cfg.FilledOrderDelay = time.Duration(delaySec) * time.Second

	cfg.OrderLevels = getInt(p, "order_levels", 3)
	if cfg.OrderLevels < 1 {
		cfg.OrderLevels = 1
	}

	cfg.MaxReprice = getInt(p, "max_reprice", 5) // 卖单最多重定价 5 次

	// ── 新参数：开关选项 ──────────────────────────────────
	cfg.UseATR = getBool(p, "use_atr", true)     // 默认启用 ATR 动态价差
	cfg.UseTrend = getBool(p, "use_trend", true) // 默认启用趋势过滤

	if v, ok := p["fee_rate"].(float64); ok && v > 0 {
		cfg.FeeRate = v
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

func getBool(p map[string]interface{}, key string, defaultVal bool) bool {
	v, ok := p[key]
	if !ok {
		return defaultVal
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return strings.ToLower(val) == "true" || val == "1"
	}
	return defaultVal
}

// ============================================================
// 状态 / 盈亏打印
// ============================================================

func (s *WaitMarketStrategy) printConfig() {
	c := s.cfg
	f := s.filters
	maxSell := c.OrderLevels
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  交易对:           %s\n", c.Market)
	fmt.Printf("  价格精度:         tickSize=%.8g（%d 位）\n", f.TickSize, f.PricePrecision)
	fmt.Printf("  数量精度:         stepSize=%.8g（%d 位）\n", f.StepSize, f.QtyPrecision)
	fmt.Printf("  基础买单价差:     %.4f%%\n", c.BidSpread*100)
	fmt.Printf("  目标盈利率:       %.4f%%\n", c.AskSpread*100)
	fmt.Printf("  每笔买单数量:     %s\n", s.fmtQty(c.OrderAmount))
	fmt.Printf("  挂单层数:         %d\n", c.OrderLevels)
	fmt.Printf("  加深价差阈值:     卖单 > %d（%d×2）时，买单价差 +%.1f%%\n",
		maxSell*2, maxSell, extraBidStep*100)
	fmt.Printf("  暂停买入阈值:     卖单 > %d（%d×3）\n", maxSell*3, maxSell)
	fmt.Printf("  买单最长存活:     %s\n", c.MaxOrderAge)
	fmt.Printf("  卖单超时重定价:   %s（最多 %d 次，降幅 %.1f%%/次，保本线 +%.1f%%）\n",
		c.MaxSellAge, c.MaxReprice,
		(1-sellRepriceFactor)*100, minSellMargin*100)
	fmt.Printf("  刷新周期:         %s\n", c.OrderRefreshTime)
	fmt.Printf("  ATR 动态价差:     %v  趋势过滤: %v\n", c.UseATR, c.UseTrend)
	fmt.Printf("  手续费率:         %.4f%%\n", c.FeeRate*100)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

func (s *WaitMarketStrategy) printStatus(midPrice float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sellCount := len(s.sellSlots)
	maxSell := s.cfg.OrderLevels

	effectiveSpread := s.effectiveBidSpread(midPrice, sellCount)

	buyState := "正常"
	switch {
	case sellCount > maxSell*3:
		buyState = "⏸️ 暂停（卖单积压 > 3×）"
	case sellCount > maxSell*2:
		buyState = fmt.Sprintf("📉 加深价差（%.4f%%）", effectiveSpread*100)
	case s.cfg.UseTrend && s.lastEMA > 0 && midPrice < s.lastEMA:
		buyState = fmt.Sprintf("🐻 熊市防御（EMA=%.4f）", s.lastEMA)
	}

	fmt.Printf("\n[%s] 中间价: %s  买单: %d/%d  卖单: %d  状态: %s\n",
		time.Now().Format("15:04:05"),
		s.fmtPrice(midPrice),
		len(s.buySlots), s.cfg.OrderLevels,
		sellCount, buyState,
	)

	if len(s.buySlots) > 0 {
		fmt.Println("  📥 买单:")
		for _, slot := range s.buySlots {
			age := time.Since(slot.PlacedAt).Round(time.Second)
			remaining := s.cfg.MaxOrderAge - time.Since(slot.PlacedAt)
			if remaining < 0 {
				remaining = 0
			}
			fmt.Printf("     层%-2d  ID:%-12d  价格:%-12s  存活:%-8s  剩余:%s\n",
				slot.Layer, slot.OrderID,
				s.fmtPrice(slot.BuyPrice), age, remaining.Round(time.Second))
		}
	}

	if sellCount > 0 {
		fmt.Printf("  📤 卖单（%d 个）:\n", sellCount)
		for _, slot := range s.sellSlots {
			age := time.Since(slot.PlacedAt).Round(time.Second)
			repriceInfo := ""
			if slot.RepriceCount > 0 {
				repriceInfo = fmt.Sprintf("  ↘已降价%d次", slot.RepriceCount)
			}
			fmt.Printf("     ID:%-12d  买入:%-12s  卖出:%-12s  数量:%-10s  存活:%s%s\n",
				slot.OrderID,
				s.fmtPrice(slot.BuyPrice),
				s.fmtPrice(slot.SellPrice),
				s.fmtQty(slot.Quantity),
				age, repriceInfo)
		}

		switch {
		case sellCount > maxSell*3:
			fmt.Printf("  🚨 卖单积压严重（%d > %d×3），已暂停挂买单！\n",
				sellCount, maxSell, maxSell*3)
		case sellCount > maxSell*2:
			fmt.Printf("  ⚠️  卖单积压（%d > %d×2），价差加深至 %.4f%%\n",
				sellCount, maxSell*2, effectiveSpread*100)
		}
	}

	// ── 优化11：每 10 轮打印一次盈亏摘要 ─────────────────
	if s.pnl.FilledRounds > 0 && s.pnl.FilledRounds%10 == 0 {
		s.printPnL()
	}
}

func (s *WaitMarketStrategy) printPnL() {
	fmt.Println("\n  ════════════ 盈亏统计 ════════════")
	fmt.Printf("  已完成轮次:  %d\n", s.pnl.FilledRounds)
	fmt.Printf("  毛利润:      %.6f USDT\n", s.pnl.TotalProfit)
	fmt.Printf("  手续费(估):  %.6f USDT\n", s.pnl.TotalFees)
	fmt.Printf("  净利润:      %.6f USDT\n", s.pnl.TotalNetProfit)
	fmt.Println("  ══════════════════════════════════")
}
