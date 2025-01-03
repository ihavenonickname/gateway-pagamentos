package main

import (
	"fmt"
	"payment-gateway/internal/cielo"

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

	cardNumber := "5024007153463100"

	fmt.Println("DetectCreditCardBrand")

	cardBrand, err := cieloApi.DetectCreditCardBrand(cardNumber)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(cardBrand)
	}

	fmt.Println()

	card := cielo.CreditCard{
		CardNumber:      cardNumber,
		Holder:          "Teste\"Holder",
		ExpirationMonth: 2,
		ExpirationYear:  2030,
		SecurityCode:    "123",
	}

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
		fmt.Printf("Card token: %s\n", cardToken)
	} else {
		fmt.Println(err.Error())
	}

	fmt.Println()

	orderId := uuid.New().String()

	fmt.Println("ProcessCreditCardPayment")

	paymentId, err := cieloApi.ProcessCreditCardPayment(orderId, 1569, 1, "LOJATESTE", card)

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(paymentId)
	}
}
