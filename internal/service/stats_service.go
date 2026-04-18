package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"jackpotTask/internal/models"
)

var ErrUserNoWagers = errors.New("user has no wagers in timeframe")

type StatsService struct {
	transactions *mongo.Collection
}

func NewStatsService(transactions *mongo.Collection) *StatsService {
	return &StatsService{transactions: transactions}
}

type CurrencyGGR struct {
	Currency  string `json:"currency" bson:"currency"`
	Wager     string `json:"wager" bson:"wager"`
	Payout    string `json:"payout" bson:"payout"`
	GGR       string `json:"ggr" bson:"ggr"`
	WagerUSD  string `json:"wagerUsd" bson:"wagerUsd"`
	PayoutUSD string `json:"payoutUsd" bson:"payoutUsd"`
	GGRUSD    string `json:"ggrUsd" bson:"ggrUsd"`
}

type DailyWagerRow struct {
	Day      string `json:"day" bson:"day"`
	Currency string `json:"currency" bson:"currency"`
	Amount   string `json:"amount" bson:"amount"`
	USD      string `json:"usd" bson:"usd"`
}

type UserWagerPercentile struct {
	UserID          string `json:"userId"`
	WagerUSD        string `json:"wagerUsd"`
	Rank            int64  `json:"rank"`
	TotalUsers      int64  `json:"totalUsers"`
	TopPercent      string `json:"topPercent"`
	PercentileScore string `json:"percentileScore"`
}

func (s *StatsService) GetGGR(ctx context.Context, from, to time.Time) ([]CurrencyGGR, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"createdAt": bson.M{"$gte": from, "$lt": to},
			"type":      bson.M{"$in": []string{models.TypeWager, models.TypePayout}},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{"currency": "$currency", "type": "$type"},
			"amount": bson.M{"$sum": "$amount"},
			"usd":    bson.M{"$sum": "$usdAmount"},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":      0,
			"currency": "$_id.currency",
			"wager": bson.M{"$cond": []any{
				bson.M{"$eq": []any{"$_id.type", models.TypeWager}}, "$amount", decimalZero(),
			}},
			"payout": bson.M{"$cond": []any{
				bson.M{"$eq": []any{"$_id.type", models.TypePayout}}, "$amount", decimalZero(),
			}},
			"wagerUsd": bson.M{"$cond": []any{
				bson.M{"$eq": []any{"$_id.type", models.TypeWager}}, "$usd", decimalZero(),
			}},
			"payoutUsd": bson.M{"$cond": []any{
				bson.M{"$eq": []any{"$_id.type", models.TypePayout}}, "$usd", decimalZero(),
			}},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":       "$currency",
			"wager":     bson.M{"$sum": "$wager"},
			"payout":    bson.M{"$sum": "$payout"},
			"wagerUsd":  bson.M{"$sum": "$wagerUsd"},
			"payoutUsd": bson.M{"$sum": "$payoutUsd"},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":       0,
			"currency":  "$_id",
			"wager":     bson.M{"$toString": "$wager"},
			"payout":    bson.M{"$toString": "$payout"},
			"ggr":       bson.M{"$toString": bson.M{"$subtract": []any{"$wager", "$payout"}}},
			"wagerUsd":  bson.M{"$toString": "$wagerUsd"},
			"payoutUsd": bson.M{"$toString": "$payoutUsd"},
			"ggrUsd":    bson.M{"$toString": bson.M{"$subtract": []any{"$wagerUsd", "$payoutUsd"}}},
		}}},
		{{Key: "$sort", Value: bson.M{"currency": 1}}},
	}

	cursor, err := s.transactions.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate ggr: %w", err)
	}
	defer cursor.Close(ctx)

	var rows []CurrencyGGR
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode ggr rows: %w", err)
	}

	return rows, nil
}

