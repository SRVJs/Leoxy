package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elrefai99/Leoxy/tools/fs"
)

var (
	attempts atomic.Uint64
	success  atomic.Uint64
	failures atomic.Uint64
	received atomic.Uint64
)

func main() {
	target := flag.String("url", "", "Leoxy URL")
	duration := flag.Duration("duration", 0, "Test duration")
	requestLimit := flag.Int("request", 0, "Requests count")
	concurrency := flag.Int("concurrency", 0, "Concurrent Worker")
	pprofURL := flag.String("pprof", "", "Optional Leoxy heap profile URL")
	profileOut := flag.String("profile-out", "docs/leoxy-heap.txt", "Heap profile output file")

	flag.Parse()

	if *target == "" || *concurrency < 1 || *duration <= 0 {
		fmt.Fprintln(os.Stderr, "url, concurrency and duration are required")
		os.Exit(2)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency * 2,
			MaxIdleConnsPerHost: *concurrency,
			MaxConnsPerHost:     *concurrency,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	defer client.CloseIdleConnections()

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	var mutex sync.Mutex
	var wg sync.WaitGroup
	latencies := make([]time.Duration, 0, 100000)

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	start := time.Now()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				slot := attempts.Add(1)
				if *requestLimit > 0 && slot > uint64(*requestLimit) {
					attempts.Add(^uint64(0))
					return
				}

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, *target, nil)
				if err != nil {
					failures.Add(1)
					return
				}

				reqStart := time.Now()
				resp, err := client.Do(req)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					failures.Add(1)
					mutex.Lock()
					latencies = append(latencies, time.Since(reqStart))
					mutex.Unlock()
					continue
				}

				n, readErr := io.Copy(io.Discard, resp.Body)
				closeErr := resp.Body.Close()
				elapsed := time.Since(reqStart)

				if readErr != nil || closeErr != nil || resp.StatusCode >= 500 {
					failures.Add(1)
				} else {
					success.Add(1)
					received.Add(uint64(n))
				}

				mutex.Lock()
				latencies = append(latencies, elapsed)
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&memAfter)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	total := success.Load()
	failed := failures.Load()

	fmt.Println("\n========== Leoxy Network ==========")
	fmt.Printf("Target: %s\n", *target)
	fmt.Printf("Duration: %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Concurrency: %v\n", *concurrency)
	fmt.Printf("Total attempts: %v\n", attempts.Load())
	fmt.Printf("Success Requests: %v\n", total)
	fmt.Printf("Failed Requests: %v\n", failed)
	fmt.Printf("Throughput: %.2f req/s\n", float64(total)/elapsed.Seconds())
	fmt.Printf("Response Throughput: %.2f MB/s\n", float64(received.Load())/elapsed.Seconds()/1e6)

	if len(latencies) > 0 {
		fmt.Printf("Latency p50: %s\n", percentile(latencies, 50))
		fmt.Printf("Latency p95: %s\n", percentile(latencies, 95))
		fmt.Printf("Latency p99: %s\n", percentile(latencies, 99))
		fmt.Printf("Latency Max: %s\n", latencies[len(latencies)-1])
	}

	fmt.Println("\n========== Process Memory ==========")
	fmt.Printf("Heap in use: %.2f MiB\n", float64(memAfter.HeapInuse)/1024/1024)
	fmt.Printf("Total allocated: %.2f MiB\n", float64(memAfter.TotalAlloc-memBefore.TotalAlloc)/1024/1024)
	fmt.Printf("Malloc allocations: %d\n", memAfter.Mallocs-memBefore.Mallocs)
	fmt.Println("(These memory/allocation values belong to the client, not leoxy)")

	if *pprofURL != "" {
		if err := fs.SaveHeapProf(client, *pprofURL, *profileOut); err != nil {
			fmt.Fprintf(os.Stderr, "Heap profile failed %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nLeoxy heap profile saved %s\n", *profileOut)
		fmt.Printf("Analyze with: go tool pprof -top %s\n", *profileOut)
	}

	if failed > 0 {
		os.Exit(1)
	}
}

func percentile(val []time.Duration, p int) time.Duration {
	index := (len(val)*p+99)/100 - 1
	if index < 0 {
		index = 0
	}
	if index >= len(val) {
		index = len(val) - 1
	}
	return val[index]
}
