package ibapi

import (
	"sync/atomic"
	"time"
)

// IbWrapper contain the funcs to handle the msg from TWS or Gateway
type IbWrapper interface {
	TickPrice(reqID int64, tickType int64, price float64, attrib TickAttrib)
	TickSize(reqID int64, tickType int64, size float64)
	OrderStatus(orderID int64, status string, filled float64, remaining float64, avgFillPrice float64, permID int64, parentID int64, lastFillPrice float64, clientID int64, whyHeld string, mktCapPrice float64)
	Error(reqID int64, errCode int64, errString string)
	OpenOrder(orderID int64, contract *Contract, order *Order, orderState *OrderState)
	UpdateAccountValue(tag string, val string, currency string, accName string)
	UpdatePortfolio(contract *Contract, position float64, marketPrice float64, marketValue float64, averageCost float64, unrealizedPNL float64, realizedPNL float64, accName string)
	UpdateAccountTime(accTime time.Time)
	NextValidID(reqID int64)
	ContractDetails(reqID int64, conDetails *ContractDetails)
	ExecDetails(reqID int64, contract *Contract, execution *Execution)
	UpdateMktDepth(reqID int64, position int64, operation int64, side int64, price float64, size float64)
	UpdateMktDepthL2(reqID int64, position int64, marketMaker string, operation int64, side int64, price float64, size float64, isSmartDepth bool)
	UpdateNewsBulletin(msgID int64, msgType int64, newsMessage string, originExchange string)
	ManagedAccounts(accountsList []string)
	ReceiveFA(faData int64, cxml string)
	HistoricalData(reqID int64, bar *BarData)
	HistoricalDataEnd(reqID int64, startDateStr string, endDateStr string)
	HistoricalDataUpdate(reqID int64, bar *BarData)
	BondContractDetails(reqID int64, conDetails *ContractDetails)
	ScannerParameters(xml string)
	ScannerData(reqID int64, rank int64, conDetails *ContractDetails, distance string, benchmark string, projection string, legs string)
	ScannerDataEnd(reqID int64)
	TickOptionComputation(reqID int64, tickType int64, tickAttrib int64, impliedVol float64, delta float64, optPrice float64, pvDiviedn float64, gamma float64, vega float64, theta float64, undPrice float64)
	TickGeneric(reqID int64, tickType int64, value float64)
	TickString(reqID int64, tickType int64, value string)
	TickEFP(reqID int64, tickType int64, basisPoints float64, formattedBasisPoints string, totalDividends float64, holdDays int64, futureLastTradeDate string, dividendImpact float64, dividendsToLastTradeDate float64)
	CurrentTime(t time.Time)
	RealtimeBar(reqID int64, time int64, open float64, high float64, low float64, close float64, volume float64, wap float64, count int64)
	FundamentalData(reqID int64, data string)
	ContractDetailsEnd(reqID int64)
	OpenOrderEnd()
	AccountDownloadEnd(accName string)
	ExecDetailsEnd(reqID int64)
	DeltaNeutralValidation(reqID int64, deltaNeutralContract DeltaNeutralContract)
	TickSnapshotEnd(reqID int64)
	MarketDataType(reqID int64, marketDataType int64)
	Position(account string, contract *Contract, position float64, avgCost float64)
	PositionEnd()
	AccountSummary(reqID int64, account string, tag string, value string, currency string)
	AccountSummaryEnd(reqID int64)
	VerifyMessageAPI(apiData string)
	VerifyCompleted(isSuccessful bool, err string)
	DisplayGroupList(reqID int64, groups string)
	DisplayGroupUpdated(reqID int64, contractInfo string)
	VerifyAndAuthMessageAPI(apiData string, xyzChallange string)
	VerifyAndAuthCompleted(isSuccessful bool, err string)
	PositionMulti(reqID int64, account string, modelCode string, contract *Contract, position float64, avgCost float64)
	PositionMultiEnd(reqID int64)
	AccountUpdateMulti(reqID int64, account string, modleCode string, tag string, value string, currency string)
	AccountUpdateMultiEnd(reqID int64)
	SecurityDefinitionOptionParameter(reqID int64, exchange string, underlyingContractID int64, tradingClass string, multiplier string, expirations []string, strikes []float64)
	SecurityDefinitionOptionParameterEnd(reqID int64)
	SoftDollarTiers(reqID int64, tiers []SoftDollarTier)
	FamilyCodes(famCodes []FamilyCode)
	SymbolSamples(reqID int64, contractDescriptions []ContractDescription)
	SmartComponents(reqID int64, smartComps []SmartComponent)
	TickReqParams(tickerID int64, minTick float64, bboExchange string, snapshotPermissions int64)
	MktDepthExchanges(depthMktDataDescriptions []DepthMktDataDescription)
	HeadTimestamp(reqID int64, headTimestamp string)
	TickNews(tickerID int64, timeStamp int64, providerCode string, articleID string, headline string, extraData string)
	NewsProviders(newsProviders []NewsProvider)
	NewsArticle(reqID int64, articleType int64, articleText string)
	HistoricalNews(reqID int64, time string, providerCode string, articleID string, headline string)
	HistoricalNewsEnd(reqID int64, hasMore bool)
	HistogramData(reqID int64, histogram []HistogramData)
	RerouteMktDataReq(reqID int64, contractID int64, exchange string)
	RerouteMktDepthReq(reqID int64, contractID int64, exchange string)
	MarketRule(marketRuleID int64, priceIncrements []PriceIncrement)
	Pnl(reqID int64, dailyPnL float64, unrealizedPnL float64, realizedPnL float64)
	PnlSingle(reqID int64, position float64, dailyPnL float64, unrealizedPnL float64, realizedPnL float64, value float64)
	HistoricalTicks(reqID int64, ticks []HistoricalTick, done bool)
	HistoricalTicksBidAsk(reqID int64, ticks []HistoricalTickBidAsk, done bool)
	HistoricalTicksLast(reqID int64, ticks []HistoricalTickLast, done bool)
	TickByTickAllLast(reqID int64, tickType int64, time int64, price float64, size float64, tickAttribLast TickAttribLast, exchange string, specialConditions string)
	TickByTickBidAsk(reqID int64, time int64, bidPrice float64, askPrice float64, bidSize float64, askSize float64, tickAttribBidAsk TickAttribBidAsk)
	TickByTickMidPoint(reqID int64, time int64, midPoint float64)
	OrderBound(reqID int64, apiClientID int64, apiOrderID int64)
	CompletedOrder(contract *Contract, order *Order, orderState *OrderState)
	CompletedOrdersEnd()
	CommissionReport(commissionReport CommissionReport)
	ConnectAck()
	ConnectionClosed()
	ReplaceFAEnd(reqID int64, text string)
	WshMetaData(reqID int64, dataJson string)
	WshEventData(reqID int64, dataJson string)
}

