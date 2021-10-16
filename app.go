package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type people struct {
	Symbol string `json:"symbol"`
	Time   int    `json:"time"`
}

func main() {
	// Check if there are any open trades. Continue if not.
	// https://binance-docs.github.io/apidocs/futures/en/#position-information-v2-user_data
	// Get kines data for an asset.
	// https://fapi.binance.com/fapi/v1/klines?symbol=BTCUSDT&interval=1d&limit=5
	// Calculate EMAs: EMA50H, EMA100H, EMA200D.
	// Cancel open orders, if any.
	// Open a new order based on calculations.
	// https://binance-docs.github.io/apidocs/futures/en/#new-order-trade

}

func test() {
	var url string = "https://fapi.binance.com/fapi/v1/ticker/price?symbol=BTCUSDT" // the same as url := cos

	spaceClient := http.Client{
		Timeout: time.Second * 2,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("User-Agent", "cb-prd-test")

	res, getErr := spaceClient.Do(req)
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

	people1 := people{}
	jsonErr := json.Unmarshal(body, &people1)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	fmt.Printf("HTTP: %s\n", res.Status)
	fmt.Println(people1.Symbol)
	fmt.Println(people1.Time)
}
