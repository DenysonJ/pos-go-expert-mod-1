package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

const (
	apiURL     = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	apiTimeout = 200 * time.Millisecond
	dbTimeout  = 10 * time.Millisecond
	serverAddr = ":8080"
	dbFile     = "prices.db"
)

type apiResponse struct {
	USDBRL Price `json:"USDBRL"`
}

type Price struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

func main() {
	db, openErr := sql.Open("sqlite", dbFile)
	if openErr != nil {
		log.Fatalf("opening database: %v", openErr)
	}
	defer db.Close()

	if migrateErr := migrate(db); migrateErr != nil {
		log.Fatalf("migrating database: %v", migrateErr)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", priceHandler(db))

	log.Printf("server listening on %s", serverAddr)
	if listenErr := http.ListenAndServe(serverAddr, mux); listenErr != nil {
		log.Fatalf("listening: %v", listenErr)
	}
}

func migrate(db *sql.DB) error {
	const schema = `
		CREATE TABLE IF NOT EXISTS prices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL,
			codein TEXT NOT NULL,
			bid TEXT NOT NULL,
			ask TEXT NOT NULL,
			high TEXT NOT NULL,
			low TEXT NOT NULL,
			create_date TEXT NOT NULL,
			recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, execErr := db.Exec(schema)
	return execErr
}

func priceHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		price, fetchErr := fetchPrice(r.Context())
		if fetchErr != nil {
			log.Printf("error fetching price: %v", fetchErr)
			http.Error(w, "failed to fetch price", http.StatusInternalServerError)
			return
		}

		if saveErr := savePrice(r.Context(), db, price); saveErr != nil {
			log.Printf("error saving price: %v", saveErr)
		}

		w.Header().Set("Content-Type", "application/json")
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{"bid": price.Bid}); encodeErr != nil {
			log.Printf("encoding response: %v", encodeErr)
		}
	}
}

func fetchPrice(parent context.Context) (Price, error) {
	ctx, cancel := context.WithTimeout(parent, apiTimeout)
	defer cancel()

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if reqErr != nil {
		return Price{}, reqErr
	}

	resp, doErr := http.DefaultClient.Do(req)
	if doErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Price{}, errors.New("timeout calling external API (200ms exceeded)")
		}
		return Price{}, doErr
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return Price{}, readErr
	}

	var parsed apiResponse
	if unmarshalErr := json.Unmarshal(body, &parsed); unmarshalErr != nil {
		return Price{}, unmarshalErr
	}

	return parsed.USDBRL, nil
}

func savePrice(parent context.Context, db *sql.DB, c Price) error {
	ctx, cancel := context.WithTimeout(parent, dbTimeout)
	defer cancel()

	const query = `
		INSERT INTO prices (code, codein, bid, ask, high, low, create_date)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, execErr := db.ExecContext(ctx, query, c.Code, c.Codein, c.Bid, c.Ask, c.High, c.Low, c.CreateDate)
	if execErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New("timeout persisting to database (10ms exceeded)")
		}
		return execErr
	}
	return nil
}
