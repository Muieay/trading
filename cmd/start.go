// Package cmd /*
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"trading/api"
	"trading/config"
	"trading/strategy"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动交易策略",
	Long: `启动已配置的交易策略。

在启动策略之前，请确保：
  1. 已通过 'bt connect' 配置 Binance API 连接
  2. 已通过 'bt config' 配置策略参数

策略将持续运行直到手动停止 (Ctrl+C)。`,
	Run: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

// runStart 执行策略启动逻辑
func runStart(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 启动交易策略")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 1. 加载 API 配置
	fmt.Println("\n📡 加载 API 配置...")
	apiConfig, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ 加载 API 配置失败: %v\n", err)
		fmt.Println("💡 提示：请先运行 'bt connect' 命令配置 API 连接")
		return
	}
	fmt.Printf("✅ API 配置加载成功 (网络: %s)\n", apiConfig.Network)

	// 2. 加载策略配置
	fmt.Println("\n⚙️  加载策略配置...")
	strategyConfig, err := loadStrategyConfig()
	if err != nil {
		fmt.Printf("❌ 加载策略配置失败: %v\n", err)
		fmt.Println("💡 提示：请先运行 'bt config' 命令配置策略参数")
		return
	}
	fmt.Printf("✅ 策略配置加载成功 (类型: %s)\n", getStrategyDisplayName(strategyConfig.Type))

	// 3. 创建 API 客户端
	fmt.Println("\n🔗 连接 Binance API...")
	client := api.NewBinanceClient(apiConfig.APIKey, apiConfig.SecretKey)

	// 测试连接
	if err := testAPIConnection(client); err != nil {
		fmt.Printf("❌ API 连接测试失败: %v\n", err)
		return
	}
	fmt.Println("✅ API 连接测试成功")

	// 4. 创建并启动策略
	fmt.Println("\n🎯 初始化策略...")
	var strategyRunner StrategyRunner
	switch strategyConfig.Type {
	case StrategyPureMarketMaking:
		strategyRunner, err = createPureMarketMakingStrategy(client, strategyConfig.Params)
	case StrategyWaitMarketMaking:
		strategyRunner, err = createWaitMarketMakingStrategy(client, strategyConfig.Params)
	default:
		err = fmt.Errorf("不支持的策略类型: %s", strategyConfig.Type)
	}

	if err != nil {
		fmt.Printf("❌ 初始化策略失败: %v\n", err)
		return
	}

	// 5. 启动策略
	fmt.Println("\n▶️  启动策略...")
	if err := strategyRunner.Start(); err != nil {
		fmt.Printf("❌ 启动策略失败: %v\n", err)
		return
	}

	fmt.Println("✅ 策略已启动")
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 策略运行中...")
	fmt.Println("💡 按 Ctrl+C 停止策略")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 6. 定期打印状态
	go printStatusPeriodically(strategyRunner)

	// 7. 等待中断信号
	waitForInterrupt(strategyRunner)
}

// ============================================================
// StrategyRunner 接口
// ============================================================

// StrategyRunner 策略运行器接口
type StrategyRunner interface {
	Start() error
	Stop() error
	GetStatus() map[string]interface{}
}

// ============================================================
// WaitMarketRunner —— WaitMarketStrategy 的 StrategyRunner 适配器
// ============================================================

// WaitMarketRunner 将 strategy.WaitMarketStrategy 包装为 StrategyRunner。
// WaitMarketStrategy.Run(ctx) 是阻塞循环，适配器在独立 goroutine 中运行它，
// 通过 context 取消实现优雅退出。
type WaitMarketRunner struct {
	inner  *strategy.WaitMarketStrategy
	params map[string]interface{} // 保存原始参数，用于 GetStatus

	mu      sync.RWMutex
	running bool
	cancel  context.CancelFunc
	done    chan struct{} // Run 返回后关闭
	runErr  error         // Run 返回的错误（如有）
}

// Start 在后台 goroutine 中启动策略主循环。
func (r *WaitMarketRunner) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("策略已在运行中")
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.done = make(chan struct{})
	r.running = true

	go func() {
		defer close(r.done)
		if err := r.inner.Run(ctx); err != nil {
			r.mu.Lock()
			r.runErr = err
			r.mu.Unlock()
			fmt.Printf("❌ 策略运行出错: %v\n", err)
		}
		r.mu.Lock()
		r.running = false
		r.mu.Unlock()
	}()

	return nil
}

