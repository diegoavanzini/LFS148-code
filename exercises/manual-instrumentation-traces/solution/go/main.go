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
)

var db UserInterface

const name = "github.com/lftraining/LFS148-code/exercises/manual-instrumentation-traces"

func getUser(w http.ResponseWriter, r *http.Request) {
	_, span := otel.GetTracerProvider().Tracer(name).Start(r.Context(), "getUser")
	defer span.End()
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

func doStuff() (*http.Response, error) {
	_, span := otel.GetTracerProvider().Tracer(name).Start(context.Background(), "doStuff")
	defer span.End()
	time.Sleep(100 * time.Millisecond)
	resp, err := http.Get("http://localhost:6000/")
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return resp, nil
}

func index(w http.ResponseWriter, r *http.Request) {
	_, span := otel.GetTracerProvider().Tracer(name).Start(context.Background(), "index")
	defer span.End()
	if _, err := doStuff(); err != nil {
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
