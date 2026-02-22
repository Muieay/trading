package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ============================================================
// 账户接口相关结构体定义
// ============================================================

// CommissionRates 手续费率
type CommissionRates struct {
	Maker  string `json:"maker"`
	Taker  string `json:"taker"`
	Buyer  string `json:"buyer"`
	Seller string `json:"seller"`
}

// AccountInfo 账户信息
type AccountInfo struct {
	MakerCommission            int             `json:"makerCommission"`
	TakerCommission            int             `json:"takerCommission"`
	BuyerCommission            int             `json:"buyerCommission"`
	SellerCommission           int             `json:"sellerCommission"`
	CommissionRates            CommissionRates `json:"commissionRates"`
	CanTrade                   bool            `json:"canTrade"`
	CanWithdraw                bool            `json:"canWithdraw"`
	CanDeposit                 bool            `json:"canDeposit"`
	Brokered                   bool            `json:"brokered"`
	RequireSelfTradePrevention bool            `json:"requireSelfTradePrevention"`
	PreventSor                 bool            `json:"preventSor"`
	UpdateTime                 int64           `json:"updateTime"`
	AccountType                string          `json:"accountType"`
	Balances                   []Balance       `json:"balances"`
	Permissions                []string        `json:"permissions"`
	UID                        int64           `json:"uid"`
}

// Balance 资产余额
type Balance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

// Order 订单信息
type Order struct {
	Symbol                  string `json:"symbol"`
	OrderId                 int64  `json:"orderId"`
	OrderListId             int64  `json:"orderListId"`
	ClientOrderId           string `json:"clientOrderId"`
	Price                   string `json:"price"`
	OrigQty                 string `json:"origQty"`
	ExecutedQty             string `json:"executedQty"`
	OrigQuoteOrderQty       string `json:"origQuoteOrderQty"`
	CummulativeQuoteQty     string `json:"cummulativeQuoteQty"`
	Status                  string `json:"status"`
	TimeInForce             string `json:"timeInForce"`
	Type                    string `json:"type"`
	Side                    string `json:"side"`
	StopPrice               string `json:"stopPrice"`
	IcebergQty              string `json:"icebergQty"`
	Time                    int64  `json:"time"`
	UpdateTime              int64  `json:"updateTime"`
	IsWorking               bool   `json:"isWorking"`
	WorkingTime             int64  `json:"workingTime"`
	SelfTradePreventionMode string `json:"selfTradePreventionMode"`
}

// OrderInList 订单列表中的简要订单信息
type OrderInList struct {
	Symbol        string `json:"symbol"`
	OrderId       int64  `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
}

// OrderList 订单列表（OCO 等）
type OrderList struct {
	OrderListId       int64         `json:"orderListId"`
	ContingencyType   string        `json:"contingencyType"`
	ListStatusType    string        `json:"listStatusType"`
	ListOrderStatus   string        `json:"listOrderStatus"`
	ListClientOrderId string        `json:"listClientOrderId"`
	TransactionTime   int64         `json:"transactionTime"`
	Symbol            string        `json:"symbol"`
	Orders            []OrderInList `json:"orders"`
}

// Trade 成交历史记录
type Trade struct {
	Symbol          string `json:"symbol"`
	Id              int64  `json:"id"`
	OrderId         int64  `json:"orderId"`
	OrderListId     int64  `json:"orderListId"`
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	QuoteQty        string `json:"quoteQty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	Time            int64  `json:"time"`
	IsBuyer         bool   `json:"isBuyer"`
	IsMaker         bool   `json:"isMaker"`
	IsBestMatch     bool   `json:"isBestMatch"`
}

// RateLimitOrder 订单速率限制
type RateLimitOrder struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
	Count         int    `json:"count"`
}

// PreventedMatch STP 阻止的撮合记录
type PreventedMatch struct {
	Symbol                  string `json:"symbol"`
	PreventedMatchId        int64  `json:"preventedMatchId"`
	TakerOrderId            int64  `json:"takerOrderId"`
	MakerSymbol             string `json:"makerSymbol"`
	MakerOrderId            int64  `json:"makerOrderId"`
	TradeGroupId            int64  `json:"tradeGroupId"`
	SelfTradePreventionMode string `json:"selfTradePreventionMode"`
	Price                   string `json:"price"`
	MakerPreventedQuantity  string `json:"makerPreventedQuantity"`
	TransactTime            int64  `json:"transactTime"`
}