// Wrapper is the default wrapper provided by this golang implement.
type Wrapper struct {
	orderID int64
}

func (w *Wrapper) GetNextOrderID() (i int64) {
	i = w.orderID
	atomic.AddInt64(&w.orderID, 1)
	return
}

func (w Wrapper) ConnectAck() {
	log.Info("<ConnectAck>...")
}

func (w Wrapper) ConnectionClosed() {
	log.Info("<ConnectionClosed>...")
}

func (w *Wrapper) NextValidID(reqID int64) {
	atomic.StoreInt64(&w.orderID, reqID)
	log.With("reqID", reqID).Info("<NextValidID>")
}

func (w Wrapper) ManagedAccounts(accountsList []string) {
	log.Info("<ManagedAccounts>", "accountList", accountsList)
}

func (w Wrapper) TickPrice(reqID int64, tickType int64, price float64, attrib TickAttrib) {
	log.With("reqID", reqID).Info("<TickPrice>", "tickType", tickType, "price", price)
}

func (w Wrapper) UpdateAccountTime(accTime time.Time) {
	log.Info("<UpdateAccountTime>", "accountTime", accTime)
}

func (w Wrapper) UpdateAccountValue(tag string, value string, currency string, account string) {
	log.Info("<UpdateAccountValue>", "tag", tag, "value", value, "currency", currency, "account", account)
}

