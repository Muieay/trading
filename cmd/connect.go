// Package cmd /*
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"trading/api"

	"trading/config"

	"github.com/spf13/cobra"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "配置 Binance API 连接参数",
	Long: `配置 Binance API 连接参数，包括选择网络类型和输入 API 密钥。

支持两种网络类型：
  • 现货网络 (Spot Network) - https://api.binance.com
  • 模拟网络 (Demo Network) - https://demo-api.binance.com

密钥将使用 AES-GCM 加密后安全存储在本地配置文件中。`,
	Run: runConnect,
}

func init() {
	rootCmd.AddCommand(connectCmd)
}

// runConnect 执行连接配置逻辑
func runConnect(cmd *cobra.Command, args []string) {
	fmt.Println("🔗 Binance API 连接配置")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 检查是否已存在配置
	if config.ConfigExists() {
		fmt.Print("⚠️  检测到已有配置，是否重新配置？(y/N): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input != "y" && input != "yes" {
			fmt.Println("❌ 已取消配置")
			return
		}
		fmt.Println()
	}

	// 选择网络类型
	network := selectNetwork()
	fmt.Printf("✅ 已选择网络: %s\n\n", getNetworkDisplayName(network))

	// 输入 API 密钥
	apiKey := inputAPIKey()
	secretKey := inputSecretKey()

	// 保存配置
	fmt.Println("\n💾 正在保存配置...")
	if err := config.SaveConfig(network, apiKey, secretKey); err != nil {
		fmt.Printf("❌ 保存配置失败: %v\n", err)
		return
	}

	fmt.Println("✅ 配置保存成功！")
	fmt.Printf("📁 配置文件位置: %s/%s\n", config.ConfigDir, config.ConfigFile)
	fmt.Printf("🔑 主密钥文件: %s/%s\n", config.ConfigDir, config.MasterKeyFile)
	fmt.Println("\n💡 提示：所有密钥均已使用 AES-GCM 加密存储")
	// 打印鉴权信息
	api.PrintBinanceAuth()
}

// selectNetwork 选择网络类型
func selectNetwork() string {
	fmt.Println("请选择网络类型：")
	fmt.Println("  1. 现货网络 (Spot Network) - https://api.binance.com")
	fmt.Println("  2. 模拟网络 (Demo Network) - https://demo-api.binance.com")
	fmt.Print("\n请输入选项 (1-2): ")

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			return "spot"
		case "2":
			return "demo"
		default:
			fmt.Print("❌ 无效选项，请输入 1 或 2: ")
		}
	}
}

// getNetworkDisplayName 获取网络显示名称
func getNetworkDisplayName(network string) string {
	switch network {
	case "spot":
		return "现货网络 (Spot Network)"
	case "demo":
		return "模拟网络 (Demo Network)"
	default:
		return network
	}
}

// inputAPIKey 输入 API Key
func inputAPIKey() string {
	fmt.Print("请输入 BINANCE_API_KEY: ")
	reader := bufio.NewReader(os.Stdin)
	for {
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		if apiKey == "" {
			fmt.Print("❌ API Key 不能为空，请重新输入: ")
			continue
		}

		return apiKey
	}
}

// inputSecretKey 输入 Secret Key
func inputSecretKey() string {
	fmt.Print("请输入 BINANCE_SECRET_KEY: ")
	reader := bufio.NewReader(os.Stdin)
	for {
		secretKey, _ := reader.ReadString('\n')
		secretKey = strings.TrimSpace(secretKey)

		if secretKey == "" {
			fmt.Print("❌ Secret Key 不能为空，请重新输入: ")
			continue
		}

		return secretKey
	}
}