// Allocation SOR 分配结果
type Allocation struct {
	Symbol          string `json:"symbol"`
	AllocationId    int64  `json:"allocationId"`
	AllocationType  string `json:"allocationType"`
	OrderId         int64  `json:"orderId"`
	OrderListId     int64  `json:"orderListId"`
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	QuoteQty        string `json:"quoteQty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	Time            int64  `json:"time"`
	IsBuyer         bool   `json:"isBuyer"`
	IsMaker         bool   `json:"isMaker"`
	IsAllocator     bool   `json:"isAllocator"`
}

// CommissionDiscount 佣金折扣（使用 BNB 支付）
type CommissionDiscount struct {
	EnabledForAccount bool   `json:"enabledForAccount"`
	EnabledForSymbol  bool   `json:"enabledForSymbol"`
	DiscountAsset     string `json:"discountAsset"`
	Discount          string `json:"discount"`
}

// AccountCommission 账户佣金费率
type AccountCommission struct {
	Symbol             string             `json:"symbol"`
	StandardCommission CommissionRates    `json:"standardCommission"`
	SpecialCommission  CommissionRates    `json:"specialCommission"`
	TaxCommission      CommissionRates    `json:"taxCommission"`
	Discount           CommissionDiscount `json:"discount"`
}

// OrderAmendment 改单记录
type OrderAmendment struct {
	Symbol            string `json:"symbol"`
	OrderId           int64  `json:"orderId"`
	ExecutionId       int64  `json:"executionId"`
	OrigClientOrderId string `json:"origClientOrderId"`
	NewClientOrderId  string `json:"newClientOrderId"`
	OrigQty           string `json:"origQty"`
	NewQty            string `json:"newQty"`
	Time              int64  `json:"time"`
}

// Filter 过滤器（泛型，以 map 存储灵活字段）
type Filter map[string]interface{}

// AccountFilters 账户过滤器
type AccountFilters struct {
	ExchangeFilters []Filter `json:"exchangeFilters"`
	SymbolFilters   []Filter `json:"symbolFilters"`
	AssetFilters    []Filter `json:"assetFilters"`
}

// ============================================================
// 请求参数结构体
// ============================================================

// GetAccountInfoParams 账户信息请求参数
type GetAccountInfoParams struct {
	OmitZeroBalances *bool // 是否隐藏零余额，默认 false
	RecvWindow       *int64
}

// GetOrderParams 查询订单请求参数
type GetOrderParams struct {
	Symbol            string // 必填
	OrderId           *int64
	OrigClientOrderId *string
	RecvWindow        *int64
}

// GetOpenOrdersParams 查询当前挂单请求参数
type GetOpenOrdersParams struct {
	Symbol     *string // 不填则返回全部交易对挂单（权重 80）
	RecvWindow *int64
}

// GetAllOrdersParams 查询所有订单请求参数
type GetAllOrdersParams struct {
	Symbol     string // 必填
	OrderId    *int64
	StartTime  *int64
	EndTime    *int64
	Limit      *int
	RecvWindow *int64
}

// GetOrderListParams 查询订单列表请求参数
type GetOrderListParams struct {
	OrderListId       *int64
	OrigClientOrderId *string
	RecvWindow        *int64
}

// GetAllOrderListParams 查询所有订单列表请求参数
type GetAllOrderListParams struct {
	FromId     *int64
	StartTime  *int64
	EndTime    *int64
	Limit      *int
	RecvWindow *int64
}

// GetMyTradesParams 账户成交历史请求参数
type GetMyTradesParams struct {
	Symbol     string // 必填
	OrderId    *int64
	StartTime  *int64
	EndTime    *int64
	FromId     *int64
	Limit      *int
	RecvWindow *int64
}

// GetRateLimitOrderParams 查询未成交订单计数请求参数
type GetRateLimitOrderParams struct {
	RecvWindow *int64
}

// GetPreventedMatchesParams 查询 Prevented Matches 请求参数
type GetPreventedMatchesParams struct {
	Symbol               string // 必填
	PreventedMatchId     *int64
	OrderId              *int64
	FromPreventedMatchId *int64
	Limit                *int
	RecvWindow           *int64
}

// GetMyAllocationsParams 查询分配结果请求参数
type GetMyAllocationsParams struct {
	Symbol           string // 必填
	StartTime        *int64
	EndTime          *int64
	FromAllocationId *int
	Limit            *int
	OrderId          *int64
	RecvWindow       *int64
}

// GetAccountCommissionParams 查询佣金费率请求参数
type GetAccountCommissionParams struct {
	Symbol string // 必填（无需签名，无需 timestamp）
}

// GetOrderAmendmentsParams 查询改单请求参数
type GetOrderAmendmentsParams struct {
	Symbol          string // 必填
	OrderId         int64  // 必填
	FromExecutionId *int64
	Limit           *int64
	RecvWindow      *int64
}

// GetMyFiltersParams 查询相关过滤器请求参数
type GetMyFiltersParams struct {
	Symbol     string // 必填
	RecvWindow *int64
}

// ============================================================
// 工具函数
// ============================================================

// signedGet 发起带签名的 GET 请求
func (c *BinanceClient) signedGet(endpoint string, params url.Values, result interface{}) error {
	params.Set("timestamp", strconv.FormatInt(getTimestamp(), 10))
	queryString := params.Encode()
	signature := c.sign(queryString)
	queryString += "&signature=" + signature

	fullURL := BaseURL + endpoint + "?" + queryString
	body, statusCode, err := c.doRequest("GET", fullURL, true)
	if err != nil {
		return err
	}
	return parseResponse(body, statusCode, result)
}

// publicGet 发起无需签名的 GET 请求（仍需 API Key）
func (c *BinanceClient) publicGet(endpoint string, params url.Values, result interface{}) error {
	queryString := params.Encode()
	fullURL := BaseURL + endpoint
	if queryString != "" {
		fullURL += "?" + queryString
	}
	body, statusCode, err := c.doRequest("GET", fullURL, true)
	if err != nil {
		return err
	}
	return parseResponse(body, statusCode, result)
}

// doRequest 执行 HTTP 请求，withAPIKey 控制是否携带 X-MBX-APIKEY 头
func (c *BinanceClient) doRequest(method, fullURL string, withAPIKey bool) ([]byte, int, error) {
	req, err := newHTTPRequest(method, fullURL)
	if err != nil {
		return nil, 0, fmt.Errorf("创建请求失败: %w", err)
	}
	if withAPIKey {
		req.Header.Set("X-MBX-APIKEY", c.APIKey)
	}
	return executeRequest(c.HTTPClient, req)
}

// parseResponse 解析 HTTP 响应体
func parseResponse(body []byte, statusCode int, result interface{}) error {
	if statusCode != 200 {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err != nil {
			return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
		}
		return &apiErr
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	return nil
}

// setOptInt64 将可选 int64 参数写入 url.Values
func setOptInt64(params url.Values, key string, v *int64) {
	if v != nil {
		params.Set(key, strconv.FormatInt(*v, 10))
	}
}

// setOptInt 将可选 int 参数写入 url.Values
func setOptInt(params url.Values, key string, v *int) {
	if v != nil {
		params.Set(key, strconv.Itoa(*v))
	}
}

// setOptString 将可选 string 参数写入 url.Values
func setOptString(params url.Values, key string, v *string) {
	if v != nil {
		params.Set(key, *v)
	}
}

// setOptBool 将可选 bool 参数写入 url.Values
func setOptBool(params url.Values, key string, v *bool) {
	if v != nil {
		if *v {
			params.Set(key, "true")
		} else {
			params.Set(key, "false")
		}
	}
}

// ============================================================
// 账户接口实现
// ============================================================

// GetAccountInfo 账户信息 (USER_DATA)
// GET /api/v3/account
// 权重: 20
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#账户信息-user_data
func (c *BinanceClient) GetAccountInfo(p *GetAccountInfoParams) (*AccountInfo, error) {
	params := url.Values{}
	if p != nil {
		setOptBool(params, "omitZeroBalances", p.OmitZeroBalances)
		setOptInt64(params, "recvWindow", p.RecvWindow)
	}
	var result AccountInfo
	if err := c.signedGet("/api/v3/account", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOrder 查询订单 (USER_DATA)
// GET /api/v3/order
// 权重: 4
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查询订单-user_data
// 注意: orderId 与 origClientOrderId 至少提供一个
func (c *BinanceClient) GetOrder(p GetOrderParams) (*Order, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	if p.OrderId == nil && p.OrigClientOrderId == nil {
		return nil, fmt.Errorf("orderId 与 origClientOrderId 至少需要提供一个")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt64(params, "orderId", p.OrderId)
	setOptString(params, "origClientOrderId", p.OrigClientOrderId)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result Order
	if err := c.signedGet("/api/v3/order", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOpenOrders 查看账户当前挂单 (USER_DATA)
// GET /api/v3/openOrders
// 权重: 带 symbol=6，不带 symbol=80
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查看账户当前挂单-user_data
func (c *BinanceClient) GetOpenOrders(p *GetOpenOrdersParams) ([]Order, error) {
	params := url.Values{}
	if p != nil {
		setOptString(params, "symbol", p.Symbol)
		setOptInt64(params, "recvWindow", p.RecvWindow)
	}
	var result []Order
	if err := c.signedGet("/api/v3/openOrders", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetAllOrders 查询所有订单，包含历史订单 (USER_DATA)
// GET /api/v3/allOrders
// 权重: 20
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查询所有订单包括历史订单-user_data
// 注意: startTime 和 endTime 间隔不能超过 24 小时
func (c *BinanceClient) GetAllOrders(p GetAllOrdersParams) ([]Order, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt64(params, "orderId", p.OrderId)
	setOptInt64(params, "startTime", p.StartTime)
	setOptInt64(params, "endTime", p.EndTime)
	setOptInt(params, "limit", p.Limit)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result []Order
	if err := c.signedGet("/api/v3/allOrders", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetOrderList 查询订单列表 (USER_DATA)
// GET /api/v3/orderList
// 权重: 4
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查询订单列表-user_data
// 注意: orderListId 与 origClientOrderId 至少提供一个
func (c *BinanceClient) GetOrderList(p GetOrderListParams) (*OrderList, error) {
	if p.OrderListId == nil && p.OrigClientOrderId == nil {
		return nil, fmt.Errorf("orderListId 与 origClientOrderId 至少需要提供一个")
	}
	params := url.Values{}
	setOptInt64(params, "orderListId", p.OrderListId)
	setOptString(params, "origClientOrderId", p.OrigClientOrderId)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result OrderList
	if err := c.signedGet("/api/v3/orderList", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAllOrderList 查询所有订单列表 (USER_DATA)
// GET /api/v3/allOrderList
// 权重: 20
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查询所有订单列表-user_data
// 注意: 提供 fromId 时不可同时提供 startTime/endTime；时间范围不超过 24 小时
func (c *BinanceClient) GetAllOrderList(p *GetAllOrderListParams) ([]OrderList, error) {
	params := url.Values{}
	if p != nil {
		setOptInt64(params, "fromId", p.FromId)
		setOptInt64(params, "startTime", p.StartTime)
		setOptInt64(params, "endTime", p.EndTime)
		setOptInt(params, "limit", p.Limit)
		setOptInt64(params, "recvWindow", p.RecvWindow)
	}
	var result []OrderList
	if err := c.signedGet("/api/v3/allOrderList", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetOpenOrderList 查询订单列表挂单 (USER_DATA)
// GET /api/v3/openOrderList
// 权重: 6
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查询订单列表挂单-user_data
func (c *BinanceClient) GetOpenOrderList(p *GetRateLimitOrderParams) ([]OrderList, error) {
	params := url.Values{}
	if p != nil {
		setOptInt64(params, "recvWindow", p.RecvWindow)
	}
	var result []OrderList
	if err := c.signedGet("/api/v3/openOrderList", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMyTrades 账户成交历史 (USER_DATA)
// GET /api/v3/myTrades
// 权重: 有 orderId=5，无 orderId=20
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#账户成交历史-user_data
// 注意: startTime 和 endTime 间隔不超过 24 小时
func (c *BinanceClient) GetMyTrades(p GetMyTradesParams) ([]Trade, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt64(params, "orderId", p.OrderId)
	setOptInt64(params, "startTime", p.StartTime)
	setOptInt64(params, "endTime", p.EndTime)
	setOptInt64(params, "fromId", p.FromId)
	setOptInt(params, "limit", p.Limit)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result []Trade
	if err := c.signedGet("/api/v3/myTrades", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetRateLimitOrder 查询未成交的订单计数 (USER_DATA)
// GET /api/v3/rateLimit/order
// 权重: 40
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查询未成交的订单计数-user_data
func (c *BinanceClient) GetRateLimitOrder(p *GetRateLimitOrderParams) ([]RateLimitOrder, error) {
	params := url.Values{}
	if p != nil {
		setOptInt64(params, "recvWindow", p.RecvWindow)
	}
	var result []RateLimitOrder
	if err := c.signedGet("/api/v3/rateLimit/order", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPreventedMatches 获取 Prevented Matches (USER_DATA)
// GET /api/v3/myPreventedMatches
// 权重: symbol 无效=2，通过 preventedMatchId 查询=2，通过 orderId 查询=20
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#获取-prevented-matches-user_data
// 支持的组合:
//   - symbol + preventedMatchId
//   - symbol + orderId
//   - symbol + orderId + fromPreventedMatchId
//   - symbol + orderId + fromPreventedMatchId + limit
func (c *BinanceClient) GetPreventedMatches(p GetPreventedMatchesParams) ([]PreventedMatch, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt64(params, "preventedMatchId", p.PreventedMatchId)
	setOptInt64(params, "orderId", p.OrderId)
	setOptInt64(params, "fromPreventedMatchId", p.FromPreventedMatchId)
	setOptInt(params, "limit", p.Limit)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result []PreventedMatch
	if err := c.signedGet("/api/v3/myPreventedMatches", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMyAllocations 查询分配结果 (USER_DATA)
// GET /api/v3/myAllocations
// 权重: 20
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查询分配结果-user_data
// 注意: startTime 和 endTime 间隔不超过 24 小时
func (c *BinanceClient) GetMyAllocations(p GetMyAllocationsParams) ([]Allocation, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt64(params, "startTime", p.StartTime)
	setOptInt64(params, "endTime", p.EndTime)
	setOptInt(params, "fromAllocationId", p.FromAllocationId)
	setOptInt(params, "limit", p.Limit)
	setOptInt64(params, "orderId", p.OrderId)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result []Allocation
	if err := c.signedGet("/api/v3/myAllocations", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetAccountCommission 查询佣金费率 (USER_DATA)
// GET /api/v3/account/commission
// 权重: 20
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查询佣金费率-user_data
// 注意: 此接口无需 timestamp，但仍需要 API Key
func (c *BinanceClient) GetAccountCommission(p GetAccountCommissionParams) (*AccountCommission, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)

	var result AccountCommission
	if err := c.publicGet("/api/v3/account/commission", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetOrderAmendments 查询改单 (USER_DATA)
// GET /api/v3/order/amendments
// 权重: 4
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查询改单-user_data
func (c *BinanceClient) GetOrderAmendments(p GetOrderAmendmentsParams) ([]OrderAmendment, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	if p.OrderId == 0 {
		return nil, fmt.Errorf("orderId 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("orderId", strconv.FormatInt(p.OrderId, 10))
	setOptInt64(params, "fromExecutionId", p.FromExecutionId)
	setOptInt64(params, "limit", p.Limit)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result []OrderAmendment
	if err := c.signedGet("/api/v3/order/amendments", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMyFilters 查询相关过滤器 (USER_DATA)
// GET /api/v3/myFilters
// 权重: 40
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/account-endpoints#查询相关过滤器-user_data
func (c *BinanceClient) GetMyFilters(p GetMyFiltersParams) (*AccountFilters, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result AccountFilters
	if err := c.signedGet("/api/v3/myFilters", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