func (w Wrapper) AccountDownloadEnd(accName string) {
	log.Info("<AccountDownloadEnd>", "accountName", accName)
}

func (w Wrapper) AccountUpdateMulti(reqID int64, account string, modelCode string, tag string, value string, currency string) {
	log.With("reqID", reqID).Info("<AccountUpdateMulti>",
		"account", account,
		"modelCode", modelCode,
		"tag", tag,
		"value", value,
		"curreny", currency,
	)
}

func (w Wrapper) AccountUpdateMultiEnd(reqID int64) {
	log.With("reqID", reqID).Info("<AccountUpdateMultiEnd>")
}

func (w Wrapper) AccountSummary(reqID int64, account string, tag string, value string, currency string) {
	log.With("reqID", reqID).Info("<AccountSummary>",
		"account", account,
		"tag", tag,
		"value", value,
		"curreny", currency,
	)

}

func (w Wrapper) AccountSummaryEnd(reqID int64) {
	log.With("reqID", reqID).Info("<AccountSummaryEnd>")
}

func (w Wrapper) VerifyMessageAPI(apiData string) {
	log.Info("<VerifyMessageAPI>", "apiData", apiData)
}

func (w Wrapper) VerifyCompleted(isSuccessful bool, err string) {
	log.Info("<VerifyCompleted>", "isSuccessful", isSuccessful, "error", err)
}

func (w Wrapper) VerifyAndAuthMessageAPI(apiData string, xyzChallange string) {
	log.Info("<VerifyMessageAPI>", "apiData", apiData, "xyzChallange", xyzChallange)
}

func (w Wrapper) VerifyAndAuthCompleted(isSuccessful bool, err string) {
	log.Info("<VerifyCompleted>", "isSuccessful", isSuccessful, "error", err)
}

func (w Wrapper) DisplayGroupList(reqID int64, groups string) {
	log.With("reqID", reqID).Info("<DisplayGroupList>", "groups", groups)
}

func (w Wrapper) DisplayGroupUpdated(reqID int64, contractInfo string) {
	log.With("reqID", reqID).Info("<DisplayGroupUpdated>", "contractInfo", contractInfo)
}

func (w Wrapper) PositionMulti(reqID int64, account string, modelCode string, contract *Contract, position float64, avgCost float64) {
	log.With("reqID", reqID).Info("<PositionMulti>",
		"account", account,
		"modelCode", modelCode,
		"contract", contract,
		"position", position,
		"avgCost", avgCost,
	)
}

func (w Wrapper) PositionMultiEnd(reqID int64) {
	log.With("reqID", reqID).Info("<PositionMultiEnd>")
}

func (w Wrapper) UpdatePortfolio(contract *Contract, position float64, marketPrice float64, marketValue float64, averageCost float64, unrealizedPNL float64, realizedPNL float64, accName string) {
	log.Info("<UpdatePortfolio>",
		"localSymbol", contract.LocalSymbol,
		"position", position,
		"marketPrice", marketPrice,
		"averageCost", averageCost,
		"unrealizedPNL", unrealizedPNL,
		"realizedPNL", realizedPNL,
		"accName", accName,
	)
}

func (w Wrapper) Position(account string, contract *Contract, position float64, avgCost float64) {
	log.Info("<UpdatePortfolio>",
		"account", account,
		"contract", contract,
		"position", position,
		"avgCost", avgCost,
	)
}

func (w Wrapper) PositionEnd() {
	log.Info("<PositionEnd>")
}