// Stop 发出取消信号并等待策略主循环退出（最多 30 秒）。
func (r *WaitMarketRunner) Stop() error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	cancel := r.cancel
	done := r.done
	r.mu.Unlock()

	cancel() // 触发 ctx.Done()

	// 等待 Run goroutine 退出
	select {
	case <-done:
		fmt.Println("✅ 挂单做市策略已停止")
	case <-time.After(30 * time.Second):
		fmt.Println("⚠️  等待策略退出超时（30s），强制终止")
	}

	r.mu.RLock()
	err := r.runErr
	r.mu.RUnlock()
	return err
}

// GetStatus 返回策略当前的运行快照，供状态打印使用。
func (r *WaitMarketRunner) GetStatus() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := map[string]interface{}{
		"running":  r.running,
		"strategy": "wait_market_making",
		"market":   getStringParam(r.params, "market", ""),
	}
	return status
}

// ============================================================
// 工厂函数
// ============================================================

// createWaitMarketMakingStrategy 解析参数、构造 WaitMarketStrategy 并包装为 StrategyRunner。
func createWaitMarketMakingStrategy(client *api.BinanceClient, params map[string]interface{}) (StrategyRunner, error) {
	inner, err := strategy.NewWaitMarketStrategy(client, params)
	if err != nil {
		return nil, fmt.Errorf("创建挂单做市策略失败: %w", err)
	}

	// 打印配置摘要
	fmt.Println("\n📋 策略配置摘要:")
	fmt.Printf("  • 交易对:         %s\n", getStringParam(params, "market", ""))
	fmt.Printf("  • 每层买单价差:   %.4f%%\n", getFloatParam(params, "bid_spread", 0.001)*100)
	fmt.Printf("  • 目标盈利率:     %.4f%%\n", getFloatParam(params, "ask_spread", 0.01)*100)
	fmt.Printf("  • 每笔买单数量:   %.6f\n", getFloatParam(params, "order_amount", 0.1))
	fmt.Printf("  • 挂单层数:       %d\n", getIntParam(params, "order_levels", 3))
	fmt.Printf("  • 刷新周期:       %v\n", getDurationParam(params, "order_refresh_time", 60))
	fmt.Printf("  • 买单最长存活:   %v\n", getDurationParam(params, "max_order_age", 300))

	return &WaitMarketRunner{
		inner:  inner,
		params: params,
	}, nil
}

// createPureMarketMakingStrategy 创建纯做市策略
func createPureMarketMakingStrategy(client *api.BinanceClient, params map[string]interface{}) (StrategyRunner, error) {
	// 构建策略配置
	config := strategy.PureMarketMakingConfig{
		Market:                            getStringParam(params, "market", "SOLUSDT"),
		BidSpread:                         getFloatParam(params, "bid_spread", 0.001),
		AskSpread:                         getFloatParam(params, "ask_spread", 0.001),
		MinimumSpread:                     getFloatParam(params, "minimum_spread", 0.0005),
		OrderRefreshTime:                  getDurationParam(params, "order_refresh_time", 10),
		MaxOrderAge:                       getDurationParam(params, "max_order_age", 300),
		OrderRefreshTolerancePct:          getFloatParam(params, "order_refresh_tolerance_pct", 0.001),
		FilledOrderDelay:                  getDurationParam(params, "filled_order_delay", 1),
		OrderAmount:                       getFloatParam(params, "order_amount", 0.1),
		OrderLevels:                       getIntParam(params, "order_levels", 3),
		OrderLevelSpread:                  getFloatParam(params, "order_level_spread", 0.001),
		OrderLevelAmount:                  getFloatParam(params, "order_level_amount", 1.5),
		InventorySkewEnabled:              getBoolParam(params, "inventory_skew_enabled", true),
		InventoryTargetBasePct:            getFloatParam(params, "inventory_target_base_pct", 0.5),
		InventoryRangeMultiplier:          getFloatParam(params, "inventory_range_multiplier", 2.0),
		InventoryPrice:                    getStringParam(params, "inventory_price", "mid"),
		PriceFloor:                        getFloatParam(params, "price_floor", 0),
		PriceCeiling:                      getFloatParam(params, "price_ceiling", 0),
		MovingPriceBandEnabled:            getBoolParam(params, "moving_price_band_enabled", false),
		PingPongEnabled:                   getBoolParam(params, "ping_pong_enabled", false),
		OrderOptimizationEnabled:          getBoolParam(params, "order_optimization_enabled", true),
		BidOrderOptimizationDepth:         getIntParam(params, "bid_order_optimization_depth", 1),
		AskOrderOptimizationDepth:         getIntParam(params, "ask_order_optimization_depth", 1),
		HangingOrdersEnabled:              getBoolParam(params, "hanging_orders_enabled", false),
		HangingOrdersCancelPct:            getFloatParam(params, "hanging_orders_cancel_pct", 0.02),
		AddTransactionCosts:               getBoolParam(params, "add_transaction_costs", true),
		PriceSource:                       getStringParam(params, "price_source", "mid"),
		PriceType:                         getStringParam(params, "price_type", "mid"),
		TakeIfCrossed:                     getBoolParam(params, "take_if_crossed", false),
		SplitOrderLevelsEnabled:           getBoolParam(params, "split_order_levels_enabled", false),
		ShouldWaitOrderCancelConfirmation: getBoolParam(params, "should_wait_order_cancel_confirmation", true),
	}

	// 打印配置摘要
	fmt.Println("\n📋 策略配置摘要:")
	fmt.Printf("  • 交易对: %s\n", config.Market)
	fmt.Printf("  • 买单价差: %.2f%%\n", config.BidSpread*100)
	fmt.Printf("  • 卖单价差: %.2f%%\n", config.AskSpread*100)
	fmt.Printf("  • 订单数量: %.4f\n", config.OrderAmount)
	fmt.Printf("  • 挂单层数: %d\n", config.OrderLevels)
	fmt.Printf("  • 刷新周期: %v\n", config.OrderRefreshTime)
	fmt.Printf("  • 库存管理: %v\n", config.InventorySkewEnabled)
	fmt.Printf("  • 订单簿优化: %v\n", config.OrderOptimizationEnabled)

	return strategy.NewPureMarketMakingStrategy(client, config), nil
}

