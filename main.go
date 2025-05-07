package main

import (
        "log"
        "net/http"
        "time"

        "website-monitor/database"
        "website-monitor/handlers"
        "website-monitor/monitor"
        "github.com/gorilla/mux"
)

func main() {
        // Initialize the database
        db, err := database.New("websites.db")
        if err != nil {
                log.Fatalf("Failed to initialize database: %v", err)
        }
        defer db.Close()

        // Create a save function to pass to the monitor
        saveWebsite := func(website *monitor.Website) {
                if err := db.SaveWebsite(website); err != nil {
                        log.Printf("Error saving website to database: %v", err)
                }
        }

        // Initialize the website monitor with the save function
        websiteMonitor := monitor.NewMonitor(saveWebsite)

        // Load websites from the database
        if err := db.LoadWebsitesToMonitor(websiteMonitor); err != nil {
                log.Printf("Error loading websites from database: %v", err)
        }

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

        // Create delete function for handlers
        deleteWebsite := func(id int) error {
                return db.DeleteWebsite(id)
        }

        // Create handlers with the monitor and delete function
        h := handlers.NewHandlers(websiteMonitor, deleteWebsite)

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
