package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

)

const appVersion = "1.0.0"

var startTime = time.Now()

type HealthResponse struct {
	Service   string  `json:"service"`
	Status    string  `json:"status"`
	Version   string  `json:"version"`
	Uptime    string  `json:"uptime"`
	Message   string  `json:"message"`
	HTTPCode  int     `json:"http_code"`
	Latency   float64 `json:"simulated_latency_ms"`
	ErrorRate float64 `json:"simulated_error_rate"`
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func healthState() (string, string, int, float64, float64) {
	switch getEnv("HEALTH_STATUS", "healthy") {
	case "degraded":
		return "degraded", "High latency detected on upstream dependencies", http.StatusOK, 1847.3, 4.2
	case "down":
		return "down", "Service unavailable - database connection failed", http.StatusServiceUnavailable, 0, 100.0
	default:
		return "healthy", "All systems operational", http.StatusOK, 42.1, 0.1
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	status, message, code, latency, errorRate := healthState()

	resp := HealthResponse{
		Service:   getEnv("SERVICE_NAME", "demo-service"),
		Status:    status,
		Version:   getEnv("VERSION", "1.0.1"),
		Uptime:    time.Since(startTime).Round(time.Second).String(),
		Message:   message,
		HTTPCode:  code,
		Latency:   latency,
		ErrorRate: errorRate,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	status, _, _, latency, errorRate := healthState()
	service := getEnv("SERVICE_NAME", "demo-service")

	// Health gauge: 1 = healthy, 0.5 = degraded, 0 = down
	healthGauge := map[string]float64{
		"healthy":  1.0,
		"degraded": 0.5,
		"down":     0.0,
	}[status]

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "# HELP service_health Current health status (1=healthy, 0.5=degraded, 0=down)\n")
	fmt.Fprintf(w, "# TYPE service_health gauge\n")
	fmt.Fprintf(w, "service_health{service=\"%s\",status=\"%s\"} %g\n\n", service, status, healthGauge)

	fmt.Fprintf(w, "# HELP service_latency_ms Simulated request latency in milliseconds\n")
	fmt.Fprintf(w, "# TYPE service_latency_ms gauge\n")
	fmt.Fprintf(w, "service_latency_ms{service=\"%s\"} %g\n\n", service, latency)

	fmt.Fprintf(w, "# HELP service_error_rate Simulated error rate percentage\n")
	fmt.Fprintf(w, "# TYPE service_error_rate gauge\n")
	fmt.Fprintf(w, "service_error_rate{service=\"%s\"} %g\n\n", service, errorRate)

	fmt.Fprintf(w, "# HELP service_uptime_seconds Service uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE service_uptime_seconds counter\n")
	fmt.Fprintf(w, "service_uptime_seconds{service=\"%s\"} %g\n", service, time.Since(startTime).Seconds())
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	status, message, _, latency, errorRate := healthState()
	service := getEnv("SERVICE_NAME", "demo-service")
	version := getEnv("VERSION", "1.0.0")

	statusColor := map[string]string{
		"healthy":  "#00c853",
		"degraded": "#ff6d00",
		"down":     "#d50000",
	}[status]

	statusEmoji := map[string]string{
		"healthy":  "✅",
		"degraded": "⚠️",
		"down":     "🔴",
	}[status]

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>%s</title>
  <meta http-equiv="refresh" content="5">
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0f0f0f; color: #fff; display: flex; align-items: center; justify-content: center; min-height: 100vh; }
    .card { background: #1a1a1a; border: 1px solid #2a2a2a; border-radius: 16px; padding: 48px; max-width: 520px; width: 100%%; text-align: center; }
    .logo { font-size: 14px; color: #666; letter-spacing: 4px; text-transform: uppercase; margin-bottom: 32px; }
    .service { font-size: 28px; font-weight: 700; margin-bottom: 8px; }
    .version { font-size: 13px; color: #666; margin-bottom: 40px; }
    .status-badge { display: inline-flex; align-items: center; gap: 8px; background: %s22; border: 1px solid %s; color: %s; padding: 10px 24px; border-radius: 100px; font-size: 15px; font-weight: 600; margin-bottom: 32px; }
    .message { color: #999; font-size: 14px; margin-bottom: 40px; line-height: 1.6; }
    .metrics { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 40px; }
    .metric { background: #222; border-radius: 12px; padding: 20px; }
    .metric-value { font-size: 24px; font-weight: 700; color: %s; }
    .metric-label { font-size: 12px; color: #666; margin-top: 4px; text-transform: uppercase; letter-spacing: 1px; }
    .uptime { font-size: 13px; color: #444; }
    .endpoints { margin-top: 24px; display: flex; gap: 8px; justify-content: center; }
    .endpoint { background: #222; border-radius: 8px; padding: 6px 14px; font-size: 12px; color: #666; font-family: monospace; }
  </style>
</head>
<body>
  <div class="card">
    <div class="logo">Walmart Engineering</div>
    <div class="service">%s</div>
    <div class="version">v%s &nbsp;·&nbsp; deploy: %s</div>
    <div class="status-badge">%s %s</div>
    <div class="message">%s</div>
    <div class="metrics">
      <div class="metric">
        <div class="metric-value">%.0fms</div>
        <div class="metric-label">Latency (p99)</div>
      </div>
      <div class="metric">
        <div class="metric-value">%.1f%%</div>
        <div class="metric-label">Error Rate</div>
      </div>
    </div>
    <div class="uptime">Uptime: %s</div>
    <div class="endpoints">
      <div class="endpoint">GET /health</div>
      <div class="endpoint">GET /metrics</div>
    </div>
  </div>
</body>
</html>`,
		service,
		statusColor, statusColor, statusColor,
		statusColor,
		service, appVersion, version,
		statusEmoji, status,
		message,
		latency, errorRate,
		time.Since(startTime).Round(time.Second).String(),
	)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, html)
}

func main() {
	port := getEnv("PORT", "8080")
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/metrics", metricsHandler)

	service := getEnv("SERVICE_NAME", "demo-service")
	status, _, _, _, _ := healthState()
	log.Printf("🚀 %s starting on port %s (status: %s)", service, port, status)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
