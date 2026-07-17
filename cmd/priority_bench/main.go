package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type submitReq struct {
	Prompt   string `json:"prompt"`
	Priority int    `json:"priority"`
}

func runBatch(n int, priority int) time.Duration {
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, _ := json.Marshal(submitReq{Prompt: "Say hello.", Priority: priority})
			resp, err := http.Post("http://127.0.0.1:9000/submit", "application/json", bytes.NewBuffer(body))
			if err != nil {
				fmt.Println("request failed:", err)
				return
			}
			defer resp.Body.Close()
		}()
	}

	wg.Wait()
	return time.Since(start)
}

func main() {
	const n = 5
	const rounds = 5

	fmt.Println("Warming up...")
	runBatch(n, 0)

	var lowTotal, highTotal time.Duration

	for i := 0; i < rounds; i++ {
		low := runBatch(n, 0)
		high := runBatch(n, 8)
		fmt.Printf("round %d: low=%v high=%v\n", i, low, high)
		lowTotal += low
		highTotal += high
	}

	fmt.Printf("\nAverage low priority:  %v\n", lowTotal/rounds)
	fmt.Printf("Average high priority: %v\n", highTotal/rounds)
}