func (w Wrapper) Pnl(reqID int64, dailyPnL float64, unrealizedPnL float64, realizedPnL float64) {
	log.With("reqID", reqID).Info("<PNL>",
		"dailyPnL", dailyPnL,
		"unrealizedPnL", unrealizedPnL,
		"realizedPnL", realizedPnL,
	)
}

func (w Wrapper) PnlSingle(reqID int64, position float64, dailyPnL float64, unrealizedPnL float64, realizedPnL float64, value float64) {
	log.With("reqID", reqID).Info("<PNLSingle>",
		"position", position,
		"dailyPnL", dailyPnL,
		"unrealizedPnL", unrealizedPnL,
		"realizedPnL", realizedPnL,
		"value", value,
	)
}

func (w Wrapper) OpenOrder(orderID int64, contract *Contract, order *Order, orderState *OrderState) {
	log.With("orderID", orderID).Info("<OpenOrder>",
		"contract", contract,
		"order", order,
		"orderState", orderState,
	)
}

func (w Wrapper) OpenOrderEnd() {
	log.Info("<OpenOrderEnd>")

}

func (w Wrapper) OrderStatus(orderID int64, status string, filled float64, remaining float64, avgFillPrice float64, permID int64, parentID int64, lastFillPrice float64, clientID int64, whyHeld string, mktCapPrice float64) {
	log.With("orderID", orderID).Info("<OrderStatus>",
		"status", status,
		"filled", filled,
		"remaining", remaining,
		"avgFillPrice", avgFillPrice,
	)
}

func (w Wrapper) ExecDetails(reqID int64, contract *Contract, execution *Execution) {
	log.With("reqID", reqID).Info("<ExecDetails>",
		"contract", contract,
		"execution", execution,
	)
}

func (w Wrapper) ExecDetailsEnd(reqID int64) {
	log.With("reqID", reqID).Info("<ExecDetailsEnd>")
}

func (w Wrapper) DeltaNeutralValidation(reqID int64, deltaNeutralContract DeltaNeutralContract) {
	log.With("reqID", reqID).Info("<DeltaNeutralValidation>",
		"deltaNeutralContract", deltaNeutralContract,
	)
}

func (w Wrapper) CommissionReport(commissionReport CommissionReport) {
	log.Info("<CommissionReport>",
		"commissionReport", commissionReport,
	)
}

func (w Wrapper) OrderBound(reqID int64, apiClientID int64, apiOrderID int64) {
	log.With("reqID", reqID).Info("<OrderBound>",
		"apiClientID", apiClientID,
		"apiOrderID", apiOrderID,
	)
}

func (w Wrapper) ContractDetails(reqID int64, conDetails *ContractDetails) {
	log.With("reqID", reqID).Info("<ContractDetails>",
		"conDetails", conDetails,
	)
}

func (w Wrapper) ContractDetailsEnd(reqID int64) {
	log.With("reqID", reqID).Info("<ContractDetailsEnd>")
}

func (w Wrapper) BondContractDetails(reqID int64, conDetails *ContractDetails) {
	log.With("reqID", reqID).Info("<BondContractDetails>",
		"conDetails", conDetails,
	)
}

func (w Wrapper) SymbolSamples(reqID int64, contractDescriptions []ContractDescription) {
	log.With("reqID", reqID).Info("<SymbolSamples>",
		"contractDescriptions", contractDescriptions,
	)
}

func (w Wrapper) SmartComponents(reqID int64, smartComps []SmartComponent) {
	log.With("reqID", reqID).Info("<SmartComponents>",
		"smartComps", smartComps,
	)
}

func (w Wrapper) MarketRule(marketRuleID int64, priceIncrements []PriceIncrement) {
	log.Info("<MarketRule>",
		"marketRuleID", marketRuleID,
		"priceIncrements", priceIncrements,
	)
}

func (w Wrapper) RealtimeBar(reqID int64, time int64, open float64, high float64, low float64, close float64, volume float64, wap float64, count int64) {
	log.With("reqID", reqID).Info("<RealtimeBar>",
		"time", time,
		"open", open,
		"high", high,
		"low", low,
		"close", close,
		"volume", volume,
		"wap", wap,
		"count", count,
	)
}

