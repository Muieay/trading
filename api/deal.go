package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ============================================================
// 交易接口相关结构体
// ============================================================

// OrderFill 成交明细（FULL 响应）
type OrderFill struct {
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	TradeId         int64  `json:"tradeId"`
	MatchType       string `json:"matchType,omitempty"` // SOR 专属
	AllocId         int64  `json:"allocId,omitempty"`   // SOR 专属
}

// OrderResponse 下单/撤单通用响应（FULL 格式，ACK/RESULT 为其子集）
type OrderResponse struct {
	Symbol                  string      `json:"symbol"`
	OrderId                 int64       `json:"orderId"`
	OrderListId             int64       `json:"orderListId"`
	ClientOrderId           string      `json:"clientOrderId"`
	OrigClientOrderId       string      `json:"origClientOrderId,omitempty"`
	TransactTime            int64       `json:"transactTime"`
	Price                   string      `json:"price,omitempty"`
	OrigQty                 string      `json:"origQty,omitempty"`
	ExecutedQty             string      `json:"executedQty,omitempty"`
	OrigQuoteOrderQty       string      `json:"origQuoteOrderQty,omitempty"`
	CummulativeQuoteQty     string      `json:"cummulativeQuoteQty,omitempty"`
	Status                  string      `json:"status,omitempty"`
	TimeInForce             string      `json:"timeInForce,omitempty"`
	Type                    string      `json:"type,omitempty"`
	Side                    string      `json:"side,omitempty"`
	WorkingTime             int64       `json:"workingTime,omitempty"`
	SelfTradePreventionMode string      `json:"selfTradePreventionMode,omitempty"`
	Fills                   []OrderFill `json:"fills,omitempty"`
	// 条件单字段
	StopPrice     string `json:"stopPrice,omitempty"`
	IcebergQty    string `json:"icebergQty,omitempty"`
	StrategyId    int64  `json:"strategyId,omitempty"`
	StrategyType  int    `json:"strategyType,omitempty"`
	TrailingDelta int64  `json:"trailingDelta,omitempty"`
	TrailingTime  int64  `json:"trailingTime,omitempty"`
	// SOR 专属字段
	WorkingFloor string `json:"workingFloor,omitempty"`
	UsedSor      bool   `json:"usedSor,omitempty"`
	// 挂钩订单字段
	PegPriceType   string `json:"pegPriceType,omitempty"`
	PegOffsetType  string `json:"pegOffsetType,omitempty"`
	PegOffsetValue int    `json:"pegOffsetValue,omitempty"`
	PeggedPrice    string `json:"peggedPrice,omitempty"`
}

// OrderReport 订单列表中的单个订单报告
type OrderReport struct {
	Symbol                  string `json:"symbol"`
	OrderId                 int64  `json:"orderId"`
	OrderListId             int64  `json:"orderListId"`
	ClientOrderId           string `json:"clientOrderId"`
	OrigClientOrderId       string `json:"origClientOrderId,omitempty"`
	TransactTime            int64  `json:"transactTime"`
	Price                   string `json:"price"`
	OrigQty                 string `json:"origQty"`
	ExecutedQty             string `json:"executedQty"`
	OrigQuoteOrderQty       string `json:"origQuoteOrderQty"`
	CummulativeQuoteQty     string `json:"cummulativeQuoteQty"`
	Status                  string `json:"status"`
	TimeInForce             string `json:"timeInForce"`
	Type                    string `json:"type"`
	Side                    string `json:"side"`
	StopPrice               string `json:"stopPrice,omitempty"`
	IcebergQty              string `json:"icebergQty,omitempty"`
	WorkingTime             int64  `json:"workingTime"`
	SelfTradePreventionMode string `json:"selfTradePreventionMode"`
}

// OrderListResponse 订单列表响应（OCO/OTO/OTOCO/OPO/OPOCO）
type OrderListResponse struct {
	OrderListId       int64         `json:"orderListId"`
	ContingencyType   string        `json:"contingencyType"`
	ListStatusType    string        `json:"listStatusType"`
	ListOrderStatus   string        `json:"listOrderStatus"`
	ListClientOrderId string        `json:"listClientOrderId"`
	TransactionTime   int64         `json:"transactionTime"`
	Symbol            string        `json:"symbol"`
	Orders            []OrderInList `json:"orders"`
	OrderReports      []OrderReport `json:"orderReports"`
}

// AmendedOrderListStatus 改单涉及的订单列表状态
type AmendedOrderListStatus struct {
	OrderListId       int64         `json:"orderListId"`
	ContingencyType   string        `json:"contingencyType"`
	ListOrderStatus   string        `json:"listOrderStatus"`
	ListClientOrderId string        `json:"listClientOrderId"`
	Symbol            string        `json:"symbol"`
	Orders            []OrderInList `json:"orders"`
}

// AmendOrderResponse 修改订单并保留优先级的响应
type AmendOrderResponse struct {
	TransactTime int64                   `json:"transactTime"`
	ExecutionId  int64                   `json:"executionId"`
	AmendedOrder OrderReport             `json:"amendedOrder"`
	ListStatus   *AmendedOrderListStatus `json:"listStatus,omitempty"`
}

