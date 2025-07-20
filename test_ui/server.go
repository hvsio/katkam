package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Serve static files from the current directory (ui)
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)
	
	// Add CORS headers for WebRTC
	corsHandler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			h.ServeHTTP(w, r)
		})
	}
	
	// Wrap the file server with CORS
	http.Handle("/static/", corsHandler(http.StripPrefix("/static/", fs)))
	
	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "service": "katkam-ui"}`))
	})
	
	// API endpoint to get backend info
	http.HandleFunc("/api/backend", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"backend_url": "ws://localhost:8080/ws/sender",
			"api_url": "http://localhost:8080/api",
			"status_url": "http://localhost:8080/api/camera/status"
		}`))
	})
	
	port := ":8081"
	
	fmt.Printf("🌟 KatKam UI Server Starting\n")
	fmt.Printf("📱 UI available at: http://localhost%s\n", port)
	fmt.Printf("🔗 Backend connection: ws://localhost:8080/ws/sender\n")
	fmt.Printf("📊 Health check: http://localhost%s/health\n", port)
	fmt.Printf("📡 Backend info: http://localhost%s/api/backend\n", port)
	fmt.Println("────────────────────────────────────────")
	fmt.Println("Ready to receive camera stream! 📹")
	
	log.Fatal(http.ListenAndServe(port, corsHandler(http.DefaultServeMux)))
}
