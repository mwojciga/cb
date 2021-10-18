package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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

	// Get kines data for an asset.
	// https://binance-docs.github.io/apidocs/futures/en/#kline-candlestick-data
	//getPriceData("BTCUSDT", "1d", 5)

	// Calculate EMAs: EMA50H, EMA100H, EMA200D.

	// Cancel open orders, if any.
	// https://binance-docs.github.io/apidocs/futures/en/#cancel-all-open-orders-trade

	// Open a new order based on calculations.
	// https://binance-docs.github.io/apidocs/futures/en/#new-order-trade
}

func getOpenPositions(symbol string) {
	// Resp: array of JSON objects.
	type PositionRisk struct {
		PositionAmt string `json:"positionAmt"`
	}

	time := getTime()
	apiEndpoint := "/fapi/v2/positionRisk"
	params := "symbol=" + symbol + "&recvWindow=" + strconv.Itoa(recvWindow) + "&timestamp=" + strconv.Itoa(time)
	resBody := sendHttpGetRequest(apiEndpoint, params, true, true)

	var positions []PositionRisk

	jsonErr := json.Unmarshal(resBody, &positions)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}
	// TODO: needs to be refactored if more then one symbol will be checked.
	if positions[0].PositionAmt != "0.000" {
		log.Print("There are already opened positions for this asset.")
		os.Exit(1)
	}
}

// TODO: this will not work as we don't receive a JSON here but an array instead.
/*
func getPriceData(symbol string, interval string, limit int) {
	// Resp: array of arrays ?
	apiEndpoint := "/fapi/v1/klines"
	// Example: https://fapi.binance.com/fapi/v1/klines?symbol=BTCUSDT&interval=1d&limit=5
	type crypto struct {
		Symbol string `json:"symbol"`
		//Price float32 `json:"price"`
		Time int `json:"time"`
	}
	params := "?symbol=" + symbol + "&interval=" + interval + "&limit=" + strconv.Itoa(limit)
	url := baseApiUrl + apiEndpoint + params
	body := sendHttpGetRequest(url, false)
	crypto1 := crypto{}
	jsonErr := json.Unmarshal(body, &crypto1)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	log.Print(crypto1.Symbol)
}
*/

func getTime() int {
	actualTime := int(time.Now().UnixMilli())
	log.Printf("Time: %d", actualTime)
	return actualTime
}

func generateSignature(params string) string {
	bskey := os.Getenv("BSKEY")
	// Create a new HMAC.
	hmac := hmac.New(sha256.New, []byte(bskey))
	hmac.Write([]byte(params))
	// Get result and encode as hexadecimal string
	signature := hex.EncodeToString(hmac.Sum(nil))
	log.Print("Generated signature: " + signature)
	return signature
}

func sendHttpGetRequest(apiEndpoint string, params string, signature bool, apikey bool) (resBody []byte) {
	reqUrl := baseApiUrl + apiEndpoint + "?" + params
	if signature {
		sig := generateSignature(params)
		reqUrl = reqUrl + "&signature=" + sig
	}

	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "cb-prd-test")

	if apikey {
		bakey := os.Getenv("BAKEY")
		req.Header.Set("X-MBX-APIKEY", bakey)
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	log.Print("URL: " + reqUrl + ", status: " + res.Status)
	log.Print(string(body))

	return body
}
