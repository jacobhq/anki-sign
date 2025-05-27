package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jacobhq/unicornhatmini-go"
)

const (
	AnkiHost   = "DESKTOP-028KM9I.local"
	Days       = 119
	Brightness = 0.1
)

var Colors = [6][3]uint8{
	{0, 0, 0},
	{144, 238, 144},
	{0, 200, 0},
	{0, 128, 0},
	{0, 100, 0},
	{0, 64, 0},
}

type ankiRequest struct {
	Action  string      `json:"action"`
	Version int         `json:"version"`
	Params  interface{} `json:"params,omitempty"`
}

type ankiResponse struct {
	Result [][]interface{} `json:"result"`
}

func getDailyReviewCounts(days int) []int {
	reviewMap := make(map[string]int)

	reqBody := ankiRequest{Action: "getNumCardsReviewedByDay", Version: 6}
	data, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(
		fmt.Sprintf("http://%s:8765", AnkiHost),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch review data: %v\n", err)
		return make([]int, days)
	}
	defer resp.Body.Close()

	var ar ankiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode Anki response: %v\n", err)
		return make([]int, days)
	}

	for _, entry := range ar.Result {
		if dateStr, ok := entry[0].(string); ok {
			if countFloat, ok := entry[1].(float64); ok {
				reviewMap[dateStr] = int(countFloat)
			}
		}
	}

	today := time.Now().Truncate(24 * time.Hour)
	history := make([]int, days)
	for i := 0; i < days; i++ {
		date := today.AddDate(0, 0, -(days - 1 - i))
		dateKey := date.Format("2006-01-02")
		history[i] = reviewMap[dateKey]
	}

	return history
}

func mapToColorBuckets(counts []int) []int {
	maxCount := 1
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	buckets := make([]int, len(counts))
	for i, c := range counts {
		if c == 0 {
			buckets[i] = 0
		} else {
			level := int(5 * c / maxCount)
			if level > 5 {
				level = 5
			}
			buckets[i] = level
		}
	}
	return buckets
}

func drawHeatmap(buckets []int) (*unicornhatmini_go.UnicornHATMini, error) {
	h, err := unicornhatmini_go.NewUnicornhatmini()
	if err != nil {
		return nil, err
	}
	h.SetBrightness(Brightness)
	h.Clear()

	for i, level := range buckets {
		x := i / unicornhatmini_go.Rows
		y := i % unicornhatmini_go.Rows
		if x < unicornhatmini_go.Cols {
			r, g, b := Colors[level][0], Colors[level][1], Colors[level][2]
			h.SetPixel(x, y, r, g, b)
		}
	}
	if err := h.Show(); err != nil {
		return nil, err
	}
	return h, nil
}

func main() {
	counts := getDailyReviewCounts(Days)
	buckets := mapToColorBuckets(counts)
	hat, err := drawHeatmap(buckets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error displaying heatmap: %v\n", err)
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("Heatmap displayed. Press Ctrl+C to exit.")
	<-sig

	hat.Clear()
	hat.Show()
	os.Exit(0)
}
