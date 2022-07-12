package main

import (
	"flag"
	"github.com/adrianliechti/go-tradfri"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "tradfri"

var (
	listenAddr   = flag.String("listen-addr", ":9368", "The address to listen on for HTTP requests.")
	gatewayAddr  = flag.String("gateway-addr", "127.0.0.1:5684", "The address of the Tradfri Gateway.")
	clientID     = flag.String("client-id", "tradfri_exporter", "The clientID to use when communicating with the gateway.")
	securityCode = flag.String("security-code", "", "The gateway security code (printed on the bottom of the gateway).")
	storagePath  = flag.String("storage-path", os.TempDir(), "Where to store generated keys.")
)

func main() {
	flag.Parse()

	psk, err := psk(*gatewayAddr, *clientID, *securityCode)
	if err != nil {
		log.Fatalf("failed to generate pre shared key: %v", err)
	}

	client, err := tradfri.New(*gatewayAddr, *clientID, psk)
	if err != nil {
		log.Fatalf("failed to start client: %v", err)
	}
	defer client.Close()

	logger, _ := zap.NewProduction()
	logger.Info("Starting nordpool_exporter")

	prometheus.MustRegister(NewTradfriCollector(namespace, logger, client))

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head><title>IKEA Tradfri Exporter</title></head>
            <body>
            <h1>IKEA Tradfri Exporter</h1>
            <p><a href="/metrics">Metrics</a></p>
            </body>
            </html>`))
	})
	srv := &http.Server{
		Addr:         *listenAddr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Info("Listening on", zap.Stringp("addr", listenAddr))
	logger.Fatal("failed to start server", zap.Error(srv.ListenAndServe()))
}

func psk(address, clientID, securityCode string) ([]byte, error) {
	filename := path.Join(*storagePath, ".tradfri_"+clientID)

	if data, err := ioutil.ReadFile(filename); err == nil {
		return data, nil
	}

	psk, err := tradfri.PSK(address, clientID, securityCode)
	if err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(filename, psk, 0644); err != nil {
		return nil, err
	}

	return psk, nil
}
