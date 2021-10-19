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

func main() {
	// Load .env file with env vars.
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error while loading .env file.")
	}

	// Check if there are any open positions. Continue if not.
	// https://binance-docs.github.io/apidocs/futures/en/#position-information-v2-user_data
	getOpenPositions("BTCUSDT")

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

	jsonErr := json.Unmarshal(resBody, &positions)
	if jsonErr != nil {
		log.Fatal(jsonErr)
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
		log.Fatal(jsonErr)
	}

	// Get 4th array element from each array and convert it to []float64.
	closePrice := make([]float64, 0)
	for _, row := range priceData {
		price, err := strconv.ParseFloat(fmt.Sprintf("%v", row[4]), 64)
		if err != nil {
			log.Fatal(err)
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
		log.Printf("[openOrders] None of the conditions to place order were not met.")
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
	log.Printf("[getTime] Time: %d", actualTime)
	return actualTime
}

func generateSignature(params string) string {
	bskey := os.Getenv("BSKEY")
	// Create a new HMAC.
	hmac := hmac.New(sha256.New, []byte(bskey))
	hmac.Write([]byte(params))
	// Get result and encode as hexadecimal string
	signature := hex.EncodeToString(hmac.Sum(nil))
	log.Printf("[generateSignature] Generated signature: %s", signature)
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
		log.Fatalf("[sendHttpRequest] Err: %s", err)
	}

	req.Header.Set("User-Agent", "cb-prd-test")

	if apikey {
		bakey := os.Getenv("BAKEY")
		req.Header.Set("X-MBX-APIKEY", bakey)
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Fatalf("[sendHttpRequest] Err: %s", getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatalf("[sendHttpRequest] Err: %s", readErr)
	}

	log.Printf("[sendHttpRequest] Req URL: %s", reqUrl)
	log.Printf("[sendHttpRequest] Res code: %s", string(res.Status))
	//log.Printf("[sendHttpRequest] Res body: %s", string(body))

	return body
}
