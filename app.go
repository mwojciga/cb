package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type crypto struct {
	Symbol string `json:"symbol"`
	Time   int    `json:"time"`
}

var baseApiUrl string = "https://fapi.binance.com"

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

func getPriceData(symbol string, interval string, limit int) {
	// Example: https://fapi.binance.com/fapi/v1/klines?symbol=BTCUSDT&interval=1d&limit=5
	url := baseApiUrl + "/fapi/v1/klines?symbol=" + symbol + "&interval=" + interval + "&limit=" + strconv.Itoa(limit)
	body := sendHttpRequest(url)
	crypto1 := crypto{}
	jsonErr := json.Unmarshal(body, &crypto1)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	fmt.Println(crypto1.Symbol)
	fmt.Println(crypto1.Time)
}

func sendHttpRequest(reqUrl string) (resBody []byte) {
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