func (w Wrapper) HistoricalData(reqID int64, bar *BarData) {
	log.With("reqID", reqID).Info("<HistoricalData>",
		"bar", bar,
	)
}

func (w Wrapper) HistoricalDataEnd(reqID int64, startDateStr string, endDateStr string) {
	log.With("reqID", reqID).Info("<HistoricalDataEnd>",
		"startDate", startDateStr,
		"endDate", endDateStr,
	)
}

func (w Wrapper) HistoricalDataUpdate(reqID int64, bar *BarData) {
	log.With("reqID", reqID).Info("<HistoricalDataUpdate>",
		"bar", bar,
	)
}

func (w Wrapper) HeadTimestamp(reqID int64, headTimestamp string) {
	log.With("reqID", reqID).Info("<HeadTimestamp>",
		"headTimestamp", headTimestamp,
	)
}

func (w Wrapper) HistoricalTicks(reqID int64, ticks []HistoricalTick, done bool) {
	log.With("reqID", reqID).Info("<HistoricalTicks>",
		"ticks", ticks,
		"done", done,
	)
}

func (w Wrapper) HistoricalTicksBidAsk(reqID int64, ticks []HistoricalTickBidAsk, done bool) {
	log.With("reqID", reqID).Info("<HistoricalTicksBidAsk>",
		"ticks", ticks,
		"done", done,
	)
}

func (w Wrapper) HistoricalTicksLast(reqID int64, ticks []HistoricalTickLast, done bool) {
	log.With("reqID", reqID).Info("<HistoricalTicksLast>",
		"ticks", ticks,
		"done", done,
	)
}

func (w Wrapper) TickSize(reqID int64, tickType int64, size float64) {
	log.With("reqID", reqID).Info("<TickSize>",
		"tickType", tickType,
		"size", size,
	)
}

func (w Wrapper) TickSnapshotEnd(reqID int64) {
	log.With("reqID", reqID).Info("<TickSnapshotEnd>")
}

func (w Wrapper) MarketDataType(reqID int64, marketDataType int64) {
	log.With("reqID", reqID).Info("<MarketDataType>",
		"marketDataType", marketDataType,
	)
}

func (w Wrapper) TickByTickAllLast(reqID int64, tickType int64, time int64, price float64, size float64, tickAttribLast TickAttribLast, exchange string, specialConditions string) {
	log.With("reqID", reqID).Info("<TickByTickAllLast>",
		"tickType", tickType,
		"time", time,
		"price", price,
		"size", size,
	)
}

func (w Wrapper) TickByTickBidAsk(reqID int64, time int64, bidPrice float64, askPrice float64, bidSize float64, askSize float64, tickAttribBidAsk TickAttribBidAsk) {
	log.With("reqID", reqID).Info("<TickByTickBidAsk>",
		"time", time,
		"bidPrice", bidPrice,
		"askPrice", askPrice,
		"bidSize", bidSize,
		"askSize", askSize,
	)
}

func (w Wrapper) TickByTickMidPoint(reqID int64, time int64, midPoint float64) {
	log.With("reqID", reqID).Info("<TickByTickMidPoint>",
		"time", time,
		"midPoint", midPoint,
	)
}

func (w Wrapper) TickString(reqID int64, tickType int64, value string) {
	log.With("reqID", reqID).Info("<TickString>",
		"tickType", tickType,
		"value", value,
	)
}

func (w Wrapper) TickGeneric(reqID int64, tickType int64, value float64) {
	log.With("reqID", reqID).Info("<TickGeneric>",
		"tickType", tickType,
		"value", value,
	)
}

func (w Wrapper) TickEFP(reqID int64, tickType int64, basisPoints float64, formattedBasisPoints string, totalDividends float64, holdDays int64, futureLastTradeDate string, dividendImpact float64, dividendsToLastTradeDate float64) {
	log.With("reqID", reqID).Info("<TickEFP>",
		"tickType", tickType,
		"basisPoints", basisPoints,
	)
}

