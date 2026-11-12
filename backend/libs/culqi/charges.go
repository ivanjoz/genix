// Package culqi is the server half of Culqi: it charges a token the browser obtained from Culqi
// Checkout (a card "tkn_" or a Yape "ype_" token, both charged the same way). It holds no business
// rules — which key, which amount and what to do with the result are the caller's.
package culqi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const chargesURL = "https://api.culqi.com/v2/charges"

var httpClient = &http.Client{Timeout: 40 * time.Second}

type ChargeRequest struct {
	// Amount is in cents.
	Amount       int32             `json:"amount"`
	CurrencyCode string            `json:"currency_code"`
	Email        string            `json:"email"`
	SourceID     string            `json:"source_id"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type Charge struct {
	Object string `json:"object"`
	ID     string `json:"id"`
	Amount int32  `json:"amount"`
	// ActionCode "REVIEW" means the bank asks for 3DS authentication: the charge was not made.
	ActionCode      string `json:"action_code"`
	UserMessage     string `json:"user_message"`
	MerchantMessage string `json:"merchant_message"`
}

// ChargeError is a charge Culqi declined. UserMessage is safe to show the shopper.
type ChargeError struct {
	UserMessage     string
	MerchantMessage string
}

func (e *ChargeError) Error() string {
	return "Culqi rechazó el cargo: " + e.UserMessage + " (" + e.MerchantMessage + ")"
}

// CreateCharge charges a token with the company's secret key. It returns a *ChargeError when Culqi
// answered and declined, and a plain error when Culqi could not be reached or answered something
// unreadable: then the caller cannot know whether the money moved.
func CreateCharge(secretKey string, request ChargeRequest) (*Charge, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequest(http.MethodPost, chargesURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+secretKey)
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("no se pudo contactar a Culqi: %w", err)
	}
	defer response.Body.Close()

	charge := Charge{}
	if err := json.NewDecoder(response.Body).Decode(&charge); err != nil {
		return nil, fmt.Errorf("respuesta ilegible de Culqi (HTTP %v): %w", response.StatusCode, err)
	}
	if response.StatusCode >= 300 || charge.Object != "charge" || charge.ActionCode == "REVIEW" {
		userMessage := charge.UserMessage
		if charge.ActionCode == "REVIEW" {
			userMessage = "Su banco requiere una autenticación adicional (3DS) que esta tienda aún no admite. Intente con otra tarjeta o con Yape."
		}
		if userMessage == "" {
			userMessage = "El pago no pudo procesarse."
		}
		return nil, &ChargeError{UserMessage: userMessage, MerchantMessage: charge.MerchantMessage}
	}
	return &charge, nil
}
