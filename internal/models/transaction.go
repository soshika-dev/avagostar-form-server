package models

import "time"

type Transaction struct {
	ID              string    `json:"id"`
	CreatedByUserID string    `json:"created_by_user_id"`
	ReceiverType    string    `json:"receiver_type"`
	ReceiverName    string    `json:"receiver_name"`
	ReceiverID      *string   `json:"receiver_id,omitempty"`
	PayerType       string    `json:"payer_type"`
	PayerName       string    `json:"payer_name"`
	PayerID         *string   `json:"payer_id,omitempty"`
	PaymentMethod   string    `json:"payment_method"`
	Currency        string    `json:"currency"`
	Amount          float64   `json:"amount"`
	Description     *string   `json:"description,omitempty"`
	DatetimeUTC     time.Time `json:"-"`
	Timezone        string    `json:"timezone"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
