package main

import (
	"encoding/json"
	"net/http"
)

type SubmitRequest struct {
	Prompt   string `json:"prompt"`
	Priority int    `json:"priority"`
}

type SubmitResponse struct {
	Text string `json:"text"`
	Err  string `json:"error,omitempty"`
}
type Server struct {
	scheduler *Scheduler
}

func NewServer(scheduler *Scheduler) *Server {
	return &Server{scheduler: scheduler}
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result := s.scheduler.Submit(req.Prompt, req.Priority)

	resp := SubmitResponse{Text: result.Text}
	if result.Err != nil {
		resp.Err = result.Err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
