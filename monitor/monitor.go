package monitor

import (
        "crypto/md5"
        "encoding/hex"
        "io"
        "log"
        "net/http"
        "sync"
        "time"
)

// Monitor keeps track of websites and checks for changes
type Monitor struct {
        websites  []*Website
        mu        sync.RWMutex
        client    *http.Client
        idCounter int
}

// NewMonitor creates a new website monitor instance
func NewMonitor() *Monitor {
        return &Monitor{
                websites: []*Website{},
                client: &http.Client{
                        Timeout: 30 * time.Second,
                },
                idCounter: 1,
        }
}

// AddWebsite adds a new website to monitor
func (m *Monitor) AddWebsite(url, name string) *Website {
        m.mu.Lock()
        defer m.mu.Unlock()

        website := &Website{
                ID:             m.idCounter,
                URL:            url,
                Name:           name,
                LastChecked:    time.Time{},
                LastHash:       "",
                HasChanged:     false,
                IsFirstCheck:   true,
                LastStatusCode: 0,
                Error:          "",
        }

        m.websites = append(m.websites, website)
        m.idCounter++

        // Immediately check the website
        go m.CheckWebsite(website)

        return website
}

// RemoveWebsite removes a website from monitoring
func (m *Monitor) RemoveWebsite(id int) bool {
        m.mu.Lock()
        defer m.mu.Unlock()

        for i, website := range m.websites {
                if website.ID == id {
                        // Remove the website from the slice
                        m.websites = append(m.websites[:i], m.websites[i+1:]...)
                        return true
                }
        }
        return false
}

// GetWebsites returns all monitored websites
func (m *Monitor) GetWebsites() []*Website {
        m.mu.RLock()
        defer m.mu.RUnlock()

        // Create a copy to avoid race conditions
        websites := make([]*Website, len(m.websites))
        copy(websites, m.websites)

        return websites
}

// GetWebsiteByID returns a website by its ID
func (m *Monitor) GetWebsiteByID(id int) *Website {
        m.mu.RLock()
        defer m.mu.RUnlock()

        for _, website := range m.websites {
                if website.ID == id {
                        return website
                }
        }
        return nil
}

// CheckWebsite performs a check on a single website
func (m *Monitor) CheckWebsite(website *Website) {
        log.Printf("Checking website: %s (%s)", website.Name, website.URL)

        resp, err := m.client.Get(website.URL)
        
        m.mu.Lock()
        defer m.mu.Unlock()

        website.LastChecked = time.Now()
        
        if err != nil {
                website.Error = err.Error()
                website.LastStatusCode = 0
                website.HasChanged = false
                log.Printf("Error checking %s: %v", website.URL, err)
                return
        }
        defer resp.Body.Close()

        website.LastStatusCode = resp.StatusCode
        
        if resp.StatusCode != http.StatusOK {
                website.Error = "Received status: " + resp.Status
                website.HasChanged = false
                log.Printf("Error status for %s: %s", website.URL, resp.Status)
                return
        }

        // Read the body content
        body, err := io.ReadAll(resp.Body)
        if err != nil {
                website.Error = "Failed to read response: " + err.Error()
                website.HasChanged = false
                log.Printf("Error reading body from %s: %v", website.URL, err)
                return
        }

        // Calculate MD5 hash of the content
        hash := md5.Sum(body)
        currentHash := hex.EncodeToString(hash[:])

        // Check if content has changed
        if website.IsFirstCheck {
                website.IsFirstCheck = false
                website.HasChanged = false
        } else if website.LastHash != currentHash {
                website.HasChanged = true
        } else {
                website.HasChanged = false
        }

        website.LastHash = currentHash
        website.Error = ""
        
        log.Printf("Check completed for %s - Changed: %v", website.URL, website.HasChanged)
}

// CheckAllWebsites checks all monitored websites for changes
func (m *Monitor) CheckAllWebsites() {
        websites := m.GetWebsites() // Get a copy to avoid holding the lock

        var wg sync.WaitGroup
        for _, website := range websites {
                wg.Add(1)
                go func(w *Website) {
                        defer wg.Done()
                        m.CheckWebsite(w)
                }(website)
        }

        wg.Wait()
        log.Printf("Completed checking all %d websites", len(websites))
}
