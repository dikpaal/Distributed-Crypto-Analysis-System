package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	// Run TUI to select coin
	symbol, err := RunTUI()
	if err != nil {
		fmt.Println("Cancelled.")
		os.Exit(0)
	}

	coinName := GetCoinName(symbol)
	fmt.Printf("\nStarting Trading Pipeline for %s...\n", coinName)

	// Create server
	server := NewServer()

	// Price channel for Binance updates
	priceChan := make(chan PriceUpdate, 100)

	// Start Binance WebSocket connection in background
	go ConnectBinance(symbol, priceChan)

	// Process incoming prices
	go func() {
		for update := range priceChan {
			server.UpdatePrice(update.Price)
		}
	}()

	// Setup HTTP routes
	http.HandleFunc("/api/price", server.HandlePrice)
	http.HandleFunc("/api/stats", server.HandleStats)
	http.HandleFunc("/ws", server.HandleWebSocket)

	// Start HTTP server in background
	go func() {
		http.ListenAndServe(":8080", nil)
	}()

	// Wait for initial data
	fmt.Println("Connecting to Binance...")
	time.Sleep(2 * time.Second)

	// Run the dashboard TUI
	if err := RunDashboard(symbol, server); err != nil {
		fmt.Printf("Dashboard error: %v\n", err)
		os.Exit(1)
	}
}
