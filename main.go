package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux" // Importing the mux package
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type PointsResponse struct {
	Points int `json:"points"`
}

type IDResponse struct {
	ID string `json:"id"`
}

var (
	receipts = make(map[string]Receipt)
	points   = make(map[string]int)
	mutex    sync.Mutex
)

func main() {
	// Create a new mux router
	r := mux.NewRouter()

	// Define routes
	r.HandleFunc("/receipts/process", handleProcessReceipt).Methods("POST")
	r.HandleFunc("/receipts/{id:[a-zA-Z0-9-]+}", handleGetPoints).Methods("GET") // Capture receipt ID as a URL parameter

	// Start the server with mux as the router
	fmt.Println("Server is running on port 8080...")
	http.ListenAndServe(":8080", r)
}

func handleProcessReceipt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var receipt Receipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Generate a new unique ID for the receipt
	id := uuid.New().String()
	// Calculate points based on the receipt
	calculatedPoints := calculatePoints(receipt)

	// Lock the receipts and points map to prevent race conditions
	mutex.Lock()
	receipts[id] = receipt
	points[id] = calculatedPoints
	mutex.Unlock()

	// Respond with the new receipt ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(IDResponse{ID: id})
}

func handleGetPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract the receipt ID from the URL path
	vars := mux.Vars(r) // Get the variables from the URL
	id := vars["id"]

	// Lock the maps to safely access data
	mutex.Lock()
	calculatedPoints, exists := points[id]
	mutex.Unlock()

	if !exists {
		http.Error(w, "No receipt found for that ID", http.StatusNotFound)
		return
	}

	// Respond with the points associated with the receipt
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PointsResponse{Points: calculatedPoints})
}
func calculatePoints(receipt Receipt) int {
	points := 0

	// Rule 1: One point for every alphanumeric character in the retailer name
	re := regexp.MustCompile(`[a-zA-Z0-9]`)
	rule1Points := len(re.FindAllString(receipt.Retailer, -1))
	points += rule1Points
	fmt.Printf("Rule 1: %d points for retailer name (%s)\n", rule1Points, receipt.Retailer)

	// Rule 2: 50 points if the total is a round dollar amount with no cents
	rule2Points := 0
	if strings.HasSuffix(receipt.Total, ".00") {
		rule2Points = 50
		points += rule2Points
	}
	fmt.Printf("Rule 2: %d points for total (%s)\n", rule2Points, receipt.Total)

	// Rule 3: 25 points if the total is a multiple of 0.25
	total, _ := strconv.ParseFloat(receipt.Total, 64)
	rule3Points := 0
	if math.Mod(total, 0.25) == 0 {
		rule3Points = 25
		points += rule3Points
	}
	fmt.Printf("Rule 3: %d points for total (%s)\n", rule3Points, receipt.Total)

	// Rule 4: 5 points for every two items on the receipt
	rule4Points := (len(receipt.Items) / 2) * 5
	points += rule4Points
	fmt.Printf("Rule 4: %d points for %d items\n", rule4Points, len(receipt.Items))

	// Rule 5: Points for item description length multiple of 3
	rule5Points := 0
	for _, item := range receipt.Items {
		descriptionLength := len(strings.TrimSpace(item.ShortDescription))
		if descriptionLength%3 == 0 {
			price, _ := strconv.ParseFloat(item.Price, 64)
			itemPoints := int(math.Ceil(price * 0.2))
			rule5Points += itemPoints
			fmt.Printf("Rule 5: +%d points for item description (%s)\n", itemPoints, item.ShortDescription)
		}
	}
	points += rule5Points
	
	// I was not sure about the rule 6 but if It was complusory to implement it i would implement it as giving below
	// rule6Points := 0
	// if total > 10.00 {
	// 	rule6Points = 5
	// 	points += rule6Points
	// }
	// fmt.Printf("Rule 6: %d points for total > 10.00\n", rule6Points)

	// Rule 7: 6 points if the day in the purchase date is odd
	rule7Points := 0
	purchaseDate, _ := time.Parse("2006-01-02", receipt.PurchaseDate)
	if purchaseDate.Day()%2 != 0 {
		rule7Points = 6
		points += rule7Points
	}
	fmt.Printf("Rule 7: %d points for odd purchase date (%s)\n", rule7Points, receipt.PurchaseDate)

	// Rule 8: 10 points if the time of purchase is after 2:00pm and before 4:00pm
	rule8Points := 0
	purchaseTime, _ := time.Parse("15:04", receipt.PurchaseTime)
	if purchaseTime.Hour() == 14 || (purchaseTime.Hour() == 15 && purchaseTime.Minute() < 60) {
		rule8Points = 10
		points += rule8Points
	}
	fmt.Printf("Rule 8: %d points for purchase time (%s)\n", rule8Points, receipt.PurchaseTime)

	// Total points
	fmt.Printf("Total points: %d\n", points)
	return points
}

