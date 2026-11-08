# rewindfs

🚧 Work in Progress

rewindfs is a small, Git-inspired filesystem versioning project. it watches files, stores snapshots, and exposes history, comparison, and recovery through an HTTP API.

The current implementation is built around Go, SQLite, and fsnotify. The focus is on getting the snapshot lifecycle and storage behavior right before adding more features.

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

The feature list below describes what is in the repo right now. It is still a learning project, so this is not being presented as production-ready.

### Working foundation

- [x] Go application and Gin HTTP server
- [x] Recursive filesystem watcher using fsnotify
- [x] Relative snapshot paths and debounced file events
- [x] Handles new directories and common atomic-save rename events
- [x] Graceful watcher and HTTP server shutdown
- [x] Snapshot creation through the API and watcher
- [x] Versioned SQLite schema initialization and migration
- [x] Database-assigned snapshot IDs
- [x] Snapshot timestamps and SHA-256 hashes
- [x] Consecutive duplicate snapshot detection
- [x] Thread-safe snapshot service and in-memory indexes
- [x] Explicit SQLite store lifecycle and configurable database path
- [x] WAL journaling and a bounded SQLite busy timeout
- [x] Snapshot lookup, listing, deletion, history, and diff endpoints
- [x] Workspace-bound snapshot paths and safe restore/recovery
- [x] Nested file lookups through the `?file=...` query parameter
- [x] Local-only API binding and same-origin request checks
- [x] Dashboard renders API data as text instead of injecting it as HTML
- [x] Unit and API tests with GitHub Actions

### Still to do

- [ ] Improve diff output beyond whole-content replacement.
- [ ] Finish dashboard history, restore, delete, and error states.
- [ ] Add metadata indexing and snapshot search.
- [ ] Add a CLI.
- [ ] Support directory rollback and multiple watch roots.
- [ ] Evaluate content-addressed blob storage and garbage collection.

---

## Architecture

The root Go module is the application that is built and run. Startup opens the configured SQLite database and workspace before the filesystem watcher begins processing events.

- `internal/api` registers HTTP routes and maps requests to snapshot service operations.
- `internal/snapshot` owns snapshot creation, hashing, duplicate detection, deletion, and thread-safe in-memory indexes.
- `internal/models` defines the snapshot data model.
- `internal/storage` owns SQLite connections, schema migrations, and snapshot persistence.
- `internal/workspace` validates relative file keys and handles safe writes inside the workspace.
- `internal/vfs` watches the configured workspace and submits changes for snapshotting.
- `internal/diff` compares snapshot content.
- `frontend` contains the browser dashboard served by the API.

See [docs/architecture.md](docs/architecture.md) for the runtime flow and package responsibilities.

---

## API

The server listens on `127.0.0.1:8080` by default, so it is only reachable from the local machine. Browser requests with a different origin are rejected too.

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/health` | Health check |
| GET | `/stats` | Snapshot, file, and hash counts |
| GET | `/files` | List tracked files |
| GET | `/files/exists?file=src/main.go` | Check whether a file has snapshots |
| GET | `/snapshots` | List all snapshots |
| GET | `/snapshots/file?file=src/main.go` | List snapshots for a file |
| GET | `/snapshot/:id` | Get one snapshot |
| POST | `/snapshot` | Create a snapshot |
| DELETE | `/snapshot/:id` | Delete a snapshot |
| GET | `/latest?file=src/main.go` | Get the latest snapshot |
| GET | `/snapshot-oldest?file=src/main.go` | Get the oldest snapshot |
| GET | `/snapshot-count?file=src/main.go` | Count snapshots for a file |
| GET | `/history?file=src/main.go` | Get file history metadata |
| GET | `/history-full?file=src/main.go` | Get full snapshot history |
| POST | `/restore/:id` | Restore a snapshot |
| POST | `/recover?file=src/main.go` | Recover a file from its latest snapshot |
| GET | `/diff/:id1/:id2` | Compare two snapshots |

The older single-segment routes are still available. use the query form for nested files.

### Create a snapshot

```bash
curl -X POST http://127.0.0.1:8080/snapshot \
  -H "Content-Type: application/json" \
  -d '{"file":"notes.txt","content":"first version"}'
```

### List snapshots

```bash
curl http://127.0.0.1:8080/snapshots
```

### Restore a snapshot

```bash
curl -X POST http://127.0.0.1:8080/restore/1
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

The watched workspace defaults to the current directory. Set `REWINDFS_ROOT` to point it somewhere else:

```bash
REWINDFS_ROOT="$HOME/projects/my-app" go run .
```

The watcher and restore/recovery API use the same canonical workspace root. Snapshot paths are stored relative to it. Restore writes use a temporary file and rename it into place after checking the path and its existing components. As with most path-based filesystem checks, a local process that can actively swap symlinks at exactly the right time is outside the protection this version can fully guarantee.

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
