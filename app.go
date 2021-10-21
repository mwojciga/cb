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

/* TODO
1. Reformat logging, delete unneccessary.
2. Replace getOpenPositions with account (same weight, more info).
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

	// Check if there are any open positions. Continue if not.
	// https://binance-docs.github.io/apidocs/futures/en/#position-information-v2-user_data
	getOpenPositions("BTCUSDT")

	getAccountData()

	// Get kines data for an asset and calculate EMAs: EMA50H, EMA100H, EMA200H.
	// https://binance-docs.github.io/apidocs/futures/en/#kline-candlestick-data
	emas := getPriceData("BTCUSDT", "1h", 600)

	// Cancel open orders, if any.
	// https://binance-docs.github.io/apidocs/futures/en/#cancel-all-open-orders-trade
	cancelOrders("BTCUSDT")

	// Open a new order based on calculations.
	// https://binance-docs.github.io/apidocs/futures/en/#new-order-trade
	openOrders("BTCUSDT", emas)
}

func getAccountData() {
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

	time := getTime()
	apiEndpoint := "/fapi/v2/account"
	params := "&recvWindow=" + strconv.Itoa(recvWindow) + "&timestamp=" + strconv.Itoa(time)
	resBody := sendHttpRequest(http.MethodGet, apiEndpoint, params, true, true)

	var account AccountData

	jsonErr := json.Unmarshal(resBody, &account)
	if jsonErr != nil {
		log.Fatalf("[getAccountData] %s", jsonErr)
	}
	log.Printf("[getAccountData] %s", account.AvailableBalance)
}

func getOpenPositions(symbol string) {
	// Resp: array of JSON objects.
	type PositionRisk struct {
		EntryPrice       string `json:"entryPrice"`
		MarginType       string `json:"marginType"`
		IsAutoAddMargin  string `json:"isAutoAddMargin"`
		IsolatedMargin   string `json:"isolatedMargin"`
		Leverage         string `json:"leverage"`
		LiquidationPrice string `json:"liquidationPrice"`
		MarkPrice        string `json:"markPrice"`
		MaxNotionalValue string `json:"maxNotionalValue"`
		PositionAmt      string `json:"positionAmt"`
		Symbol           string `json:"symbol"`
		UnRealizedProfit string `json:"unRealizedProfit"`
		PositionSide     string `json:"positionSide"`
		UpdateTime       int    `json:"updateTime"`
	}

	time := getTime()
	apiEndpoint := "/fapi/v2/positionRisk"
	params := "symbol=" + symbol + "&recvWindow=" + strconv.Itoa(recvWindow) + "&timestamp=" + strconv.Itoa(time)
	resBody := sendHttpRequest(http.MethodGet, apiEndpoint, params, true, true)

	var positions []PositionRisk

	err := json.Unmarshal(resBody, &positions)
	if err != nil {
		log.Fatalf("[getOpenPositions] %s", err)
	}
	// TODO: needs to be refactored if more then one symbol will be checked.
	if positions[0].PositionAmt != "0.000" {
		log.Printf("[getOpenPositions] There are already opened positions for this asset.")
		// TODO: Check if TP/SL are placed.
		// TODO: Place TP and SL orders (STOP_MARKET, TAKE_PROFIT_MARKET)
		os.Exit(1)
	}
	log.Printf("[getOpenPositions] There are no opened positions for this asset. Continuing.")
}

func getPriceData(symbol string, interval string, limit int) map[string]float64 {
	// Resp: JSON array of arrays
	type PriceData []interface{}

	apiEndpoint := "/fapi/v1/klines"
	params := "symbol=" + symbol + "&interval=" + interval + "&limit=" + strconv.Itoa(limit)
	resBody := sendHttpRequest(http.MethodGet, apiEndpoint, params, true, true)

	var priceData []PriceData

	jsonErr := json.Unmarshal(resBody, &priceData)
	if jsonErr != nil {
		log.Fatalf("[getPriceData] %s", jsonErr)
	}

	// Get 4th array element from each array and convert it to []float64.
	closePrice := make([]float64, 0)
	for _, row := range priceData {
		price, err := strconv.ParseFloat(fmt.Sprintf("%v", row[4]), 64)
		if err != nil {
			log.Fatalf("[getPriceData] %s", err)
		}
		closePrice = append(closePrice, price)
	}

	log.Printf("[getPriceData] Price: %v", closePrice[len(closePrice)-2])
	ema50 := indicator.Ema(50, closePrice)
	ema100 := indicator.Ema(100, closePrice)
	ema200 := indicator.Ema(200, closePrice)
	emas := map[string]float64{
		"ema50":  ema50[len(ema50)-2],
		"ema100": ema100[len(ema100)-2],
		"ema200": ema200[len(ema200)-2],
	}
	log.Printf("[getPriceData] EMA50: %0.2f", emas["ema50"])
	log.Printf("[getPriceData] EMA100: %0.2f", emas["ema100"])
	log.Printf("[getPriceData] EMA200: %0.2f", emas["ema200"])

	return emas
}

func cancelOrders(symbol string) {
	time := getTime()
	apiEndpoint := "/fapi/v1/allOpenOrders"
	params := "symbol=" + symbol + "&recvWindow=" + strconv.Itoa(recvWindow) + "&timestamp=" + strconv.Itoa(time)
	sendHttpRequest(http.MethodDelete, apiEndpoint, params, true, true)

	log.Printf("[cancelOrders] Open orders cancelled.")
}

func openOrders(symbol string, emas map[string]float64) {
	/*
		Calculate where to open orders.

		Options:
		1. EMA50 > EMA100 > EMA200 - long
		2. EMA200 > EMA100 > EMA50 - short // Not covered yet!
		3. Others: not covered.
	*/

	var price float64
	if emas["ema50"] > emas["ema100"] && emas["ema100"] > emas["ema200"] {
		price = emas["ema100"]
		log.Printf("[openOrders] Condition for placing a long was met.")
	} else {
		log.Printf("[openOrders] None of the conditions to place order were met.")
		os.Exit(1)
	}

	// Fixed for testing purposes only.
	quantity := "0.05"

	// Open orders.
	time := getTime()
	apiEndpoint := "/fapi/v1/order"
	params := "symbol=" + symbol + "&recvWindow=" + strconv.Itoa(recvWindow) + "&timestamp=" + strconv.Itoa(time) + "&side=BUY" + "&positionSide=BOTH" + "&type=LIMIT" + "&timeInforce=GTC" + "&newClientOrderId=cbTestOrder" + "&price=" + fmt.Sprintf("%0.2f", price) + "&quantity=" + quantity
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
