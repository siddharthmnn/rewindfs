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
- [x] Shared content blobs keyed by SHA-256 across different files
- [x] Reference-counted blob cleanup and a consistency garbage collector
- [x] Thread-safe snapshot service and in-memory indexes
- [x] Explicit SQLite store lifecycle and configurable database path
- [x] WAL journaling and a bounded SQLite busy timeout
- [x] Snapshot lookup, listing, deletion, history, search, and diff endpoints
- [x] Bounded line-oriented snapshot diffs
- [x] Paginated snapshot search by workspace-relative file path
- [x] Workspace-bound snapshot paths and safe restore/recovery
- [x] Nested file lookups through the `?file=...` query parameter
- [x] Local-only API binding and same-origin request checks
- [x] Dashboard renders API data as text instead of injecting it as HTML
- [x] Dashboard snapshot filtering and latest-file recovery action
- [x] Unit and API tests with GitHub Actions

### Still to do

- [x] Add a Go CLI backed by the local HTTP API.
- [ ] Support directory rollback and multiple watch roots.
- [x] Content-addressed storage with cross-file deduplication and reference-counted garbage collection.

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

See [docs/architecture.md](docs/architecture.md) for the runtime flow and package responsibilities. [Installation, configuration, and release details](docs/installation.md) are covered separately.

---

## API

The server listens on `127.0.0.1:8080` by default. It accepts the local `127.0.0.1:8080` and `localhost:8080` hostnames, rejects unexpected Host/Origin values, and limits HTTP request bodies to 16 MiB by default. Configure `REWINDFS_MAX_REQUEST_BODY_BYTES` to change the JSON request limit (1 KiB–256 MiB); archive imports retain their separate 64 MiB cap. `GET /health` checks liveness, while `GET /ready` checks SQLite and workspace availability.

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/health` | Health check |
| GET | `/stats` | Snapshot, file, and hash counts |
| GET | `/files` | List tracked files |
| GET | `/files/exists?file=src/main.go` | Check whether a file has snapshots |
| GET | `/snapshots` | List all snapshots |
| GET | `/snapshots/search?q=src/&limit=25&offset=0` | Search and paginate snapshots by file path |
| GET | `/snapshots/file?file=src/main.go` | List snapshots for a file |
| GET | `/snapshot/:id` | Get one snapshot |
| POST | `/maintenance/gc` | Repair blob references and remove unreferenced content |
| GET | `/export` | Download a versioned snapshot archive |
| POST | `/import` | Validate and import snapshot history from an archive |
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

The older single-segment routes are still available. use the query form for nested files. Snapshot search returns `items`, `total`, `limit`, and `offset`; search matches file paths, not snapshot contents. Archive import returns `imported`, `skipped`, and the new local `snapshot_ids`.

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
REWINDFS_ROOT="$HOME/projects/my-app" REWINDFS_IGNORE="*.log,build/,tmp/" go run .
```

Set `REWINDFS_IGNORE` to add comma-separated ignore patterns. A pattern without a slash matches any path component; a pattern ending in `/` excludes that directory and its contents.

The watcher and restore/recovery API use the same canonical workspace root. Snapshot paths are stored relative to it. Restore writes use a temporary file and rename it into place after checking the path and its existing components. As with most path-based filesystem checks, a local process that can actively swap symlinks at exactly the right time is outside the protection this version can fully guarantee.

Set `REWINDFS_LOG_LEVEL` to `debug`, `info`, `warn`, or `error`, and `REWINDFS_LOG_FORMAT` to `text` or `json` to control structured application logs. `REWINDFS_MAX_REQUEST_BODY_BYTES` defaults to 16 MiB and accepts values from 1 KiB through 256 MiB.

SQLite uses WAL journaling and a five-second busy timeout. The schema is versioned with SQLite's `user_version` pragma and migrations run inside a transaction. Existing v1 databases are upgraded in place: snapshot metadata, IDs, and the AUTOINCREMENT high-water mark are preserved while repeated content moves into a shared blob table. Deleting a snapshot decrements the blob reference count, and `rewindfs gc` can repair counts and remove unreferenced content.

The application uses SQLite through `github.com/mattn/go-sqlite3`, which requires CGO and a C compiler in environments where the driver is built from source.

---


## Command-line interface

The CLI talks to the running local RewindFS API. Start the application in one terminal, then run commands from another terminal:

```bash
go run ./cmd/rewindfs status
go run ./cmd/rewindfs files
go run ./cmd/rewindfs snapshots
go run ./cmd/rewindfs history src/main.go
go run ./cmd/rewindfs diff 12 18
go run ./cmd/rewindfs restore 12
go run ./cmd/rewindfs recover src/main.go
go run ./cmd/rewindfs gc
go run ./cmd/rewindfs export ./backups/rewindfs.zip
go run ./cmd/rewindfs import ./backups/rewindfs.zip
```

Restore, recovery, deletion, garbage collection, archive import, and overwriting an existing export ask for confirmation. Add `--yes` to restore, recover, garbage collection, or archive import when running an intentional scripted operation. Set `REWINDFS_API` to use another API address; the default is `http://127.0.0.1:8080`.


### Snapshot archives

Export and import preserve snapshot history, timestamps, file keys, and content hashes in a versioned ZIP format. Identical content is stored once in the archive. Import assigns new local snapshot IDs, skips consecutive duplicates, validates every file key and blob checksum, and commits the batch in one SQLite transaction. It does **not** write imported file contents into the workspace; use restore or recover after importing when you actually want to materialize a version.

```bash
go run ./cmd/rewindfs export ./backups/rewindfs.zip
go run ./cmd/rewindfs import ./backups/rewindfs.zip
```

Archive uploads and downloads are limited to 64 MiB. The importer also checks the manifest version, ZIP entry names, decompressed sizes, SHA-256 values, and path traversal attempts.

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
