package monitor

import (
        "time"
)

// Website represents a website being monitored
type Website struct {
        ID             int       `json:"id"`
        URL            string    `json:"url"`
        Name           string    `json:"name"`
        LastChecked    time.Time `json:"lastChecked"`
        LastHash       string    `json:"lastHash"`
        HasChanged     bool      `json:"hasChanged"`
        IsFirstCheck   bool      `json:"isFirstCheck"`
        LastStatusCode int       `json:"lastStatusCode"`
        Error          string    `json:"error"`
        
        // PKI authentication fields
        UsePKI            bool   `json:"usePKI"`            // Whether to use PKI authentication
        ClientCertPath    string `json:"clientCertPath"`    // Path to client certificate file
        ClientKeyPath     string `json:"clientKeyPath"`     // Path to client key file
        SkipTLSVerify     bool   `json:"skipTLSVerify"`     // Whether to skip TLS verification (insecure)
        CustomRootCAPath  string `json:"customRootCAPath"`  // Path to custom root CA certificate
}