// ============================================================
// 辅助函数（原有）
// ============================================================

// testAPIConnection 测试 API 连接
func testAPIConnection(client *api.BinanceClient) error {
	_, err := client.GetAccountInfo(nil)
	return err
}

// printStatusPeriodically 定期打印策略状态
func printStatusPeriodically(runner StrategyRunner) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		status := runner.GetStatus()
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("📊 策略状态 [%s]\n", time.Now().Format("15:04:05"))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if running, ok := status["running"].(bool); ok {
			fmt.Printf("  运行状态: %v\n", running)
		}
		if market, ok := status["market"].(string); ok {
			fmt.Printf("  交易对: %s\n", market)
		}
		if activeOrders, ok := status["active_orders"].(int); ok {
			fmt.Printf("  活跃订单: %d\n", activeOrders)
		}
		if baseBalance, ok := status["base_balance"].(float64); ok {
			fmt.Printf("  基础资产余额: %.8f\n", baseBalance)
		}
		if quoteBalance, ok := status["quote_balance"].(float64); ok {
			fmt.Printf("  计价资产余额: %.8f\n", quoteBalance)
		}
		if lastMidPrice, ok := status["last_mid_price"].(float64); ok {
			fmt.Printf("  最新中间价: %.8f\n", lastMidPrice)
		}
		if inventorySkew, ok := status["inventory_skew"].(float64); ok {
			fmt.Printf("  库存偏移: %.4f%%\n", inventorySkew*100)
		}

		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	}
}

// waitForInterrupt 等待中断信号
func waitForInterrupt(runner StrategyRunner) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	fmt.Println("\n\n⏸️  收到停止信号，正在停止策略...")
	if err := runner.Stop(); err != nil {
		log.Printf("❌ 停止策略失败: %v", err)
	} else {
		fmt.Println("✅ 策略已停止")
	}

	fmt.Println("👋 再见！")
	os.Exit(0)
}

// ============================================================
// 参数提取辅助函数（原有）
// ============================================================

func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return defaultValue
}

func getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if v, ok := params[key].(float64); ok {
		return int(v)
	}
	if v, ok := params[key].(int); ok {
		return v
	}
	return defaultValue
}

func getFloatParam(params map[string]interface{}, key string, defaultValue float64) float64 {
	if v, ok := params[key].(float64); ok {
		return v
	}
	if v, ok := params[key].(int); ok {
		return float64(v)
	}
	return defaultValue
}

func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := params[key].(bool); ok {
		return v
	}
	return defaultValue
}

func getDurationParam(params map[string]interface{}, key string, defaultSeconds int) time.Duration {
	if v, ok := params[key].(float64); ok {
		return time.Duration(v) * time.Second
	}
	if v, ok := params[key].(int); ok {
		return time.Duration(v) * time.Second
	}
	return time.Duration(defaultSeconds) * time.Second
}
