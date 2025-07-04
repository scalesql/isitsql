package app

import (
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/scalesql/isitsql/internal/failure"
	"github.com/prometheus/client_golang/prometheus"
)

// consider: https://github.com/go-chi/chi/blob/master/middleware/nocache.go

// panicHandler is the outer middleware wrapper
// func panicHandler(h httprouter.Handle) httprouter.Handle {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		defer failure.HandlePanic()
// 		h(w, r, ps)
// 	}
// }

var (
	httpRequestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests received",
	}, []string{"status", "endpoint"})

	httpRequestSeconds = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total_seconds",
		Help: "Total seconds to process HTTP requests",
	}, []string{"status", "endpoint"})

	responseTimeHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds",
		Help: "Histogram of response time for handler in seconds",
		// Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		Buckets: []float64{.000_001, .000_01, .000_1, .001, .01, .1, 1, 2, 5, 10},
	})
)

func init() {
	prometheus.MustRegister(httpRequestCounter)
	prometheus.MustRegister(responseTimeHistogram)
	prometheus.MustRegister(httpRequestSeconds)
}

// Middleware to monitor HTTP requests
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the ResponseWriter to capture the status code
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		start := time.Now()

		// Process the request
		next.ServeHTTP(recorder, r)
		dur := time.Since(start)
		path := r.URL.Path // Path can be adjusted for aggregation (e.g., `/users/:id` → `/users/{id}`)
		var endpoint string
		top := topfolder(path)
		switch top {
		case "api":
			endpoint = "api"
		case "metrics":
			endpoint = "metrics"
		default:
			endpoint = "html"
		}
		status := strconv.Itoa(recorder.statusCode)

		// Increase the counters
		httpRequestCounter.WithLabelValues(status, endpoint).Inc()
		httpRequestSeconds.WithLabelValues(status, endpoint).Add(dur.Seconds())
		responseTimeHistogram.Observe(dur.Seconds())
	})
}

// topfolder returns the first folder in a path
// /metrics/isitsql returns "metrics"
func topfolder(url string) string {
	url = path.Clean(url)
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// Helper to capture HTTP status codes
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// panicHandler is the outer middleware wrapper
func panicHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer failure.HandlePanic()
		next.ServeHTTP(w, r)
	})
}
