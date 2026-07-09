package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type ChatResponse struct {
	Choices []Choice `json:"choices"`
}
type Choice struct {
	Message Message `json:"message"`
}

func sendChatRequest(prompt string) (string, error) {
	reqBody := ChatRequest{
		Model: "mlx-community/Qwen2.5-0.5B-Instruct-4bit",
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	resp, err := http.Post(
		"http://127.0.0.1:8080/v1/chat/completions",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned in response")
	}
	return chatResp.Choices[0].Message.Content, nil
}
func main() {
	scheduler := NewScheduler(50)
	scheduler.Start()

	const numRequests = 8
	var wg sync.WaitGroup

	overallStart := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start := time.Now()
			result := scheduler.Submit("Say hello in one short sentence.")
			elapsed := time.Since(start)
			if result.Err != nil {
				fmt.Printf("request %d failed: %v\n", i, result.Err)
				return
			}
			fmt.Printf("request %d finished in %v: %s\n", i, elapsed, result.Text)
		}(i)
	}

	wg.Wait()
	overallElapsed := time.Since(overallStart)
	fmt.Printf("\nAll %d requests finished in %v (wall clock)\n", numRequests, overallElapsed)
}

func timedChatRequest(id int, prompt string, results chan<- time.Duration) {
	start := time.Now()
	_, err := sendChatRequest(prompt)
	if err != nil {
		fmt.Printf("request %d failed: %v\n", id, err)
		results <- 0
		return
	}
	elapsed := time.Since(start)
	fmt.Printf("request %d finished in %v\n", id, elapsed)
	results <- elapsed
}
