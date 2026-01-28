package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Reporter struct {
	ControlPlaneURL string
	HTTPClient      *http.Client
	WorkerName      string
}

func (r Reporter) enabled() bool {
	return r.ControlPlaneURL != ""
}

func (r Reporter) client() *http.Client {
	if r.HTTPClient != nil {
		return r.HTTPClient
	}
	return &http.Client{Timeout: 3 * time.Second}
}

type ReportAttemptRequest struct {
	Attempt     int        `json:"attempt"`
	Status      string     `json:"status"` // running/success/fail/dead
	AsynqTaskID string     `json:"asynq_task_id,omitempty"`
	Error       string     `json:"error,omitempty"`
	WorkerName  string     `json:"worker_name,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	DurationMs  *int       `json:"duration_ms,omitempty"`
	TraceID     string     `json:"trace_id,omitempty"`
	SpanID      string     `json:"span_id,omitempty"`
}

func (r Reporter) ReportAttempt(ctx context.Context, taskID string, req ReportAttemptRequest) error {
	if !r.enabled() {
		return nil
	}
	if req.WorkerName == "" {
		req.WorkerName = r.WorkerName
	}

	b, _ := json.Marshal(req)
	u := fmt.Sprintf("%s/api/v1/tasks/%s/report-attempt", r.ControlPlaneURL, taskID)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client().Do(httpReq)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("report attempt failed: status=%d", resp.StatusCode)
	}
	return nil
}
