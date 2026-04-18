package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	TypeWager  = "Wager"
	TypePayout = "Payout"
)

type Transaction struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	CreatedAt time.Time           `bson:"createdAt" json:"createdAt"`
	UserID    primitive.ObjectID  `bson:"userId" json:"userId"`
	RoundID   string              `bson:"roundId" json:"roundId"`
	Type      string              `bson:"type" json:"type"`
	Amount    primitive.Decimal128 `bson:"amount" json:"amount"`
	Currency  string              `bson:"currency" json:"currency"`
	USDAmount primitive.Decimal128 `bson:"usdAmount" json:"usdAmount"`
}