// CancelReplaceResponse 撤单再下单响应
type CancelReplaceResponse struct {
	CancelResult     string         `json:"cancelResult"`
	NewOrderResult   string         `json:"newOrderResult"`
	CancelResponse   *OrderResponse `json:"cancelResponse"`
	NewOrderResponse *OrderResponse `json:"newOrderResponse"`
}

// TestOrderCommissionRates 测试下单时的佣金率（computeCommissionRates=true 时返回）
type TestOrderCommissionRates struct {
	StandardCommissionForOrder CommissionRates    `json:"standardCommissionForOrder"`
	SpecialCommissionForOrder  CommissionRates    `json:"specialCommissionForOrder"`
	TaxCommissionForOrder      CommissionRates    `json:"taxCommissionForOrder"`
	Discount                   CommissionDiscount `json:"discount"`
}

// ============================================================
// 请求参数结构体
// ============================================================

// PlaceOrderParams 下单参数 POST /api/v3/order
type PlaceOrderParams struct {
	Symbol                  string  // 必填
	Side                    string  // 必填 BUY/SELL
	Type                    string  // 必填 LIMIT/MARKET/STOP_LOSS 等
	TimeInForce             *string // LIMIT 等必填
	Quantity                *string
	QuoteOrderQty           *string
	Price                   *string
	NewClientOrderId        *string
	StrategyId              *int64
	StrategyType            *int
	StopPrice               *string
	TrailingDelta           *int64
	IcebergQty              *string
	NewOrderRespType        *string // ACK/RESULT/FULL
	SelfTradePreventionMode *string
	PegPriceType            *string
	PegOffsetValue          *int
	PegOffsetType           *string
	RecvWindow              *int64
}

// TestOrderParams 测试下单参数 POST /api/v3/order/test
type TestOrderParams struct {
	PlaceOrderParams
	ComputeCommissionRates *bool
}

// CancelOrderParams 撤销订单参数 DELETE /api/v3/order
type CancelOrderParams struct {
	Symbol             string // 必填
	OrderId            *int64
	OrigClientOrderId  *string
	NewClientOrderId   *string
	CancelRestrictions *string // ONLY_NEW / ONLY_PARTIALLY_FILLED
	RecvWindow         *int64
}

// CancelOpenOrdersParams 撤销交易对全部挂单参数 DELETE /api/v3/openOrders
type CancelOpenOrdersParams struct {
	Symbol     string // 必填
	RecvWindow *int64
}

// CancelReplaceParams 撤单再下单参数 POST /api/v3/order/cancelReplace
type CancelReplaceParams struct {
	Symbol                     string // 必填
	Side                       string // 必填
	Type                       string // 必填
	CancelReplaceMode          string // 必填 STOP_ON_FAILURE/ALLOW_FAILURE
	CancelOrderId              *int64
	CancelOrigClientOrderId    *string
	CancelNewClientOrderId     *string
	TimeInForce                *string
	Quantity                   *string
	QuoteOrderQty              *string
	Price                      *string
	NewClientOrderId           *string
	StrategyId                 *int64
	StrategyType               *int
	StopPrice                  *string
	TrailingDelta              *int64
	IcebergQty                 *string
	NewOrderRespType           *string
	SelfTradePreventionMode    *string
	CancelRestrictions         *string
	OrderRateLimitExceededMode *string // DO_NOTHING/CANCEL_ONLY
	PegPriceType               *string
	PegOffsetValue             *int
	PegOffsetType              *string
	RecvWindow                 *int64
}

// AmendOrderParams 修改订单并保留优先级参数 PUT /api/v3/order/amend/keepPriority
type AmendOrderParams struct {
	Symbol            string // 必填
	NewQty            string // 必填，必须 > 0 且 < 原始数量
	OrderId           *int64 // orderId 与 origClientOrderId 至少提供一个
	OrigClientOrderId *string
	NewClientOrderId  *string
	RecvWindow        *int64
}

// PlaceOCOLegacyParams 发送旧版 OCO 订单参数 POST /api/v3/order/oco（已弃用）
type PlaceOCOLegacyParams struct {
	Symbol                  string // 必填
	Side                    string // 必填
	Quantity                string // 必填
	Price                   string // 必填（限价腿）
	StopPrice               string // 必填（止损腿）
	ListClientOrderId       *string
	LimitClientOrderId      *string
	LimitStrategyId         *int64
	LimitStrategyType       *int
	LimitIcebergQty         *string
	TrailingDelta           *int64
	StopClientOrderId       *string
	StopStrategyId          *int64
	StopStrategyType        *int
	StopLimitPrice          *string
	StopIcebergQty          *string
	StopLimitTimeInForce    *string // GTC/FOK/IOC
	NewOrderRespType        *string
	SelfTradePreventionMode *string
	RecvWindow              *int64
}

// OCOLeg 新版 OCO 中的单腿配置（above/below 腿共用）
type OCOLeg struct {
	Type           string // 必填（aboveType/belowType）
	ClientOrderId  *string
	IcebergQty     *int64
	Price          *string
	StopPrice      *string
	TrailingDelta  *int64
	TimeInForce    *string
	StrategyId     *int64
	StrategyType   *int
	PegPriceType   *string
	PegOffsetType  *string
	PegOffsetValue *int
}