func (s *StatsService) GetDailyWagerVolume(ctx context.Context, from, to time.Time) ([]DailyWagerRow, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"createdAt": bson.M{"$gte": from, "$lt": to},
			"type":      models.TypeWager,
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"day":      bson.M{"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$createdAt"}},
				"currency": "$currency",
			},
			"amount": bson.M{"$sum": "$amount"},
			"usd":    bson.M{"$sum": "$usdAmount"},
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":      0,
			"day":      "$_id.day",
			"currency": "$_id.currency",
			"amount":   bson.M{"$toString": "$amount"},
			"usd":      bson.M{"$toString": "$usd"},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "day", Value: 1}, {Key: "currency", Value: 1}}}},
	}

	cursor, err := s.transactions.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate daily wagers: %w", err)
	}
	defer cursor.Close(ctx)

	var rows []DailyWagerRow
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode daily wager rows: %w", err)
	}

	return rows, nil
}

func (s *StatsService) GetUserWagerPercentile(ctx context.Context, userID primitive.ObjectID, from, to time.Time) (UserWagerPercentile, error) {
	targetTotal, err := s.getUserWagerUSDTotal(ctx, userID, from, to)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return UserWagerPercentile{}, ErrUserNoWagers
		}
		return UserWagerPercentile{}, err
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"createdAt": bson.M{"$gte": from, "$lt": to},
			"type":      models.TypeWager,
		}}},
		{{Key: "$group", Value: bson.M{"_id": "$userId", "totalUSD": bson.M{"$sum": "$usdAmount"}}}},
		{{Key: "$group", Value: bson.M{
			"_id":        nil,
			"totalUsers": bson.M{"$sum": 1},
			"usersAbove": bson.M{"$sum": bson.M{"$cond": []any{bson.M{"$gt": []any{"$totalUSD", targetTotal}}, 1, 0}}},
		}}},
	}

	cursor, err := s.transactions.Aggregate(ctx, pipeline)
	if err != nil {
		return UserWagerPercentile{}, fmt.Errorf("aggregate percentile: %w", err)
	}
	defer cursor.Close(ctx)

	var out []struct {
		TotalUsers int64 `bson:"totalUsers"`
		UsersAbove int64 `bson:"usersAbove"`
	}
	if err := cursor.All(ctx, &out); err != nil {
		return UserWagerPercentile{}, fmt.Errorf("decode percentile: %w", err)
	}
	if len(out) == 0 || out[0].TotalUsers == 0 {
		return UserWagerPercentile{}, ErrUserNoWagers
	}

	rank := out[0].UsersAbove + 1
	total := out[0].TotalUsers
	topPercent := (float64(rank) / float64(total)) * 100.0
	percentileScore := (1.0 - (float64(rank-1) / float64(total))) * 100.0

	return UserWagerPercentile{
		UserID:          userID.Hex(),
		WagerUSD:        targetTotal.String(),
		Rank:            rank,
		TotalUsers:      total,
		TopPercent:      fmt.Sprintf("%.2f", topPercent),
		PercentileScore: fmt.Sprintf("%.2f", percentileScore),
	}, nil
}

func (s *StatsService) getUserWagerUSDTotal(ctx context.Context, userID primitive.ObjectID, from, to time.Time) (primitive.Decimal128, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"createdAt": bson.M{"$gte": from, "$lt": to},
			"type":      models.TypeWager,
			"userId":    userID,
		}}},
		{{Key: "$group", Value: bson.M{"_id": "$userId", "totalUSD": bson.M{"$sum": "$usdAmount"}}}},
	}

	cursor, err := s.transactions.Aggregate(ctx, pipeline)
	if err != nil {
		return primitive.Decimal128{}, fmt.Errorf("aggregate user total: %w", err)
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		if cursor.Err() != nil {
			return primitive.Decimal128{}, cursor.Err()
		}
		return primitive.Decimal128{}, mongo.ErrNoDocuments
	}

	var doc struct {
		TotalUSD primitive.Decimal128 `bson:"totalUSD"`
	}
	if err := cursor.Decode(&doc); err != nil {
		return primitive.Decimal128{}, fmt.Errorf("decode user total: %w", err)
	}

	return doc.TotalUSD, nil
}

func decimalZero() primitive.Decimal128 {
	z, _ := primitive.ParseDecimal128("0")
	return z
}

