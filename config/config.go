package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	ConfigDir     = ".bt"
	ConfigFile    = "config.json"
	MasterKeyFile = "master.key"
)

// Config 配置结构体
type Config struct {
	Network   string `json:"network"`    // "spot" 或 "demo"
	APIKey    string `json:"api_key"`    // 加密后的API Key
	SecretKey string `json:"secret_key"` // 加密后的Secret Key
	BaseURL   string `json:"base_url"`   // API基础URL
}

// AppConfig 应用配置结构体（解密后的）
type AppConfig struct {
	Network   string
	APIKey    string
	SecretKey string
	BaseURL   string
}

// generateMasterKey 生成主密钥
func generateMasterKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256需要32字节密钥
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("生成主密钥失败: %w", err)
	}
	return key, nil
}

// saveMasterKey 保存主密钥到文件
func saveMasterKey(key []byte) error {
	configPath := filepath.Join(ConfigDir, MasterKeyFile)

	// 确保目录存在
	if err := os.MkdirAll(ConfigDir, 0700); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 保存密钥文件，设置权限为600
	if err := os.WriteFile(configPath, key, 0600); err != nil {
		return fmt.Errorf("保存主密钥失败: %w", err)
	}

	return nil
}

// loadMasterKey 加载主密钥
func loadMasterKey() ([]byte, error) {
	configPath := filepath.Join(ConfigDir, MasterKeyFile)

	// 检查密钥文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 如果不存在则生成新的主密钥
		key, err := generateMasterKey()
		if err != nil {
			return nil, err
		}

		if err := saveMasterKey(key); err != nil {
			return nil, err
		}

		return key, nil
	}

	// 读取现有密钥
	key, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取主密钥失败: %w", err)
	}

	// 验证密钥长度
	if len(key) != 32 {
		return nil, fmt.Errorf("主密钥长度不正确")
	}

	return key, nil
}

// encrypt 使用AES-GCM加密数据
func encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 使用AES-GCM解密数据
func decrypt(ciphertext string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("密文格式错误")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// SaveConfig 保存配置到文件
func SaveConfig(network, apiKey, secretKey string) error {
	// 根据网络类型确定BaseURL
	var baseURL string
	switch network {
	case "spot":
		baseURL = "https://api.binance.com"
	case "demo":
		baseURL = "https://demo-api.binance.com"
	default:
		return fmt.Errorf("不支持的网络类型: %s", network)
	}

	// 加载或生成主密钥
	masterKey, err := loadMasterKey()
	if err != nil {
		return fmt.Errorf("获取主密钥失败: %w", err)
	}

	// 加密API密钥
	encryptedAPIKey, err := encrypt(apiKey, masterKey)
	if err != nil {
		return fmt.Errorf("加密API Key失败: %w", err)
	}

	// 加密Secret Key
	encryptedSecretKey, err := encrypt(secretKey, masterKey)
	if err != nil {
		return fmt.Errorf("加密Secret Key失败: %w", err)
	}

	// 创建配置对象
	config := &Config{
		Network:   network,
		APIKey:    encryptedAPIKey,
		SecretKey: encryptedSecretKey,
		BaseURL:   baseURL,
	}

	// 确保配置目录存在
	if err := os.MkdirAll(ConfigDir, 0700); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 保存配置文件
	configPath := filepath.Join(ConfigDir, ConfigFile)
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

// LoadConfig 加载并解密配置
func LoadConfig() (*AppConfig, error) {
	configPath := filepath.Join(ConfigDir, ConfigFile)

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在，请先运行 'bt connect' 命令")
	}

	// 读取配置文件
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 加载主密钥
	masterKey, err := loadMasterKey()
	if err != nil {
		return nil, fmt.Errorf("获取主密钥失败: %w", err)
	}

	// 解密API Key
	apiKey, err := decrypt(config.APIKey, masterKey)
	if err != nil {
		return nil, fmt.Errorf("解密API Key失败: %w", err)
	}

	// 解密Secret Key
	secretKey, err := decrypt(config.SecretKey, masterKey)
	if err != nil {
		return nil, fmt.Errorf("解密Secret Key失败: %w", err)
	}

	return &AppConfig{
		Network:   config.Network,
		APIKey:    apiKey,
		SecretKey: secretKey,
		BaseURL:   config.BaseURL,
	}, nil
}

// ConfigExists 检查配置是否存在
func ConfigExists() bool {
	configPath := filepath.Join(ConfigDir, ConfigFile)
	_, err := os.Stat(configPath)
	return err == nil
}