// PlaceOCOParams 新版 OCO 订单参数 POST /api/v3/orderList/oco
type PlaceOCOParams struct {
	Symbol                  string // 必填
	Side                    string // 必填 BUY/SELL
	Quantity                string // 必填
	AboveLeg                OCOLeg // 必填（上方腿）
	BelowLeg                OCOLeg // 必填（下方腿）
	ListClientOrderId       *string
	AboveClientOrderId      *string
	BelowClientOrderId      *string
	NewOrderRespType        *string
	SelfTradePreventionMode *string
	RecvWindow              *int64
}

// WorkingOrder OTO/OTOCO/OPO/OPOCO 中的生效订单配置
type WorkingOrder struct {
	Type           string // 必填 LIMIT/LIMIT_MAKER
	Side           string // 必填
	Price          string // 必填
	Quantity       string // 必填
	ClientOrderId  *string
	IcebergQty     *string
	TimeInForce    *string
	StrategyId     *int64
	StrategyType   *int
	PegPriceType   *string
	PegOffsetType  *string
	PegOffsetValue *int
}

// PendingOrder OTO 中的待处理订单配置
type PendingOrder struct {
	Type           string // 必填
	Side           string // 必填
	Quantity       string // 必填
	ClientOrderId  *string
	Price          *string
	StopPrice      *string
	TrailingDelta  *string
	IcebergQty     *string
	TimeInForce    *string
	StrategyId     *int64
	StrategyType   *int
	PegPriceType   *string
	PegOffsetType  *string
	PegOffsetValue *int
}

// PlaceOTOParams OTO 订单参数 POST /api/v3/orderList/oto
type PlaceOTOParams struct {
	Symbol                  string       // 必填
	Working                 WorkingOrder // 必填
	Pending                 PendingOrder // 必填
	ListClientOrderId       *string
	NewOrderRespType        *string
	SelfTradePreventionMode *string
	RecvWindow              *int64
}

// PlaceOTOCOParams OTOCO 订单参数 POST /api/v3/orderList/otoco
type PlaceOTOCOParams struct {
	Symbol                    string       // 必填
	Working                   WorkingOrder // 必填
	PendingSide               string       // 必填
	PendingQuantity           string       // 必填
	PendingAbove              OCOLeg       // 必填（上方待处理腿）
	PendingBelow              OCOLeg       // 可选（下方待处理腿）
	PendingAboveClientOrderId *string
	PendingBelowClientOrderId *string
	ListClientOrderId         *string
	NewOrderRespType          *string
	SelfTradePreventionMode   *string
	RecvWindow                *int64
}

// PlaceOPOParams OPO 订单参数 POST /api/v3/orderList/opo
type PlaceOPOParams struct {
	Symbol                  string       // 必填
	Working                 WorkingOrder // 必填
	Pending                 PendingOrder // 必填
	ListClientOrderId       *string
	NewOrderRespType        *string
	SelfTradePreventionMode *string
	RecvWindow              *int64
}

// PlaceOPOCOParams OPOCO 订单参数 POST /api/v3/orderList/opoco
type PlaceOPOCOParams struct {
	Symbol                    string       // 必填
	Working                   WorkingOrder // 必填
	PendingSide               string       // 必填
	PendingAbove              OCOLeg       // 必填（上方待处理腿）
	PendingBelow              OCOLeg       // 可选（下方待处理腿）
	PendingAboveClientOrderId *string
	PendingBelowClientOrderId *string
	ListClientOrderId         *string
	NewOrderRespType          *string
	SelfTradePreventionMode   *string
	RecvWindow                *int64
}

// CancelOrderListParams 取消订单列表参数 DELETE /api/v3/orderList
type CancelOrderListParams struct {
	Symbol            string // 必填
	OrderListId       *int64
	ListClientOrderId *string
	NewClientOrderId  *string
	RecvWindow        *int64
}

// PlaceSOROrderParams 下 SOR 订单参数 POST /api/v3/sor/order
type PlaceSOROrderParams struct {
	Symbol                  string // 必填
	Side                    string // 必填
	Type                    string // 必填 LIMIT/MARKET
	Quantity                string // 必填
	TimeInForce             *string
	Price                   *string
	NewClientOrderId        *string
	StrategyId              *int64
	StrategyType            *int
	IcebergQty              *string
	NewOrderRespType        *string
	SelfTradePreventionMode *string
	RecvWindow              *int64
}

// TestSOROrderParams 测试 SOR 下单参数 POST /api/v3/sor/order/test
type TestSOROrderParams struct {
	PlaceSOROrderParams
	ComputeCommissionRates *bool
}

// ============================================================
// 工具函数：构建 working/pending 订单公共参数
// ============================================================

