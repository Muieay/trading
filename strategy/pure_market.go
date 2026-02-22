package strategy

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"trading/api"
)

// ============================================================
// 纯做市策略配置
// ============================================================

// PureMarketMakingConfig 纯做市策略配置
type PureMarketMakingConfig struct {
	// 一、基础交易设置
	Market string // 交易对，如 "SOLUSDT"

	// 二、报价与点差设置
	BidSpread     float64 // 买单价差（百分比，如 0.001 = 0.1%）
	AskSpread     float64 // 卖单价差（百分比）
	MinimumSpread float64 // 最小点差保护（百分比）

	// 三、订单刷新与跟价机制
	OrderRefreshTime         time.Duration // 刷新周期
	MaxOrderAge              time.Duration // 订单最长存活时间
	OrderRefreshTolerancePct float64       // 价格变化触发刷新阈值（百分比）
	FilledOrderDelay         time.Duration // 成交后延迟下单

	// 四、订单数量与规模控制
	OrderAmount float64 // 每笔订单数量

	// 五、分层挂单
	OrderLevels      int     // 挂单层数
	OrderLevelSpread float64 // 层级价差间隔（百分比）
	OrderLevelAmount float64 // 每层数量变化（倍数，如 1.0 表示相同）

	// 六、库存管理
	InventorySkewEnabled     bool    // 启用库存平衡
	InventoryTargetBasePct   float64 // 目标基础资产比例（0-1）
	InventoryRangeMultiplier float64 // 库存调整范围
	InventoryPrice           string  // 库存成本定价方式（last/mid/custom）

	// 七、价格区间控制
	PriceFloor             float64 // 价格下限
	PriceCeiling           float64 // 价格上限
	MovingPriceBandEnabled bool    // 动态价格带

	// 八、Ping-Pong 成交模式
	PingPongEnabled bool // 启用 ping-pong

	// 九、订单簿优化
	OrderOptimizationEnabled  bool // 启用最优价跳价
	BidOrderOptimizationDepth int  // 买单优化深度
	AskOrderOptimizationDepth int  // 卖单优化深度

	// 十、Hanging Orders
	HangingOrdersEnabled   bool    // 启用挂单保留
	HangingOrdersCancelPct float64 // 偏离取消阈值（百分比）

	// 十一、手续费与利润保护
	AddTransactionCosts bool // 计入手续费报价

	// 十二、价格来源与定价方式
	PriceSource string // 价格来源（mid/last/best_bid/best_ask/external）
	PriceType   string // 定价类型（mid/last）

	// 十三、成交行为控制
	TakeIfCrossed bool // 价格交叉直接成交

	// 十四、高级订单结构控制
	SplitOrderLevelsEnabled bool // 分层差异化配置

	// 十五、安全与同步机制
	ShouldWaitOrderCancelConfirmation bool // 等待撤单确认
}

// ============================================================
// 纯做市策略执行器
// ============================================================

// PureMarketMakingStrategy 纯做市策略
type PureMarketMakingStrategy struct {
	client *api.BinanceClient
	config PureMarketMakingConfig

	// 运行时状态
	mu              sync.RWMutex
	activeOrders    map[int64]*api.Order // 活跃订单
	lastRefreshTime time.Time
	lastFilledTime  time.Time
	lastMidPrice    float64
	baseBalance     float64
	quoteBalance    float64
	running         bool
	stopChan        chan struct{}

	// 交易规则缓存
	pricePrecision    int // 价格精度
	quantityPrecision int // 数量精度
}

// NewPureMarketMakingStrategy 创建纯做市策略实例
func NewPureMarketMakingStrategy(client *api.BinanceClient, config PureMarketMakingConfig) *PureMarketMakingStrategy {
	// 设置默认值
	if config.OrderRefreshTime == 0 {
		config.OrderRefreshTime = 10 * time.Second
	}
	if config.MaxOrderAge == 0 {
		config.MaxOrderAge = 5 * time.Minute
	}
	if config.OrderRefreshTolerancePct == 0 {
		config.OrderRefreshTolerancePct = 0.001 // 0.1%
	}
	if config.FilledOrderDelay == 0 {
		config.FilledOrderDelay = 1 * time.Second
	}
	if config.OrderLevels == 0 {
		config.OrderLevels = 1
	}
	if config.OrderLevelAmount == 0 {
		config.OrderLevelAmount = 1.0
	}
	if config.InventoryTargetBasePct == 0 {
		config.InventoryTargetBasePct = 0.5
	}
	if config.InventoryRangeMultiplier == 0 {
		config.InventoryRangeMultiplier = 1.0
	}
	if config.PriceSource == "" {
		config.PriceSource = "mid"
	}
	if config.PriceType == "" {
		config.PriceType = "mid"
	}

	return &PureMarketMakingStrategy{
		client:            client,
		config:            config,
		activeOrders:      make(map[int64]*api.Order),
		stopChan:          make(chan struct{}),
		pricePrecision:    2, // 默认价格精度
		quantityPrecision: 2, // 默认数量精度
	}
}

