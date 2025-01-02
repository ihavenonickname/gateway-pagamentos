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
		viper.GetString("cielo.baseQueryApiUrl"))

	orderId := uuid.New().String()

	paymentId, err := cieloApi.ProcessCreditCardPayment(orderId, 1569, 1, "LOJATESTE", cielo.CreditCard{
		CardNumber:      "1234123412341231",
		Holder:          "Teste\"Holder",
		ExpirationMonth: 2,
		ExpirationYear:  2030,
		SecurityCode:    "123",
	})

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(paymentId)
}