func applyWorkingOrder(params url.Values, prefix string, w WorkingOrder) {
	params.Set(prefix+"Type", w.Type)
	params.Set(prefix+"Side", w.Side)
	params.Set(prefix+"Price", w.Price)
	params.Set(prefix+"Quantity", w.Quantity)
	setOptString(params, prefix+"ClientOrderId", w.ClientOrderId)
	setOptString(params, prefix+"IcebergQty", w.IcebergQty)
	setOptString(params, prefix+"TimeInForce", w.TimeInForce)
	setOptInt64(params, prefix+"StrategyId", w.StrategyId)
	if w.StrategyType != nil {
		params.Set(prefix+"StrategyType", strconv.Itoa(*w.StrategyType))
	}
	setOptString(params, prefix+"PegPriceType", w.PegPriceType)
	setOptString(params, prefix+"PegOffsetType", w.PegOffsetType)
	if w.PegOffsetValue != nil {
		params.Set(prefix+"PegOffsetValue", strconv.Itoa(*w.PegOffsetValue))
	}
}

func applyPendingOrder(params url.Values, prefix string, p PendingOrder) {
	params.Set(prefix+"Type", p.Type)
	params.Set(prefix+"Side", p.Side)
	params.Set(prefix+"Quantity", p.Quantity)
	setOptString(params, prefix+"ClientOrderId", p.ClientOrderId)
	setOptString(params, prefix+"Price", p.Price)
	setOptString(params, prefix+"StopPrice", p.StopPrice)
	setOptString(params, prefix+"TrailingDelta", p.TrailingDelta)
	setOptString(params, prefix+"IcebergQty", p.IcebergQty)
	setOptString(params, prefix+"TimeInForce", p.TimeInForce)
	setOptInt64(params, prefix+"StrategyId", p.StrategyId)
	if p.StrategyType != nil {
		params.Set(prefix+"StrategyType", strconv.Itoa(*p.StrategyType))
	}
	setOptString(params, prefix+"PegPriceType", p.PegPriceType)
	setOptString(params, prefix+"PegOffsetType", p.PegOffsetType)
	if p.PegOffsetValue != nil {
		params.Set(prefix+"PegOffsetValue", strconv.Itoa(*p.PegOffsetValue))
	}
}

func applyOCOLeg(params url.Values, prefix string, leg OCOLeg) {
	params.Set(prefix+"Type", leg.Type)
	setOptString(params, prefix+"ClientOrderId", leg.ClientOrderId)
	if leg.IcebergQty != nil {
		params.Set(prefix+"IcebergQty", strconv.FormatInt(*leg.IcebergQty, 10))
	}
	setOptString(params, prefix+"Price", leg.Price)
	setOptString(params, prefix+"StopPrice", leg.StopPrice)
	if leg.TrailingDelta != nil {
		params.Set(prefix+"TrailingDelta", strconv.FormatInt(*leg.TrailingDelta, 10))
	}
	setOptString(params, prefix+"TimeInForce", leg.TimeInForce)
	setOptInt64(params, prefix+"StrategyId", leg.StrategyId)
	if leg.StrategyType != nil {
		params.Set(prefix+"StrategyType", strconv.Itoa(*leg.StrategyType))
	}
	setOptString(params, prefix+"PegPriceType", leg.PegPriceType)
	setOptString(params, prefix+"PegOffsetType", leg.PegOffsetType)
	if leg.PegOffsetValue != nil {
		params.Set(prefix+"PegOffsetValue", strconv.Itoa(*leg.PegOffsetValue))
	}
}

// signedPost 发起带签名的 POST 请求
func (c *BinanceClient) signedPost(endpoint string, params url.Values, result interface{}) error {
	params.Set("timestamp", strconv.FormatInt(getTimestamp(), 10))
	queryString := params.Encode()
	signature := c.sign(queryString)
	queryString += "&signature=" + signature

	fullURL := BaseURL + endpoint + "?" + queryString
	body, statusCode, err := c.doRequest("POST", fullURL, true)
	if err != nil {
		return err
	}
	return parseResponse(body, statusCode, result)
}

// signedPut 发起带签名的 PUT 请求
func (c *BinanceClient) signedPut(endpoint string, params url.Values, result interface{}) error {
	params.Set("timestamp", strconv.FormatInt(getTimestamp(), 10))
	queryString := params.Encode()
	signature := c.sign(queryString)
	queryString += "&signature=" + signature

	fullURL := BaseURL + endpoint + "?" + queryString
	body, statusCode, err := c.doRequest("PUT", fullURL, true)
	if err != nil {
		return err
	}
	return parseResponse(body, statusCode, result)
}

// signedDelete 发起带签名的 DELETE 请求
func (c *BinanceClient) signedDelete(endpoint string, params url.Values, result interface{}) error {
	params.Set("timestamp", strconv.FormatInt(getTimestamp(), 10))
	queryString := params.Encode()
	signature := c.sign(queryString)
	queryString += "&signature=" + signature

	fullURL := BaseURL + endpoint + "?" + queryString
	body, statusCode, err := c.doRequest("DELETE", fullURL, true)
	if err != nil {
		return err
	}
	return parseResponse(body, statusCode, result)
}

// ============================================================
// 交易接口实现
// ============================================================