func (w Wrapper) TickReqParams(reqID int64, minTick float64, bboExchange string, snapshotPermissions int64) {
	log.With("reqID", reqID).Info("<TickReqParams>",
		"minTick", minTick,
		"bboExchange", bboExchange,
		"snapshotPermissions", snapshotPermissions,
	)
}
func (w Wrapper) MktDepthExchanges(depthMktDataDescriptions []DepthMktDataDescription) {
	log.Info("<MktDepthExchanges>",
		"depthMktDataDescriptions", depthMktDataDescriptions,
	)
}

/*Returns the order book.

tickerId -  the request's identifier
position -  the order book's row being updated
operation - how to refresh the row:
	0 = insert (insert this new order into the row identified by 'position')
	1 = update (update the existing order in the row identified by 'position')
	2 = delete (delete the existing order at the row identified by 'position').
side -  0 for ask, 1 for bid
price - the order's price
size -  the order's size*/
func (w Wrapper) UpdateMktDepth(reqID int64, position int64, operation int64, side int64, price float64, size float64) {
	log.With("reqID", reqID).Info("<UpdateMktDepth>",
		"position", position,
		"operation", operation,
		"side", side,
		"price", price,
		"size", size,
	)
}

func (w Wrapper) UpdateMktDepthL2(reqID int64, position int64, marketMaker string, operation int64, side int64, price float64, size float64, isSmartDepth bool) {
	log.With("reqID", reqID).Info("<UpdateMktDepthL2>",
		"position", position,
		"marketMaker", marketMaker,
		"operation", operation,
		"side", side,
		"price", price,
		"size", size,
		"isSmartDepth", isSmartDepth,
	)
}

func (w Wrapper) TickOptionComputation(reqID int64, tickType int64, tickAttrib int64, impliedVol float64, delta float64, optPrice float64, pvDiviedn float64, gamma float64, vega float64, theta float64, undPrice float64) {
	log.With("reqID", reqID).Info("<TickOptionComputation>",
		"tickType", tickType,
		"tickAttrib", tickAttrib,
		"impliedVol", impliedVol,
		"delta", delta,
		"optPrice", optPrice,
		"pvDiviedn", pvDiviedn,
		"gamma", gamma,
		"vega", vega,
		"theta", theta,
		"undPrice", undPrice,
	)
}

func (w Wrapper) FundamentalData(reqID int64, data string) {
	log.With("reqID", reqID).Info("<FundamentalData>",
		"data", data,
	)
}

func (w Wrapper) ScannerParameters(xml string) {
	log.Info("<ScannerParameters>",
		"xml", xml,
	)

}

func (w Wrapper) ScannerData(reqID int64, rank int64, conDetails *ContractDetails, distance string, benchmark string, projection string, legs string) {
	log.With("reqID", reqID).Info("<ScannerData>",
		"rank", rank,
		"conDetails", conDetails,
		"distance", distance,
		"benchmark", benchmark,
		"projection", projection,
		"legs", legs,
	)
}

func (w Wrapper) ScannerDataEnd(reqID int64) {
	log.With("reqID", reqID).Info("<ScannerDataEnd>")
}

func (w Wrapper) HistogramData(reqID int64, histogram []HistogramData) {
	log.With("reqID", reqID).Info("<HistogramData>",
		"histogram", histogram,
	)
}

func (w Wrapper) RerouteMktDataReq(reqID int64, contractID int64, exchange string) {
	log.With("reqID", reqID).Info("<RerouteMktDataReq>",
		"contractID", contractID,
		"exchange", exchange,
	)
}

func (w Wrapper) RerouteMktDepthReq(reqID int64, contractID int64, exchange string) {
	log.With("reqID", reqID).Info("<RerouteMktDepthReq>",
		"contractID", contractID,
		"exchange", exchange,
	)
}

