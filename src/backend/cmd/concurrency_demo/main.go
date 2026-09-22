package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"unihub-workshop/internal/middleware"
	"unihub-workshop/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	WorkshopID string
	TotalUsers int
	TotalSlots int
	Workers    int
	Mode       string // "gate" (true 12k simultaneous barrier) or "pool" (worker pool)
	Duration   time.Duration
	Cycles     int
	Loop       bool
	APIBaseURL string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPass     string
	DBName     string
	RedisAddr  string
	RabbitURL  string
	AuthSecret string
}

type RequestResult struct {
	StatusCode int
	Latency    time.Duration
	StatusMsg  string
	Err        error
}

type UserCredential struct {
	ID        string
	StudentID string
	Token     string
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.WorkshopID, "workshop", "11111111-1111-1111-1111-111111111111", "Target workshop UUID")
	flag.IntVar(&cfg.TotalUsers, "users", 12000, "Number of concurrent users to simulate")
	flag.IntVar(&cfg.TotalSlots, "slots", 1000, "Number of workshop slots available")
	flag.IntVar(&cfg.Workers, "workers", 500, "Number of concurrent HTTP worker routines (for pool mode)")
	flag.StringVar(&cfg.Mode, "mode", "gate", "Burst mode: 'gate' (true simultaneous 12k barrier with starting gun) or 'pool' (worker pool)")
	flag.DurationVar(&cfg.Duration, "duration", 0*time.Second, "Target duration for requests in pool mode (default 0 for unthrottled)")
	flag.IntVar(&cfg.Cycles, "cycles", 1, "Number of test cycles to execute")
	flag.BoolVar(&cfg.Loop, "loop", false, "Run continuously in a loop with reset")
	flag.StringVar(&cfg.APIBaseURL, "api", "http://localhost:8080", "UniHub API base URL")
	flag.StringVar(&cfg.DBHost, "dbhost", "localhost", "PostgreSQL host")
	flag.StringVar(&cfg.DBPort, "dbport", "5433", "PostgreSQL port")
	flag.StringVar(&cfg.DBUser, "dbuser", "unihub", "PostgreSQL user")
	flag.StringVar(&cfg.DBPass, "dbpass", "unihub_secret", "PostgreSQL password")
	flag.StringVar(&cfg.DBName, "dbname", "unihub_workshop", "PostgreSQL database name")
	flag.StringVar(&cfg.RedisAddr, "redis", "localhost:6379", "Redis address")
	flag.StringVar(&cfg.RabbitURL, "rabbit", "amqp://guest:guest@localhost:5672/", "RabbitMQ URL")
	flag.StringVar(&cfg.AuthSecret, "secret", "default-secret", "JWT Auth secret")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Println("================================================================================")
	fmt.Println("   UNIHUB WORKSHOP — HIGH CONCURRENCY LOAD TEST & IDEMPOTENCY DEMO              ")
	fmt.Println("================================================================================")
	fmt.Printf(" Target Workshop : %s (%d slots)\n", cfg.WorkshopID, cfg.TotalSlots)
	fmt.Printf(" Backend Target  : %s (Running on FULL %d CPU Cores)\n", cfg.APIBaseURL, runtime.NumCPU())
	fmt.Println("================================================================================")

	// 1. Connect to PostgreSQL
	pgDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
	pool, err := pgxpool.New(ctx, pgDSN)
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	// 2. Connect to Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[FATAL] Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	// 3. Connect to RabbitMQ
	rabbitConn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("[FATAL] Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	// 4. Pre-fetch distinct student users
	fmt.Printf("[INIT] Fetching %d distinct student users from database...\n", cfg.TotalUsers)
	users, err := fetchStudentUsers(ctx, pool, cfg.TotalUsers)
	if err != nil {
		log.Fatalf("[FATAL] Could not fetch student users: %v", err)
	}
	if len(users) < cfg.TotalUsers {
		log.Fatalf("[FATAL] Not enough student users in DB! Need %d, found %d", cfg.TotalUsers, len(users))
	}
	fmt.Printf("       Loaded %d users successfully.\n", len(users))

	// 5. Pre-generate JWT tokens for all users to minimize test-time overhead
	fmt.Printf("[INIT] Pre-generating %d JWT bearer tokens...\n", len(users))
	for i := range users {
		token, err := middleware.GenerateJWT(cfg.AuthSecret, users[i].ID, model.RoleStudent)
		if err != nil {
			log.Fatalf("[FATAL] Failed to generate JWT for user %s: %v", users[i].ID, err)
		}
		users[i].Token = token
	}
	fmt.Printf("       All %d tokens generated in memory.\n", len(users))

	// Pre-create HTTP client with high connection pool
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        20000,
			MaxIdleConnsPerHost: 20000,
			MaxConnsPerHost:     20000,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
	}

	cycle := 1
	for {
		if ctx.Err() != nil {
			fmt.Println("\n[INFO] Stopped by user signal.")
			break
		}

		fmt.Printf("\n================================================================================\n")
		fmt.Printf(" >>> CYCLE #%d STARTING <<<\n", cycle)
		fmt.Printf("================================================================================\n")

		// Step A: Reset State
		fmt.Println("[RESET] Cleaning up database, Redis cache, and RabbitMQ queues...")
		if err := resetState(ctx, pool, rdb, rabbitConn, cfg); err != nil {
			log.Fatalf("[FATAL] Reset failed: %v", err)
		}
		fmt.Printf("        State cleanly reset: Available seats = %d, Registrations = 0, Redis cache cleared.\n", cfg.TotalSlots)

		// Step B: Run the 12000-user rush
		fmt.Printf("\n[BURST] Blasting %d registrations (%d concurrent workers)...\n", cfg.TotalUsers, cfg.Workers)
		burstStart := time.Now()
		results := runBurst(ctx, httpClient, cfg, users)
		burstDuration := time.Since(burstStart)

		// Step C & D: Wait for RabbitMQ workers to finalize registrations in PostgreSQL
		fmt.Println("\n[DRAIN] Waiting for background workers on 2 CPUs to finalize registrations in PostgreSQL...")
		var stats DBStats
		deadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(deadline) {
			var err error
			stats, err = verifyDBState(ctx, pool, cfg)
			if err == nil && (stats.TotalSuccess >= stats.Capacity || stats.AvailableSeats == 0) {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		// Step E: Print Cycle Report
		printCycleReport(cycle, cfg, burstDuration, results, stats)

		cycle++
		if !cfg.Loop && cycle > cfg.Cycles {
			break
		}

		fmt.Println("\n[PAUSE] Next cycle in 2 seconds (Press Ctrl+C to stop)...")
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}

	fmt.Println("\n================================================================================")
	fmt.Println("   DEMO COMPLETED SUCCESSFULLY!                                                 ")
	fmt.Println("================================================================================")
}

func fetchStudentUsers(ctx context.Context, pool *pgxpool.Pool, count int) ([]UserCredential, error) {
	rows, err := pool.Query(ctx, "SELECT id, user_id FROM users WHERE role = 'STUDENT' ORDER BY id LIMIT $1", count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserCredential
	for rows.Next() {
		var u UserCredential
		if err := rows.Scan(&u.ID, &u.StudentID); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func resetState(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client, rabbitConn *amqp.Connection, cfg Config) error {
	// 1. PostgreSQL Clean
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "DELETE FROM notifications WHERE registration_id IN (SELECT id FROM registrations WHERE workshop_id = $1)", cfg.WorkshopID); err != nil {
		return fmt.Errorf("postgres delete notifications: %w", err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM registrations WHERE workshop_id = $1", cfg.WorkshopID); err != nil {
		return fmt.Errorf("postgres delete registrations: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE workshops SET 
			available_seats = $2, 
			capacity = $2, 
			status = 'PUBLISHED',
			registration_start_time = NOW() - INTERVAL '1 hour',
			registration_end_time = NOW() + INTERVAL '1 day'
		WHERE id = $1
	`, cfg.WorkshopID, cfg.TotalSlots); err != nil {
		return fmt.Errorf("postgres update workshop: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres commit reset tx: %w", err)
	}

	// 2. Redis Clean
	keys := []string{
		fmt.Sprintf("waitingroom:%s", cfg.WorkshopID),
		fmt.Sprintf("waitingroom:active:%s", cfg.WorkshopID),
		fmt.Sprintf("waitingroom:heartbeat:%s", cfg.WorkshopID),
		fmt.Sprintf("workshop:seats:%s", cfg.WorkshopID),
		fmt.Sprintf("workshop:%s", cfg.WorkshopID),
		fmt.Sprintf("workshop:meta:%s", cfg.WorkshopID),
	}
	if err := rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}

	// 3. RabbitMQ Purge
	ch, err := rabbitConn.Channel()
	if err == nil {
		_, _ = ch.QueuePurge("registration_queue", false)
		_, _ = ch.QueuePurge("notification_queue", false)
		ch.Close()
	}

	return nil
}

func runBurst(ctx context.Context, client *http.Client, cfg Config, users []UserCredential) []RequestResult {
	results := make([]RequestResult, len(users))
	url := fmt.Sprintf("%s/api/v1/registrations", cfg.APIBaseURL)
	reqBodyBytes, _ := json.Marshal(map[string]string{
		"workshop_id": cfg.WorkshopID,
	})

	var completed int64

	// Progress reporter goroutine
	stopProgress := make(chan struct{})
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopProgress:
				return
			case <-ticker.C:
				done := atomic.LoadInt64(&completed)
				fmt.Printf("\r       -> Progress: %d / %d requests sent (%.1f%%)...", done, len(users), float64(done)*100/float64(len(users)))
			}
		}
	}()

	if cfg.Mode == "gate" {
		// =========================================================================
		// MODE: STARTING GATE BARRIER (TRUE 12,000 SIMULTANEOUS GOROUTINES IN-FLIGHT)
		// =========================================================================
		var wg sync.WaitGroup
		var readyWg sync.WaitGroup
		startGun := make(chan struct{})

		fmt.Printf("       [GATE] Spawning and aligning all %d student goroutines at the starting line...\n", len(users))
		readyWg.Add(len(users))

		for i := 0; i < len(users); i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// Pre-allocate HTTP request before the gun so no CPU cycles are wasted
				req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBodyBytes))
				if err != nil {
					results[idx] = RequestResult{Err: err}
					readyWg.Done()
					atomic.AddInt64(&completed, 1)
					return
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+users[idx].Token)

				// Signal that this student is standing at the gate ready
				readyWg.Done()

				// WAIT for the starting gun! All 12,000 students block here!
				<-startGun

				start := time.Now()
				resp, err := client.Do(req)
				latency := time.Since(start)

				if err != nil {
					results[idx] = RequestResult{Err: err, Latency: latency}
				} else {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()

					var apiResp model.APIResponse
					_ = json.Unmarshal(body, &apiResp)

					results[idx] = RequestResult{
						StatusCode: resp.StatusCode,
						Latency:    latency,
						StatusMsg:  apiResp.Message,
					}
				}
				atomic.AddInt64(&completed, 1)
			}(i)
		}

		// Wait until ALL 12,000 student goroutines are alive, initialized, and at the starting gate
		readyWg.Wait()
		fmt.Printf("       [GATE] 🎯 ALL %d STUDENTS STANDING AT THE GATE! FIRING STARTING GUN NOW! 🚀\n", len(users))

		// FIRE THE STARTING GUN! All 12,000 requests release at the EXACT same microsecond!
		close(startGun)

		wg.Wait()
		close(stopProgress)
		fmt.Printf("\r       -> Progress: %d / %d requests sent (100.0%%)   \n", len(users), len(users))
		return results
	}

	// =========================================================================
	// MODE: WORKER POOL (Workers pulling tasks with optional pacing)
	// =========================================================================
	var wg sync.WaitGroup
	workerCount := cfg.Workers
	if workerCount <= 0 {
		workerCount = 500
	}
	taskCh := make(chan int, len(users))
	for i := range users {
		taskCh <- i
	}
	close(taskCh)

	// Pacing: distribute task dispatch evenly across cfg.Duration if duration > 0
	var pacingCh <-chan time.Time
	if cfg.Duration > 0 {
		interval := cfg.Duration / time.Duration(len(users))
		if interval > 0 {
			pacingTicker := time.NewTicker(interval)
			defer pacingTicker.Stop()
			pacingCh = pacingTicker.C
		}
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range taskCh {
				if pacingCh != nil {
					<-pacingCh
				}

				req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBodyBytes))
				if err != nil {
					results[idx] = RequestResult{Err: err}
					atomic.AddInt64(&completed, 1)
					continue
				}

				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+users[idx].Token)

				start := time.Now()
				resp, err := client.Do(req)
				latency := time.Since(start)

				if err != nil {
					results[idx] = RequestResult{Err: err, Latency: latency}
				} else {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()

					var apiResp model.APIResponse
					_ = json.Unmarshal(body, &apiResp)

					results[idx] = RequestResult{
						StatusCode: resp.StatusCode,
						Latency:    latency,
						StatusMsg:  apiResp.Message,
					}
				}
				atomic.AddInt64(&completed, 1)
			}
		}()
	}

	wg.Wait()
	close(stopProgress)
	fmt.Printf("\r       -> Progress: %d / %d requests sent (100.0%%)   \n", len(users), len(users))

	return results
}

type DBStats struct {
	TotalSuccess   int
	AvailableSeats int
	Capacity       int
}

func verifyDBState(ctx context.Context, pool *pgxpool.Pool, cfg Config) (DBStats, error) {
	var stats DBStats
	err := pool.QueryRow(ctx, `
		SELECT 
			COALESCE(SUM(CASE WHEN status = 'SUCCESS' THEN 1 ELSE 0 END), 0) AS total_success
		FROM registrations 
		WHERE workshop_id = $1;
	`, cfg.WorkshopID).Scan(&stats.TotalSuccess)
	if err != nil {
		return stats, err
	}

	err = pool.QueryRow(ctx, `
		SELECT capacity, available_seats FROM workshops WHERE id = $1;
	`, cfg.WorkshopID).Scan(&stats.Capacity, &stats.AvailableSeats)
	return stats, err
}

func printCycleReport(cycle int, cfg Config, duration time.Duration, results []RequestResult, stats DBStats) {
	statusCounts := make(map[int]int)
	var latencies []time.Duration
	var errCount int

	for _, r := range results {
		if r.Err != nil {
			errCount++
			continue
		}
		statusCounts[r.StatusCode]++
		latencies = append(latencies, r.Latency)
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var minLat, maxLat, avgLat, p50, p95, p99 time.Duration
	if len(latencies) > 0 {
		minLat = latencies[0]
		maxLat = latencies[len(latencies)-1]
		var sum time.Duration
		for _, l := range latencies {
			sum += l
		}
		avgLat = sum / time.Duration(len(latencies))
		p50 = latencies[len(latencies)*50/100]
		p95 = latencies[len(latencies)*95/100]
		p99 = latencies[len(latencies)*99/100]
	}

	rps := float64(len(results)) / duration.Seconds()

	fmt.Println("\n--------------------------------------------------------------------------------")
	fmt.Printf("               CYCLE #%d BENCHMARK RESULTS (FULL %d CPU CORES)                 \n", cycle, runtime.NumCPU())
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf(" [Traffic Profile]      : %d total users in %.3f seconds (%.1f req/sec)\n", len(results), duration.Seconds(), rps)
	// Sort and print all status codes received
	var codes []int
	for c := range statusCounts {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	for _, code := range codes {
		name := http.StatusText(code)
		if code == http.StatusAccepted {
			name = "Accepted (Admitted & Enqueued to RabbitMQ)"
		} else if code == http.StatusTooManyRequests {
			name = "Too Many Requests (Held in Virtual Waiting Queue)"
		} else if code == http.StatusConflict {
			name = "Conflict (Redis Seat Shield: Cache Full)"
		}
		fmt.Printf(" [HTTP %d] %-42s: %d requests\n", code, name, statusCounts[code])
	}
	if errCount > 0 {
		fmt.Printf(" [Connection Errors]                         : %d\n", errCount)
	}

	fmt.Println("\n [Latency Metrics]")
	fmt.Printf("   Min Latency : %8.2f ms\n", float64(minLat.Microseconds())/1000.0)
	fmt.Printf("   Avg Latency : %8.2f ms\n", float64(avgLat.Microseconds())/1000.0)
	fmt.Printf("   p50 (Median): %8.2f ms\n", float64(p50.Microseconds())/1000.0)
	fmt.Printf("   p95         : %8.2f ms\n", float64(p95.Microseconds())/1000.0)
	fmt.Printf("   p99         : %8.2f ms\n", float64(p99.Microseconds())/1000.0)
	fmt.Printf("   Max Latency : %8.2f ms\n", float64(maxLat.Microseconds())/1000.0)

	fmt.Println("\n [Database Verification]")
	fmt.Printf("   Workshop Capacity     : %d\n", stats.Capacity)
	fmt.Printf("   Available Seats in DB : %d\n", stats.AvailableSeats)
	fmt.Printf("   SUCCESS Registrations : %d\n", stats.TotalSuccess)

	overbooking := stats.TotalSuccess - stats.Capacity
	if overbooking > 0 {
		fmt.Printf("   VERDICT               : ❌ OVERBOOKING DETECTED! (%d extra seats sold)\n", overbooking)
	} else if stats.TotalSuccess == stats.Capacity && stats.AvailableSeats == 0 {
		fmt.Printf("   VERDICT               : ✅ 100%%%% PERFECT CONSISTENCY! Exactly %d seats, 0 Overbooking!\n", stats.Capacity)
	} else {
		fmt.Printf("   VERDICT               : ⚠️ Partial (%d / %d registered)\n", stats.TotalSuccess, stats.Capacity)
	}
	fmt.Println("--------------------------------------------------------------------------------")
}
