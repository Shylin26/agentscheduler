package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

type Result struct {
	Text string
	Err  error
}

type ScheduledRequest struct {
	Prompt     string
	PrefixHash string
	Priority   int
	Response   chan Result
}

type Scheduler struct {
	highQueue   chan ScheduledRequest
	normalQueue chan ScheduledRequest
}

func NewScheduler(queueSize int) *Scheduler {
	return &Scheduler{
		highQueue:   make(chan ScheduledRequest, queueSize),
		normalQueue: make(chan ScheduledRequest, queueSize),
	}
}
func (s *Scheduler) nextRequest(forceNormal bool) ScheduledRequest {
	if !forceNormal {
		select {
		case req := <-s.highQueue:
			return req
		default:
		}
	}

	select {
	case req := <-s.highQueue:
		return req
	case req := <-s.normalQueue:
		return req
	}
}

const prefixLength = 100

func computePrefixHash(prompt string) string {
	prefix := prompt
	if len(prefix) > prefixLength {
		prefix = prefix[:prefixLength]
	}
	sum := sha256.Sum256([]byte(prefix))
	return fmt.Sprintf("%x", sum)
}

func sortBatch(batch []ScheduledRequest) {
	sort.Slice(batch, func(i, j int) bool {
		if batch[i].Priority != batch[j].Priority {
			return batch[i].Priority > batch[j].Priority
		}
		return batch[i].PrefixHash < batch[j].PrefixHash
	})
}

const highPriorityThreshold = 5
const highPriorityWindow = 2 * time.Millisecond
const normalWindow = 10 * time.Millisecond

var batchCount int

func (s *Scheduler) collectBatch(maxBatchSize int) []ScheduledRequest {
	batch := make([]ScheduledRequest, 0, maxBatchSize)

	batchCount++
	forceNormal := batchCount%4 == 0

	first := s.nextRequest(forceNormal)
	batch = append(batch, first)

	isHighPriorityBatch := first.Priority >= highPriorityThreshold
	window := normalWindow
	sourceQueue := s.normalQueue
	if isHighPriorityBatch {
		window = highPriorityWindow
		sourceQueue = s.highQueue
	}

	timeout := time.After(window)

	for len(batch) < maxBatchSize {
		select {
		case req := <-sourceQueue:
			batch = append(batch, req)
		case <-timeout:
			sortBatch(batch)
			return batch
		}
	}

	sortBatch(batch)
	return batch
}
func (s *Scheduler) Start() {
	go func() {
		for {
			batch := s.collectBatch(8)

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

func (s *Scheduler) Submit(prompt string, priority int) Result {
	req := ScheduledRequest{
		Prompt:     prompt,
		PrefixHash: computePrefixHash(prompt),
		Priority:   priority,
		Response:   make(chan Result),
	}

	if priority >= highPriorityThreshold {
		s.highQueue <- req
	} else {
		s.normalQueue <- req
	}

	return <-req.Response
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
