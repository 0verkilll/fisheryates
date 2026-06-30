package main

import (
	"fmt"
	"sync"

	"github.com/0verkilll/fisheryates"
	"github.com/0verkilll/securerandom"
	"github.com/0verkilll/sha1"
)

func main() {
	fmt.Println("Concurrent Fisher-Yates Usage")
	fmt.Println("==============================")
	fmt.Println()

	// Demonstrate safe concurrent usage patterns
	demonstrateGoroutinePerInstance()
	demonstrateSyncPool()
}

// demonstrateGoroutinePerInstance shows the simplest safe pattern:
// each goroutine creates its own instances
func demonstrateGoroutinePerInstance() {
	fmt.Println("Pattern 1: One instance per goroutine")
	fmt.Println("-------------------------------------")

	var wg sync.WaitGroup
	results := make(chan string, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine creates its own instances
			hasher := sha1.NewSHA1(sha1.NewBigEndian())
			random := securerandom.NewSecureRandom(hasher)
			fy := fisheryates.NewFisherYates()

			// Seed with unique value per goroutine
			random.Seed([]byte(fmt.Sprintf("goroutine-%d", id)))

			perm, err := fy.Generate(5, random)
			if err != nil {
				results <- fmt.Sprintf("Goroutine %d: error: %v", id, err)
				return
			}
			results <- fmt.Sprintf("Goroutine %d: %v", id, perm)
		}(i)
	}

	wg.Wait()
	close(results)

	for result := range results {
		fmt.Println(result)
	}
	fmt.Println()
}

// demonstrateSyncPool shows the high-performance pattern using sync.Pool
func demonstrateSyncPool() {
	fmt.Println("Pattern 2: sync.Pool for high throughput")
	fmt.Println("-----------------------------------------")

	// Pool for FisherYates instances (stateless, safe to reuse)
	fyPool := &sync.Pool{
		New: func() interface{} {
			return fisheryates.NewFisherYates()
		},
	}

	// Pool for PRNG instances (must be re-seeded before each use)
	randomPool := &sync.Pool{
		New: func() interface{} {
			hasher := sha1.NewSHA1(sha1.NewBigEndian())
			return securerandom.NewSecureRandom(hasher)
		},
	}

	// Pool for reusable buffers (zero-allocation pattern)
	bufferPool := &sync.Pool{
		New: func() interface{} {
			buf := make([]int, 0, 100) // Pre-allocate capacity
			return &buf
		},
	}

	var wg sync.WaitGroup
	results := make(chan string, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Get instances from pools
			fy := fyPool.Get().(*fisheryates.FisherYates)
			random := randomPool.Get().(*securerandom.SecureRandom)
			bufPtr := bufferPool.Get().(*[]int)
			buf := *bufPtr

			// IMPORTANT: Always re-seed before use
			random.Seed([]byte(fmt.Sprintf("pool-worker-%d", id)))

			// Use GenerateInto for zero-allocation
			perm, err := fy.GenerateInto(buf, 10, random)
			if err != nil {
				results <- fmt.Sprintf("Worker %d: error: %v", id, err)
			} else {
				results <- fmt.Sprintf("Worker %d: %v", id, perm[:5])
			}

			// Return instances to pools
			*bufPtr = perm[:0] // Reset length, keep capacity
			bufferPool.Put(bufPtr)
			fyPool.Put(fy)
			randomPool.Put(random)
		}(i)
	}

	wg.Wait()
	close(results)

	for result := range results {
		fmt.Println(result)
	}
	fmt.Println()

	fmt.Println("Key Points:")
	fmt.Println("- FisherYates is stateless, safe to share via pool")
	fmt.Println("- SecureRandom MUST be re-seeded before each use from pool")
	fmt.Println("- Use GenerateInto with buffer pool for zero allocations")
}
