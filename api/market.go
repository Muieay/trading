package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// ============================================================
// 行情接口相关结构体
// ============================================================

// OrderBookEntry 深度价格条目 [价格, 数量]
type OrderBookEntry [2]string

// OrderBook 深度信息
type OrderBook struct {
	LastUpdateId int64            `json:"lastUpdateId"`
	Bids         []OrderBookEntry `json:"bids"` // 买单 [价格, 数量]，从高到低
	Asks         []OrderBookEntry `json:"asks"` // 卖单 [价格, 数量]，从低到高
}

// MarketTrade 近期成交 / 历史成交记录
type MarketTrade struct {
	Id           int64  `json:"id"`
	Price        string `json:"price"`
	Qty          string `json:"qty"`
	QuoteQty     string `json:"quoteQty"`
	Time         int64  `json:"time"`
	IsBuyerMaker bool   `json:"isBuyerMaker"`
	IsBestMatch  bool   `json:"isBestMatch"`
}

// AggTrade 归集成交记录
type AggTrade struct {
	AggId        int64  `json:"a"` // 归集成交 ID
	Price        string `json:"p"` // 成交价
	Qty          string `json:"q"` // 成交量
	FirstTradeId int64  `json:"f"` // 被归集的首个成交 ID
	LastTradeId  int64  `json:"l"` // 被归集的末个成交 ID
	Time         int64  `json:"T"` // 成交时间
	IsBuyerMaker bool   `json:"m"` // 是否为主动卖出单
	IsBestMatch  bool   `json:"M"` // 是否为最优撮合
}

// Kline K线数据（每根K线为一个数组）
// 字段顺序: [开盘时间, 开盘价, 最高价, 最低价, 收盘价, 成交量,
//
//	收盘时间, 成交额, 成交笔数, 主动买入成交量, 主动买入成交额, 忽略]
type Kline [12]json.RawMessage

// KlineData 解析后的 K 线字段（便于使用）
type KlineData struct {
	OpenTime                 int64  // 开盘时间（毫秒时间戳）
	Open                     string // 开盘价
	High                     string // 最高价
	Low                      string // 最低价
	Close                    string // 收盘价
	Volume                   string // 成交量（base asset）
	CloseTime                int64  // 收盘时间（毫秒时间戳）
	QuoteAssetVolume         string // 成交额（quote asset）
	TradeCount               int64  // 成交笔数
	TakerBuyBaseAssetVolume  string // 主动买入成交量
	TakerBuyQuoteAssetVolume string // 主动买入成交额
}

// ParseKline 将原始 Kline 解析为 KlineData
func ParseKline(k Kline) (KlineData, error) {
	var d KlineData
	if err := json.Unmarshal(k[0], &d.OpenTime); err != nil {
		return d, fmt.Errorf("解析开盘时间失败: %w", err)
	}
	if err := json.Unmarshal(k[1], &d.Open); err != nil {
		return d, err
	}
	if err := json.Unmarshal(k[2], &d.High); err != nil {
		return d, err
	}
	if err := json.Unmarshal(k[3], &d.Low); err != nil {
		return d, err
	}
	if err := json.Unmarshal(k[4], &d.Close); err != nil {
		return d, err
	}
	if err := json.Unmarshal(k[5], &d.Volume); err != nil {
		return d, err
	}
	if err := json.Unmarshal(k[6], &d.CloseTime); err != nil {
		return d, err
	}
	if err := json.Unmarshal(k[7], &d.QuoteAssetVolume); err != nil {
		return d, err
	}
	if err := json.Unmarshal(k[8], &d.TradeCount); err != nil {
		return d, err
	}
	if err := json.Unmarshal(k[9], &d.TakerBuyBaseAssetVolume); err != nil {
		return d, err
	}
	if err := json.Unmarshal(k[10], &d.TakerBuyQuoteAssetVolume); err != nil {
		return d, err
	}
	return d, nil
}

// AvgPrice 当前平均价格
type AvgPrice struct {
	Mins      int    `json:"mins"`      // 计算均价的分钟数
	Price     string `json:"price"`     // 均价
	CloseTime int64  `json:"closeTime"` // 统计截止时间
}