// Start 启动策略
func (s *PureMarketMakingStrategy) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("策略已在运行中")
	}
	s.running = true
	s.mu.Unlock()

	log.Printf("启动纯做市策略: %s", s.config.Market)

	// 检测交易对的精度
	s.detectPrecision()

	// 启动主循环
	go s.mainLoop()

	return nil
}

// Stop 停止策略
func (s *PureMarketMakingStrategy) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("策略未运行")
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopChan)
	log.Printf("停止纯做市策略: %s", s.config.Market)

	// 撤销所有挂单
	return s.cancelAllOrders()
}

// mainLoop 主循环
func (s *PureMarketMakingStrategy) mainLoop() {
	ticker := time.NewTicker(s.config.OrderRefreshTime)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := s.tick(); err != nil {
				log.Printf("策略执行错误: %v", err)
			}
		}
	}
}

// tick 单次执行周期
func (s *PureMarketMakingStrategy) tick() error {
	// 1. 更新账户余额
	if err := s.updateBalances(); err != nil {
		return fmt.Errorf("更新余额失败: %w", err)
	}

	// 2. 更新活跃订单状态
	if err := s.updateActiveOrders(); err != nil {
		return fmt.Errorf("更新订单状态失败: %w", err)
	}

	// 3. 检查是否需要刷新订单
	if s.shouldRefreshOrders() {
		if err := s.refreshOrders(); err != nil {
			return fmt.Errorf("刷新订单失败: %w", err)
		}
	}

	return nil
}

// ============================================================
// 核心逻辑实现
// ============================================================

// updateBalances 更新账户余额
func (s *PureMarketMakingStrategy) updateBalances() error {
	accountInfo, err := s.client.GetAccountInfo(nil)
	if err != nil {
		return err
	}

	// 解析交易对（如 SOLUSDT -> SOL + USDT）
	baseAsset, quoteAsset := s.parseSymbol(s.config.Market)

	for _, balance := range accountInfo.Balances {
		if balance.Asset == baseAsset {
			s.baseBalance, _ = strconv.ParseFloat(balance.Free, 64)
		}
		if balance.Asset == quoteAsset {
			s.quoteBalance, _ = strconv.ParseFloat(balance.Free, 64)
		}
	}

	return nil
}

// updateActiveOrders 更新活跃订单状态
func (s *PureMarketMakingStrategy) updateActiveOrders() error {
	symbol := s.config.Market
	orders, err := s.client.GetOpenOrders(&api.GetOpenOrdersParams{Symbol: &symbol})
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 清空旧的活跃订单
	s.activeOrders = make(map[int64]*api.Order)

	// 更新活跃订单
	for i := range orders {
		s.activeOrders[orders[i].OrderId] = &orders[i]
	}

	return nil
}

// shouldRefreshOrders 判断是否需要刷新订单
func (s *PureMarketMakingStrategy) shouldRefreshOrders() bool {
	// 检查是否超过刷新周期
	if time.Since(s.lastRefreshTime) >= s.config.OrderRefreshTime {
		return true
	}

	// 检查订单是否超过最长存活时间
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, order := range s.activeOrders {
		orderAge := time.Since(time.UnixMilli(order.Time))
		if orderAge >= s.config.MaxOrderAge {
			return true
		}
	}

	// 检查价格变化是否超过阈值
	currentMidPrice, err := s.getMidPrice()
	if err == nil && s.lastMidPrice > 0 {
		priceChange := math.Abs(currentMidPrice-s.lastMidPrice) / s.lastMidPrice
		if priceChange >= s.config.OrderRefreshTolerancePct {
			return true
		}
	}

	// 检查成交后延迟
	if !s.lastFilledTime.IsZero() && time.Since(s.lastFilledTime) < s.config.FilledOrderDelay {
		return false
	}

	return false
}

