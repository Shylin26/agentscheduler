package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Result struct {
	Text string
	Err  error
}

type ScheduledRequest struct {
	Prompt   string
	Response chan Result
}

type Scheduler struct {
	queue chan ScheduledRequest
}

func NewScheduler(queueSize int) *Scheduler {
	return &Scheduler{
		queue: make(chan ScheduledRequest, queueSize),
	}
}

func (s *Scheduler) Start() {
	go func() {
		for {
			batch := s.collectBatch(8, 10*time.Millisecond)

			prompts := make([]string, len(batch))
			for i, req := range batch {
				prompts[i] = req.Prompt
			}

			completions, err := sendBatchRequest(prompts)
			if err != nil {
				for _, req := range batch {
					req.Response <- Result{Err: err}
				}
				continue
			}

			if len(completions) != len(batch) {
				mismatchErr := fmt.Errorf("batch size mismatch: sent %d prompts, got %d completions", len(batch), len(completions))
				for _, req := range batch {
					req.Response <- Result{Err: mismatchErr}
				}
				continue
			}

			for i, req := range batch {
				req.Response <- Result{Text: completions[i]}
			}
		}
	}()
}

func (s *Scheduler) Submit(prompt string) Result {
	req := ScheduledRequest{
		Prompt:   prompt,
		Response: make(chan Result),
	}

	s.queue <- req

	return <-req.Response
}

func (s *Scheduler) collectBatch(maxBatchSize int, window time.Duration) []ScheduledRequest {
	batch := make([]ScheduledRequest, 0, maxBatchSize)

	first := <-s.queue
	batch = append(batch, first)

	timeout := time.After(window)

	for len(batch) < maxBatchSize {
		select {
		case req := <-s.queue:
			batch = append(batch, req)
		case <-timeout:
			return batch
		}
	}

	return batch
}

type BatchRequest struct {
	Prompts []string `json:"prompts"`
}

type BatchResponse struct {
	Completions []string `json:"completions"`
}

func sendBatchRequest(prompts []string) ([]string, error) {
	reqBody := BatchRequest{Prompts: prompts}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	resp, err := http.Post(
		"http://127.0.0.1:8081",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send batch request: %w", err)
	}
	defer resp.Body.Close()

	var batchResp BatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode batch response: %w", err)
	}

	return batchResp.Completions, nil
}
