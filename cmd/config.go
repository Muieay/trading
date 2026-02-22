// Package cmd /*
package cmd

/*
策略配置命令
*/
import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"trading/config"

	"github.com/spf13/cobra"
)

const (
	StrategyConfigFile = "strategy.json"
	BackupFilePrefix   = "strategy."
	BackupFileSuffix   = ".json"
	MaxBackupFiles     = 10 // 保留最近5个备份
)

// StrategyType 策略类型
type StrategyType string

const (
	StrategyPureMarketMaking StrategyType = "pure_market_making"
)

const (
	StrategyWaitMarketMaking StrategyType = "wait_market_making"
)

// StrategyConfig 策略配置
type StrategyConfig struct {
	Type   StrategyType           `json:"type"`
	Params map[string]interface{} `json:"params"`
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置交易策略参数",
	Long: `交互式配置交易策略参数。

支持的策略类型：
  • 纯做市策略 (Pure Market Making) - 双边挂单做市

配置将保存到本地文件，可通过 'bt start' 命令启动策略。`,
	Run: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

// runConfig 执行策略配置逻辑
func runConfig(cmd *cobra.Command, args []string) {
	fmt.Println("⚙️  交易策略配置")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 检查是否已存在配置
	if strategyConfigExists() {
		fmt.Print("⚠️  检测到已有策略配置，是否重新配置？(y/N): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input != "y" && input != "yes" {
			fmt.Println("❌ 已取消配置")
			return
		}
		fmt.Println()
	}

	// 选择策略类型
	strategyType := selectStrategyType()
	fmt.Printf("✅ 已选择策略: %s\n\n", getStrategyDisplayName(strategyType))

	// 配置策略参数
	var params map[string]interface{}
	switch strategyType {
	case StrategyPureMarketMaking:
		params = configurePureMarketMaking()
	case StrategyWaitMarketMaking:
		params = configureWaitMarketMaking()
	default:
		fmt.Printf("❌ 不支持的策略类型: %s\n", strategyType)
		return
	}

	// 保存配置
	config := StrategyConfig{
		Type:   strategyType,
		Params: params,
	}

	fmt.Println("\n💾 正在保存策略配置...")
	if err := saveStrategyConfig(config); err != nil {
		fmt.Printf("❌ 保存配置失败: %v\n", err)
		return
	}

	fmt.Println("✅ 策略配置保存成功！")
	fmt.Printf("📁 配置文件位置: %s/%s\n", getConfigDir(), StrategyConfigFile)
	fmt.Println("\n💡 提示：使用 'bt start' 命令启动策略")
}

// selectStrategyType 选择策略类型
func selectStrategyType() StrategyType {
	fmt.Println("请选择策略类型：")
	fmt.Println("  1. 纯做市策略 (Pure Market Making)")
	fmt.Println("     双边挂单做市，支持分层、库存管理、订单簿优化等高级功能")
	fmt.Println("  2. 时间市场策略 (Wait Market Making)")
	fmt.Println("     双边挂单做市，挂长线钓小鱼")
	fmt.Print("\n请输入选项 (1-2): ")

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1", "":
			return StrategyPureMarketMaking
		case "2":
			return StrategyWaitMarketMaking
		default:
			fmt.Print("❌ 无效选项 ")
		}
	}
}

// getStrategyDisplayName 获取策略显示名称
func getStrategyDisplayName(strategyType StrategyType) string {
	switch strategyType {
	case StrategyPureMarketMaking:
		return "纯做市策略 (Pure Market Making)"
	case StrategyWaitMarketMaking:
		return "时间市场策略 (Wait Market Making)"
	default:
		return string(strategyType)
	}
}