// PlaceOrder 下单 (TRADE)
// POST /api/v3/order
// 权重: 1 | 未成交订单计数: 1
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#下单-trade
// 响应类型由 NewOrderRespType 控制：ACK(最快) / RESULT / FULL(默认，含成交明细)
func (c *BinanceClient) PlaceOrder(p PlaceOrderParams) (*OrderResponse, error) {
	if p.Symbol == "" || p.Side == "" || p.Type == "" {
		return nil, fmt.Errorf("symbol、side、type 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("side", p.Side)
	params.Set("type", p.Type)
	setOptString(params, "timeInForce", p.TimeInForce)
	setOptString(params, "quantity", p.Quantity)
	setOptString(params, "quoteOrderQty", p.QuoteOrderQty)
	setOptString(params, "price", p.Price)
	setOptString(params, "newClientOrderId", p.NewClientOrderId)
	setOptInt64(params, "strategyId", p.StrategyId)
	if p.StrategyType != nil {
		params.Set("strategyType", strconv.Itoa(*p.StrategyType))
	}
	setOptString(params, "stopPrice", p.StopPrice)
	setOptInt64(params, "trailingDelta", p.TrailingDelta)
	setOptString(params, "icebergQty", p.IcebergQty)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptString(params, "pegPriceType", p.PegPriceType)
	setOptString(params, "pegOffsetType", p.PegOffsetType)
	if p.PegOffsetValue != nil {
		params.Set("pegOffsetValue", strconv.Itoa(*p.PegOffsetValue))
	}
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result OrderResponse
	if err := c.signedPost("/api/v3/order", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TestOrder 测试下单接口 (TRADE)
// POST /api/v3/order/test
// 权重: 无 computeCommissionRates=1，有=20
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#测试下单接口-trade
// 说明: 不会真实提交到撮合引擎。返回 {} 或佣金信息（当 computeCommissionRates=true）
func (c *BinanceClient) TestOrder(p TestOrderParams) (*TestOrderCommissionRates, error) {
	if p.Symbol == "" || p.Side == "" || p.Type == "" {
		return nil, fmt.Errorf("symbol、side、type 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("side", p.Side)
	params.Set("type", p.Type)
	setOptString(params, "timeInForce", p.TimeInForce)
	setOptString(params, "quantity", p.Quantity)
	setOptString(params, "quoteOrderQty", p.QuoteOrderQty)
	setOptString(params, "price", p.Price)
	setOptString(params, "newClientOrderId", p.NewClientOrderId)
	setOptInt64(params, "strategyId", p.StrategyId)
	if p.StrategyType != nil {
		params.Set("strategyType", strconv.Itoa(*p.StrategyType))
	}
	setOptString(params, "stopPrice", p.StopPrice)
	setOptInt64(params, "trailingDelta", p.TrailingDelta)
	setOptString(params, "icebergQty", p.IcebergQty)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptBool(params, "computeCommissionRates", p.ComputeCommissionRates)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	// 返回 {} 时直接忽略（不设置 computeCommissionRates 时）
	var raw json.RawMessage
	if err := c.signedPost("/api/v3/order/test", params, &raw); err != nil {
		return nil, err
	}
	if string(raw) == "{}" || len(raw) == 0 {
		return nil, nil
	}
	var result TestOrderCommissionRates
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("解析测试下单响应失败: %w", err)
	}
	return &result, nil
}

// CancelOrder 撤销订单 (TRADE)
// DELETE /api/v3/order
// 权重: 1
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#撤销订单-trade
// 注意: orderId 与 origClientOrderId 至少提供一个
func (c *BinanceClient) CancelOrder(p CancelOrderParams) (*OrderResponse, error) {
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
	setOptString(params, "newClientOrderId", p.NewClientOrderId)
	setOptString(params, "cancelRestrictions", p.CancelRestrictions)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result OrderResponse
	if err := c.signedDelete("/api/v3/order", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CancelOpenOrders 撤销单一交易对的所有挂单 (TRADE)
// DELETE /api/v3/openOrders
// 权重: 1
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#撤销单一交易对的所有挂单-trade
// 说明: 也会撤销来自订单列表（OCO等）的挂单，返回混合列表（普通订单 + 订单列表）
func (c *BinanceClient) CancelOpenOrders(p CancelOpenOrdersParams) ([]json.RawMessage, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result []json.RawMessage
	if err := c.signedDelete("/api/v3/openOrders", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CancelAndReplaceOrder 撤消挂单再下单 (TRADE)
// POST /api/v3/order/cancelReplace
// 权重: 1 | 未成交订单计数: 1
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#撤消挂单再下单-trade
// 注意: cancelOrderId 与 cancelOrigClientOrderId 至少提供一个
func (c *BinanceClient) CancelAndReplaceOrder(p CancelReplaceParams) (*CancelReplaceResponse, error) {
	if p.Symbol == "" || p.Side == "" || p.Type == "" || p.CancelReplaceMode == "" {
		return nil, fmt.Errorf("symbol、side、type、cancelReplaceMode 为必填参数")
	}
	if p.CancelOrderId == nil && p.CancelOrigClientOrderId == nil {
		return nil, fmt.Errorf("cancelOrderId 与 cancelOrigClientOrderId 至少需要提供一个")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("side", p.Side)
	params.Set("type", p.Type)
	params.Set("cancelReplaceMode", p.CancelReplaceMode)
	setOptInt64(params, "cancelOrderId", p.CancelOrderId)
	setOptString(params, "cancelOrigClientOrderId", p.CancelOrigClientOrderId)
	setOptString(params, "cancelNewClientOrderId", p.CancelNewClientOrderId)
	setOptString(params, "timeInForce", p.TimeInForce)
	setOptString(params, "quantity", p.Quantity)
	setOptString(params, "quoteOrderQty", p.QuoteOrderQty)
	setOptString(params, "price", p.Price)
	setOptString(params, "newClientOrderId", p.NewClientOrderId)
	setOptInt64(params, "strategyId", p.StrategyId)
	if p.StrategyType != nil {
		params.Set("strategyType", strconv.Itoa(*p.StrategyType))
	}
	setOptString(params, "stopPrice", p.StopPrice)
	setOptInt64(params, "trailingDelta", p.TrailingDelta)
	setOptString(params, "icebergQty", p.IcebergQty)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptString(params, "cancelRestrictions", p.CancelRestrictions)
	setOptString(params, "orderRateLimitExceededMode", p.OrderRateLimitExceededMode)
	setOptString(params, "pegPriceType", p.PegPriceType)
	setOptString(params, "pegOffsetType", p.PegOffsetType)
	if p.PegOffsetValue != nil {
		params.Set("pegOffsetValue", strconv.Itoa(*p.PegOffsetValue))
	}
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result CancelReplaceResponse
	if err := c.signedPost("/api/v3/order/cancelReplace", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// AmendOrderKeepPriority 修改订单并保留优先级 (TRADE)
// PUT /api/v3/order/amend/keepPriority
// 权重: 4 | 未成交订单计数: 0
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#修改订单并保留优先级-trade
// 说明: 用于减少现有订单的数量，不会影响订单在队列中的优先级
func (c *BinanceClient) AmendOrderKeepPriority(p AmendOrderParams) (*AmendOrderResponse, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	if p.NewQty == "" {
		return nil, fmt.Errorf("newQty 为必填参数")
	}
	if p.OrderId == nil && p.OrigClientOrderId == nil {
		return nil, fmt.Errorf("orderId 与 origClientOrderId 至少需要提供一个")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("newQty", p.NewQty)
	setOptInt64(params, "orderId", p.OrderId)
	setOptString(params, "origClientOrderId", p.OrigClientOrderId)
	setOptString(params, "newClientOrderId", p.NewClientOrderId)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result AmendOrderResponse
	if err := c.signedPut("/api/v3/order/amend/keepPriority", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PlaceOCOLegacy 发送旧版 OCO 订单 - 已弃用 (TRADE)
// POST /api/v3/order/oco
// 权重: 1 | 未成交订单计数: 2
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#发送新-oco-订单---已弃用-trade
// 说明: 已弃用，推荐使用 PlaceOCO（POST /api/v3/orderList/oco）
func (c *BinanceClient) PlaceOCOLegacy(p PlaceOCOLegacyParams) (*OrderListResponse, error) {
	if p.Symbol == "" || p.Side == "" || p.Quantity == "" || p.Price == "" || p.StopPrice == "" {
		return nil, fmt.Errorf("symbol、side、quantity、price、stopPrice 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("side", p.Side)
	params.Set("quantity", p.Quantity)
	params.Set("price", p.Price)
	params.Set("stopPrice", p.StopPrice)
	setOptString(params, "listClientOrderId", p.ListClientOrderId)
	setOptString(params, "limitClientOrderId", p.LimitClientOrderId)
	setOptInt64(params, "limitStrategyId", p.LimitStrategyId)
	if p.LimitStrategyType != nil {
		params.Set("limitStrategyType", strconv.Itoa(*p.LimitStrategyType))
	}
	setOptString(params, "limitIcebergQty", p.LimitIcebergQty)
	setOptInt64(params, "trailingDelta", p.TrailingDelta)
	setOptString(params, "stopClientOrderId", p.StopClientOrderId)
	setOptInt64(params, "stopStrategyId", p.StopStrategyId)
	if p.StopStrategyType != nil {
		params.Set("stopStrategyType", strconv.Itoa(*p.StopStrategyType))
	}
	setOptString(params, "stopLimitPrice", p.StopLimitPrice)
	setOptString(params, "stopIcebergQty", p.StopIcebergQty)
	setOptString(params, "stopLimitTimeInForce", p.StopLimitTimeInForce)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result OrderListResponse
	if err := c.signedPost("/api/v3/order/oco", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PlaceOCO 发送新 OCO 订单 (TRADE)
// POST /api/v3/orderList/oco
// 权重: 1 | 未成交订单计数: 2
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#new-order-list---oco-trade
// 说明: 上方腿支持 STOP_LOSS_LIMIT/STOP_LOSS/LIMIT_MAKER/TAKE_PROFIT/TAKE_PROFIT_LIMIT
//
//	下方腿支持 STOP_LOSS/STOP_LOSS_LIMIT/TAKE_PROFIT/TAKE_PROFIT_LIMIT
func (c *BinanceClient) PlaceOCO(p PlaceOCOParams) (*OrderListResponse, error) {
	if p.Symbol == "" || p.Side == "" || p.Quantity == "" {
		return nil, fmt.Errorf("symbol、side、quantity 为必填参数")
	}
	if p.AboveLeg.Type == "" || p.BelowLeg.Type == "" {
		return nil, fmt.Errorf("aboveType 与 belowType 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("side", p.Side)
	params.Set("quantity", p.Quantity)
	setOptString(params, "listClientOrderId", p.ListClientOrderId)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptInt64(params, "recvWindow", p.RecvWindow)
	applyOCOLeg(params, "above", p.AboveLeg)
	applyOCOLeg(params, "below", p.BelowLeg)

	var result OrderListResponse
	if err := c.signedPost("/api/v3/orderList/oco", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PlaceOTO 发送 OTO 订单 (TRADE)
// POST /api/v3/orderList/oto
// 权重: 1 | 未成交订单计数: 2
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#new-order-list---oto-trade
// 说明: 生效订单完全成交后，待处理订单才会自动提交
func (c *BinanceClient) PlaceOTO(p PlaceOTOParams) (*OrderListResponse, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	if p.Working.Type == "" || p.Working.Side == "" || p.Working.Price == "" || p.Working.Quantity == "" {
		return nil, fmt.Errorf("working 订单的 type、side、price、quantity 为必填参数")
	}
	if p.Pending.Type == "" || p.Pending.Side == "" || p.Pending.Quantity == "" {
		return nil, fmt.Errorf("pending 订单的 type、side、quantity 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptString(params, "listClientOrderId", p.ListClientOrderId)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptInt64(params, "recvWindow", p.RecvWindow)
	applyWorkingOrder(params, "working", p.Working)
	applyPendingOrder(params, "pending", p.Pending)

	var result OrderListResponse
	if err := c.signedPost("/api/v3/orderList/oto", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PlaceOTOCO 发送 OTOCO 订单 (TRADE)
// POST /api/v3/orderList/otoco
// 权重: 1 | 未成交订单计数: 3
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#new-order-list---otoco-trade
// 说明: 生效订单完全成交后，待处理的 OCO 组合（上方+下方）才会自动提交
func (c *BinanceClient) PlaceOTOCO(p PlaceOTOCOParams) (*OrderListResponse, error) {
	if p.Symbol == "" || p.PendingSide == "" || p.PendingQuantity == "" {
		return nil, fmt.Errorf("symbol、pendingSide、pendingQuantity 为必填参数")
	}
	if p.Working.Type == "" || p.Working.Side == "" || p.Working.Price == "" || p.Working.Quantity == "" {
		return nil, fmt.Errorf("working 订单的 type、side、price、quantity 为必填参数")
	}
	if p.PendingAbove.Type == "" {
		return nil, fmt.Errorf("pendingAboveType 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("pendingSide", p.PendingSide)
	params.Set("pendingQuantity", p.PendingQuantity)
	setOptString(params, "listClientOrderId", p.ListClientOrderId)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptInt64(params, "recvWindow", p.RecvWindow)
	applyWorkingOrder(params, "working", p.Working)
	applyOCOLeg(params, "pendingAbove", p.PendingAbove)
	if p.PendingBelow.Type != "" {
		applyOCOLeg(params, "pendingBelow", p.PendingBelow)
	}

	var result OrderListResponse
	if err := c.signedPost("/api/v3/orderList/otoco", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PlaceOPO 发送 OPO 订单 (TRADE)
// POST /api/v3/orderList/opo
// 权重: 1 | 未成交订单计数: 2
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#new-order-list---opotrade
// 说明: OPO（One-Pending-Order）生效订单完全成交后，待处理订单自动提交（类似 OTO）
func (c *BinanceClient) PlaceOPO(p PlaceOPOParams) (*OrderListResponse, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	if p.Working.Type == "" || p.Working.Side == "" || p.Working.Price == "" || p.Working.Quantity == "" {
		return nil, fmt.Errorf("working 订单的 type、side、price、quantity 为必填参数")
	}
	if p.Pending.Type == "" || p.Pending.Side == "" || p.Pending.Quantity == "" {
		return nil, fmt.Errorf("pending 订单的 type、side、quantity 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptString(params, "listClientOrderId", p.ListClientOrderId)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptInt64(params, "recvWindow", p.RecvWindow)
	applyWorkingOrder(params, "working", p.Working)
	applyPendingOrder(params, "pending", p.Pending)

	var result OrderListResponse
	if err := c.signedPost("/api/v3/orderList/opo", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PlaceOPOCO 发送 OPOCO 订单 (TRADE)
// POST /api/v3/orderList/opoco
// 权重: 1 | 未成交订单计数: 3
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#new-order-list---opoco-trade
// 说明: 生效订单完全成交后，待处理的 OCO 组合才会自动提交
func (c *BinanceClient) PlaceOPOCO(p PlaceOPOCOParams) (*OrderListResponse, error) {
	if p.Symbol == "" || p.PendingSide == "" {
		return nil, fmt.Errorf("symbol、pendingSide 为必填参数")
	}
	if p.Working.Type == "" || p.Working.Side == "" || p.Working.Price == "" || p.Working.Quantity == "" {
		return nil, fmt.Errorf("working 订单的 type、side、price、quantity 为必填参数")
	}
	if p.PendingAbove.Type == "" {
		return nil, fmt.Errorf("pendingAboveType 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("pendingSide", p.PendingSide)
	setOptString(params, "listClientOrderId", p.ListClientOrderId)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptInt64(params, "recvWindow", p.RecvWindow)
	applyWorkingOrder(params, "working", p.Working)
	applyOCOLeg(params, "pendingAbove", p.PendingAbove)
	if p.PendingBelow.Type != "" {
		applyOCOLeg(params, "pendingBelow", p.PendingBelow)
	}

	var result OrderListResponse
	if err := c.signedPost("/api/v3/orderList/opoco", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CancelOrderList 取消订单列表 (TRADE)
// DELETE /api/v3/orderList
// 权重: 1
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#取消订单列表-trade
// 注意: orderListId 与 listClientOrderId 至少提供一个；取消列表中任一订单将取消整个列表
func (c *BinanceClient) CancelOrderList(p CancelOrderListParams) (*OrderListResponse, error) {
	if p.Symbol == "" {
		return nil, fmt.Errorf("symbol 为必填参数")
	}
	if p.OrderListId == nil && p.ListClientOrderId == nil {
		return nil, fmt.Errorf("orderListId 与 listClientOrderId 至少需要提供一个")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	setOptInt64(params, "orderListId", p.OrderListId)
	setOptString(params, "listClientOrderId", p.ListClientOrderId)
	setOptString(params, "newClientOrderId", p.NewClientOrderId)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result OrderListResponse
	if err := c.signedDelete("/api/v3/orderList", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PlaceSOROrder 下 SOR 订单 (TRADE)
// POST /api/v3/sor/order
// 权重: 1 | 未成交订单计数: 1
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#下-sor-订单-trade
// 说明: 使用智能订单路由，仅支持 LIMIT 和 MARKET 类型，不支持 quoteOrderQty
func (c *BinanceClient) PlaceSOROrder(p PlaceSOROrderParams) (*OrderResponse, error) {
	if p.Symbol == "" || p.Side == "" || p.Type == "" || p.Quantity == "" {
		return nil, fmt.Errorf("symbol、side、type、quantity 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("side", p.Side)
	params.Set("type", p.Type)
	params.Set("quantity", p.Quantity)
	setOptString(params, "timeInForce", p.TimeInForce)
	setOptString(params, "price", p.Price)
	setOptString(params, "newClientOrderId", p.NewClientOrderId)
	setOptInt64(params, "strategyId", p.StrategyId)
	if p.StrategyType != nil {
		params.Set("strategyType", strconv.Itoa(*p.StrategyType))
	}
	setOptString(params, "icebergQty", p.IcebergQty)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var result OrderResponse
	if err := c.signedPost("/api/v3/sor/order", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// TestSOROrder 测试 SOR 下单接口 (TRADE)
// POST /api/v3/sor/order/test
// 权重: 无 computeCommissionRates=1，有=20
// 文档: https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/rest-api/trading-endpoints#测试-sor-下单接口-trade
// 说明: 不会真实提交到撮合引擎
func (c *BinanceClient) TestSOROrder(p TestSOROrderParams) (*TestOrderCommissionRates, error) {
	if p.Symbol == "" || p.Side == "" || p.Type == "" || p.Quantity == "" {
		return nil, fmt.Errorf("symbol、side、type、quantity 为必填参数")
	}
	params := url.Values{}
	params.Set("symbol", p.Symbol)
	params.Set("side", p.Side)
	params.Set("type", p.Type)
	params.Set("quantity", p.Quantity)
	setOptString(params, "timeInForce", p.TimeInForce)
	setOptString(params, "price", p.Price)
	setOptString(params, "newClientOrderId", p.NewClientOrderId)
	setOptInt64(params, "strategyId", p.StrategyId)
	if p.StrategyType != nil {
		params.Set("strategyType", strconv.Itoa(*p.StrategyType))
	}
	setOptString(params, "icebergQty", p.IcebergQty)
	setOptString(params, "newOrderRespType", p.NewOrderRespType)
	setOptString(params, "selfTradePreventionMode", p.SelfTradePreventionMode)
	setOptBool(params, "computeCommissionRates", p.ComputeCommissionRates)
	setOptInt64(params, "recvWindow", p.RecvWindow)

	var raw json.RawMessage
	if err := c.signedPost("/api/v3/sor/order/test", params, &raw); err != nil {
		return nil, err
	}
	if string(raw) == "{}" || len(raw) == 0 {
		return nil, nil
	}
	var result TestOrderCommissionRates
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("解析测试 SOR 下单响应失败: %w", err)
	}
	return &result, nil
}