// refreshOrders 刷新订单
func (s *PureMarketMakingStrategy) refreshOrders() error {
	log.Printf("刷新订单: %s", s.config.Market)

	// 1. 撤销所有现有订单
	if err := s.cancelAllOrders(); err != nil {
		return fmt.Errorf("撤销订单失败: %w", err)
	}

	// 2. 获取当前市场价格
	midPrice, err := s.getMidPrice()
	if err != nil {
		return fmt.Errorf("获取市场价格失败: %w", err)
	}
	s.lastMidPrice = midPrice

	// 3. 检查价格区间限制
	if s.config.PriceFloor > 0 && midPrice < s.config.PriceFloor {
		log.Printf("价格低于下限，暂停下单: %.8f < %.8f", midPrice, s.config.PriceFloor)
		return nil
	}
	if s.config.PriceCeiling > 0 && midPrice > s.config.PriceCeiling {
		log.Printf("价格高于上限，暂停下单: %.8f > %.8f", midPrice, s.config.PriceCeiling)
		return nil
	}

	// 4. 计算库存偏移
	inventorySkew := s.calculateInventorySkew()

	// 5. 获取手续费率
	makerFee := 0.001 // 默认 0.1%
	if s.config.AddTransactionCosts {
		commission, err := s.client.GetAccountCommission(api.GetAccountCommissionParams{
			Symbol: s.config.Market,
		})
		if err == nil {
			makerFee, _ = strconv.ParseFloat(commission.StandardCommission.Maker, 64)
		}
	}

	// 6. 下买单和卖单（分层）
	for level := 0; level < s.config.OrderLevels; level++ {
		// 计算该层的价差和数量
		levelSpread := s.config.OrderLevelSpread * float64(level)
		levelAmount := s.config.OrderAmount * math.Pow(s.config.OrderLevelAmount, float64(level))

		// 买单
		bidSpread := s.config.BidSpread + levelSpread
		if s.config.AddTransactionCosts {
			bidSpread += makerFee
		}
		bidSpread += inventorySkew // 库存调整

		bidPrice := midPrice * (1 - bidSpread)
		if err := s.placeBidOrder(bidPrice, levelAmount, level); err != nil {
			log.Printf("下买单失败 (层级 %d): %v", level, err)
		}

		// 卖单
		askSpread := s.config.AskSpread + levelSpread
		if s.config.AddTransactionCosts {
			askSpread += makerFee
		}
		askSpread -= inventorySkew // 库存调整（反向）

		askPrice := midPrice * (1 + askSpread)
		if err := s.placeAskOrder(askPrice, levelAmount, level); err != nil {
			log.Printf("下卖单失败 (层级 %d): %v", level, err)
		}
	}

	s.lastRefreshTime = time.Now()
	return nil
}

// calculateInventorySkew 计算库存偏移
func (s *PureMarketMakingStrategy) calculateInventorySkew() float64 {
	if !s.config.InventorySkewEnabled {
		return 0
	}

	// 计算当前基础资产比例
	totalValue := s.baseBalance*s.lastMidPrice + s.quoteBalance
	if totalValue == 0 {
		return 0
	}

	currentBasePct := (s.baseBalance * s.lastMidPrice) / totalValue
	targetBasePct := s.config.InventoryTargetBasePct

	// 计算偏移量
	deviation := currentBasePct - targetBasePct
	skew := deviation * s.config.InventoryRangeMultiplier

	// 限制偏移范围
	maxSkew := 0.05 // 最大 5% 偏移
	if skew > maxSkew {
		skew = maxSkew
	} else if skew < -maxSkew {
		skew = -maxSkew
	}

	return skew
}

// placeBidOrder 下买单
func (s *PureMarketMakingStrategy) placeBidOrder(price, quantity float64, level int) error {
	// 订单簿优化：跳价到最优价
	if s.config.OrderOptimizationEnabled && level < s.config.BidOrderOptimizationDepth {
		optimizedPrice, err := s.getOptimizedBidPrice(price)
		if err == nil {
			price = optimizedPrice
		}
	}

	// 格式化价格和数量
	priceStr := s.formatPrice(price)
	qtyStr := s.formatQuantity(quantity)

	// 下单
	timeInForce := "GTC"
	resp, err := s.client.PlaceOrder(api.PlaceOrderParams{
		Symbol:      s.config.Market,
		Side:        "BUY",
		Type:        "LIMIT",
		TimeInForce: &timeInForce,
		Price:       &priceStr,
		Quantity:    &qtyStr,
	})

	if err != nil {
		return err
	}

	log.Printf("买单已下 (层级 %d): 价格=%.8f, 数量=%.8f, OrderID=%d", level, price, quantity, resp.OrderId)
	return nil
}

