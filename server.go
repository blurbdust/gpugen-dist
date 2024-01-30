package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type NumberPool struct {
	mu              sync.Mutex
	available       []int
	checkedOut      map[int]string
	checkoutHistory map[string]time.Time
	checkoutAt      map[time.Time]int
	fileName        string
}

var pool *NumberPool

func init() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Create a number pool with a maximum number of 4096
	pool = NewNumberPool(4096, "pool.json")

	// Load the pool from a file
	if err := pool.LoadFromFile(); err != nil {
		fmt.Println("Error loading pool:", err)
	}
}

func NewNumberPool(size int, fileName string) *NumberPool {
	pool := &NumberPool{
		available:       make([]int, size),
		checkedOut:      make(map[int]string),
		checkoutHistory: make(map[string]time.Time),
		checkoutAt:      make(map[time.Time]int),
		fileName:        fileName,
	}

	for i := 0; i < size; i++ {
		pool.available[i] = i
	}

	return pool
}

func (p *NumberPool) Checkout(addr string) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	lastCheckedOut, ok := p.checkoutHistory[addr]
	if ok && time.Since(lastCheckedOut) < 72*time.Hour {
		return 0, fmt.Errorf("Error! IP address %s has already checked out a number in the last 24 hours", addr)
	}

	// Check if there are any available numbers
	if len(p.available) == 0 {
		return 0, errors.New("Error! No available numbers")
	}

	// Choose a random number from the available pool
	idx := rand.Intn(len(p.available))
	num := p.available[idx]

	// Remove the chosen number from the available pool
	p.available = append(p.available[:idx], p.available[idx+1:]...)

	// Add the chosen number to the checked out pool and update the checkout time
	p.checkedOut[num] = addr
	p.checkoutHistory[addr] = time.Now()
	p.checkoutAt[p.checkoutHistory[addr]] = num

	return num, nil
}

func (p *NumberPool) ReleaseExpired() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for ip, checkoutTime := range p.checkoutHistory {
		if time.Since(checkoutTime) >= 72*time.Hour {
			num := p.checkoutAt[checkoutTime]
			tmp := make([]int, 1)
			tmp[0] = p.checkoutAt[checkoutTime]
			p.available = append(p.available, tmp...)
			delete(p.checkedOut, num)
			delete(p.checkoutHistory, ip)
			delete(p.checkoutAt, checkoutTime)
			fmt.Println("Cleared stale checkout for ", num)
		}
	}
}

func (p *NumberPool) CheckedOutBy(num int) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.checkedOut[num]
}

func (p *NumberPool) SaveToFile() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	file, err := os.Create(p.fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	if err := enc.Encode(p.checkedOut); err != nil {
		return err
	}

	return nil
}

func (p *NumberPool) LoadFromFile() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	file, err := os.Open(p.fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	if err := dec.Decode(&p.checkedOut); err != nil {
		return err
	}

	// Populate the available pool based on the numbers that are not checked out
	for i := 0; i < len(p.available); i++ {
		if _, ok := p.checkedOut[i]; !ok {
			p.available[i] = i
		}
	}

	return nil
}

func main() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Start the HTTP server
	go func() {
		if err := http.ListenAndServe(":65080", http.HandlerFunc(handler)); err != nil {
			fmt.Printf("HTTP server error: %s\n", err)
			os.Exit(1)
		}
	}()

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			fmt.Println("Interrupt triggered.")
			pool.ReleaseExpired()
		case <-sigChan:
			fmt.Printf("Received CTRL+C\n")
			// Write the pool to a file on exit
			if err := pool.SaveToFile(); err != nil {
				fmt.Println("Error saving pool:", err)
			}

			// Stop the HTTP server gracefully
			fmt.Println("Stopping server...")
			return
		default:
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		handleGet(w, r)
	} else if r.Method == "PUT" {
		pool.handlePut(w, r)
	} else if r.Method == "OPTIONS" {
		pool.handleOptions(w, r)
	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	// Check out a number from the pool using the remote IP address
	num, err := pool.Checkout(strings.Split(r.RemoteAddr, ":")[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert the number to a string
	numStr := strconv.Itoa(num)

	// Write the number to the response
	fmt.Fprintf(w, "%s\n", numStr)

	fmt.Printf("GET %s : %d\n", r.RemoteAddr, numStr)
}

func (p *NumberPool) handleOptions(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if r.Method == http.MethodOptions {
		num := p.checkoutAt[p.checkoutHistory[strings.Split(r.RemoteAddr, ":")[0]]]
		if num == 0 {
			num = -1
		}
		fmt.Fprintf(w, "%d\n", num)
		// Send a success response
		w.WriteHeader(http.StatusOK)
		fmt.Printf("OPTIONS %s : %d\n", r.RemoteAddr, num)
	}

}

func (p *NumberPool) handlePut(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()

	addr := strings.Split(r.RemoteAddr, ":")[0]
	num, _ := strconv.Atoi(r.RequestURI[1:])

	if r.Method == http.MethodPut {
		p.mu.Unlock()
		if pool.CheckedOutBy(num) != addr {
			http.Error(w, "Error! Forbidden", http.StatusForbidden)
			return
		}
		// Save the request body to a file
		fileName := r.RequestURI[1:] + ".txt"
		file, err := os.Create(fileName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		if _, err := io.Copy(file, r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Printf("PUT %s : %d\n", r.RemoteAddr, num)
		fmt.Fprintf(w, "Saved data to file %s\n", fileName)
		p.mu.Lock()
		delete(p.checkoutHistory, addr)
		p.mu.Unlock()
		return
	}

	// Send a success response
	w.WriteHeader(http.StatusOK)
}

