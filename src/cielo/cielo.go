package cielo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/valyala/fastjson"
	"io"
	"net/http"
	"regexp"
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
	Number          string
	Holder          string
	ExpirationMonth int
	ExpirationYear  int
	SecurityCode    string
}

type CreditCardPayment struct {
	OrderId        string
	Amount         int
	Installments   int
	SoftDescriptor string
}

var digitRegex = regexp.MustCompile(`^\d+$`)
var digitAndLetterRegex = regexp.MustCompile(`^[a-zA-Z\d]+$`)

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

func (cieloApi *CieloApi) ProcessCreditCardPayment(payment CreditCardPayment, card CreditCard) (string, error) {
	err := validateCreditCardFields(card)

	if err != nil {
		return "", err
	}

	if payment.OrderId == "" {
		return "", errors.New("field OrderId cannot be empty")
	}

	if len(payment.OrderId) > 50 {
		return "", errors.New("field OrderId cannot exceed 50 characters")
	}

	if !digitAndLetterRegex.MatchString(payment.OrderId) {
		return "", errors.New("field OrderId can contain letters and digits only")
	}

	if payment.Amount < 1 {
		return "", errors.New("field Amount cannot be less than 1")
	}

	if payment.Installments < 1 {
		return "", errors.New("field Installments cannot be less than 1")
	}

	if payment.SoftDescriptor == "" {
		return "", errors.New("field SoftDescriptor cannot be empty")
	}

	if len(payment.SoftDescriptor) > 13 {
		return "", errors.New("field SoftDescriptor cannot exceed 13 characters")
	}

	if !digitAndLetterRegex.MatchString(payment.SoftDescriptor) {
		return "", errors.New("field SoftDescriptor can contain letters and digits only")
	}

	// TODO Accept a tokenized card
	// TODO Accept recurrent payment
	// https://docs.cielo.com.br/ecommerce-cielo/reference/criar-pagamento-credito

	payload, err := json.Marshal(map[string]interface{}{
		"MerchantOrderId": payment.OrderId,
		"Payment": map[string]interface{}{
			"Type":           "CreditCard",
			"Amount":         payment.Amount,
			"Currency":       "BRL",
			"Country":        "BRA",
			"Installments":   payment.Installments,
			"Capture":        true,
			"SoftDescriptor": payment.SoftDescriptor,
			"CreditCard": map[string]interface{}{
				"CardNumber":     card.Number,
				"Holder":         card.Holder,
				"ExpirationDate": fmt.Sprintf("%02d/%04d", card.ExpirationMonth, card.ExpirationYear),
				"SecurityCode":   card.SecurityCode,
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

func (cieloApi *CieloApi) ValidateCreditCard(card CreditCard) error {
	err := validateCreditCardFields(card)

	if err != nil {
		return err
	}

	payload, err := json.Marshal(map[string]interface{}{
		"CardType":       "CreditCard",
		"CardNumber":     card.Number,
		"Holder":         card.Holder,
		"ExpirationDate": fmt.Sprintf("%02d/%04d", card.ExpirationMonth, card.ExpirationYear),
		"SecurityCode":   card.SecurityCode,
		"SaveCard":       false,
	})

	if err != nil {
		return fmt.Errorf("failed to create request body: %w", err)
	}

	req, err := http.NewRequest("POST", cieloApi.commandBaseUrl+"/1/zeroauth", bytes.NewBuffer(payload))

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("MerchantId", cieloApi.merchantID)
	req.Header.Add("MerchantKey", cieloApi.merchantKey)

	res, err := cieloApi.client.Do(req)

	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("cielo api responded with status code %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)

	var p fastjson.Parser

	jsonDoc, err := p.ParseBytes(body)

	if err != nil {
		return fmt.Errorf("failed to parse response body: %w", err)
	}

	isValid := jsonDoc.GetBool("Valid")

	if !isValid {
		errorMessage := jsonDoc.GetStringBytes("ReturnMessage")

		if errorMessage == nil {
			return errors.New("unknown reason")
		}

		return fmt.Errorf("card is not valid: %s", errorMessage)
	}

	return nil
}

func (cieloApi *CieloApi) DetectCreditCardBrand(cardNumber string) (string, error) {
	url := fmt.Sprintf("%s/1/cardBin/%.9s", cieloApi.queryBaseUrl, cardNumber)

	req, err := http.NewRequest("GET", url, nil)

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

	if res.StatusCode != 200 {
		return "", fmt.Errorf("cielo api responded with status code %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)

	var p fastjson.Parser

	jsonDoc, err := p.ParseBytes(body)

	if err != nil {
		return "", fmt.Errorf("failed to parse response body: %w", err)
	}

	cardBrand := jsonDoc.GetStringBytes("Provider")

	if len(cardBrand) == 0 {
		return "", errors.New("could not find Provider field in response body")
	}

	return string(cardBrand), nil
}

func (cieloApi *CieloApi) TokenizeCreditCard(customerName string, card CreditCard) (string, error) {
	err := validateCreditCardFields(card)

	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(map[string]interface{}{
		"CustomerName":   customerName,
		"CardNumber":     card.Number,
		"Holder":         card.Holder,
		"ExpirationDate": fmt.Sprintf("%02d/%04d", card.ExpirationMonth, card.ExpirationYear),
		"SecurityCode":   card.SecurityCode,
	})

	if err != nil {
		return "", fmt.Errorf("failed to create request body: %w", err)
	}

	req, err := http.NewRequest("POST", cieloApi.commandBaseUrl+"/1/card", bytes.NewBuffer(payload))

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

	cardToken := jsonDoc.GetStringBytes("CardToken")

	if len(cardToken) == 0 {
		return "", errors.New("could not find CardToken field in response body")
	}

	return string(cardToken), nil
}

func validateCreditCardFields(card CreditCard) error {
	if len(card.Number) < 7 {
		return errors.New("field Number cannot be less than 7 characters")
	}

	if len(card.Number) > 19 {
		return errors.New("field Number cannot exceed 19 characters")
	}

	if !digitRegex.MatchString(card.Number) {
		return errors.New("field Number must contain only digits")
	}

	if card.Holder == "" {
		return errors.New("field Holder name cannot be empty")
	}

	if len(card.Holder) > 25 {
		return errors.New("field Holder name cannot exceed 25 characters")
	}

	if card.ExpirationMonth < 1 || card.ExpirationMonth > 12 {
		return errors.New("field ExpirationMonth must be between 1 and 12")
	}

	if card.ExpirationYear < time.Now().Year() {
		return errors.New("field ExpirationYear must be the current year or later")
	}

	if card.SecurityCode == "" {
		return errors.New("field SecurityCode name cannot be empty")
	}

	if len(card.SecurityCode) > 4 {
		return errors.New("field SecurityCode name cannot exceed 4 characters")
	}

	if !digitRegex.MatchString(card.SecurityCode) {
		return errors.New("field SecurityCode must contain only digits")
	}

	return nil
}
