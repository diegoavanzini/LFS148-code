package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

var db UserInterface

const name = "github.com/lftraining/LFS148-code/exercises/manual-instrumentation-traces"

func getUser(w http.ResponseWriter, r *http.Request) {
	propagator := otel.GetTextMapPropagator()
	previous_ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	_, span := otel.GetTracerProvider().Tracer(name).Start(previous_ctx, "users")
	defer span.End()
	span.SetAttributes(
		attribute.String("http.request.method", r.Method),
		attribute.String("url.full", r.URL.Path),
	)
	user, status := db.GetUser(123)
	var data any = map[string]any{}
	if user != nil {
		data = user
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func doStuff(ctx context.Context) (*http.Response, error) {
	spanCtx, span := otel.GetTracerProvider().Tracer(name).Start(ctx, "doStuff")
	defer span.End()

	time.Sleep(100 * time.Millisecond)

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:6000/", nil)
	// Inject trace context into request headers
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(spanCtx, propagation.HeaderCarrier(req.Header))

	resp, err := http.DefaultClient.Do(req)

	// resp, err := http.Get("http://localhost:6000/")
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	// debug
	// jsonHeaders, err := json.MarshalIndent(resp.Request.Header, "", " ")
	log.Printf("Headers included in outbound request: %v", resp.Request.Header)
	// ...

	return resp, nil
}

func index(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.GetTracerProvider().Tracer(name).Start(r.Context(), "index")
	defer span.End()
	span.SetAttributes(
		attribute.String("http.request.method", r.Method),
		attribute.String("url.full", r.URL.Path),
	)
	propagator := otel.GetTextMapPropagator()
	ctx = propagator.Extract(ctx, propagation.HeaderCarrier(r.Header))

	if _, err := doStuff(ctx); err != nil {
		log.Printf("doStuff: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	currentTime := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")
	fmt.Fprintf(w, "Hello, World from GoLang! It's currently %s", currentTime)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(ctx)

	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	db = NewChaosClient(NewFakerClient())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /users", getUser)
	mux.HandleFunc("GET /{$}", index)

	srvErr := make(chan error, 1)
	go func() {
		log.Println("listening on 0.0.0.0:5000")
		srvErr <- http.ListenAndServe("0.0.0.0:5000", mux)
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting HTTP server.
		err = errors.Join(err, otelShutdown(context.Background()))
		return
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

}
