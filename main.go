package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jacobhq/unicornhatmini-go"
)

const (
	ANKI_HOST  = "DESKTOP-028KM9I.local"
	DAYS       = 119
	BRIGHTNESS = 1.0
)

var COLORS = [6][3]uint8{
	{0, 0, 0},
	{144, 238, 144},
	{0, 200, 0},
	{0, 128, 0},
	{0, 100, 0},
	{0, 64, 0},
}

type AnkiResponse struct {
	Result [][]interface{} `json:"result"`
	Error  interface{}     `json:"error"`
}

func getDailyReviewCounts(days int) []int {
	url := fmt.Sprintf("http://%s:8765", ANKI_HOST)
	payload := map[string]interface{}{
		"action":  "getNumCardsReviewedByDay",
		"version": 6,
	}
	jsonPayload, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println("Failed to fetch review data:", err)
		return make([]int, days)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to read response:", err)
		return make([]int, days)
	}

	var res AnkiResponse
	if err := json.Unmarshal(body, &res); err != nil {
		fmt.Println("Failed to parse response:", err)
		return make([]int, days)
	}

	if res.Error != nil {
		fmt.Println("API error:", res.Error)
		return make([]int, days)
	}

	// Build a map[time.Time]int for quick lookup
	reviewMap := make(map[time.Time]int)
	for _, entry := range res.Result {
		if len(entry) != 2 {
			continue
		}
		dateStr, ok1 := entry[0].(string)
		countFloat, ok2 := entry[1].(float64)
		if !ok1 || !ok2 {
			continue
		}
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		reviewMap[date] = int(countFloat)
	}

	today := time.Now().Truncate(24 * time.Hour)
	history := make([]int, days)
	for i := days - 1; i >= 0; i-- {
		date := today.AddDate(0, 0, -i)
		if val, found := reviewMap[date]; found {
			history[days-1-i] = val
		} else {
			history[days-1-i] = 0
		}
	}

	return history
}

func mapToColorBuckets(counts []int) []int {
	maxCount := 1
	for _, v := range counts {
		if v > maxCount {
			maxCount = v
		}
	}
	buckets := make([]int, len(counts))
	for i, count := range counts {
		if count == 0 {
			buckets[i] = 0
		} else {
			level := 5 * count / maxCount
			if level > 5 {
				level = 5
			}
			buckets[i] = level
		}
	}
	return buckets
}

func drawHeatmap(buckets []int) (*unicornhatmini_go.UnicornHATMini, error) {
	uhm, err := unicornhatmini_go.NewUnicornhatmini()
	if err != nil {
		return nil, fmt.Errorf("failed to init unicornhatmini: %w", err)
	}
	uhm.SetBrightness(BRIGHTNESS)
	uhm.Clear()

	for i, level := range buckets {
		x := i / 7
		y := i % 7
		if x < 17 {
			r, g, b := COLORS[level][0], COLORS[level][1], COLORS[level][2]
			uhm.SetPixel(x, y, r, g, b)
		}
	}

	if err := uhm.Show(); err != nil {
		return nil, err
	}

	return uhm, nil
}

func main() {
	counts := getDailyReviewCounts(DAYS)
	buckets := mapToColorBuckets(counts)

	uhm, err := drawHeatmap(buckets)
	if err != nil {
		fmt.Println("Error displaying heatmap:", err)
		return
	}

	fmt.Println("Heatmap displayed. Press Ctrl+C to exit.")

	// Handle Ctrl+C to clear the display before exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("Exiting and clearing display.")
	uhm.Clear()
	uhm.Show()
}
