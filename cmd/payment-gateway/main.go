package main

import (
	"fmt"
	"payment-gateway/internal/cielo"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("appconfig")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()

	if err != nil {
		fmt.Println(err)

		return
	}

	viper.SetConfigName("appconfig.dev")

	err = viper.MergeInConfig()

	if err != nil {
		fmt.Println(err)

		return
	}

	cieloApi := cielo.NewCieloApi(
		viper.GetString("cielo.merchantId"),
		viper.GetString("cielo.merchantKey"),
		viper.GetString("cielo.baseApiUrl"),
		viper.GetString("cielo.baseQueryApiUrl"),
	)

	card := cielo.CreditCard{
		Number:          "5024007153463100",
		Holder:          "Teste\"Holder",
		ExpirationMonth: 2,
		ExpirationYear:  2030,
		SecurityCode:    "123",
	}

	fmt.Println("DetectCreditCardBrand")

	cardBrand, err := cieloApi.DetectCreditCardBrand(card.Number)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(cardBrand)
	}

	fmt.Println()

	fmt.Println("ValidateCreditCard")

	err = cieloApi.ValidateCreditCard(card)

	if err == nil {
		fmt.Println("Card is valid")
	} else {
		fmt.Println(err.Error())
	}

	fmt.Println()

	fmt.Println("TokenizeCreditCard")

	cardToken, err := cieloApi.TokenizeCreditCard("Gabriel Teste", card)

	if err == nil {
		fmt.Println(cardToken)
	} else {
		fmt.Println(err.Error())
	}

	fmt.Println()

	fmt.Println("ProcessCreditCardPayment")

	payment := cielo.CreditCardPayment{
		OrderId:        strings.ReplaceAll(uuid.New().String(), "-", ""),
		Amount:         1569,
		Installments:   1,
		SoftDescriptor: "LOJATESTE",
	}

	paymentId, err := cieloApi.ProcessCreditCardPayment(payment, card)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(paymentId)
	}
}
