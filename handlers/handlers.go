package handlers

import (
        "encoding/json"
        "html/template"
        "net/http"
        "strconv"

        "website-monitor/monitor"
        "github.com/gorilla/mux"
)

// Handlers contains the HTTP handlers for the application
type Handlers struct {
        Monitor *monitor.Monitor
        tmpl    *template.Template
}

// NewHandlers creates a new Handlers instance
func NewHandlers(monitor *monitor.Monitor) *Handlers {
        tmpl := template.Must(template.ParseFiles("templates/index.html"))
        return &Handlers{
                Monitor: monitor,
                tmpl:    tmpl,
        }
}

// Dashboard renders the main dashboard
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
        h.tmpl.Execute(w, nil)
}

// GetWebsites returns all monitored websites as JSON
func (h *Handlers) GetWebsites(w http.ResponseWriter, r *http.Request) {
        websites := h.Monitor.GetWebsites()
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(websites)
}

// AddWebsite adds a new website to monitor
func (h *Handlers) AddWebsite(w http.ResponseWriter, r *http.Request) {
        var data struct {
                URL  string `json:"url"`
                Name string `json:"name"`
        }

        // Parse the request body
        err := json.NewDecoder(r.Body).Decode(&data)
        if err != nil {
                http.Error(w, "Invalid request format", http.StatusBadRequest)
                return
        }

        // Validate inputs
        if data.URL == "" {
                http.Error(w, "URL is required", http.StatusBadRequest)
                return
        }

        // Set default name if not provided
        if data.Name == "" {
                data.Name = data.URL
        }

        // Add the website to the monitor
        website := h.Monitor.AddWebsite(data.URL, data.Name)

        // Return the new website as JSON
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(website)
}

// RemoveWebsite removes a website from monitoring
func (h *Handlers) RemoveWebsite(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        id, err := strconv.Atoi(vars["id"])
        if err != nil {
                http.Error(w, "Invalid ID format", http.StatusBadRequest)
                return
        }

        // Try to remove the website
        success := h.Monitor.RemoveWebsite(id)
        if !success {
                http.Error(w, "Website not found", http.StatusNotFound)
                return
        }

        // Return success
        w.WriteHeader(http.StatusNoContent)
}

// CheckWebsite manually triggers a check for a specific website
func (h *Handlers) CheckWebsite(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        id, err := strconv.Atoi(vars["id"])
        if err != nil {
                http.Error(w, "Invalid ID format", http.StatusBadRequest)
                return
        }

        // Find the website
        website := h.Monitor.GetWebsiteByID(id)
        if website == nil {
                http.Error(w, "Website not found", http.StatusNotFound)
                return
        }

        // Check the website
        h.Monitor.CheckWebsite(website)

        // Return the updated website
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(website)
}