func (w Wrapper) SecurityDefinitionOptionParameter(reqID int64, exchange string, underlyingContractID int64, tradingClass string, multiplier string, expirations []string, strikes []float64) {
	log.With("reqID", reqID).Info("<SecurityDefinitionOptionParameter>",
		"exchange", exchange,
		"underlyingContractID", underlyingContractID,
		"tradingClass", tradingClass,
		"multiplier", multiplier,
		"expirations", expirations,
		"strikes", strikes,
	)
}

func (w Wrapper) SecurityDefinitionOptionParameterEnd(reqID int64) {
	log.With("reqID", reqID).Info("<SecurityDefinitionOptionParameterEnd>")
}

func (w Wrapper) SoftDollarTiers(reqID int64, tiers []SoftDollarTier) {
	log.With("reqID", reqID).Info("<SoftDollarTiers>",
		"tiers", tiers,
	)
}

func (w Wrapper) FamilyCodes(famCodes []FamilyCode) {
	log.Info("<FamilyCodes>",
		"famCodes", famCodes,
	)
}

func (w Wrapper) NewsProviders(newsProviders []NewsProvider) {
	log.Info("<NewsProviders>",
		"newsProviders", newsProviders,
	)
}

func (w Wrapper) TickNews(tickerID int64, timeStamp int64, providerCode string, articleID string, headline string, extraData string) {
	log.With("tickerID", tickerID).Info("<TickNews>",
		"timeStamp", timeStamp,
		"providerCode", providerCode,
		"articleID", articleID,
		"headline", headline,
		"extraData", extraData,
	)
}

func (w Wrapper) NewsArticle(reqID int64, articleType int64, articleText string) {
	log.With("reqID", reqID).Info("<NewsArticle>",
		"articleType", articleType,
		"articleText", articleText,
	)
}

func (w Wrapper) HistoricalNews(reqID int64, time string, providerCode string, articleID string, headline string) {
	log.With("reqID", reqID).Info("<HistoricalNews>",
		"time", time,
		"providerCode", providerCode,
		"articleID", articleID,
		"headline", headline,
	)
}

func (w Wrapper) HistoricalNewsEnd(reqID int64, hasMore bool) {
	log.With("reqID", reqID).Info("<HistoricalNewsEnd>",
		"hasMore", hasMore,
	)
}

func (w Wrapper) UpdateNewsBulletin(msgID int64, msgType int64, newsMessage string, originExch string) {
	log.With("msgID", msgID).Info("<UpdateNewsBulletin>",
		"msgType", msgType,
		"newsMessage", newsMessage,
		"originExch", originExch,
	)
}

func (w Wrapper) ReceiveFA(faData int64, cxml string) {
	log.Info("<ReceiveFA>",
		"faData", faData,
		"cxml", cxml,
	)
}

func (w Wrapper) CurrentTime(t time.Time) {
	log.Info("<CurrentTime>",
		"time", t,
	)
}

func (w Wrapper) Error(reqID int64, errCode int64, errString string) {
	log.With("reqID", reqID).Info("<Error>",
		"errCode", errCode,
		"errString", errString,
	)
}

func (w Wrapper) CompletedOrder(contract *Contract, order *Order, orderState *OrderState) {
	log.Info("<CompletedOrder>",
		"contract", contract,
		"order", order,
		"orderState", orderState,
	)
}

func (w Wrapper) CompletedOrdersEnd() {
	log.Info("<CompletedOrdersEnd>:")
}

func (w Wrapper) ReplaceFAEnd(reqID int64, text string) {
	log.With("reqID", reqID).Info("<ReplaceFAEnd>", "text", text)
}

func (w Wrapper) WshMetaData(reqID int64, dataJson string) {
	log.With("reqID", reqID).Info("<WshMetaData>", "dataJson", dataJson)
}
func (w Wrapper) WshEventData(reqID int64, dataJson string) {
	log.With("reqID", reqID).Info("<WshEventData>", "dataJson", dataJson)
}