// configurePureMarketMaking 配置纯做市策略
func configurePureMarketMaking() map[string]interface{} {
	reader := bufio.NewReader(os.Stdin)
	params := make(map[string]interface{})

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 纯做市策略参数配置")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 一、基础交易设置
	fmt.Println("\n【一、基础交易设置】")
	params["market"] = config.InputString(reader, "交易对 (如 SOLUSDT)", "SOLUSDT")

	// 二、报价与点差设置
	fmt.Println("\n【二、报价与点差设置】")
	params["bid_spread"] = config.InputFloat(reader, "买单价差 (百分比，如 0.1 表示 0.1%)", 0.1) / 100
	params["ask_spread"] = config.InputFloat(reader, "卖单价差 (百分比)", 0.1) / 100
	params["minimum_spread"] = config.InputFloat(reader, "最小点差保护 (百分比)", 0.05) / 100

	// 三、订单刷新与跟价机制
	fmt.Println("\n【三、订单刷新与跟价机制】")
	params["order_refresh_time"] = config.InputInt(reader, "刷新周期 (秒)", 10)
	params["max_order_age"] = config.InputInt(reader, "订单最长存活时间 (秒)", 300)
	params["order_refresh_tolerance_pct"] = config.InputFloat(reader, "价格变化触发刷新阈值 (百分比)", 0.1) / 100
	params["filled_order_delay"] = config.InputInt(reader, "成交后延迟下单 (秒)", 1)

	// 四、订单数量与规模控制
	fmt.Println("\n【四、订单数量与规模控制】")
	params["order_amount"] = config.InputFloat(reader, "每笔订单数量 (基础资产)", 0.1)

	// 五、分层挂单
	fmt.Println("\n【五、分层挂单 (Ladder Orders)】")
	params["order_levels"] = config.InputInt(reader, "挂单层数", 3)
	params["order_level_spread"] = config.InputFloat(reader, "层级价差间隔 (百分比)", 0.1) / 100
	params["order_level_amount"] = config.InputFloat(reader, "每层数量变化倍数 (如 1.5)", 1.5)

	// 六、库存管理
	fmt.Println("\n【六、库存管理 (Inventory Skew)】")
	params["inventory_skew_enabled"] = config.InputBool(reader, "启用库存平衡", true)
	if params["inventory_skew_enabled"].(bool) {
		params["inventory_target_base_pct"] = config.InputFloat(reader, "目标基础资产比例 (0-100)", 50) / 100
		params["inventory_range_multiplier"] = config.InputFloat(reader, "库存调整范围倍数", 2.0)
		params["inventory_price"] = config.InputString(reader, "库存成本定价 (mid/last)", "mid")
	}

	// 七、价格区间控制
	fmt.Println("\n【七、价格区间控制 (市场保护)】")
	params["price_floor"] = config.InputFloat(reader, "价格下限 (0 表示无限制)", 0)
	params["price_ceiling"] = config.InputFloat(reader, "价格上限 (0 表示无限制)", 0)
	params["moving_price_band_enabled"] = config.InputBool(reader, "启用动态价格带", false)

	// 八、Ping-Pong 成交模式
	fmt.Println("\n【八、Ping-Pong 成交模式】")
	params["ping_pong_enabled"] = config.InputBool(reader, "启用 Ping-Pong 模式", false)

	// 九、订单簿优化
	fmt.Println("\n【九、订单簿优化 (提高成交率)】")
	params["order_optimization_enabled"] = config.InputBool(reader, "启用最优价跳价", true)
	if params["order_optimization_enabled"].(bool) {
		params["bid_order_optimization_depth"] = config.InputInt(reader, "买单优化深度 (层数)", 1)
		params["ask_order_optimization_depth"] = config.InputInt(reader, "卖单优化深度 (层数)", 1)
	}

	// 十、Hanging Orders
	fmt.Println("\n【十、Hanging Orders (保留挂单策略)】")
	params["hanging_orders_enabled"] = config.InputBool(reader, "启用挂单保留", false)
	if params["hanging_orders_enabled"].(bool) {
		params["hanging_orders_cancel_pct"] = config.InputFloat(reader, "偏离取消阈值 (百分比)", 2.0) / 100
	}

	// 十一、手续费与利润保护
	fmt.Println("\n【十一、手续费与利润保护】")
	params["add_transaction_costs"] = config.InputBool(reader, "计入手续费报价", true)

	// 十二、价格来源与定价方式
	fmt.Println("\n【十二、价格来源与定价方式】")
	params["price_source"] = config.InputString(reader, "价格来源 (mid/last/best_bid/best_ask)", "mid")
	params["price_type"] = config.InputString(reader, "定价类型 (mid/last)", "mid")

	// 十三、成交行为控制
	fmt.Println("\n【十三、成交行为控制】")
	params["take_if_crossed"] = config.InputBool(reader, "价格交叉直接成交", false)

	// 十四、高级订单结构控制
	fmt.Println("\n【十四、高级订单结构控制】")
	params["split_order_levels_enabled"] = config.InputBool(reader, "分层差异化配置", false)

	// 十五、安全与同步机制
	fmt.Println("\n【十五、安全与同步机制】")
	params["should_wait_order_cancel_confirmation"] = config.InputBool(reader, "等待撤单确认", true)

	return params
}

