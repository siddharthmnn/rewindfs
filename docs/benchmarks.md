# Storage benchmarks

RewindFS includes benchmarks for the SQLite snapshot storage path. Run them
from the repository root:

```bash
go test ./internal/storage -run '^$' -bench 'BenchmarkSnapshot' -benchmem
```

The current benchmark set measures:

- `BenchmarkSnapshotInsert`: unique file content and unique content hashes.
- `BenchmarkSnapshotInsertSharedContent`: multiple snapshot rows referring to
  one shared blob, exercising the deduplication/ref-count update path.
- `BenchmarkSnapshotLoadAll`: loading a seeded 1,000-snapshot history.

For more stable comparisons, use multiple samples and compare them with
`benchstat`:

```bash
go test ./internal/storage -run '^$' -bench 'BenchmarkSnapshot' -benchmem -count=5 > before.txt
# make the storage change being evaluated
go test ./internal/storage -run '^$' -bench 'BenchmarkSnapshot' -benchmem -count=5 > after.txt
benchstat before.txt after.txt
```

Run the same Go version and machine setup for both samples. These benchmarks
are local performance measurements, not CI pass/fail thresholds. They help
spot regressions and compare storage changes; they don't replace profiling a
representative workload.
