# Architecture

This document describes the application that currently lives on `main`. It is specific to the implementation in the repository; planned features are called out as limitations rather than presented as completed work.

## Application boundary

The repository has one active Go application, defined by the root `go.mod`. Run commands such as `go test ./...` and `go run .` from the repository root.

An earlier `backend/` directory contained a separate Go module, a second SQLite schema, sample files, and a small blob-storage experiment. It was not imported by the root application, so it was removed from the active tree. Its content-addressed storage direction can be revisited inside the main application with an explicit migration and lifecycle design.

## Runtime flow

1. `main.go` calls `api.Initialize`, which opens the configured SQLite database and loads persisted snapshots.
2. Only after initialization succeeds, the filesystem watcher starts. The Gin server then begins accepting HTTP requests.
3. `internal/vfs` reads changed file content and submits it through the API's snapshot entry point.
4. `internal/api` maps HTTP requests to the snapshot service. `internal/snapshot` validates the file key, calculates the SHA-256 hash, checks the latest version for duplicates, and coordinates persistence.
5. `internal/storage` owns a `Store` instance. It applies schema migrations transactionally, inserts new snapshot rows, and returns the ID assigned by SQLite. The service updates its in-memory indexes only after the insert succeeds.
6. The dashboard in `frontend/` reads snapshot, history, statistics, and diff data through the HTTP API.

Snapshot creation and deletion are serialized by the service. Read operations use a read lock and return copies of stored values so callers cannot mutate the service's indexes.

## Package responsibilities

| Path | Responsibility |
|---|---|
| `main.go` | Initialize the application, start the watcher, and run the API server |
| `internal/api` | HTTP routes, request/response mapping, application initialization, and the SQLite repository adapter |
| `internal/snapshot` | Snapshot lifecycle, hashing, duplicate checks, indexes, and concurrency control |
| `internal/models` | Snapshot data model |
| `internal/storage` | SQLite connection lifecycle, schema migrations, and snapshot persistence |
| `internal/vfs` | Filesystem event monitoring |
| `internal/diff` | Snapshot content comparison |
| `frontend` | Browser dashboard |

The snapshot service depends on a small repository interface. The API package supplies the SQLite-backed implementation, while service tests use an in-memory repository. Storage tests open independent temporary database files, so test runs do not share or overwrite a developer's runtime database.

## Snapshot lifecycle

Creating a snapshot follows this order:

1. Reject an empty file key.
2. Hash the supplied content.
3. Check whether the latest snapshot for that file already has the same hash.
4. Insert a new row into SQLite. SQLite's `AUTOINCREMENT` sequence assigns the ID.
5. Add the persisted snapshot to the service's ordered slice and lookup indexes.

If persistence fails, the service leaves its in-memory state unchanged. Deleting a snapshot follows the same rule in reverse: the database deletion must succeed before the in-memory indexes are updated.

Using the database-assigned ID matters when the highest-numbered snapshot is deleted. Computing the next ID from the largest row still present could reuse a deleted ID after a restart; SQLite's sequence retains the high-water mark.

## SQLite lifecycle and migrations

The API owns one storage `Store` for the lifetime of the application. The store opens the configured path, applies connection settings, checks that SQLite is reachable, and completes migrations before it is made available to the snapshot service.

The default database path remains `./rewindfs.db` so existing installations continue to use their current database. Set `REWINDFS_DB_PATH` to select another location. Parent directories are created when needed.

The schema version is tracked through SQLite's `PRAGMA user_version`. Migrations run in a transaction, so a failed schema change is rolled back rather than leaving a partially initialized database. Existing databases with the original snapshots table are recognized as version 0 and receive the initial version marker without losing rows. The application refuses to open a database whose schema version is newer than it supports.

SQLite connections use WAL journaling, a five-second busy timeout, foreign-key enforcement, and a single pooled connection per store. The single-connection pool keeps per-connection settings consistent and avoids opening independent in-process connections to the same local database; the busy timeout gives other processes time to release SQLite locks.

## Data and runtime files

SQLite is the persistence layer for the active application. The database file defaults to `rewindfs.db` in the process working directory and can be relocated with `REWINDFS_DB_PATH`. It is local runtime data and must stay out of version control.

The current schema stores file path, content, hash, and creation time in the same snapshot row. The earlier blob prototype is not wired into this schema. Moving to content-addressed blobs will require an explicit migration and lifecycle design, including how references are counted and how unreferenced blobs are collected.

## Decisions for upcoming work

- Keep one buildable Go module until a genuinely separate executable is needed.
- Keep SQLite as the metadata store and source of snapshot IDs.
- Keep snapshot lifecycle rules in the snapshot service.
- Keep HTTP handlers focused on transport, validation, and response mapping.
- Treat paths received through the API as untrusted input. Restore and recovery must be confined to a configured root.
- Keep database files, SQLite sidecars, and generated blobs out of commits.
- Add new storage formats only when they are integrated with the active application and covered by tests.

## Known limitations

The current implementation is a working prototype, not yet a hardened filesystem service. Remaining limitations include:

- The watcher monitors only the path passed to it, does not recurse into subdirectories, and identifies files by basename.
- Restore and recovery do not yet enforce a configured filesystem root.
- The current diff implementation reports changed content as whole old/new strings.
- The dashboard interpolates API data into HTML and needs safe DOM rendering.
- The HTTP middleware and handlers need a separate validation and error-handling pass.

These limitations are tracked in the project roadmap and should be removed in the relevant implementation sections.
