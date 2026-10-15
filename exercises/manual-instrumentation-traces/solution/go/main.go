package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

var db UserInterface

func getUser(w http.ResponseWriter, r *http.Request) {
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
	time.Sleep(100 * time.Millisecond)
	resp, err := http.Get("http://localhost:6000/")
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return resp, nil
}

func index(w http.ResponseWriter, r *http.Request) {
	if _, err := doStuff(); err != nil {
		log.Printf("doStuff: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	currentTime := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05")
	fmt.Fprintf(w, "Hello, World! It's currently %s", currentTime)
}

func main() {
	db = NewChaosClient(NewFakerClient())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /users", getUser)
	mux.HandleFunc("GET /{$}", index)

	log.Println("listening on 0.0.0.0:5000")
	log.Fatal(http.ListenAndServe("0.0.0.0:5000", mux))
}
