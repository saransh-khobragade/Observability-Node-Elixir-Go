package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// LogEntry represents a structured log entry - Consistent format across all services
type LogEntry map[string]interface{}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

var (
	serviceName = "go-service"
	logger      = log.New(os.Stdout, "", 0)
	
	// Prometheus metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)
	
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// log creates a structured log entry with consistent format
// Core fields at top level, request/context fields nested in "fields" object
func log(level, message string, additionalFields map[string]interface{}) {
	entry := LogEntry{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"service":   serviceName,
		"message":   message,
	}
	
	// Nest additional fields in "fields" object (consistent structure)
	if len(additionalFields) > 0 {
		entry["fields"] = additionalFields
	}
	
	jsonData, _ := json.Marshal(entry)
	logger.Println(string(jsonData))
}

// Logger convenience methods
var logInfo = func(message string, fields map[string]interface{}) {
	log("INFO", message, fields)
}

var logWarn = func(message string, fields map[string]interface{}) {
	log("WARN", message, fields)
}

var logError = func(message string, fields map[string]interface{}) {
	log("ERROR", message, fields)
}

var logDebug = func(message string, fields map[string]interface{}) {
	log("DEBUG", message, fields)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:  "healthy",
		Service: "go",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Go Service is running!"))
}

func initTracing() {
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		otlpEndpoint = "otel-collector:4317"
	}
	
	// Initialize OTLP trace exporter
	traceExp, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(otlpEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		logError("Failed to create OTLP trace exporter", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	
	// Initialize OTLP metrics exporter
	metricExp, err := otlpmetricgrpc.New(
		context.Background(),
		otlpmetricgrpc.WithEndpoint(otlpEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		logError("Failed to create OTLP metrics exporter", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		logError("Failed to create resource", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(traceExp),
		tracesdk.WithResource(res),
	)
	
	mp := metricsdk.NewMeterProvider(
		metricsdk.WithReader(metricsdk.NewPeriodicReader(metricExp)),
		metricsdk.WithResource(res),
	)
	
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	
	logInfo("OpenTelemetry SDK initialized", map[string]interface{}{
		"otlp_endpoint": otlpEndpoint,
	})
}

// loggingMiddleware logs all HTTP requests and responses
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Log incoming request
		logInfo("Incoming HTTP request", map[string]interface{}{
			"remote_addr": r.RemoteAddr,
			"method":      r.Method,
			"path":        r.URL.Path,
			"user_agent":  r.UserAgent(),
		})
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start).Seconds()
		statusCode := wrapped.statusCode
		
		// Determine log level based on status code
		var logFunc func(string, map[string]interface{})
		if statusCode >= 500 {
			logFunc = logError
		} else if statusCode >= 400 {
			logFunc = logWarn
		} else {
			logFunc = logInfo
		}
		
		// Log response
		logFunc("HTTP request completed", map[string]interface{}{
			"remote_addr":     r.RemoteAddr,
			"method":          r.Method,
			"path":            r.URL.Path,
			"status":          statusCode,
			"duration_seconds": duration,
		})
		
		// Update metrics
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	initTracing()
	
	r := mux.NewRouter()
	r.Use(otelmux.Middleware(serviceName))
	r.Use(loggingMiddleware)
	
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/", rootHandler).Methods("GET")
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	logInfo("Go service starting", map[string]interface{}{
		"port": 8080,
	})
	
	log.Fatal(http.ListenAndServe(":8080", r))
}
