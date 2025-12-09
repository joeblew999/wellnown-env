// Example: NATS Micro service behind Narun gateway
//
// This registers a tiny NATS Micro service ("narun.hello") and replies to
// requests forwarded by narun-gw. It expects a JSON body with an optional
// "name" field and returns a greeting.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/openpcc/bhttp"
)

type helloRequest struct {
	Name string `json:"name"`
}

type helloResponse struct {
	Message string `json:"message"`
	Server  string `json:"server"`
	Time    string `json:"time"`
}

func main() {
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	healthAddr := getEnv("NARUN_HEALTH_ADDR", ":8090")

	nc, err := nats.Connect(natsURL,
		nats.Name("narun-hello"),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		log.Fatalf("connect to NATS: %v", err)
	}
	defer nc.Drain()

	log.Printf("connected to NATS at %s", natsURL)

	_, err = micro.AddService(nc, micro.Config{
		Name:    "narun_hello",
		Version: "0.0.1",
		Endpoint: &micro.EndpointConfig{
			Subject: "narun_hello",
			Handler: micro.HandlerFunc(handleHello),
		},
	})
	if err != nil {
		log.Fatalf("register micro service: %v", err)
	}

	log.Printf("service 'narun.hello' ready")

	// Simple HTTP health endpoint for readiness probes
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})
		if err := http.ListenAndServe(healthAddr, mux); err != nil {
			log.Printf("health server error: %v", err)
		}
	}()

	// Block until signal
	waitForSignal()
	log.Println("shutting down")
}

func handleHello(req micro.Request) {
	// Decode BHTTP request from Narun gateway into an HTTP request
	decoder := &bhttp.RequestDecoder{}
	httpReq, err := decoder.DecodeRequest(context.Background(), bytes.NewReader(req.Data()))
	if err != nil {
		_ = req.Error("bad_request", fmt.Sprintf("decode BHTTP: %v", err), nil)
		return
	}

	var body helloRequest
	if httpReq.Body != nil {
		defer httpReq.Body.Close()
		if err := json.NewDecoder(httpReq.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			_ = req.Error("bad_request", fmt.Sprintf("invalid JSON: %v", err), nil)
			return
		}
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "NATS user"
	}

	resp := helloResponse{
		Message: fmt.Sprintf("Hello, %s!", name),
		Server:  "narun_hello",
		Time:    time.Now().Format(time.RFC3339),
	}

	// Encode HTTP response as BHTTP so narun-gw can decode it
	payload, err := json.Marshal(resp)
	if err != nil {
		_ = req.Error("internal_error", "failed to encode response", nil)
		return
	}

	httpResp := &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		ContentLength: int64(len(payload)),
		Body:          io.NopCloser(bytes.NewReader(payload)),
	}

	enc := &bhttp.ResponseEncoder{}
	msg, err := enc.EncodeResponse(httpResp)
	if err != nil {
		_ = req.Error("internal_error", fmt.Sprintf("encode BHTTP: %v", err), nil)
		return
	}

	out, err := io.ReadAll(msg)
	if err != nil {
		_ = req.Error("internal_error", "failed to serialize response", nil)
		return
	}

	if err := req.Respond(out); err != nil {
		log.Printf("respond error: %v", err)
	}
}

func waitForSignal() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
