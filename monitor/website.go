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
}
