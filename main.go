package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/horizon-exporter/collector"
	"github.com/horizon-exporter/horizon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		listenIP      = flag.String("web.listen-ip", "", "IP address to listen on (empty = all interfaces)")
		listenPort    = flag.Int("web.listen-port", 9888, "Port to listen on for metrics")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
		horizonURL    = flag.String("horizon.url", "http://localhost", "Base URL of the Laravel application")
		horizonToken  = flag.String("horizon.token", "", "Bearer token for Horizon API authentication")
		skipTLSVerify = flag.Bool("horizon.tls-skip-verify", false, "Skip TLS verification for Horizon API")
	)
	flag.Parse()

	if *horizonURL == "" {
		log.Fatal("--horizon.url is required")
	}

	listenAddr := fmt.Sprintf("%s:%d", *listenIP, *listenPort)

	client := horizon.NewClient(*horizonURL, *horizonToken, *skipTLSVerify)
	col := collector.New(client)

	reg := prometheus.NewRegistry()
	reg.MustRegister(col)

	http.Handle(*metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head><title>Horizon Exporter</title></head><body>
<h1>Laravel Horizon Exporter</h1>
<p><a href="` + *metricsPath + `">Metrics</a></p>
</body></html>`))
	})
	http.HandleFunc("/-/healthy", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("Starting Horizon exporter on %s, scraping %s", listenAddr, *horizonURL)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}
