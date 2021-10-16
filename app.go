package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

var baseApiUrl string = "https://fapi.binance.com"
var recvWindow int = 10000000

var httpClient http.Client = http.Client{
	Timeout: time.Second * 2,
}

func main() {
	// Check if there are any open trades. Continue if not.
	// https://binance-docs.github.io/apidocs/futures/en/#position-information-v2-user_data

	// Get kines data for an asset.
	// https://binance-docs.github.io/apidocs/futures/en/#kline-candlestick-data
	getPriceData("BTCUSDT", "1d", 5)

	// Calculate EMAs: EMA50H, EMA100H, EMA200D.

	// Cancel open orders, if any.

	// Open a new order based on calculations.
	// https://binance-docs.github.io/apidocs/futures/en/#new-order-trade
}

// TODO: this will not work as we don't receive a JSON here but an array instead.
func getPriceData(symbol string, interval string, limit int) {
	// Example: https://fapi.binance.com/fapi/v1/klines?symbol=BTCUSDT&interval=1d&limit=5
	type crypto struct {
		Symbol string `json:"symbol"`
		//Price float32 `json:"price"`
		Time int `json:"time"`
	}
	params := "symbol=" + symbol + "&interval=" + interval + "&limit=" + strconv.Itoa(limit)
	url := baseApiUrl + "/fapi/v1/klines?" + params
	body := sendHttpGetRequest(url, false)
	crypto1 := crypto{}
	jsonErr := json.Unmarshal(body, &crypto1)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	log.Print(crypto1.Symbol)
}

// TODO
func getTime() {

}

// TODO
func generateSignature(params string) string {
	signature := ""
	return signature
}

func sendHttpGetRequest(reqUrl string, signature bool) (resBody []byte) {
	if signature {
		sig := generateSignature()
		reqUrl = reqUrl + "&signature=" + sig
	}

	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "cb-prd-test")

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

	return body
}
