package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	serverURL      = "http://localhost:8080/cotacao"
	requestTimeout = 300 * time.Millisecond
	outputFile     = "price.txt"
)

type priceResponse struct {
	Bid string `json:"bid"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, nil)
	if reqErr != nil {
		log.Fatalf("building request: %v", reqErr)
	}

	resp, doErr := http.DefaultClient.Do(req)
	if doErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Fatalf("timeout calling server (300ms exceeded): %v", doErr)
		}
		log.Fatalf("calling server: %v", doErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("server returned status %d", resp.StatusCode)
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatalf("reading response body: %v", readErr)
	}

	var cotacao priceResponse
	if unmarshalErr := json.Unmarshal(body, &cotacao); unmarshalErr != nil {
		log.Fatalf("decoding response: %v", unmarshalErr)
	}

	content := fmt.Sprintf("Dólar: %s", cotacao.Bid)
	if writeErr := os.WriteFile(outputFile, []byte(content), 0644); writeErr != nil {
		log.Fatalf("writing file: %v", writeErr)
	}

	log.Printf("saved cotacao to %s -> %s", outputFile, content)
}
