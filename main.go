package main

import (
        "log"
        "net/http"
        "time"

        "website-monitor/handlers"
        "website-monitor/monitor"
        "github.com/gorilla/mux"
)

func main() {
        // Initialize the website monitor
        websiteMonitor := monitor.NewMonitor()

        // Start the background monitoring process
        go func() {
                ticker := time.NewTicker(5 * time.Minute)
                defer ticker.Stop()

                log.Println("Starting website monitoring service...")
                for {
                        websiteMonitor.CheckAllWebsites()
                        <-ticker.C
                }
        }()

        // Set up the router
        r := mux.NewRouter()

        // Create handlers with the monitor
        h := handlers.NewHandlers(websiteMonitor)

        // API routes
        r.HandleFunc("/api/websites", h.GetWebsites).Methods("GET")
        r.HandleFunc("/api/websites", h.AddWebsite).Methods("POST")
        r.HandleFunc("/api/websites/{id}", h.RemoveWebsite).Methods("DELETE")
        r.HandleFunc("/api/websites/{id}/check", h.CheckWebsite).Methods("POST")

        // HTML routes
        r.HandleFunc("/", h.Dashboard).Methods("GET")

        // Serve static files
        r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

        // Start the server
        log.Println("Starting server on :5000...")
        log.Fatal(http.ListenAndServe("0.0.0.0:5000", r))
}
