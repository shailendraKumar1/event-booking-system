package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const baseURL = "http://localhost:8080"

type result struct {
	success  int64
	failed   int64
	dropped  int64
	totalMs  int64
	minMs    int64
	maxMs    int64
}

func main() {
	url := baseURL
	if v := os.Getenv("BASE_URL"); v != "" {
		url = v
	}

	fmt.Println("=== Event Booking System - Stress Test ===")
	fmt.Printf("Target: %s\n\n", url)

	// Phase 1: Setup - register users and create event
	fmt.Println("[Phase 1] Setting up test data...")
	orgToken := register(url, "stressorg@test.com", "password123", "Stress Organizer", "ORGANIZER")
	if orgToken == "" {
		orgToken = login(url, "stressorg@test.com", "password123")
	}
	custTokens := make([]string, 0)
	for i := 0; i < 50; i++ {
		email := fmt.Sprintf("stresscust%d@test.com", i)
		token := register(url, email, "password123", fmt.Sprintf("Customer %d", i), "CUSTOMER")
		if token == "" {
			token = login(url, email, "password123")
		}
		custTokens = append(custTokens, token)
	}
	eventID := createEvent(url, orgToken, "Stress Test Concert", 10000)
	fmt.Printf("  Created event: %s with 10000 capacity\n", eventID)
	fmt.Printf("  Registered %d customers\n\n", len(custTokens))

	// Phase 2: Concurrent booking stress test
	concurrencyLevels := []int{10, 50, 100, 200, 500, 1000}
	for _, c := range concurrencyLevels {
		evtID := createEvent(url, orgToken, fmt.Sprintf("Stress Event C%d", c), c*2)
		r := stressBooking(url, custTokens, evtID, c, 2)
		printResult(fmt.Sprintf("Booking (concurrency=%d)", c), r, c)
	}

	// Phase 3: Concurrent read stress test
	fmt.Println("\n[Phase 3] Read endpoint stress test...")
	for _, c := range []int{100, 500, 1000, 2000} {
		r := stressRead(url, eventID, c)
		printResult(fmt.Sprintf("GET /events/%s (concurrency=%d)", eventID, c), r, c)
	}

	// Phase 4: Mixed workload
	fmt.Println("\n[Phase 4] Mixed workload stress test...")
	mixEvtID := createEvent(url, orgToken, "Mixed Workload Event", 50000)
	for _, c := range []int{50, 100, 200, 500} {
		r := stressMixed(url, orgToken, custTokens, mixEvtID, c)
		printResult(fmt.Sprintf("Mixed (concurrency=%d, 70%% reads + 30%% bookings)", c), r, c)
	}

	fmt.Println("\n=== Stress Test Complete ===")
}

func stressBooking(url string, tokens []string, eventID string, concurrency, qty int) result {
	var res result
	res.minMs = 999999
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			token := tokens[idx%len(tokens)]
			t := time.Now()
			body := fmt.Sprintf(`{"quantity":%d}`, qty)
			req, _ := http.NewRequest("POST", url+"/api/events/"+eventID+"/bookings", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			elapsed := time.Since(t).Milliseconds()
			atomic.AddInt64(&res.totalMs, elapsed)
			updateMinMax(&res, elapsed)
			if err != nil {
				atomic.AddInt64(&res.dropped, 1)
				return
			}
			io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 201 {
				atomic.AddInt64(&res.success, 1)
			} else {
				atomic.AddInt64(&res.failed, 1)
			}
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("  Total wall time: %s\n", elapsed.Round(time.Millisecond))
	return res
}

func stressRead(url, eventID string, concurrency int) result {
	var res result
	res.minMs = 999999
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t := time.Now()
			resp, err := http.Get(url + "/api/events/" + eventID)
			elapsed := time.Since(t).Milliseconds()
			atomic.AddInt64(&res.totalMs, elapsed)
			updateMinMax(&res, elapsed)
			if err != nil {
				atomic.AddInt64(&res.dropped, 1)
				return
			}
			io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 {
				atomic.AddInt64(&res.success, 1)
			} else {
				atomic.AddInt64(&res.failed, 1)
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("  Total wall time: %s\n", elapsed.Round(time.Millisecond))
	return res
}

func stressMixed(url, orgToken string, custTokens []string, eventID string, concurrency int) result {
	var res result
	res.minMs = 999999
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			t := time.Now()
			var resp *http.Response
			var err error
			if idx%10 < 7 {
				resp, err = http.Get(url + "/api/events/" + eventID)
			} else {
				token := custTokens[idx%len(custTokens)]
				body := `{"quantity":1}`
				req, _ := http.NewRequest("POST", url+"/api/events/"+eventID+"/bookings", bytes.NewBufferString(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+token)
				resp, err = http.DefaultClient.Do(req)
			}
			elapsed := time.Since(t).Milliseconds()
			atomic.AddInt64(&res.totalMs, elapsed)
			updateMinMax(&res, elapsed)
			if err != nil {
				atomic.AddInt64(&res.dropped, 1)
				return
			}
			io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				atomic.AddInt64(&res.success, 1)
			} else {
				atomic.AddInt64(&res.failed, 1)
			}
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("  Total wall time: %s\n", elapsed.Round(time.Millisecond))
	return res
}

func printResult(label string, r result, total int) {
	avg := int64(0)
	count := r.success + r.failed + r.dropped
	if count > 0 {
		avg = r.totalMs / count
	}
	successRate := float64(r.success) / float64(total) * 100
	fmt.Printf("  %-55s | OK: %4d | Fail: %4d | Drop: %4d | Avg: %4dms | Min: %4dms | Max: %4dms | Rate: %.1f%%\n",
		label, r.success, r.failed, r.dropped, avg, r.minMs, r.maxMs, successRate)
}

func updateMinMax(r *result, ms int64) {
	for {
		old := atomic.LoadInt64(&r.minMs)
		if ms >= old || atomic.CompareAndSwapInt64(&r.minMs, old, ms) {
			break
		}
	}
	for {
		old := atomic.LoadInt64(&r.maxMs)
		if ms <= old || atomic.CompareAndSwapInt64(&r.maxMs, old, ms) {
			break
		}
	}
}

func register(url, email, password, name, role string) string {
	body := fmt.Sprintf(`{"email":"%s","password":"%s","name":"%s","role":"%s"}`, email, password, name, role)
	resp, err := http.Post(url+"/api/auth/register", "application/json", bytes.NewBufferString(body))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	if data, ok := res["data"].(map[string]interface{}); ok {
		if token, ok := data["token"].(string); ok {
			return token
		}
	}
	return ""
}

func login(url, email, password string) string {
	body := fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)
	resp, err := http.Post(url+"/api/auth/login", "application/json", bytes.NewBufferString(body))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	if data, ok := res["data"].(map[string]interface{}); ok {
		if token, ok := data["token"].(string); ok {
			return token
		}
	}
	return ""
}

func createEvent(url, token, title string, capacity int) string {
	body := fmt.Sprintf(`{"title":"%s","description":"Stress test event","venue":"Test Venue","starts_at":"2026-12-01T18:00:00Z","capacity":%d}`, title, capacity)
	req, _ := http.NewRequest("POST", url+"/api/events", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	if data, ok := res["data"].(map[string]interface{}); ok {
		if id, ok := data["id"].(string); ok {
			return id
		}
	}
	return ""
}