// placeAskOrder 下卖单
func (s *PureMarketMakingStrategy) placeAskOrder(price, quantity float64, level int) error {
	// 订单簿优化：跳价到最优价
	if s.config.OrderOptimizationEnabled && level < s.config.AskOrderOptimizationDepth {
		optimizedPrice, err := s.getOptimizedAskPrice(price)
		if err == nil {
			price = optimizedPrice
		}
	}

	// 格式化价格和数量
	priceStr := s.formatPrice(price)
	qtyStr := s.formatQuantity(quantity)

	// 下单
	timeInForce := "GTC"
	resp, err := s.client.PlaceOrder(api.PlaceOrderParams{
		Symbol:      s.config.Market,
		Side:        "SELL",
		Type:        "LIMIT",
		TimeInForce: &timeInForce,
		Price:       &priceStr,
		Quantity:    &qtyStr,
	})

	if err != nil {
		return err
	}

	log.Printf("卖单已下 (层级 %d): 价格=%.8f, 数量=%.8f, OrderID=%d", level, price, quantity, resp.OrderId)
	return nil
}

// cancelAllOrders 撤销所有订单
func (s *PureMarketMakingStrategy) cancelAllOrders() error {
	// 先检查是否有活跃订单
	s.mu.RLock()
	hasOrders := len(s.activeOrders) > 0
	s.mu.RUnlock()

	// 如果没有订单，直接返回
	if !hasOrders {
		return nil
	}

	// 撤销所有订单
	_, err := s.client.CancelOpenOrders(api.CancelOpenOrdersParams{
		Symbol: s.config.Market,
	})

	// 忽略"没有订单"的错误
	if err != nil {
		// 检查是否是"没有订单"的错误
		errMsg := err.Error()
		if !strings.Contains(errMsg, "Unknown order") && !strings.Contains(errMsg, "-2011") {
			return err
		}
		// 如果是"没有订单"错误，继续执行
		log.Printf("撤单时没有找到订单，继续执行")
	}

	// 等待撤单确认
	if s.config.ShouldWaitOrderCancelConfirmation {
		time.Sleep(500 * time.Millisecond)
	}

	s.mu.Lock()
	s.activeOrders = make(map[int64]*api.Order)
	s.mu.Unlock()

	return nil
}

// ============================================================
// 辅助函数
// ============================================================

// getMidPrice 获取中间价
func (s *PureMarketMakingStrategy) getMidPrice() (float64, error) {
	switch s.config.PriceSource {
	case "mid":
		return s.getMidPriceFromOrderBook()
	case "last":
		return s.getLastPrice()
	case "best_bid":
		return s.getBestBidPrice()
	case "best_ask":
		return s.getBestAskPrice()
	default:
		return s.getMidPriceFromOrderBook()
	}
}

// getMidPriceFromOrderBook 从订单簿获取中间价
func (s *PureMarketMakingStrategy) getMidPriceFromOrderBook() (float64, error) {
	bookTicker, err := s.client.GetBookTicker(api.GetBookTickerParams{
		Symbol: &s.config.Market,
	})
	if err != nil {
		return 0, err
	}

	if len(bookTicker) == 0 {
		return 0, fmt.Errorf("未获取到订单簿数据")
	}

	bidPrice, _ := strconv.ParseFloat(bookTicker[0].BidPrice, 64)
	askPrice, _ := strconv.ParseFloat(bookTicker[0].AskPrice, 64)

	return (bidPrice + askPrice) / 2, nil
}

// getLastPrice 获取最新成交价
func (s *PureMarketMakingStrategy) getLastPrice() (float64, error) {
	symbol := s.config.Market
	ticker, err := s.client.GetTickerPrice(api.GetTickerPriceParams{
		Symbol: &symbol,
	})
	if err != nil {
		return 0, err
	}

	if len(ticker) == 0 {
		return 0, fmt.Errorf("未获取到价格数据")
	}

	price, _ := strconv.ParseFloat(ticker[0].Price, 64)
	return price, nil
}

// getBestBidPrice 获取最优买价
func (s *PureMarketMakingStrategy) getBestBidPrice() (float64, error) {
	bookTicker, err := s.client.GetBookTicker(api.GetBookTickerParams{
		Symbol: &s.config.Market,
	})
	if err != nil {
		return 0, err
	}

	if len(bookTicker) == 0 {
		return 0, fmt.Errorf("未获取到订单簿数据")
	}

	price, _ := strconv.ParseFloat(bookTicker[0].BidPrice, 64)
	return price, nil
}