// Ticker24hr 24小时价格变动情况（FULL 格式）
type Ticker24hr struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	PrevClosePrice     string `json:"prevClosePrice,omitempty"`
	LastPrice          string `json:"lastPrice"`
	LastQty            string `json:"lastQty,omitempty"`
	BidPrice           string `json:"bidPrice,omitempty"`
	BidQty             string `json:"bidQty,omitempty"`
	AskPrice           string `json:"askPrice,omitempty"`
	AskQty             string `json:"askQty,omitempty"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	FirstId            int64  `json:"firstId"`
	LastId             int64  `json:"lastId"`
	Count              int64  `json:"count"`
}

// Ticker24hrMini 24小时价格变动情况（MINI 格式）
type Ticker24hrMini struct {
	Symbol      string `json:"symbol"`
	OpenPrice   string `json:"openPrice"`
	HighPrice   string `json:"highPrice"`
	LowPrice    string `json:"lowPrice"`
	LastPrice   string `json:"lastPrice"`
	Volume      string `json:"volume"`
	QuoteVolume string `json:"quoteVolume"`
	OpenTime    int64  `json:"openTime"`
	CloseTime   int64  `json:"closeTime"`
	FirstId     int64  `json:"firstId"`
	LastId      int64  `json:"lastId"`
	Count       int64  `json:"count"`
}

// TradingDayTicker 交易日行情（FULL 格式，与 Ticker24hr 结构相同）
type TradingDayTicker = Ticker24hr

// TradingDayTickerMini 交易日行情（MINI 格式）
type TradingDayTickerMini = Ticker24hrMini

// TickerPrice 最新价格
type TickerPrice struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// BookTicker 最优挂单
type BookTicker struct {
	Symbol   string `json:"symbol"`
	BidPrice string `json:"bidPrice"` // 最优买单价
	BidQty   string `json:"bidQty"`   // 最优买单量
	AskPrice string `json:"askPrice"` // 最优卖单价
	AskQty   string `json:"askQty"`   // 最优卖单量
}

// RollingWindowTicker 滚动窗口价格变动统计（FULL 格式，与 Ticker24hr 结构相同）
type RollingWindowTicker = Ticker24hr

// RollingWindowTickerMini 滚动窗口价格变动统计（MINI 格式）
type RollingWindowTickerMini = Ticker24hrMini

// ============================================================
// 请求参数结构体
// ============================================================

// GetDepthParams 深度信息请求参数
type GetDepthParams struct {
	Symbol       string  // 必填
	Limit        *int    // 默认 100，最大 5000
	SymbolStatus *string // TRADING / HALT / BREAK
}

// GetTradesParams 近期成交请求参数
type GetTradesParams struct {
	Symbol string // 必填
	Limit  *int   // 默认 500，最大 1000
}

// GetHistoricalTradesParams 历史成交请求参数
type GetHistoricalTradesParams struct {
	Symbol string // 必填
	Limit  *int   // 默认 500，最大 1000
	FromId *int64 // 从哪一条成交 ID 开始返回
}

// GetAggTradesParams 归集成交请求参数
type GetAggTradesParams struct {
	Symbol    string // 必填
	FromId    *int64 // 从该 ID 开始返回
	StartTime *int64
	EndTime   *int64
	Limit     *int // 默认 500，最大 1000
}

// GetKlinesParams K线数据请求参数
type GetKlinesParams struct {
	Symbol    string // 必填
	Interval  string // 必填，如 1m/5m/1h/1d 等
	StartTime *int64
	EndTime   *int64
	TimeZone  *string // 默认 0 (UTC)
	Limit     *int    // 默认 500，最大 1000
}

// GetAvgPriceParams 当前平均价格请求参数
type GetAvgPriceParams struct {
	Symbol string // 必填
}

// GetTicker24hrParams 24小时价格变动请求参数
type GetTicker24hrParams struct {
	Symbol       *string  // symbol 与 symbols 不可同时使用
	Symbols      []string // 多个交易对
	Type         *string  // FULL(默认) / MINI
	SymbolStatus *string  // TRADING / HALT / BREAK
}

// GetTradingDayTickerParams 交易日行情请求参数
type GetTradingDayTickerParams struct {
	Symbol       *string // symbol 与 symbols 必须提供其一
	Symbols      []string
	TimeZone     *string
	Type         *string // FULL(默认) / MINI
	SymbolStatus *string
}

// GetTickerPriceParams 最新价格请求参数
type GetTickerPriceParams struct {
	Symbol       *string // symbol 与 symbols 不可同时使用
	Symbols      []string
	SymbolStatus *string
}

// GetBookTickerParams 最优挂单请求参数
type GetBookTickerParams struct {
	Symbol       *string // symbol 与 symbols 不可同时使用
	Symbols      []string
	SymbolStatus *string
}

// GetRollingWindowTickerParams 滚动窗口价格变动统计请求参数
type GetRollingWindowTickerParams struct {
	Symbol       *string // symbol 与 symbols 必须提供其一
	Symbols      []string
	WindowSize   *string // 默认 1d，支持 1m-59m / 1h-23h / 1d-7d
	Type         *string // FULL(默认) / MINI
	SymbolStatus *string
}

// ============================================================
// 工具函数
// ============================================================

// noAuthGet 发起不需要鉴权的公开 GET 请求（行情接口无需 API Key）
func (c *BinanceClient) noAuthGet(endpoint string, params url.Values, result interface{}) error {
	queryString := params.Encode()
	fullURL := BaseURL + endpoint
	if queryString != "" {
		fullURL += "?" + queryString
	}
	body, statusCode, err := c.doRequest("GET", fullURL, false)
	if err != nil {
		return err
	}
	return parseResponse(body, statusCode, result)
}

// symbolsToJSON 将字符串切片转为 JSON 数组字符串，用于 symbols 参数
func symbolsToJSON(symbols []string) string {
	b, _ := json.Marshal(symbols)
	return string(b)
}

// ============================================================
// 行情接口实现（均为公开接口，无需签名）
// ============================================================

// GetDepth 深度信息
// GET /api/v3/depth
// 权重: limit 1-100=5, 101-500=25, 501-1000=50, 1001-5000=250
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#深度信息
func (c *BinanceClient) GetDepth(p GetDepthParams) (*OrderBook, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt(params, "limit", p.Limit)
	setOptString(params, "symbolStatus", p.SymbolStatus)

	var result OrderBook
	if err := c.noAuthGet("/api/v3/depth", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTrades 近期成交
// GET /api/v3/trades
// 权重: 25
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#近期成交
func (c *BinanceClient) GetTrades(p GetTradesParams) ([]MarketTrade, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt(params, "limit", p.Limit)

	var result []MarketTrade
	if err := c.noAuthGet("/api/v3/trades", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetHistoricalTrades 查询历史成交
// GET /api/v3/historicalTrades
// 权重: 25
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#查询历史成交
// 注意: 此接口需要 API Key（X-MBX-APIKEY），但无需签名
func (c *BinanceClient) GetHistoricalTrades(p GetHistoricalTradesParams) ([]MarketTrade, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt(params, "limit", p.Limit)
	setOptInt64(params, "fromId", p.FromId)

	// historicalTrades 需要 API Key 但不需要签名
	var result []MarketTrade
	if err := c.publicGet("/api/v3/historicalTrades", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetAggTrades 近期成交（归集）
// GET /api/v3/aggTrades
// 权重: 4
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#近期成交归集
// 说明: 同一 taker 同一时间同一价格与多个 maker 的成交会合并为一条
func (c *BinanceClient) GetAggTrades(p GetAggTradesParams) ([]AggTrade, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt64(params, "fromId", p.FromId)
	setOptInt64(params, "startTime", p.StartTime)
	setOptInt64(params, "endTime", p.EndTime)
	setOptInt(params, "limit", p.Limit)

	var result []AggTrade
	if err := c.noAuthGet("/api/v3/aggTrades", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetKlines K 线数据
// GET /api/v3/klines
// 权重: 2
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#k线数据
// 支持的 interval: 1s / 1m 3m 5m 15m 30m / 1h 2h 4h 6h 8h 12h / 1d 3d / 1w / 1M
// 提示: 返回原始 []Kline，可调用 ParseKline() 解析每条记录为 KlineData
func (c *BinanceClient) GetKlines(p GetKlinesParams) ([]Kline, error) {
	if p.Symbol == "" || p.Interval == "" {
		return nil, fmt.Errorf("symbol 和 interval 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("interval", p.Interval)
	setOptInt64(params, "startTime", p.StartTime)
	setOptInt64(params, "endTime", p.EndTime)
	setOptString(params, "timeZone", p.TimeZone)
	setOptInt(params, "limit", p.Limit)

	var result []Kline
	if err := c.noAuthGet("/api/v3/klines", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetUIKlines UI K 线数据（针对图表展示优化）
// GET /api/v3/uiKlines
// 权重: 2
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#uik线数据
// 说明: 参数与响应格式和 GetKlines 完全相同，数据经过图表优化处理
func (c *BinanceClient) GetUIKlines(p GetKlinesParams) ([]Kline, error) {
	if p.Symbol == "" || p.Interval == "" {
		return nil, fmt.Errorf("symbol 和 interval 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("interval", p.Interval)
	setOptInt64(params, "startTime", p.StartTime)
	setOptInt64(params, "endTime", p.EndTime)
	setOptString(params, "timeZone", p.TimeZone)
	setOptInt(params, "limit", p.Limit)

	var result []Kline
	if err := c.noAuthGet("/api/v3/uiKlines", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetAvgPrice 当前平均价格
// GET /api/v3/avgPrice
// 权重: 2
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#当前平均价格
func (c *BinanceClient) GetAvgPrice(p GetAvgPriceParams) (*AvgPrice, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)

	var result AvgPrice
	if err := c.noAuthGet("/api/v3/avgPrice", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTicker24hr 24 小时价格变动情况（FULL 格式）
// GET /api/v3/ticker/24hr
// 权重: 单个 symbol=2，symbols 1-20=2，21-100=40，>=101 或不传=80
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#24hr价格变动情况
// 说明: 不传任何 symbol 参数会返回所有交易对，权重极高(80)，谨慎使用
// 响应: 传 symbol 返回单个对象，传 symbols 或不传返回数组
func (c *BinanceClient) GetTicker24hr(p GetTicker24hrParams) ([]Ticker24hr, error) {
	params := url.Values{}
	if p.Symbol != nil {
		params.Set("symbol", *p.Symbol)
	} else if len(p.Symbols) > 0 {
		params.Set("symbols", symbolsToJSON(p.Symbols))
	}
	setOptString(params, "type", p.Type)
	setOptString(params, "symbolStatus", p.SymbolStatus)

	// 根据是否传 symbol 决定响应格式（单对象 vs 数组）
	raw, statusCode, err := c.doRequest("GET", BaseURL+"/api/v3/ticker/24hr?"+params.Encode(), false)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPStatus(raw, statusCode); err != nil {
		return nil, err
	}

	// 尝试解析为数组，兼容单个对象
	return parseSingleOrArray[Ticker24hr](raw)
}

// GetTicker24hrMini 24 小时价格变动情况（MINI 格式）
// GET /api/v3/ticker/24hr?type=MINI
// 权重: 同 GetTicker24hr
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#24hr价格变动情况
func (c *BinanceClient) GetTicker24hrMini(p GetTicker24hrParams) ([]Ticker24hrMini, error) {
	mini := "MINI"
	p.Type = &mini

	params := url.Values{}
	if p.Symbol != nil {
		params.Set("symbol", *p.Symbol)
	} else if len(p.Symbols) > 0 {
		params.Set("symbols", symbolsToJSON(p.Symbols))
	}
	params.Set("type", "MINI")
	setOptString(params, "symbolStatus", p.SymbolStatus)

	raw, statusCode, err := c.doRequest("GET", BaseURL+"/api/v3/ticker/24hr?"+params.Encode(), false)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPStatus(raw, statusCode); err != nil {
		return nil, err
	}
	return parseSingleOrArray[Ticker24hrMini](raw)
}

// GetTradingDayTicker 交易日行情（FULL 格式）
// GET /api/v3/ticker/tradingDay
// 权重: 每个交易对 4 权重，超过 50 个时上限为 200
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#交易日行情ticker
// 说明: symbol 或 symbols 必须提供其一，最多 100 个交易对
func (c *BinanceClient) GetTradingDayTicker(p GetTradingDayTickerParams) ([]TradingDayTicker, error) {
	if p.Symbol == nil && len(p.Symbols) == 0 {
		return nil, fmt.Errorf("symbol 或 symbols 必须提供其一")
	}
	params := url.Values{}
	if p.Symbol != nil {
		params.Set("symbol", *p.Symbol)
	} else {
		params.Set("symbols", symbolsToJSON(p.Symbols))
	}
	setOptString(params, "timeZone", p.TimeZone)
	setOptString(params, "type", p.Type)
	setOptString(params, "symbolStatus", p.SymbolStatus)

	raw, statusCode, err := c.doRequest("GET", BaseURL+"/api/v3/ticker/tradingDay?"+params.Encode(), false)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPStatus(raw, statusCode); err != nil {
		return nil, err
	}
	return parseSingleOrArray[TradingDayTicker](raw)
}

// GetTradingDayTickerMini 交易日行情（MINI 格式）
// GET /api/v3/ticker/tradingDay?type=MINI
// 权重: 同 GetTradingDayTicker
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#交易日行情ticker
func (c *BinanceClient) GetTradingDayTickerMini(p GetTradingDayTickerParams) ([]TradingDayTickerMini, error) {
	if p.Symbol == nil && len(p.Symbols) == 0 {
		return nil, fmt.Errorf("symbol 或 symbols 必须提供其一")
	}
	params := url.Values{}
	if p.Symbol != nil {
		params.Set("symbol", *p.Symbol)
	} else {
		params.Set("symbols", symbolsToJSON(p.Symbols))
	}
	params.Set("type", "MINI")
	setOptString(params, "timeZone", p.TimeZone)
	setOptString(params, "symbolStatus", p.SymbolStatus)

	raw, statusCode, err := c.doRequest("GET", BaseURL+"/api/v3/ticker/tradingDay?"+params.Encode(), false)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPStatus(raw, statusCode); err != nil {
		return nil, err
	}
	return parseSingleOrArray[TradingDayTickerMini](raw)
}

// GetTickerPrice 最新价格
// GET /api/v3/ticker/price
// 权重: 单 symbol=2，symbols 或不传=4
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#最新价格接口
// 说明: 不传参数返回所有交易对价格；symbol 与 symbols 不可同时使用
func (c *BinanceClient) GetTickerPrice(p GetTickerPriceParams) ([]TickerPrice, error) {
	params := url.Values{}
	if p.Symbol != nil {
		params.Set("symbol", *p.Symbol)
	} else if len(p.Symbols) > 0 {
		params.Set("symbols", symbolsToJSON(p.Symbols))
	}
	setOptString(params, "symbolStatus", p.SymbolStatus)

	raw, statusCode, err := c.doRequest("GET", BaseURL+"/api/v3/ticker/price?"+params.Encode(), false)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPStatus(raw, statusCode); err != nil {
		return nil, err
	}
	return parseSingleOrArray[TickerPrice](raw)
}

// GetBookTicker 最优挂单（最高买单/最低卖单）
// GET /api/v3/ticker/bookTicker
// 权重: 单 symbol=2，symbols 或不传=4
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#最优挂单接口
// 说明: 不传参数返回所有交易对；symbol 与 symbols 不可同时使用
func (c *BinanceClient) GetBookTicker(p GetBookTickerParams) ([]BookTicker, error) {
	params := url.Values{}
	if p.Symbol != nil {
		params.Set("symbol", *p.Symbol)
	} else if len(p.Symbols) > 0 {
		params.Set("symbols", symbolsToJSON(p.Symbols))
	}
	setOptString(params, "symbolStatus", p.SymbolStatus)

	raw, statusCode, err := c.doRequest("GET", BaseURL+"/api/v3/ticker/bookTicker?"+params.Encode(), false)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPStatus(raw, statusCode); err != nil {
		return nil, err
	}
	return parseSingleOrArray[BookTicker](raw)
}

// GetRollingWindowTicker 滚动窗口价格变动统计（FULL 格式）
// GET /api/v3/ticker
// 权重: 每个交易对 4 权重，超过 50 个时上限为 200
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#滚动窗口价格变动统计
// 说明: symbol 或 symbols 必须提供其一；windowSize 支持 1m-59m / 1h-23h / 1d-7d
// 注意: openTime 从整分钟起始，实际统计范围比 windowSize 多不超过 59999ms
func (c *BinanceClient) GetRollingWindowTicker(p GetRollingWindowTickerParams) ([]RollingWindowTicker, error) {
	if p.Symbol == nil && len(p.Symbols) == 0 {
		return nil, fmt.Errorf("symbol 或 symbols 必须提供其一")
	}
	params := url.Values{}
	if p.Symbol != nil {
		params.Set("symbol", *p.Symbol)
	} else {
		params.Set("symbols", symbolsToJSON(p.Symbols))
	}
	setOptString(params, "windowSize", p.WindowSize)
	setOptString(params, "type", p.Type)
	setOptString(params, "symbolStatus", p.SymbolStatus)

	raw, statusCode, err := c.doRequest("GET", BaseURL+"/api/v3/ticker?"+params.Encode(), false)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPStatus(raw, statusCode); err != nil {
		return nil, err
	}
	return parseSingleOrArray[RollingWindowTicker](raw)
}

// GetRollingWindowTickerMini 滚动窗口价格变动统计（MINI 格式）
// GET /api/v3/ticker?type=MINI
// 权重: 同 GetRollingWindowTicker
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/market-data-endpoints#滚动窗口价格变动统计
func (c *BinanceClient) GetRollingWindowTickerMini(p GetRollingWindowTickerParams) ([]RollingWindowTickerMini, error) {
	if p.Symbol == nil && len(p.Symbols) == 0 {
		return nil, fmt.Errorf("symbol 或 symbols 必须提供其一")
	}
	params := url.Values{}
	if p.Symbol != nil {
		params.Set("symbol", *p.Symbol)
	} else {
		params.Set("symbols", symbolsToJSON(p.Symbols))
	}
	params.Set("type", "MINI")
	setOptString(params, "windowSize", p.WindowSize)
	setOptString(params, "symbolStatus", p.SymbolStatus)

	raw, statusCode, err := c.doRequest("GET", BaseURL+"/api/v3/ticker?"+params.Encode(), false)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPStatus(raw, statusCode); err != nil {
		return nil, err
	}
	return parseSingleOrArray[RollingWindowTickerMini](raw)
}

// ============================================================
// 内部工具：单对象/数组统一解析
// ============================================================

// checkHTTPStatus 检查 HTTP 状态码
func checkHTTPStatus(body []byte, statusCode int) error {
	if statusCode != 200 {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
		}
		return &apiErr
	}
	return nil
}

// parseSingleOrArray 将响应体解析为切片，兼容单个对象（自动包装为 []T）
func parseSingleOrArray[T any](data []byte) ([]T, error) {
	if len(data) == 0 {
		return nil, nil
	}
	// 以第一个字符区分数组和对象
	if data[0] == '[' {
		var result []T
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("解析数组响应失败: %w", err)
		}
		return result, nil
	}
	// 单个对象，包装为切片
	var single T
	if err := json.Unmarshal(data, &single); err != nil {
		return nil, fmt.Errorf("解析对象响应失败: %w", err)
	}
	return []T{single}, nil
}