// configureWaitMarketMaking 时间市场策略
func configureWaitMarketMaking() map[string]interface{} {
	reader := bufio.NewReader(os.Stdin)
	params := make(map[string]interface{})

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 时间做市策略参数配置")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 一、基础交易设置
	fmt.Println("\n【一、基础交易设置】")
	params["market"] = config.InputString(reader, "交易对 (如 SOLUSDT)", "SOLUSDT")

	// 二、报价与点差设置
	fmt.Println("\n【二、报价与点差设置】")
	params["bid_spread"] = config.InputFloat(reader, "买单价差 (百分比，如 0.1 表示 0.1%)", 0.1) / 100
	params["ask_spread"] = config.InputFloat(reader, "卖单价差 (百分比)", 0.1) / 100

	// 三、订单刷新与跟价机制
	fmt.Println("\n【三、订单刷新与跟价机制】")
	params["order_refresh_time"] = config.InputInt(reader, "刷新周期 (秒)", 60)
	params["max_order_age"] = config.InputInt(reader, "订单最长存活时间 (秒)", 300)
	params["filled_order_delay"] = config.InputInt(reader, "成交后延迟下单 (秒)", 1)

	// 四、订单数量与规模控制
	fmt.Println("\n【四、订单数量与规模控制】")
	params["order_amount"] = config.InputFloat(reader, "每笔订单数量 (基础资产)", 0.1)

	// 五、分层挂单
	fmt.Println("\n【五、分层挂单 (Ladder Orders)】")
	params["order_levels"] = config.InputInt(reader, "挂单层数", 3)

	return params
}

// saveStrategyConfig 保存策略配置，支持多版本备份
func saveStrategyConfig(config StrategyConfig) error {
	configDir := getConfigDir()

	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	configPath := filepath.Join(configDir, StrategyConfigFile)

	// 如果配置文件已存在，先创建带时间戳的备份
	if fileExists(configPath) {
		if err := backupStrategyConfigWithTimestamp(); err != nil {
			return fmt.Errorf("备份原策略文件失败: %w", err)
		}
		// 清理旧备份，只保留最近 MaxBackupFiles 个
		if err := cleanOldBackups(configDir); err != nil {
			// 清理失败只记录日志，不影响主流程
			log.Printf("警告: 清理旧备份失败: %v", err)
		}
	}

	// 保存新配置文件
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	// 设置配置文件权限为600
	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("设置配置文件权限失败: %w", err)
	}

	return nil
}

// backupStrategyConfigWithTimestamp 创建带时间戳的备份文件
func backupStrategyConfigWithTimestamp() error {
	configDir := getConfigDir()
	sourcePath := filepath.Join(configDir, StrategyConfigFile)

	// 生成时间戳，格式：YYYYMMDD_HHMMSS
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("%s%s%s", BackupFilePrefix, timestamp, BackupFileSuffix)
	backupPath := filepath.Join(configDir, backupFilename)

	// 读取原文件内容
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("读取原配置文件失败: %w", err)
	}

	// 写入备份文件，权限设为0600
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return fmt.Errorf("写入备份文件失败: %w", err)
	}

	return nil
}

