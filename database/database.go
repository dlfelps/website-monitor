package database

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.etcd.io/bbolt"
	"website-monitor/monitor"
)

// DB represents the database
type DB struct {
	bolt *bbolt.DB
}

// WebsitesBucket is the name of the bucket where websites are stored
const WebsitesBucket = "websites"

// CounterBucket is the name of the bucket where counters are stored
const CounterBucket = "counters"

// IDCounterKey is the key for the website ID counter
const IDCounterKey = "website_id_counter"

// New creates a new database connection
func New(path string) (*DB, error) {
	// Open the database file
	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("could not open db: %v", err)
	}

	// Initialize buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		// Create websites bucket if it doesn't exist
		_, err := tx.CreateBucketIfNotExists([]byte(WebsitesBucket))
		if err != nil {
			return fmt.Errorf("could not create websites bucket: %v", err)
		}

		// Create counter bucket if it doesn't exist
		counterBucket, err := tx.CreateBucketIfNotExists([]byte(CounterBucket))
		if err != nil {
			return fmt.Errorf("could not create counter bucket: %v", err)
		}

		// Initialize ID counter if it doesn't exist
		if counterBucket.Get([]byte(IDCounterKey)) == nil {
			err = counterBucket.Put([]byte(IDCounterKey), []byte("1"))
			if err != nil {
				return fmt.Errorf("could not initialize ID counter: %v", err)
			}
		}

		return nil
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	return &DB{bolt: db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.bolt.Close()
}

// SaveWebsite saves a website to the database
func (db *DB) SaveWebsite(website *monitor.Website) error {
	return db.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(WebsitesBucket))

		// Convert website to JSON
		buf, err := json.Marshal(website)
		if err != nil {
			return fmt.Errorf("could not marshal website: %v", err)
		}

		// Save website with ID as key
		key := fmt.Sprintf("%d", website.ID)
		return b.Put([]byte(key), buf)
	})
}

// GetWebsites returns all websites from the database
func (db *DB) GetWebsites() ([]*monitor.Website, error) {
	var websites []*monitor.Website

	err := db.bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(WebsitesBucket))

		return b.ForEach(func(k, v []byte) error {
			var website monitor.Website
			if err := json.Unmarshal(v, &website); err != nil {
				return fmt.Errorf("could not unmarshal website: %v", err)
			}
			websites = append(websites, &website)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return websites, nil
}

// DeleteWebsite deletes a website from the database
func (db *DB) DeleteWebsite(id int) error {
	return db.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(WebsitesBucket))
		key := fmt.Sprintf("%d", id)
		return b.Delete([]byte(key))
	})
}

// GetNextID returns the next available website ID and increments the counter
func (db *DB) GetNextID() (int, error) {
	var id int

	err := db.bolt.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(CounterBucket))
		
		// Get current ID
		idBytes := b.Get([]byte(IDCounterKey))
		if idBytes == nil {
			return fmt.Errorf("ID counter not found")
		}
		
		// Parse ID
		_, err := fmt.Sscanf(string(idBytes), "%d", &id)
		if err != nil {
			return fmt.Errorf("could not parse ID counter: %v", err)
		}
		
		// Increment and save
		newID := id + 1
		err = b.Put([]byte(IDCounterKey), []byte(fmt.Sprintf("%d", newID)))
		if err != nil {
			return fmt.Errorf("could not update ID counter: %v", err)
		}
		
		return nil
	})

	if err != nil {
		return 0, err
	}

	return id, nil
}

// LoadWebsitesToMonitor loads all websites from the database into the monitor
func (db *DB) LoadWebsitesToMonitor(m *monitor.Monitor) error {
	websites, err := db.GetWebsites()
	if err != nil {
		return fmt.Errorf("could not get websites: %v", err)
	}

	log.Printf("Loading %d websites from database", len(websites))
	for _, website := range websites {
		m.AddExistingWebsite(website)
	}

	// Get the highest ID to set the counter
	highestID := 0
	for _, website := range websites {
		if website.ID > highestID {
			highestID = website.ID
		}
	}

	if highestID > 0 {
		m.SetIDCounter(highestID + 1)
	}

	return nil
}