// Package metrics provides a lightweight, stdlib-only instrumentation layer
// for the EduGraph API. It exposes atomic counters and gauges via a JSON
// endpoint on an internal-only port (default :9090) that Prometheus can
// scrape using a custom text-format exporter, or that Grafana can query
// directly via the JSON datasource plugin.
//
// Note: github.com/prometheus/client_golang would provide native
// /metrics in Prometheus text format but requires adding it to go.mod/go.sum.
// This implementation uses only stdlib so it builds with zero extra
// dependencies. Swap to client_golang by running:
//
//	go get github.com/prometheus/client_golang/prometheus
//	go get github.com/prometheus/client_golang/prometheus/promauto
//	go get github.com/prometheus/client_golang/prometheus/promhttp
//
// and replacing the handlers below with promhttp.Handler().
package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// ── Counters (monotonically increasing) ──────────────────────────────────────

var (
	HTTPRequestsTotal    atomic.Int64 // all completed HTTP requests
	HTTP2xxTotal         atomic.Int64
	HTTP4xxTotal         atomic.Int64
	HTTP5xxTotal         atomic.Int64
	ExamSubmissionsTotal atomic.Int64
	GCKTJobsTotal        atomic.Int64
	GCKTJobsFailed       atomic.Int64
	SkillStateUpserts    atomic.Int64
	WorkerJobsTotal      atomic.Int64
	WorkerJobsFailed     atomic.Int64
)

// ── Gauges (point-in-time values) ────────────────────────────────────────────

var (
	ActiveExamSessions atomic.Int64

	// Queue depths — updated by the poller goroutine in main.go.
	QueueCurriculumDepth  atomic.Int64
	QueueExamDepth        atomic.Int64
	QueueAnswerKeyDepth   atomic.Int64
	QueueGapDepth         atomic.Int64
	QueueStudyPlanDepth   atomic.Int64
	QueueEmbeddingDepth   atomic.Int64
	QueueGCKTDepth        atomic.Int64
)

// ── Histograms (simplified: only p50/p95/p99 approximations via reservoir) ──

// RequestDurationBuckets accumulates request duration samples in milliseconds.
// A proper histogram would use reservoir sampling; this is a lightweight
// alternative that records totals and a sum for average calculation.
var (
	HTTPDurationSumMS  atomic.Int64 // sum of all request durations in ms
	HTTPDurationCount  atomic.Int64 // number of samples (== HTTPRequestsTotal)
)

// RecordRequest increments the counters for a completed HTTP request. Call
// this from the instrumentation middleware.
func RecordRequest(status int, durationMS int64) {
	HTTPRequestsTotal.Add(1)
	HTTPDurationSumMS.Add(durationMS)
	HTTPDurationCount.Add(1)
	switch {
	case status >= 500:
		HTTP5xxTotal.Add(1)
	case status >= 400:
		HTTP4xxTotal.Add(1)
	default:
		HTTP2xxTotal.Add(1)
	}
}

// snapshot is the JSON shape served by the /metrics endpoint.
type snapshot struct {
	Timestamp          string `json:"timestamp"`
	HTTPRequestsTotal  int64  `json:"http_requests_total"`
	HTTP2xxTotal       int64  `json:"http_2xx_total"`
	HTTP4xxTotal       int64  `json:"http_4xx_total"`
	HTTP5xxTotal       int64  `json:"http_5xx_total"`
	AvgRequestDurationMS float64 `json:"avg_request_duration_ms"`
	ActiveExamSessions int64  `json:"active_exam_sessions"`
	ExamSubmissions    int64  `json:"exam_submissions_total"`
	GCKTJobsTotal      int64  `json:"gckt_jobs_total"`
	GCKTJobsFailed     int64  `json:"gckt_jobs_failed"`
	SkillStateUpserts  int64  `json:"skill_state_upserts"`
	WorkerJobsTotal    int64  `json:"worker_jobs_total"`
	WorkerJobsFailed   int64  `json:"worker_jobs_failed"`
	QueueDepths        map[string]int64 `json:"queue_depths"`
}

func collect() snapshot {
	count := HTTPDurationCount.Load()
	var avg float64
	if count > 0 {
		avg = float64(HTTPDurationSumMS.Load()) / float64(count)
	}
	return snapshot{
		Timestamp:            time.Now().UTC().Format(time.RFC3339),
		HTTPRequestsTotal:    HTTPRequestsTotal.Load(),
		HTTP2xxTotal:         HTTP2xxTotal.Load(),
		HTTP4xxTotal:         HTTP4xxTotal.Load(),
		HTTP5xxTotal:         HTTP5xxTotal.Load(),
		AvgRequestDurationMS: avg,
		ActiveExamSessions:   ActiveExamSessions.Load(),
		ExamSubmissions:      ExamSubmissionsTotal.Load(),
		GCKTJobsTotal:        GCKTJobsTotal.Load(),
		GCKTJobsFailed:       GCKTJobsFailed.Load(),
		SkillStateUpserts:    SkillStateUpserts.Load(),
		WorkerJobsTotal:      WorkerJobsTotal.Load(),
		WorkerJobsFailed:     WorkerJobsFailed.Load(),
		QueueDepths: map[string]int64{
			"queue:curriculum:parse":    QueueCurriculumDepth.Load(),
			"queue:exam:parse":          QueueExamDepth.Load(),
			"queue:exam:answerkey":      QueueAnswerKeyDepth.Load(),
			"queue:gap:analyze":         QueueGapDepth.Load(),
			"queue:studyplan:generate":  QueueStudyPlanDepth.Load(),
			"queue:embedding:generate":  QueueEmbeddingDepth.Load(),
			"queue:gckt:trace":          QueueGCKTDepth.Load(),
		},
	}
}

// Handler serves the metrics snapshot as JSON. Mount on an internal port
// (9090) that is NOT exposed via the public load balancer in production.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(collect())
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"status":"ok"}`)
	})
	return mux
}

// InstrumentHandler wraps an HTTP handler with request-level metrics recording.
func InstrumentHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusCapture{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rw, r)
		RecordRequest(rw.code, time.Since(start).Milliseconds())
	})
}

type statusCapture struct {
	http.ResponseWriter
	code int
}

func (s *statusCapture) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

// StartQueuePoller starts a goroutine that refreshes queue depth gauges every
// 30 s. Call from main.go after the Redis client is initialised.
func StartQueuePoller(ctx context.Context, redis *goredis.Client) {
	queues := []struct {
		name  string
		gauge *atomic.Int64
	}{
		{"queue:curriculum:parse", &QueueCurriculumDepth},
		{"queue:exam:parse", &QueueExamDepth},
		{"queue:exam:answerkey", &QueueAnswerKeyDepth},
		{"queue:gap:analyze", &QueueGapDepth},
		{"queue:studyplan:generate", &QueueStudyPlanDepth},
		{"queue:embedding:generate", &QueueEmbeddingDepth},
		{"queue:gckt:trace", &QueueGCKTDepth},
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, q := range queues {
					n, err := redis.LLen(ctx, q.name).Result()
					if err == nil {
						q.gauge.Store(n)
					}
				}
			}
		}
	}()
}
