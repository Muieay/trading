// Package cmd /*
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
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

// StrategyRunner 策略运行器接口
type StrategyRunner interface {
	Start() error
	Stop() error
	GetStatus() map[string]interface{}
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

// testAPIConnection 测试 API 连接
func testAPIConnection(client *api.BinanceClient) error {
	// 尝试获取账户信息
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

// 参数提取辅助函数
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
