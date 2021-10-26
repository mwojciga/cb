package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cinar/indicator"
	"github.com/joho/godotenv"
)

var baseApiUrl string = "https://fapi.binance.com"
var recvWindow int = 10000000

var httpClient http.Client = http.Client{
	Timeout: time.Second * 2,
}

type AccountData struct {
	FeeTier                     int    `json:"feeTier"`
	CanTrade                    bool   `json:"canTrade"`
	CanDeposit                  bool   `json:"canDeposit"`
	CanWithdraw                 bool   `json:"canWithdraw"`
	UpdateTime                  int    `json:"updateTime"`
	TotalInitialMargin          string `json:"totalInitialMargin"`
	TotalMaintMargin            string `json:"totalMaintMargin"`
	TotalWalletBalance          string `json:"totalWalletBalance"`
	TotalUnrealizedProfit       string `json:"totalUnrealizedProfit"`
	TotalMarginBalance          string `json:"totalMarginBalance"`
	TotalPositionInitialMargin  string `json:"totalPositionInitialMargin"`
	TotalOpenOrderInitialMargin string `json:"totalOpenOrderInitialMargin"`
	TotalCrossWalletBalance     string `json:"totalCrossWalletBalance"`
	TotalCrossUnPnl             string `json:"totalCrossUnPnl"`
	AvailableBalance            string `json:"availableBalance"`
	MaxWithdrawAmount           string `json:"maxWithdrawAmount"`
	Assets                      []struct {
		Asset                  string `json:"asset"`
		WalletBalance          string `json:"walletBalance"`
		UnrealizedProfit       string `json:"unrealizedProfit"`
		MarginBalance          string `json:"marginBalance"`
		MaintMargin            string `json:"maintMargin"`
		InitialMargin          string `json:"initialMargin"`
		PositionInitialMargin  string `json:"positionInitialMargin"`
		OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
		CrossWalletBalance     string `json:"crossWalletBalance"`
		CrossUnPnl             string `json:"crossUnPnl"`
		AvailableBalance       string `json:"availableBalance"`
		MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
		MarginAvailable        bool   `json:"marginAvailable"`
		UpdateTime             int64  `json:"updateTime"`
	} `json:"assets"`
	Positions []struct {
		Symbol                 string `json:"symbol"`
		InitialMargin          string `json:"initialMargin"`
		MaintMargin            string `json:"maintMargin"`
		UnrealizedProfit       string `json:"unrealizedProfit"`
		PositionInitialMargin  string `json:"positionInitialMargin"`
		OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
		Leverage               string `json:"leverage"`
		Isolated               bool   `json:"isolated"`
		EntryPrice             string `json:"entryPrice"`
		MaxNotional            string `json:"maxNotional"`
		PositionSide           string `json:"positionSide"`
		PositionAmt            string `json:"positionAmt"`
		UpdateTime             int    `json:"updateTime"`
	} `json:"positions"`
}

type NewOrderData struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	PositionSide     string  `json:"positionSide"`
	Type             string  `json:"type"`
	TimeInforce      string  `json:"timeInforce"`
	NewClientOrderId string  `json:"newClientOrderId"`
	Price            float64 `json:"price"`
	StopPrice        float64 `json:"stopPrice"`
	Quantity         float64 `json:"quantity"`
}

