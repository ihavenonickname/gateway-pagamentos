package cielo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/valyala/fastjson"
	"io"
	"net/http"
	"time"
)

type CieloApi struct {
	merchantID     string
	merchantKey    string
	commandBaseUrl string
	queryBaseUrl   string
	client         *http.Client
}

type CreditCard struct {
	CardNumber      string
	Holder          string
	ExpirationMonth int
	ExpirationYear  int
	SecurityCode    string
}

type httpRetryTransport struct{}

var backoffDurations = []time.Duration{
	0 * time.Second,
	1 * time.Second,
	5 * time.Second,
	15 * time.Second,
}

func (t *httpRetryTransport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	for _, backoffDuration := range backoffDurations {
		time.Sleep(backoffDuration)

		res, err = http.DefaultTransport.RoundTrip(req)

		if err == nil && res.StatusCode < 500 {
			return res, nil
		}
	}

	if err != nil {
		return nil, fmt.Errorf("too many retries: %w", err)
	}

	if res != nil {
		return nil, fmt.Errorf("too many retries: received %d status code", res.StatusCode)
	}

	return nil, errors.New("Too many retries")
}

func NewCieloApi(merchantID string, merchantKey string, commandBaseUrl string, queryBaseUrl string) *CieloApi {
	return &CieloApi{
		merchantID:     merchantID,
		merchantKey:    merchantKey,
		commandBaseUrl: commandBaseUrl,
		queryBaseUrl:   queryBaseUrl,
		client: &http.Client{
			Transport: &httpRetryTransport{},
			Timeout:   30 * time.Second,
		},
	}
}

func (cieloApi *CieloApi) ProcessCreditCardPayment(orderId string, amount int, installments int, softDescriptor string, card CreditCard) (string, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"MerchantOrderId": orderId,
		"Payment": map[string]interface{}{
			"Type":           "CreditCard",
			"Amount":         amount,
			"Currency":       "BRL",
			"Country":        "BRA",
			"Installments":   installments,
			"Capture":        true,
			"SoftDescriptor": softDescriptor,
			"CreditCard": map[string]interface{}{
				"CardNumber":     card.CardNumber,
				"Holder":         card.Holder,
				"ExpirationDate": fmt.Sprintf("%02d/%04d", card.ExpirationMonth, card.ExpirationYear),
				"SecurityCode":   card.SecurityCode,
				"Brand":          "Visa",
			},
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to create request body: %w", err)
	}

	req, err := http.NewRequest("POST", cieloApi.commandBaseUrl+"/1/sales", bytes.NewBuffer(payload))

	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("MerchantId", cieloApi.merchantID)
	req.Header.Add("MerchantKey", cieloApi.merchantKey)

	res, err := cieloApi.client.Do(req)

	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}

	defer res.Body.Close()

	if res.StatusCode != 201 {
		return "", fmt.Errorf("cielo api responded with status code %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)

	var p fastjson.Parser

	jsonDoc, err := p.ParseBytes(body)

	if err != nil {
		return "", fmt.Errorf("failed to parse response body: %w", err)
	}

	paymentId := jsonDoc.GetStringBytes("Payment", "PaymentId")

	if len(paymentId) == 0 {
		return "", errors.New("transaction confirmed, but could not get PaymentId")
	}

	return string(paymentId), nil
}
