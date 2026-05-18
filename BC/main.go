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
	mux.HandleFunc("/api/add-chapter", handlers.AddChapterHandler)
	mux.HandleFunc("/api/shift-chapter", handlers.ShiftChapterHandler)

	handler := middleware.EnableCORS(mux)
	log.Println("🚀 Server running at http://localhost:9000")
	log.Fatal(http.ListenAndServe(":9000", handler))
}