type OrderData struct {
	AvgPrice      string `json:"avgPrice"`
	ClientOrderID string `json:"clientOrderId"`
	CumQuote      string `json:"cumQuote"`
	ExecutedQty   string `json:"executedQty"`
	OrderID       int    `json:"orderId"`
	OrigQty       string `json:"origQty"`
	OrigType      string `json:"origType"`
	Price         string `json:"price"`
	ReduceOnly    bool   `json:"reduceOnly"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	Status        string `json:"status"`
	StopPrice     string `json:"stopPrice"`
	ClosePosition bool   `json:"closePosition"`
	Symbol        string `json:"symbol"`
	Time          int64  `json:"time"`
	TimeInForce   string `json:"timeInForce"`
	Type          string `json:"type"`
	ActivatePrice string `json:"activatePrice"`
	PriceRate     string `json:"priceRate"`
	UpdateTime    int64  `json:"updateTime"`
	WorkingType   string `json:"workingType"`
	PriceProtect  bool   `json:"priceProtect"`
}

/* TODO
3. TP/SL
4. Add other assets (maybe a DB with configs?)
5. Dockerize, scale.
*/

func main() {
	// Load .env file with env vars.
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error while loading .env file.")
	}

	account := getAccountData()

	// Check if there are any open positions. Continue if not.
	// https://binance-docs.github.io/apidocs/futures/en/#position-information-v2-user_data
	checkOpenPositions("BTCUSDT", account)

	// Get kines data for an asset and calculate EMAs: EMA50H, EMA100H, EMA200H.
	// https://binance-docs.github.io/apidocs/futures/en/#kline-candlestick-data
	asset := getAssetData("BTCUSDT", "1h", 600)

	newOrder := calculateOrder("BTCUSDT", asset, account)

	// Cancel open orders, if any.
	// https://binance-docs.github.io/apidocs/futures/en/#cancel-all-open-orders-trade
	cancelOrders("BTCUSDT")

	// Open a new order based on calculations.
	// https://binance-docs.github.io/apidocs/futures/en/#new-order-trade
	openOrder(newOrder, false)
}

func getAccountData() AccountData {

	time := getTime()
	apiEndpoint := "/fapi/v2/account"
	params := "&recvWindow=" + strconv.Itoa(recvWindow) + "&timestamp=" + strconv.Itoa(time)
	resBody := sendHttpRequest(http.MethodGet, apiEndpoint, params, true, true)

	var account AccountData

	jsonErr := json.Unmarshal(resBody, &account)
	if jsonErr != nil {
		log.Fatalf("[getAccountData] %s", jsonErr)
	}

	return account
}

func getOpenOrders(symbol string) []OrderData {
	time := getTime()
	apiEndpoint := "/fapi/v1/openOrders"
	params := "&symbol=" + symbol + "&recvWindow=" + strconv.Itoa(recvWindow) + "&timestamp=" + strconv.Itoa(time)
	resBody := sendHttpRequest(http.MethodGet, apiEndpoint, params, true, true)

	var order []OrderData

	jsonErr := json.Unmarshal(resBody, &order)
	if jsonErr != nil {
		log.Fatalf("[getOpenOrders] %s", jsonErr)
	}

	return order
}

func checkOpenPositions(symbol string, account AccountData) {
	for _, position := range account.Positions {
		if position.Symbol == symbol {
			positionAmt, err := strconv.ParseFloat(position.PositionAmt, 32)
			if err != nil {
				log.Fatalf("[checkOpenPositions] Can't parse position size.")
			}
			if positionAmt != 0 {
				log.Printf("[checkOpenPositions] There are already opened positions for %s asset.", symbol)
				// TODO: Check if TP/SL are placed.
				orders := getOpenOrders(symbol)
				if len(orders) != 0 {
					for _, order := range orders {
						log.Printf("[checkOpenPositions] Orders: %s", order.ClientOrderID)
					}
				} else {
					// Place TP and SL orders (STOP_MARKET, TAKE_PROFIT_MARKET)
					log.Printf("[checkOpenPositions] No SL/TP found.")
					entryPrice, err := strconv.ParseFloat(position.EntryPrice, 32)
					log.Printf("[checkOpenPositions] Amt %f, entry %f", positionAmt, entryPrice)
					if err != nil {
						log.Fatalf("[checkOpenPositions] Can't parse EntryPrice.")
					}
					// Common values for TP and SL.
					var newOrder NewOrderData
					newOrder.Quantity = positionAmt
					newOrder.Side = "SELL"
					newOrder.PositionSide = "BOTH"
					newOrder.TimeInforce = "GTC"
					newOrder.Symbol = symbol

					// SL
					newOrder.StopPrice = 0.98 * entryPrice // 2% for SL.
					newOrder.Type = "STOP_MARKET"
					log.Printf("[checkOpenPositions] Opening a SL at %0.2f", newOrder.StopPrice)
					openOrder(newOrder, true)
					// TP
					newOrder.StopPrice = 1.02 * entryPrice // 2% for TP.
					newOrder.Type = "TAKE_PROFIT_MARKET"
					log.Printf("[checkOpenPositions] Opening a TP at %0.2f", newOrder.StopPrice)
					openOrder(newOrder, true)
				}
				os.Exit(1)
			}
			break
		}
		// TODO: If BTCUSDT is not found, it will continue anyways :(
	}
	log.Printf("[checkOpenPositions] There are no opened positions for %s asset. Continuing.", symbol)
}

func getAssetData(symbol string, interval string, limit int) map[string]float64 {
	// Resp: JSON array of arrays
	type PriceData []interface{}

	apiEndpoint := "/fapi/v1/klines"
	params := "symbol=" + symbol + "&interval=" + interval + "&limit=" + strconv.Itoa(limit)
	resBody := sendHttpRequest(http.MethodGet, apiEndpoint, params, true, true)

	var priceData []PriceData

	jsonErr := json.Unmarshal(resBody, &priceData)
	if jsonErr != nil {
		log.Fatalf("[getAssetData] %s", jsonErr)
	}

	// Get 4th array element from each array and convert it to []float64.
	closePrice := make([]float64, 0)
	for _, row := range priceData {
		price, err := strconv.ParseFloat(fmt.Sprintf("%v", row[4]), 64)
		if err != nil {
			log.Fatalf("[getAssetData] %s", err)
		}
		closePrice = append(closePrice, price)
	}

	ema50 := indicator.Ema(50, closePrice)
	ema100 := indicator.Ema(100, closePrice)
	ema200 := indicator.Ema(200, closePrice)
	asset := map[string]float64{
		"currentPrice": closePrice[len(closePrice)-2],
		"ema50":        ema50[len(ema50)-2],
		"ema100":       ema100[len(ema100)-2],
		"ema200":       ema200[len(ema200)-2],
	}
	log.Printf("[getAssetData] Current price: %0.2f", asset["currentPrice"])
	log.Printf("[getAssetData] EMA50: %0.2f", asset["ema50"])
	log.Printf("[getAssetData] EMA100: %0.2f", asset["ema100"])
	log.Printf("[getAssetData] EMA200: %0.2f", asset["ema200"])

	return asset
}

func cancelOrders(symbol string) {
	time := getTime()
	apiEndpoint := "/fapi/v1/allOpenOrders"
	params := "symbol=" + symbol + "&recvWindow=" + strconv.Itoa(recvWindow) + "&timestamp=" + strconv.Itoa(time)
	sendHttpRequest(http.MethodDelete, apiEndpoint, params, true, true)

	log.Printf("[cancelOrders] Open orders cancelled.")
}

func calculateOrder(symbol string, asset map[string]float64, account AccountData) NewOrderData {
	var newOrder NewOrderData
	newOrder.Symbol = symbol

	/*
		Calculate where to open orders.

		Options:
		1. EMA50 > EMA100 > EMA200 - long
		2. EMA200 > EMA100 > EMA50 - short // Not covered yet!
		3. Others: not covered.
	*/
	if asset["ema50"] > asset["ema100"] && asset["ema100"] > asset["ema200"] {
		log.Printf("[calculateOrder] Condition for placing a long was met.")
		newOrder.Price = asset["ema200"]
		// Set the vars for a long here.
		newOrder.Side = "BUY"
		newOrder.PositionSide = "BOTH"
		newOrder.Type = "LIMIT"
		newOrder.TimeInforce = "GTC"

	} else {
		log.Printf("[calculateOrder] None of the conditions to place order were met.")
		os.Exit(1)
	}

	// Calculate quantity.
	balance, err := strconv.ParseFloat(account.AvailableBalance, 32)
	if err != nil {
		log.Fatalf("[calculateOrder] Can't calculate balance.")
	}
	for _, position := range account.Positions {
		if position.Symbol == "BTCUSDT" {
			leverage, err := strconv.ParseFloat(position.Leverage, 32)
			if err != nil {
				log.Fatalf("[calculateOrder] Can't calculate margin.")
			}
			newOrder.Quantity = balance * leverage / asset["currentPrice"]
		}
	}

	return newOrder
}

func openOrder(newOrder NewOrderData, tpsl bool) {
	// Open orders.
	time := getTime()
	apiEndpoint := "/fapi/v1/order"
	var params string
	if tpsl {
		params = fmt.Sprintf("symbol=%s&recvWindow=%d&timestamp=%d&side=%s&positionSide=%s&type=%s&timeInforce=%s&stopPrice=%0.2f&quantity=%0.3f", newOrder.Symbol, recvWindow, time, newOrder.Side, newOrder.PositionSide, newOrder.Type, newOrder.TimeInforce, newOrder.StopPrice, newOrder.Quantity)
	} else {
		params = fmt.Sprintf("symbol=%s&recvWindow=%d&timestamp=%d&side=%s&positionSide=%s&type=%s&timeInforce=%s&price=%0.2f&quantity=%0.3f", newOrder.Symbol, recvWindow, time, newOrder.Side, newOrder.PositionSide, newOrder.Type, newOrder.TimeInforce, newOrder.Price, newOrder.Quantity)
	}
	sendHttpRequest(http.MethodPost, apiEndpoint, params, true, true)
}

func getTime() int {
	actualTime := int(time.Now().UnixMilli())
	return actualTime
}

func generateSignature(params string) string {
	bskey := os.Getenv("BSKEY")
	// Create a new HMAC.
	hmac := hmac.New(sha256.New, []byte(bskey))
	hmac.Write([]byte(params))
	// Get result and encode as hexadecimal string
	signature := hex.EncodeToString(hmac.Sum(nil))
	return signature
}

func sendHttpRequest(method string, apiEndpoint string, params string, signature bool, apikey bool) (resBody []byte) {
	reqUrl := baseApiUrl + apiEndpoint + "?" + params
	if signature {
		sig := generateSignature(params)
		reqUrl = reqUrl + "&signature=" + sig
	}

	req, err := http.NewRequest(method, reqUrl, nil)
	if err != nil {
		log.Fatalf("[sendHttpRequest] %s", err)
	}

	req.Header.Set("User-Agent", "cb-prd-test")

	if apikey {
		bakey := os.Getenv("BAKEY")
		req.Header.Set("X-MBX-APIKEY", bakey)
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Fatalf("[sendHttpRequest] %s", getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatalf("[sendHttpRequest] %s", readErr)
	}

	log.Printf("[sendHttpRequest] Req URL: %s", reqUrl)
	log.Printf("[sendHttpRequest] Res code: %s", string(res.Status))
	//log.Printf("[sendHttpRequest] Res body: %s", string(body))

	return body
}
