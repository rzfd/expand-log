package response

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	validationFailureCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expense_tracker_validation_failures_total",
			Help: "Count of validation failures grouped by error code.",
		},
		[]string{"error_code"},
	)

	httpRequestsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "expense_tracker_http_requests_total",
			Help: "Total HTTP requests grouped by method, route, status, and outcome.",
		},
		[]string{"method", "route", "status", "outcome"},
	)
)

func init() {
	prometheus.MustRegister(validationFailureCounter)
	prometheus.MustRegister(httpRequestsCounter)
}

func IncValidationFailure(errorCode string) {
	if errorCode == "" {
		errorCode = "unknown"
	}
	validationFailureCounter.WithLabelValues(errorCode).Inc()
}

func ObserveHTTPRequest(method, route string, status int) {
	if route == "" {
		route = "unknown"
	}
	outcome := "success"
	if status >= 400 {
		outcome = "failure"
	}
	httpRequestsCounter.WithLabelValues(method, route, strconv.Itoa(status), outcome).Inc()
}
