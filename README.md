# Rinha de Backend 2026 — Fraud Detection (Go)

This project implements a high-performance fraud scoring API using an **exact KNN (k=5)** approach optimized for low latency and constrained CPU environments.

---

## Overview

The system receives a transaction payload and computes a fraud score based on similarity against a reference dataset using **Euclidean distance in a 14-dimensional feature space**.

Key characteristics:

- Dataset: ~100,000 vectors
- Dimensions: 14
- KNN: exact search (k = 5)
- Language: Go
- Deployment: Docker + NGINX load balancing
- CPU budget: 1.0 core (Rinha constraint)

---

## Architecture

```
k6 / client
    ↓
NGINX (0.10 CPU)
    ↓
2x API instances (0.45 CPU each)
    ↓
In-memory dataset + KNN search
```

- NGINX performs load balancing between API instances
- Each API instance loads the full dataset into memory at startup
- Requests are stateless and processed independently

---

## Performance

Baseline (final version):

- ~580–600 requests/sec
- p50 ≈ 2 ms
- p95 ≈ 55–70 ms
- p99 ≈ 75–80 ms
- 0 errors under load

Performance was validated using `k6` with up to 20 VUs.

---

## Implementation Details

### 1. Exact KNN

The system performs a full scan over the dataset:

```go
for i := 0; i < dataset.Count; i++ {
    distance := squaredEuclidean(query, reference)
    insertFixedNeighbor(...)
}
```

Despite being O(N), the implementation is heavily optimized.

---

### 2. Zero Allocation Design

- No heap allocations in the hot path
- Fixed-size arrays (`[5]Neighbor`) used for top-k
- Reused memory throughout execution

```
0 B/op
0 allocs/op
```

---

### 3. Distance Optimization

The Euclidean distance function is **fully unrolled**:

```go
d0 := query[0] - reference[0]
...
d13 := query[13] - reference[13]
```

Benefits:

- eliminates loop overhead
- removes bounds checks
- improves CPU pipeline utilization

---

### 4. CPU-Constrained Design

Extensive benchmarking was done under Rinha constraints:

- 1 API (0.9 CPU) → worse tail latency
- 3 APIs (0.3 CPU each) → worse throughput
- **2 APIs (0.45 CPU each) → best balance**

Final choice:

```yaml
api1: 0.45 CPU
api2: 0.45 CPU
nginx: 0.10 CPU
```

---

### 5. Profiling

CPU profiling revealed:

- ~73% of CPU time spent in distance computation
- ~96% inside KNN overall

This guided optimization efforts toward:

- distance function
- memory access patterns

---

## Benchmarking

Microbenchmark:

```
BenchmarkExactKNN_SearchInto_100k_14d
~835,000 ns/op
0 allocs/op
```

Load test:

```
k6 run test.js
```

---

## How to Run

```bash
docker compose up --build
```

API will be available at:

```
http://localhost:9999/fraud-score
```

---

## Future Improvements

This implementation prioritizes correctness and predictability.

Next steps would include:

- Approximate Nearest Neighbor (ANN)
- IVF (centroid-based partitioning)
- Reducing comparisons from 100k → 5k–20k per request

These approaches can significantly increase throughput while maintaining acceptable accuracy.

---

## Summary

This solution focuses on:

- correctness (exact KNN)
- performance under CPU limits
- zero allocations
- predictable latency behavior

It serves as a strong baseline for further optimization using ANN techniques.
