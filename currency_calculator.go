package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func getConversionRate(fromCurrency, toCurrency, apiKey string) (float64, error) {
	url := fmt.Sprintf("https://api.freecurrencyapi.com/v1/latest?apikey=%s&currencies=%s&base_currency=%s", apiKey, toCurrency, fromCurrency)
	resp, err := http.Get(url)
	if err != nil {
		logger.WithError(err).Error("Failed to get response from currency API")
		return 0, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.WithError(err).Error("Failed to read response body")
		return 0, err
	}

	bodyString := string(bodyBytes)
	logger.Infof("Currency API response: %s", bodyString)

	var result struct {
		Data map[string]float64 `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		logger.WithError(err).Error("JSON decoding failed")
		return 0, err
	}

	rate, exists := result.Data[toCurrency]
	if !exists {
		errMsg := fmt.Sprintf("Conversion rate for %s to %s not found", fromCurrency, toCurrency)
		logger.Error(errMsg)
		return 0, fmt.Errorf(errMsg)
	}

	return rate, nil
}