// getBestAskPrice 获取最优卖价
func (s *PureMarketMakingStrategy) getBestAskPrice() (float64, error) {
	bookTicker, err := s.client.GetBookTicker(api.GetBookTickerParams{
		Symbol: &s.config.Market,
	})
	if err != nil {
		return 0, err
	}

	if len(bookTicker) == 0 {
		return 0, fmt.Errorf("未获取到订单簿数据")
	}

	price, _ := strconv.ParseFloat(bookTicker[0].AskPrice, 64)
	return price, nil
}

// getOptimizedBidPrice 获取优化后的买价（跳价到最优价前一档）
func (s *PureMarketMakingStrategy) getOptimizedBidPrice(targetPrice float64) (float64, error) {
	limit := 10
	depth, err := s.client.GetDepth(api.GetDepthParams{
		Symbol: s.config.Market,
		Limit:  &limit,
	})
	if err != nil {
		return targetPrice, err
	}

	if len(depth.Bids) == 0 {
		return targetPrice, nil
	}

	// 获取最优买价
	bestBidPrice, _ := strconv.ParseFloat(depth.Bids[0][0], 64)

	// 如果目标价格高于最优买价，则跳价到最优买价 + 一个最小价格单位
	if targetPrice >= bestBidPrice {
		tickSize := s.getTickSize()
		return bestBidPrice + tickSize, nil
	}

	return targetPrice, nil
}

// getOptimizedAskPrice 获取优化后的卖价（跳价到最优价前一档）
func (s *PureMarketMakingStrategy) getOptimizedAskPrice(targetPrice float64) (float64, error) {
	limit := 10
	depth, err := s.client.GetDepth(api.GetDepthParams{
		Symbol: s.config.Market,
		Limit:  &limit,
	})
	if err != nil {
		return targetPrice, err
	}

	if len(depth.Asks) == 0 {
		return targetPrice, nil
	}

	// 获取最优卖价
	bestAskPrice, _ := strconv.ParseFloat(depth.Asks[0][0], 64)

	// 如果目标价格低于最优卖价，则跳价到最优卖价 - 一个最小价格单位
	if targetPrice <= bestAskPrice {
		tickSize := s.getTickSize()
		return bestAskPrice - tickSize, nil
	}

	return targetPrice, nil
}

// getTickSize 获取最小价格变动单位
func (s *PureMarketMakingStrategy) getTickSize() float64 {
	// 简化实现，实际应从交易规则中获取
	if s.lastMidPrice > 1000 {
		return 0.1
	} else if s.lastMidPrice > 100 {
		return 0.01
	} else if s.lastMidPrice > 1 {
		return 0.001
	}
	return 0.0001
}

// parseSymbol 解析交易对
func (s *PureMarketMakingStrategy) parseSymbol(symbol string) (baseAsset, quoteAsset string) {
	// 简化实现：假设 USDT/BUSD/USDC 为计价货币
	for _, quote := range []string{"USDT", "BUSD", "USDC", "BTC", "ETH", "BNB"} {
		if len(symbol) > len(quote) && symbol[len(symbol)-len(quote):] == quote {
			return symbol[:len(symbol)-len(quote)], quote
		}
	}
	// 默认返回
	return symbol[:3], symbol[3:]
}

// formatPrice 格式化价格
func (s *PureMarketMakingStrategy) formatPrice(price float64) string {
	// 使用检测到的精度
	format := fmt.Sprintf("%%.%df", s.pricePrecision)
	return fmt.Sprintf(format, price)
}

// formatQuantity 格式化数量
func (s *PureMarketMakingStrategy) formatQuantity(quantity float64) string {
	// 使用检测到的精度
	format := fmt.Sprintf("%%.%df", s.quantityPrecision)
	return fmt.Sprintf(format, quantity)
}

