<div align="center">

# 🚀 Binance 自动化交易机器人 (bt)

[![Go Version](https://img.shields.io/github/go-mod/go-version/LogicApex/crypto-auto/trading)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)]()

</div>

一个功能强大、安全可靠的 Binance 加密货币自动化交易机器人，支持多种做市策略和高级交易功能。

## 🌟 特性

### 🔧 核心功能
- **双策略支持**：纯做市策略 + 挂单做市策略
- **安全认证**：AES-256 加密存储 API 密钥
- **实时监控**：30秒间隔状态报告
- **智能风控**：库存管理、价格区间控制、订单簿优化
- **灵活配置**：支持多层挂单、动态调参

### 📊 策略特性

#### 纯做市策略 (Pure Market Making)
- 多层挂单支持（可配置层数）
- 智能库存平衡
- 订单簿优化跳价
- 动态价格带控制
- Ping-Pong 成交模式
- Hanging Orders 机制

#### 挂单做市策略 (Wait Market Making)
- 精准价差控制
- 自适应刷新机制
- 成交延迟处理
- 最长订单存活时间管理

## 📁 项目结构

```
trading/
├── .bt/                    # 配置文件目录
├── api/                    # Binance API 集成
│   ├── account.go          # 账户相关接口
│   ├── binance_auth.go     # 认证模块
│   ├── deal.go             # 交易接口
│   └── market.go           # 市场数据接口
├── cmd/                    # 命令行界面
├── config/                 # 配置管理
├── strategy/               # 交易策略
│   ├── pure_market.go      # 纯做市策略
│   └── wait_market.go      # 挂单做市策略
├── main.go                 # 程序入口
├── go.mod                  # Go 模块文件
└── README.md               # 本文档
```

## 🚀 快速开始

### 环境要求

- Go 1.24 或更高版本
- Binance API Key 和 Secret Key

### 安装步骤

1. **克隆项目**
```bash
git clone https://github.com/Muieay/trading.git
```

2. **编译项目**
```bash
cd trading

go build -o bt
```

3. **验证安装**
```bash
./bt --help
```

### 基本使用流程

#### 1. 配置 Binance API 连接
```bash
./bt connect
```
按照提示输入：
- 网络类型（spot/demo）
- API Key
- Secret Key

> 💡 **安全提醒**：API 密钥将使用 AES-256 加密存储在 `.bt/config.json` 中

#### 2. 配置交易策略
```bash
./bt config
```
选择策略类型并配置参数：
- 交易对（如 SOLUSDT）
- 买卖价差
- 订单数量
- 挂单层数等

#### 3. 启动交易策略
```bash
./bt start
```

程序将：
- ✅ 验证 API 连接
- ✅ 加载策略配置
- ✅ 启动做市策略
- 📊 每30秒显示状态报告

> ⚠️ **停止策略**：按 `Ctrl+C` 安全退出

## 🛠️ 详细命令说明

### connect - 配置 API 连接

```bash
./bt connect [flags]
```

**参数**：
- `-n, --network string` 网络类型：spot（实盘）或 demo（模拟盘）
- `-k, --api-key string` Binance API Key
- `-s, --secret-key string` Binance Secret Key

**示例**：
```bash
# 交互式配置
./bt connect

# 命令行直接配置
./bt connect -n spot -k "your_api_key" -s "your_secret_key"
```

### config - 配置交易策略

```bash
./bt config [flags]
```

**子命令**：
- `pure-market` 纯做市策略
- `wait-market` 挂单做市策略

**示例**：
```bash
# 配置纯做市策略
./bt config pure-market

# 配置挂单做市策略
./bt config wait-market
```

### start - 启动策略

```bash
./bt start
```

**功能**：
- 加载已保存的 API 配置和策略配置
- 测试 API 连接
- 初始化并启动选定的策略
- 实时显示策略状态

## ⚙️ 策略配置详解

### 纯做市策略配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `market` | string | SOLUSDT | 交易对 |
| `bid_spread` | float | 0.001 | 买单价差 (0.1%) |
| `ask_spread` | float | 0.001 | 卖单价差 (0.1%) |
| `order_amount` | float | 0.1 | 每笔订单数量 |
| `order_levels` | int | 3 | 挂单层数 |
| `order_refresh_time` | duration | 10s | 刷新周期 |
| `inventory_skew_enabled` | bool | true | 启用库存平衡 |
| `order_optimization_enabled` | bool | true | 启用订单簿优化 |

### 挂单做市策略配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `market` | string | SOLUSDT | 交易对 |
| `bid_spread` | float | 0.001 | 买单价差 (0.1%) |
| `ask_spread` | float | 0.01 | 卖单价差 (1%) |
| `order_amount` | float | 0.1 | 每笔买单数量 |
| `order_levels` | int | 3 | 挂单层数 |
| `order_refresh_time` | duration | 60s | 刷新周期 |
| `max_order_age` | duration | 300s | 订单最长存活时间 |

## 🔒 安全特性

### 数据加密
- API 密钥使用 AES-256-GCM 加密
- 主密钥随机生成并安全存储
- 配置文件权限设置为 600

### 风险控制
- 价格区间限制
- 最小点差保护
- 库存风险管理
- 订单生命周期管理

## 📊 状态监控

程序运行时会定期显示以下信息：

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 策略状态 [14:30:25]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  运行状态: true
  交易对: SOLUSDT
  活跃订单: 6
  基础资产余额: 10.50000000
  计价资产余额: 1250.75000000
  最新中间价: 119.12000000
  库存偏移: 2.3500%
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## 🐛 故障排除

### 常见问题

**1. API 连接失败**
```
❌ API 连接测试失败: Invalid API-key, IP, or permissions.
```
**解决方案**：
- 检查 API Key 和 Secret Key 是否正确
- 确认 API 权限包含 Spot & Margin Trading
- 验证服务器 IP 是否在白名单中

**2. 配置文件不存在**
```
❌ 加载 API 配置失败: 配置文件不存在，请先运行 'bt connect' 命令
```
**解决方案**：
```bash
./bt connect  # 重新配置 API 连接
```

**3. 策略配置缺失**
```
❌ 加载策略配置失败: 策略配置不存在，请先运行 'bt config' 命令
```
**解决方案**：
```bash
./bt config pure-market  # 配置纯做市策略
# 或
./bt config wait-market  # 配置挂单做市策略
```

### 日志查看

所有操作都会输出详细日志，便于调试和监控。

## 📈 性能优化建议

### 参数调优
1. **价差设置**：根据市场波动性调整（建议 0.1%-0.5%）
2. **订单数量**：从小额开始，逐步增加
3. **刷新频率**：高频交易设为 5-10s，低频设为 30-60s
4. **挂单层数**：新手建议 1-2 层，有经验可增至 5 层

### 环境建议
- 使用稳定的网络连接
- 保持系统时间同步
- 定期备份配置文件

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 开发环境搭建
```bash
git clone https://github.com/Muieay/trading.git
cd trading
go mod tidy
go build
```

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## ⚠️ 免责声明

> 本软件仅供学习和研究使用。数字货币交易存在高风险，可能导致本金全部损失。使用者需自行承担所有交易风险，作者不对任何交易损失负责。

> 请在充分了解做市策略原理和风险后再进行实盘交易，建议先在模拟环境中测试。

---

<p align="center">
  Made with ❤️ by <a href="https://muieay.github.io">Muieay</a>
</p>