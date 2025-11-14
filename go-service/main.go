package main

import (
	"context"
	"encoding/json"
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

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

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

func structuredLog(level, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Service:   serviceName,
		Message:   message,
		Fields:    fields,
	}
	
	jsonData, _ := json.Marshal(entry)
	logger.Println(string(jsonData))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	fields := map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"method":      r.Method,
		"path":        r.URL.Path,
	}
	structuredLog("INFO", "Health check requested", fields)
	
	response := HealthResponse{
		Status:  "healthy",
		Service: "go",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	
	duration := time.Since(start).Seconds()
	httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "200").Inc()
	httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	
	fields["status"] = "healthy"
	fields["duration_seconds"] = duration
	structuredLog("INFO", "Health check completed", fields)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	fields := map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"method":      r.Method,
		"path":        r.URL.Path,
	}
	structuredLog("INFO", "Root endpoint accessed", fields)
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Go Service is running!"))
	
	duration := time.Since(start).Seconds()
	httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, "200").Inc()
	httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
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
		log.Printf("Failed to create OTLP trace exporter: %v", err)
		return
	}
	
	// Initialize OTLP metrics exporter
	metricExp, err := otlpmetricgrpc.New(
		context.Background(),
		otlpmetricgrpc.WithEndpoint(otlpEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		log.Printf("Failed to create OTLP metrics exporter: %v", err)
		return
	}
	
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		log.Printf("Failed to create resource: %v", err)
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
}

func main() {
	initTracing()
	
	r := mux.NewRouter()
	r.Use(otelmux.Middleware(serviceName))
	
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/", rootHandler).Methods("GET")
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	structuredLog("INFO", "Go service starting", map[string]interface{}{
		"port": 8080,
	})
	
	log.Fatal(http.ListenAndServe(":8080", r))
}