// detectPrecision 检测交易对的价格和数量精度
func (s *PureMarketMakingStrategy) detectPrecision() {
	// 获取当前价格作为参考
	midPrice, err := s.getMidPrice()
	if err != nil {
		log.Printf("⚠️  无法获取价格，使用默认精度")
		return
	}

	// 特定交易对的精度配置（基于 Binance 实际规则）
	symbolPrecision := map[string]struct{ price, quantity int }{
		"BTCUSDT":   {2, 5},
		"ETHUSDT":   {2, 4},
		"BNBUSDT":   {2, 2},
		"SOLUSDT":   {2, 1}, // SOL/USDT: 价格 2 位，数量 1 位
		"ADAUSDT":   {4, 0},
		"DOGEUSDT":  {5, 0},
		"XRPUSDT":   {4, 0},
		"DOTUSDT":   {3, 1},
		"MATICUSDT": {4, 0},
		"LTCUSDT":   {2, 3},
	}

	// 检查是否有预定义的精度
	if precision, ok := symbolPrecision[s.config.Market]; ok {
		s.pricePrecision = precision.price
		s.quantityPrecision = precision.quantity
		log.Printf("✅ 使用预定义精度 [%s] - 价格: %d 位, 数量: %d 位",
			s.config.Market, s.pricePrecision, s.quantityPrecision)
		return
	}

	// 如果没有预定义，根据价格范围自动检测
	if midPrice >= 1000 {
		s.pricePrecision = 2
		s.quantityPrecision = 3
	} else if midPrice >= 100 {
		s.pricePrecision = 2
		s.quantityPrecision = 2
	} else if midPrice >= 10 {
		s.pricePrecision = 2
		s.quantityPrecision = 2
	} else if midPrice >= 1 {
		s.pricePrecision = 2
		s.quantityPrecision = 1
	} else if midPrice >= 0.1 {
		s.pricePrecision = 4
		s.quantityPrecision = 1
	} else if midPrice >= 0.01 {
		s.pricePrecision = 5
		s.quantityPrecision = 0
	} else {
		s.pricePrecision = 6
		s.quantityPrecision = 0
	}

	log.Printf("✅ 自动检测精度 [%s, 价格: %.2f] - 价格: %d 位, 数量: %d 位",
		s.config.Market, midPrice, s.pricePrecision, s.quantityPrecision)
}

// ============================================================
// 状态查询
// ============================================================

// GetStatus 获取策略状态
func (s *PureMarketMakingStrategy) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"running":        s.running,
		"market":         s.config.Market,
		"active_orders":  len(s.activeOrders),
		"base_balance":   s.baseBalance,
		"quote_balance":  s.quoteBalance,
		"last_mid_price": s.lastMidPrice,
		"last_refresh":   s.lastRefreshTime,
		"inventory_skew": s.calculateInventorySkew(),
	}
}

// ============================================================
// 示例配置
// ============================================================

// DefaultConfig 返回默认配置示例
func DefaultConfig() PureMarketMakingConfig {
	return PureMarketMakingConfig{
		// 一、基础交易设置
		Market: "SOLUSDT",

		// 二、报价与点差设置
		BidSpread:     0.001,  // 0.1%
		AskSpread:     0.001,  // 0.1%
		MinimumSpread: 0.0005, // 0.05%

		// 三、订单刷新与跟价机制
		OrderRefreshTime:         10 * time.Second,
		MaxOrderAge:              5 * time.Minute,
		OrderRefreshTolerancePct: 0.001, // 0.1%
		FilledOrderDelay:         1 * time.Second,

		// 四、订单数量与规模控制
		OrderAmount: 0.1, // 每笔 0.1 SOL

		// 五、分层挂单
		OrderLevels:      3,     // 3 层
		OrderLevelSpread: 0.001, // 每层额外 0.1%
		OrderLevelAmount: 1.5,   // 每层数量增加 1.5 倍

		// 六、库存管理
		InventorySkewEnabled:     true,
		InventoryTargetBasePct:   0.5, // 目标 50% 基础资产
		InventoryRangeMultiplier: 2.0, // 调整倍数
		InventoryPrice:           "mid",

		// 七、价格区间控制
		PriceFloor:             0, // 无下限
		PriceCeiling:           0, // 无上限
		MovingPriceBandEnabled: false,

		// 八、Ping-Pong 成交模式
		PingPongEnabled: false,

		// 九、订单簿优化
		OrderOptimizationEnabled:  true,
		BidOrderOptimizationDepth: 1, // 仅优化第一层
		AskOrderOptimizationDepth: 1,

		// 十、Hanging Orders
		HangingOrdersEnabled:   false,
		HangingOrdersCancelPct: 0.02, // 2%

		// 十一、手续费与利润保护
		AddTransactionCosts: true,

		// 十二、价格来源与定价方式
		PriceSource: "mid",
		PriceType:   "mid",

		// 十三、成交行为控制
		TakeIfCrossed: false,

		// 十四、高级订单结构控制
		SplitOrderLevelsEnabled: false,

		// 十五、安全与同步机制
		ShouldWaitOrderCancelConfirmation: true,
	}
}