// cleanOldBackups 清理旧备份，只保留最近的 MaxBackupFiles 个
func cleanOldBackups(configDir string) error {
	// 获取所有备份文件
	pattern := filepath.Join(configDir, BackupFilePrefix+"*"+BackupFileSuffix)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	if len(matches) <= MaxBackupFiles {
		return nil // 不需要清理
	}

	// 按修改时间排序（从旧到新）
	sort.Slice(matches, func(i, j int) bool {
		infoI, _ := os.Stat(matches[i])
		infoJ, _ := os.Stat(matches[j])
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// 删除超出数量的旧文件
	toDelete := matches[:len(matches)-MaxBackupFiles]
	for _, f := range toDelete {
		if err := os.Remove(f); err != nil {
			log.Printf("警告: 删除旧备份文件 %s 失败: %v", f, err)
		}
	}

	return nil
}

// loadStrategyConfig 加载策略配置，如果主文件损坏则尝试从最新备份恢复
func loadStrategyConfig() (*StrategyConfig, error) {
	configPath := filepath.Join(getConfigDir(), StrategyConfigFile)

	// 检查配置文件是否存在
	if !fileExists(configPath) {
		// 尝试从最新备份恢复
		if err := restoreFromLatestBackup(); err != nil {
			return nil, fmt.Errorf("策略配置文件不存在且无法从备份恢复: %w", err)
		}
	}

	// 读取配置文件
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	defer file.Close()

	var config StrategyConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		// 解析失败，尝试从备份恢复
		log.Printf("配置文件解析失败，尝试从备份恢复: %v", err)
		if restoreErr := restoreFromLatestBackup(); restoreErr != nil {
			return nil, fmt.Errorf("配置文件损坏且无法从备份恢复: %w", restoreErr)
		}
		// 重新加载
		return loadStrategyConfig()
	}

	return &config, nil
}

// restoreFromLatestBackup 从最新的备份文件恢复
func restoreFromLatestBackup() error {
	configDir := getConfigDir()
	latestBackup, err := findLatestBackup(configDir)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(latestBackup)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %w", err)
	}

	configPath := filepath.Join(configDir, StrategyConfigFile)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("恢复配置文件失败: %w", err)
	}

	log.Printf("已从备份 %s 恢复配置文件", filepath.Base(latestBackup))
	return nil
}

// findLatestBackup 查找最新的备份文件（按修改时间）
func findLatestBackup(configDir string) (string, error) {
	pattern := filepath.Join(configDir, BackupFilePrefix+"*"+BackupFileSuffix)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("没有找到备份文件")
	}

	// 按修改时间排序（从新到旧）
	sort.Slice(matches, func(i, j int) bool {
		infoI, _ := os.Stat(matches[i])
		infoJ, _ := os.Stat(matches[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return matches[0], nil
}

// strategyConfigExists 检查策略配置是否存在
func strategyConfigExists() bool {
	configPath := filepath.Join(getConfigDir(), StrategyConfigFile)
	return fileExists(configPath)
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

// getConfigDir 获取配置目录
func getConfigDir() string {
	return ".bt"
}

// convertToStrategyConfig 将通用配置转换为策略特定配置
func convertToPureMarketMakingConfig(params map[string]interface{}) map[string]interface{} {
	config := make(map[string]interface{})

	// 转换时间参数
	if v, ok := params["order_refresh_time"].(int); ok {
		config["order_refresh_time"] = time.Duration(v) * time.Second
	}
	if v, ok := params["max_order_age"].(int); ok {
		config["max_order_age"] = time.Duration(v) * time.Second
	}
	if v, ok := params["filled_order_delay"].(int); ok {
		config["filled_order_delay"] = time.Duration(v) * time.Second
	}

	// 复制其他参数
	for k, v := range params {
		if k != "order_refresh_time" && k != "max_order_age" && k != "filled_order_delay" {
			config[k] = v
		}
	}

	return config
}
