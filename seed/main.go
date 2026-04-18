package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"jackpotTask/internal/config"
	"jackpotTask/internal/models"
	"jackpotTask/internal/store"
)

var currencyRates = map[string]float64{
	"BTC": 62000.0,
	"ETH": 3200.0,
	"USDT": 1.0,
}

var currencies = []string{"BTC", "ETH", "USDT"}

func main() {
	rounds := flag.Int("rounds", 2_000_000, "number of game rounds to generate")
	users := flag.Int("users", 1000, "number of unique users to generate")
	batchRounds := flag.Int("batch-rounds", 2500, "how many rounds to write per mongo batch")
	seed := flag.Int64("seed", time.Now().UnixNano(), "seed for pseudo-random generation")
	flag.Parse()

	if *rounds < 2_000_000 {
		log.Fatalf("rounds must be >= 2000000")
	}
	if *users < 500 {
		log.Fatalf("users must be >= 500")
	}
	if *batchRounds <= 0 {
		log.Fatalf("batch-rounds must be > 0")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	mongoStore, err := store.NewMongoStore(ctx, cfg.MongoURI, cfg.MongoDatabase, cfg.MongoTimeout)
	if err != nil {
		log.Fatalf("init mongo: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = mongoStore.Close(shutdownCtx)
	}()

	if err := mongoStore.Transactions.Drop(ctx); err != nil {
		log.Fatalf("drop transactions collection: %v", err)
	}
	if err := mongoStore.EnsureIndexes(ctx); err != nil {
		log.Fatalf("ensure indexes: %v", err)
	}

	rng := rand.New(rand.NewSource(*seed))
	userIDs := make([]primitive.ObjectID, *users)
	for i := range userIDs {
		userIDs[i] = primitive.NewObjectID()
	}

	start := time.Now()
	generatedRounds := 0
	for generatedRounds < *rounds {
		remaining := *rounds - generatedRounds
		batch := *batchRounds
		if remaining < batch {
			batch = remaining
		}

		docs := make([]any, 0, batch*2)
		for i := 0; i < batch; i++ {
			roundID := primitive.NewObjectID().Hex()
			userID := userIDs[rng.Intn(len(userIDs))]
			currency := currencies[rng.Intn(len(currencies))]
			baseTime := randomTimeInPastYear(rng)

			wagerAmountFloat := randomAmount(rng, currency)
			payoutFactor := 0.2 + rng.Float64()*1.6 // Payout range 20%-180% of wager.
			payoutAmountFloat := wagerAmountFloat * payoutFactor

			wagerAmount := decimalFromFloat(wagerAmountFloat)
			payoutAmount := decimalFromFloat(payoutAmountFloat)

			wagerUSD := decimalFromFloat(wagerAmountFloat * currencyRates[currency])
			payoutUSD := decimalFromFloat(payoutAmountFloat * currencyRates[currency])

			wagerTime := baseTime
			payoutTime := baseTime.Add(time.Duration(rng.Intn(240)+1) * time.Second)

			wager := models.Transaction{
				ID:        primitive.NewObjectID(),
				CreatedAt: wagerTime,
				UserID:    userID,
				RoundID:   roundID,
				Type:      models.TypeWager,
				Amount:    wagerAmount,
				Currency:  currency,
				USDAmount: wagerUSD,
			}

			payout := models.Transaction{
				ID:        primitive.NewObjectID(),
				CreatedAt: payoutTime,
				UserID:    userID,
				RoundID:   roundID,
				Type:      models.TypePayout,
				Amount:    payoutAmount,
				Currency:  currency,
				USDAmount: payoutUSD,
			}

			docs = append(docs, wager, payout)
		}

		insertCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		_, err := mongoStore.Transactions.InsertMany(insertCtx, docs)
		cancel()
		if err != nil {
			log.Fatalf("insert batch at round %d: %v", generatedRounds, err)
		}

		generatedRounds += batch
		if generatedRounds%100_000 == 0 || generatedRounds == *rounds {
			elapsed := time.Since(start).Round(time.Second)
			log.Printf("seeded rounds=%d/%d transactions=%d elapsed=%s", generatedRounds, *rounds, generatedRounds*2, elapsed)
		}
	}

	log.Printf("done seeding %d rounds (%d transactions) in %s", *rounds, *rounds*2, time.Since(start).Round(time.Second))
}

func randomTimeInPastYear(rng *rand.Rand) time.Time {
	now := time.Now().UTC()
	secondsBack := rng.Int63n(int64(365 * 24 * time.Hour / time.Second))
	return now.Add(-time.Duration(secondsBack) * time.Second)
}

func randomAmount(rng *rand.Rand, currency string) float64 {
	switch currency {
	case "BTC":
		return 0.0001 + rng.Float64()*0.02
	case "ETH":
		return 0.001 + rng.Float64()*0.3
	default:
		return 1 + rng.Float64()*500
	}
}

func decimalFromFloat(v float64) primitive.Decimal128 {
	d, err := primitive.ParseDecimal128(fmt.Sprintf("%.8f", v))
	if err != nil {
		panic(err)
	}
	return d
}

