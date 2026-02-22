package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"trading/config"
)

// BaseURL 将根据配置动态设置
var BaseURL = "https://demo-api.binance.com"

// BinanceClient Binance API 客户端
type BinanceClient struct {
	APIKey     string
	SecretKey  string
	HTTPClient *http.Client
}

// APIError Binance API 错误结构体
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Binance API Error [%d]: %s", e.Code, e.Message)
}

// NewBinanceClient 创建新的 Binance 客户端
func NewBinanceClient(apiKey, secretKey string) *BinanceClient {
	return &BinanceClient{
		APIKey:    apiKey,
		SecretKey: secretKey,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// sign 使用 HMAC-SHA256 对查询字符串进行签名
func (c *BinanceClient) sign(queryString string) string {
	mac := hmac.New(sha256.New, []byte(c.SecretKey))
	mac.Write([]byte(queryString))
	return hex.EncodeToString(mac.Sum(nil))
}

// getTimestamp 获取当前时间戳（毫秒）
func getTimestamp() int64 {
	return time.Now().UnixMilli()
}

// newHTTPRequest 创建 HTTP 请求
func newHTTPRequest(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// executeRequest 执行 HTTP 请求并返回响应体和状态码
func executeRequest(client *http.Client, req *http.Request) ([]byte, int, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("读取响应失败: %w", err)
	}
	return body, resp.StatusCode, nil
}

// ============================================================
// main：鉴权连通测试入口
// ============================================================

func PrintBinanceAuth() {
	// 从配置文件加载API密钥
	appConfig, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		fmt.Println("💡 请先运行 'bt connect' 命令配置API密钥")
		os.Exit(1)
	}

	// 设置BaseURL
	BaseURL = appConfig.BaseURL

	client := NewBinanceClient(appConfig.APIKey, appConfig.SecretKey)

	fmt.Println("🔐 Binance API 鉴权测试")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📡 请求地址: %s/api/v3/account\n", BaseURL)
	fmt.Printf("🌐 网络类型: %s\n", getNetworkType(BaseURL))
	fmt.Printf("🕐 请求时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 调用 account.go 中封装的 GetAccountInfo
	omitZero := true
	accountInfo, err := client.GetAccountInfo(&GetAccountInfoParams{
		OmitZeroBalances: &omitZero,
	})
	if err != nil {
		fmt.Printf("❌ 连通失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ 鉴权成功，API 连通正常！")
	fmt.Println()
	fmt.Printf("👤 账户 UID:      %d\n", accountInfo.UID)
	fmt.Printf("📋 账户类型:      %s\n", accountInfo.AccountType)
	fmt.Printf("💹 允许交易:      %v\n", accountInfo.CanTrade)
	fmt.Printf("💸 允许提现:      %v\n", accountInfo.CanWithdraw)
	fmt.Printf("💰 允许充值:      %v\n", accountInfo.CanDeposit)
	fmt.Printf("📊 Maker 手续费:  %d bps\n", accountInfo.MakerCommission)
	fmt.Printf("📊 Taker 手续费:  %d bps\n", accountInfo.TakerCommission)
	fmt.Printf("🔑 账户权限:      %v\n", accountInfo.Permissions)
	fmt.Printf("🕐 更新时间:      %s\n",
		time.UnixMilli(accountInfo.UpdateTime).Format("2006-01-02 15:04:05"))

	fmt.Println()
	fmt.Println("💼 持仓余额（非零资产）:")
	fmt.Println("   ┌─────────────┬──────────────────────┬──────────────────────┐")
	fmt.Println("   │ 资产        │ 可用余额              │ 冻结余额              │")
	fmt.Println("   ├─────────────┼──────────────────────┼──────────────────────┤")

	hasBalance := false
	for _, b := range accountInfo.Balances {
		if b.Free != "0.00000000" || b.Locked != "0.00000000" {
			fmt.Printf("   │ %-11s │ %-20s │ %-20s │\n", b.Asset, b.Free, b.Locked)
			hasBalance = true
		}
	}
	if !hasBalance {
		fmt.Println("   │ （暂无非零资产）                                           │")
	}
	fmt.Println("   └─────────────┴──────────────────────┴──────────────────────┘")
}

// getNetworkType 根据BaseURL返回网络类型
func getNetworkType(baseURL string) string {
	switch baseURL {
	case "https://api.binance.com":
		return "现货网络 (Spot Network)"
	case "https://demo-api.binance.com":
		return "模拟网络 (Demo Network)"
	default:
		return "未知网络"
	}
}
