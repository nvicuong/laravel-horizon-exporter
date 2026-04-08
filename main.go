package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/horizon-exporter/collector"
	"github.com/horizon-exporter/horizon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	var (
		listenAddr    = flag.String("web.listen-address", ":9888", "Address to listen on for metrics")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
		horizonURL    = flag.String("horizon.url", "http://localhost", "Base URL of the Laravel application")
		horizonToken  = flag.String("horizon.token", "", "Bearer token for Horizon API authentication")
		skipTLSVerify = flag.Bool("horizon.tls-skip-verify", false, "Skip TLS verification for Horizon API")
	)
	flag.Parse()

	if *horizonURL == "" {
		log.Fatal("--horizon.url is required")
	}

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

	log.Printf("Starting Horizon exporter on %s, scraping %s", *listenAddr, *horizonURL)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}
