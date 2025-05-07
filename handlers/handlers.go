package handlers

import (
        "encoding/json"
        "fmt"
        "html/template"
        "io"
        "log"
        "net/http"
        "os"
        "strconv"
        "time"

        "website-monitor/monitor"
        "github.com/gorilla/mux"
)

// Handlers contains the HTTP handlers for the application
type Handlers struct {
        Monitor       *monitor.Monitor
        tmpl          *template.Template
        deleteFromDB  func(int) error // Function to delete website from database
}

// NewHandlers creates a new Handlers instance
func NewHandlers(monitor *monitor.Monitor, deleteFunc func(int) error) *Handlers {
        tmpl := template.Must(template.ParseFiles("templates/index.html"))
        return &Handlers{
                Monitor:      monitor,
                tmpl:         tmpl,
                deleteFromDB: deleteFunc,
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
                URL             string `json:"url"`
                Name            string `json:"name"`
                UsePKI          bool   `json:"usePKI"`
                ClientCertPath  string `json:"clientCertPath"`
                ClientKeyPath   string `json:"clientKeyPath"`
                SkipTLSVerify   bool   `json:"skipTLSVerify"`
                CustomRootCAPath string `json:"customRootCAPath"`
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

        // Check for client certificate if PKI is enabled
        if data.UsePKI && data.ClientCertPath != "" && data.ClientKeyPath == "" {
                http.Error(w, "Client key path is required when client certificate is provided", http.StatusBadRequest)
                return
        }

        if data.UsePKI && data.ClientKeyPath != "" && data.ClientCertPath == "" {
                http.Error(w, "Client certificate path is required when client key is provided", http.StatusBadRequest)
                return
        }

        // Add the website to the monitor with PKI configuration if needed
        var website *monitor.Website
        if data.UsePKI {
                website = h.Monitor.AddWebsiteWithPKI(
                        data.URL, 
                        data.Name, 
                        data.UsePKI, 
                        data.ClientCertPath, 
                        data.ClientKeyPath, 
                        data.SkipTLSVerify, 
                        data.CustomRootCAPath,
                )
        } else {
                website = h.Monitor.AddWebsite(data.URL, data.Name)
        }

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

        // Try to remove the website from memory
        success := h.Monitor.RemoveWebsite(id)
        if !success {
                http.Error(w, "Website not found", http.StatusNotFound)
                return
        }

        // Delete from database if delete function is provided
        if h.deleteFromDB != nil {
                if err := h.deleteFromDB(id); err != nil {
                        // Log the error but don't fail the request since the website 
                        // is already removed from memory
                        log.Printf("Error deleting website %d from database: %v", id, err)
                }
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

// UploadCertificate handles certificate file uploads
func (h *Handlers) UploadCertificate(w http.ResponseWriter, r *http.Request) {
        // Limit file size to 5MB
        r.ParseMultipartForm(5 << 20)
        
        // Get the certificate type (client, key, or CA)
        certType := r.FormValue("type")
        if certType == "" {
                http.Error(w, "Certificate type is required", http.StatusBadRequest)
                return
        }
        
        // Get the file from form data
        file, header, err := r.FormFile("file")
        if err != nil {
                http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
                return
        }
        defer file.Close()
        
        // Read the file content
        fileBytes, err := io.ReadAll(file)
        if err != nil {
                http.Error(w, "Failed to read file: "+err.Error(), http.StatusInternalServerError)
                return
        }
        
        // Create a directory for certificates if it doesn't exist
        err = os.MkdirAll("./certs", 0755)
        if err != nil {
                http.Error(w, "Failed to create certs directory: "+err.Error(), http.StatusInternalServerError)
                return
        }
        
        // Create a unique filename based on current timestamp and original filename
        timestamp := time.Now().Unix()
        uniqueFilename := fmt.Sprintf("%d_%s", timestamp, header.Filename)
        filePath := fmt.Sprintf("./certs/%s", uniqueFilename)
        
        // Save the file
        err = os.WriteFile(filePath, fileBytes, 0644)
        if err != nil {
                http.Error(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
                return
        }
        
        log.Printf("Certificate file saved: %s", filePath)
        
        response := map[string]string{
                "filePath": filePath,
                "type":     certType,
        }
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
}
