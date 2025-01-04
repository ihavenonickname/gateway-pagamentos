package handlers

import (
	"gateway-pagamentos/cielo"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PaymentHandler struct {
	cieloApi *cielo.CieloApi
}

func NewPaymentHandler(cieloApi *cielo.CieloApi) *PaymentHandler {
	return &PaymentHandler{
		cieloApi: cieloApi,
	}
}

func (deps *PaymentHandler) PostPayment(c echo.Context) error {
	card := cielo.CreditCard{
		Number:          "5024007153463100",
		Holder:          "Teste\"Holder",
		ExpirationMonth: 2,
		ExpirationYear:  2030,
		SecurityCode:    "123",
	}

	payment := cielo.CreditCardPayment{
		OrderId:        strings.ReplaceAll(uuid.New().String(), "-", ""),
		Amount:         1569,
		Installments:   1,
		SoftDescriptor: "LOJATESTE",
	}

	paymentId, err := deps.cieloApi.ProcessCreditCardPayment(payment, card)

	if err != nil {
		// TODO
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Payment processing failed"})
	}

	return c.JSON(http.StatusOK, paymentId)
}
