# Architecture

This document describes the application that currently lives on `main`. It is specific to the implementation in the repository; planned features are called out as limitations rather than presented as completed work.

## Application boundary

The repository has one active Go application, defined by the root `go.mod`. Run commands such as `go test ./...` and `go run .` from the repository root.

An earlier `backend/` directory contained a separate Go module, a second SQLite schema, sample files, and a small blob-storage experiment. It was not imported by the root application, so it was removed from the active tree. Its content-addressed storage direction can be revisited inside the main application with an explicit migration and lifecycle design.

## Runtime flow

1. `main.go` calls `api.Initialize`, which opens SQLite and loads persisted snapshots.
2. Only after initialization succeeds, the filesystem watcher starts. The Gin server then begins accepting HTTP requests.
3. `internal/vfs` reads changed file content and submits it through the API's snapshot entry point.
4. `internal/api` maps HTTP requests to the snapshot service. `internal/snapshot` validates the file key, calculates the SHA-256 hash, checks the latest version for duplicates, and coordinates persistence.
5. `internal/storage` inserts new snapshot rows and returns the ID assigned by SQLite. The service updates its in-memory indexes only after the insert succeeds.
6. The dashboard in `frontend/` reads snapshot, history, statistics, and diff data through the HTTP API.

Snapshot creation and deletion are serialized by the service. Read operations use a read lock and return copies of stored values so callers cannot mutate the service's indexes.

## Package responsibilities

| Path | Responsibility |
|---|---|
| `main.go` | Initialize the application, start the watcher, and run the API server |
| `internal/api` | HTTP routes, request/response mapping, and the SQLite repository adapter |
| `internal/snapshot` | Snapshot lifecycle, hashing, duplicate checks, indexes, and concurrency control |
| `internal/models` | Snapshot data model |
| `internal/storage` | SQLite schema and snapshot persistence |
| `internal/vfs` | Filesystem event monitoring |
| `internal/diff` | Snapshot content comparison |
| `frontend` | Browser dashboard |

The snapshot service depends on a small repository interface. The API package supplies the SQLite-backed implementation, while service tests use an in-memory repository to exercise lifecycle behavior without sharing a database file.

## Snapshot lifecycle

Creating a snapshot follows this order:

1. Reject an empty file key.
2. Hash the supplied content.
3. Check whether the latest snapshot for that file already has the same hash.
4. Insert a new row into SQLite. SQLite's `AUTOINCREMENT` sequence assigns the ID.
5. Add the persisted snapshot to the service's ordered slice and lookup indexes.

If persistence fails, the service leaves its in-memory state unchanged. Deleting a snapshot follows the same rule in reverse: the database deletion must succeed before the in-memory indexes are updated.

Using the database-assigned ID matters when the highest-numbered snapshot is deleted. Computing the next ID from the largest row still present could reuse a deleted ID after a restart; SQLite's sequence retains the high-water mark.

## Data and runtime files

SQLite is the persistence layer for the active application. The database file is named `rewindfs.db` and is created relative to the process working directory. It is local runtime data and must stay out of version control.

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
- SQLite integration tests still share a local database filename and need stronger isolation.
- The HTTP middleware and handlers need a separate validation and error-handling pass.

These limitations are tracked in the project roadmap and should be removed in the relevant implementation sections.
