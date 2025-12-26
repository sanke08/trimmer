package main

import (
	"log"
	"net/http"

	"github.com/sanke08/videoprocessor/handlers"
	"github.com/sanke08/videoprocessor/middleware"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/scan", handlers.ScanHandler)
	mux.HandleFunc("/api/process", handlers.ProcessHandler)
	mux.HandleFunc("/api/status", handlers.StatusHandler)

	handler := middleware.EnableCORS(mux)
	log.Println("🚀 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
