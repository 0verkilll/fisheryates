# Concurrent Usage Example

Demonstrates safe patterns for using Fisher-Yates in concurrent applications.

## Running

```bash
go run main.go
```

## Expected Output

```
Concurrent Fisher-Yates Usage
==============================

Pattern 1: One instance per goroutine
-------------------------------------
Goroutine 0: [3 1 4 0 2]
Goroutine 1: [2 4 0 3 1]
Goroutine 2: [4 2 1 0 3]
Goroutine 3: [1 0 3 4 2]
Goroutine 4: [0 3 2 1 4]

Pattern 2: sync.Pool for high throughput
-----------------------------------------
Worker 0: [6 8 5 2 1]
Worker 1: [3 7 1 9 4]
...

Key Points:
- FisherYates is stateless, safe to share via pool
- SecureRandom MUST be re-seeded before each use from pool
- Use GenerateInto with buffer pool for zero allocations
```

## Thread Safety

**FisherYates** is stateless and safe to share between goroutines. However, the **RandomSource** (SecureRandom) maintains internal state and is NOT thread-safe.

## Pattern 1: One Instance Per Goroutine

Simplest and safest approach. Each goroutine creates its own instances:

```go
go func() {
    hasher := sha1.NewSHA1(sha1.NewBigEndian())
    random := securerandom.NewSecureRandom(hasher)
    fy := fisheryates.NewFisherYates()

    random.Seed([]byte("unique-seed"))
    perm, _ := fy.Generate(100, random)
}()
```

**Pros:** Simple, no synchronization needed
**Cons:** More allocations for high-throughput scenarios

## Pattern 2: sync.Pool for High Throughput

Reuse instances via sync.Pool for performance-critical applications:

```go
fyPool := &sync.Pool{
    New: func() interface{} {
        return fisheryates.NewFisherYates()
    },
}

randomPool := &sync.Pool{
    New: func() interface{} {
        hasher := sha1.NewSHA1(sha1.NewBigEndian())
        return securerandom.NewSecureRandom(hasher)
    },
}

bufferPool := &sync.Pool{
    New: func() interface{} {
        buf := make([]int, 0, 1000)
        return &buf
    },
}

// In goroutine:
fy := fyPool.Get().(*fisheryates.FisherYates)
random := randomPool.Get().(*securerandom.SecureRandom)
bufPtr := bufferPool.Get().(*[]int)

random.Seed([]byte("unique-seed"))  // ALWAYS re-seed!
perm, _ := fy.GenerateInto(*bufPtr, size, random)

// Return to pools
fyPool.Put(fy)
randomPool.Put(random)
*bufPtr = perm[:0]
bufferPool.Put(bufPtr)
```

**Pros:** Minimal allocations, high throughput
**Cons:** More complex, must remember to re-seed

## Critical Rules

1. **Never share SecureRandom between goroutines** without synchronization
2. **Always re-seed** SecureRandom when getting from pool
3. **FisherYates is safe** to share (stateless)
4. Use **GenerateInto** with buffer pools for zero-allocation
