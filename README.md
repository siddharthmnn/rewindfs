# rewindfs

🚧 Work in Progress

rewindfs is a small, Git-inspired filesystem versioning project. It watches files, stores snapshots, and exposes history, comparison, and recovery operations through an HTTP API.

The current implementation is a learning project built around Go, SQLite, and fsnotify. The focus is on getting the snapshot lifecycle and storage behavior right before adding more features.

---

## Goals

- Track file changes over time.
- Create snapshots when watched files change.
- Keep snapshot metadata across restarts.
- Restore an earlier version of a file.
- Recover files from their latest stored snapshot.
- Compare snapshots and inspect file history.
- Avoid storing the same consecutive file version more than once.

---

## Current progress

The repository has a Go API, SQLite persistence, a filesystem watcher, a dedicated snapshot service, basic history and lookup endpoints, and a browser dashboard.

The feature list below describes the current implementation, not a claim that every path is production-ready. Watcher behavior, API validation, restore safety, and broader integration coverage still need work.

### Working foundation

- [x] Go application and Gin HTTP server
- [x] Filesystem watcher using fsnotify
- [x] Snapshot creation through the API and watcher
- [x] Versioned SQLite schema initialization
- [x] Transactional schema initialization and migration
- [x] Database-assigned snapshot IDs
- [x] Snapshot timestamps and SHA-256 hashes
- [x] Consecutive duplicate snapshot detection
- [x] Thread-safe snapshot service and in-memory indexes
- [x] Explicit SQLite store lifecycle and configurable database path
- [x] WAL journaling and a bounded SQLite busy timeout
- [x] Isolated storage tests using temporary databases
- [x] Persistence errors surfaced to callers
- [x] Snapshot lookup, listing, and deletion endpoints
- [x] Basic file history and latest/oldest snapshot lookups
- [x] Snapshot comparison endpoint
- [x] Restore and deleted-file recovery endpoints
- [x] Basic statistics and file listing endpoints
- [x] Initial dashboard
- [x] Unit tests and GitHub Actions test workflow

### Still to do

- [ ] Make watcher paths relative to a configured root and support nested directories.
- [ ] Restrict restore and recovery paths to the watched root.
- [ ] Improve diff output beyond whole-content replacement.
- [ ] Finish dashboard history, restore, delete, and error states.
- [ ] Add integration tests for the API, watcher, and recovery paths.
- [ ] Add metadata indexing and snapshot search.
- [ ] Add a CLI.
- [ ] Support directory rollback and multiple watch roots.
- [ ] Evaluate content-addressed blob storage and garbage collection.

---

## Architecture

The root Go module is the application that is built and run. Startup opens the configured SQLite database and loads existing snapshots before the filesystem watcher begins processing events.

- `internal/api` registers HTTP routes and maps requests to snapshot service operations.
- `internal/snapshot` owns snapshot creation, hashing, duplicate detection, ID handling through the repository, deletion, and thread-safe in-memory indexes.
- `internal/models` defines the snapshot data model.
- `internal/storage` owns SQLite connections, schema migrations, and snapshot persistence.
- `internal/vfs` watches the configured filesystem path and submits changes for snapshotting.
- `internal/diff` compares snapshot content.
- `frontend` contains the browser dashboard served by the API.

See [docs/architecture.md](docs/architecture.md) for the runtime flow, package responsibilities, and known limitations.

SQLite stores snapshot metadata and content together. Content-addressed blob storage is not part of the active application yet.

---

## API

The server listens on port `8080`.

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/health` | Health check |
| GET | `/stats` | Snapshot, file, and hash counts |
| GET | `/files` | List tracked files |
| GET | `/files/:name/exists` | Check whether a file has snapshots |
| GET | `/snapshots` | List all snapshots |
| GET | `/snapshots/file/:name` | List snapshots for a file |
| GET | `/snapshot/:id` | Get one snapshot |
| POST | `/snapshot` | Create a snapshot |
| DELETE | `/snapshot/:id` | Delete a snapshot |
| GET | `/latest/:file` | Get the latest snapshot for a file |
| GET | `/snapshot-oldest/:file` | Get the oldest snapshot for a file |
| GET | `/snapshot-count/:file` | Count snapshots for a file |
| GET | `/history/:file` | Get file history metadata |
| GET | `/history-full/:file` | Get full snapshot history |
| GET | `/snapshot-latest-id/:file` | Get the latest snapshot ID |
| GET | `/snapshot-first-id/:file` | Get the first snapshot ID |
| POST | `/restore/:id` | Restore a snapshot |
| POST | `/recover/:file` | Recover a file from its latest snapshot |
| GET | `/diff/:id1/:id2` | Compare two snapshots |

Restore/recovery path validation and consistent API error handling are still being hardened.

### Create a snapshot

```bash
curl -X POST http://localhost:8080/snapshot \
  -H "Content-Type: application/json" \
  -d '{"file":"notes.txt","content":"first version"}'
```

### List snapshots

```bash
curl http://localhost:8080/snapshots
```

### Restore a snapshot

```bash
curl -X POST http://localhost:8080/restore/1
```

---

## Development

Run the application from the repository root:

```bash
go run .
```

Run the test suite from the same directory:

```bash
go test ./...
```

The database defaults to `./rewindfs.db` for compatibility with existing installations. Set `REWINDFS_DB_PATH` to use an explicit location:

```bash
REWINDFS_DB_PATH="$HOME/.local/share/rewindfs/rewindfs.db" go run .
```

SQLite uses WAL journaling and a five-second busy timeout. The schema is versioned with SQLite's `user_version` pragma and migrations run inside a transaction. Existing databases are upgraded in place without replacing their snapshot rows.

The application uses SQLite through `github.com/mattn/go-sqlite3`, which requires CGO and a C compiler in environments where the driver is built from source.

---

## Tech

- Go
- Gin
- SQLite
- fsnotify
- SHA-256
- HTML, CSS, and JavaScript
- GitHub Actions

## Contributors

### Siddharth S Menon

Snapshot APIs, SQLite integration, history and lookup endpoints, diff functionality, dashboard, tests, and CI.

### Tanisha Khoria

Snapshot engine groundwork, filesystem watcher, and early storage design.
