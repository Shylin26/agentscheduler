package main

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
		for req := range s.queue {
			text, err := sendChatRequest(req.Prompt)
			req.Response <- Result{Text: text, Err: err}
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
