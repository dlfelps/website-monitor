package monitor

import (
        "crypto/md5"
        "crypto/tls"
        "crypto/x509"
        "encoding/hex"
        "fmt"
        "io"
        "log"
        "net/http"
        "os"
        "sync"
        "time"
)

// Monitor keeps track of websites and checks for changes
type Monitor struct {
        websites  []*Website
        mu        sync.RWMutex
        client    *http.Client
        idCounter int
        saveFunc  func(*Website) // Function to save website changes to database
}

// NewMonitor creates a new website monitor instance
func NewMonitor(saveFunction func(*Website)) *Monitor {
        return &Monitor{
                websites: []*Website{},
                client: &http.Client{
                        Timeout: 30 * time.Second,
                },
                idCounter: 1,
                saveFunc: saveFunction,
        }
}

// AddWebsite adds a new website to monitor
func (m *Monitor) AddWebsite(url, name string) *Website {
        return m.AddWebsiteWithPKI(url, name, false, "", "", false, "")
}

// AddWebsiteWithPKI adds a new website with PKI configuration to monitor
func (m *Monitor) AddWebsiteWithPKI(url, name string, usePKI bool, clientCertPath, clientKeyPath string, skipTLSVerify bool, customRootCAPath string) *Website {
        m.mu.Lock()
        defer m.mu.Unlock()

        website := &Website{
                ID:               m.idCounter,
                URL:              url,
                Name:             name,
                LastChecked:      time.Time{},
                LastHash:         "",
                HasChanged:       false,
                IsFirstCheck:     true,
                LastStatusCode:   0,
                Error:            "",
                UsePKI:           usePKI,
                ClientCertPath:   clientCertPath,
                ClientKeyPath:    clientKeyPath,
                SkipTLSVerify:    skipTLSVerify,
                CustomRootCAPath: customRootCAPath,
        }

        m.websites = append(m.websites, website)
        m.idCounter++

        // Save website to database if save function is provided
        if m.saveFunc != nil {
                m.saveFunc(website)
        }

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
                        
                        // This function doesn't use website.saveFunc because
                        // we can't access individual websites once they're deleted,
                        // so this is handled externally in the handlers package
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

        var resp *http.Response
        var err error

        if website.UsePKI {
                // Configure TLS for this specific website
                client, err := createClientWithPKI(website)
                if err != nil {
                        m.mu.Lock()
                        website.LastChecked = time.Now()
                        website.Error = "PKI configuration error: " + err.Error()
                        website.LastStatusCode = 0
                        website.HasChanged = false
                        if m.saveFunc != nil {
                                m.saveFunc(website)
                        }
                        m.mu.Unlock()
                        log.Printf("PKI configuration error for %s: %v", website.URL, err)
                        return
                }
                
                // Use the custom client to make the request
                resp, err = client.Get(website.URL)
        } else {
                // Use the default client for non-PKI websites
                resp, err = m.client.Get(website.URL)
        }
        
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
        
        // Save website to database if save function is provided
        if m.saveFunc != nil {
                m.saveFunc(website)
        }
        
        log.Printf("Check completed for %s - Changed: %v", website.URL, website.HasChanged)
}

// createClientWithPKI creates an HTTP client with PKI authentication for a specific website
func createClientWithPKI(website *Website) (*http.Client, error) {
        // Start with default TLS config
        tlsConfig := &tls.Config{
                MinVersion: tls.VersionTLS12,
        }
        
        // Load client certificate if specified
        if website.ClientCertPath != "" && website.ClientKeyPath != "" {
                cert, err := tls.LoadX509KeyPair(website.ClientCertPath, website.ClientKeyPath)
                if err != nil {
                        return nil, err
                }
                tlsConfig.Certificates = []tls.Certificate{cert}
        }
        
        // Configure custom root CA if specified
        if website.CustomRootCAPath != "" {
                caCert, err := os.ReadFile(website.CustomRootCAPath)
                if err != nil {
                        return nil, err
                }
                
                caCertPool := x509.NewCertPool()
                if !caCertPool.AppendCertsFromPEM(caCert) {
                        return nil, fmt.Errorf("failed to append CA certificate")
                }
                
                tlsConfig.RootCAs = caCertPool
        }
        
        // Configure insecure TLS verification if specified
        if website.SkipTLSVerify {
                tlsConfig.InsecureSkipVerify = true
        }
        
        // Create and return a custom HTTP client with the configured TLS
        return &http.Client{
                Timeout: 30 * time.Second,
                Transport: &http.Transport{
                        TLSClientConfig: tlsConfig,
                },
        }, nil
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

// AddExistingWebsite adds a website that was loaded from the database
func (m *Monitor) AddExistingWebsite(website *Website) {
        m.mu.Lock()
        defer m.mu.Unlock()

        m.websites = append(m.websites, website)
}

// SetIDCounter sets the ID counter for new websites
func (m *Monitor) SetIDCounter(id int) {
        m.mu.Lock()
        defer m.mu.Unlock()
        
        m.idCounter = id
}
