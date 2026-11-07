# Architecture

This document describes the application that currently lives on `main`. It is intentionally specific to the code in the repository; it is not a description of features that are only planned.

## Application boundary

The repository has one active Go application, defined by the root `go.mod`. Run commands such as `go test ./...` and `go run .` from the repository root.

An earlier `backend/` directory contained a separate Go module, a second SQLite schema, sample files, and a small blob-storage experiment. It was not imported by the root application, and its database and sample data were not application fixtures. That prototype has been removed from the active tree. Its design can be revisited as part of a proper content-addressed storage implementation rather than maintained as a second, disconnected application.

## Runtime flow

1. `main.go` starts the filesystem watcher in a goroutine and starts the Gin server.
2. `internal/vfs` receives filesystem events, reads file content, and submits snapshot requests to the API package.
3. `internal/api` calculates the content hash, assigns an ID, updates its in-memory indexes, and asks the storage package to persist the snapshot.
4. `internal/storage` stores the snapshot record in SQLite.
5. The dashboard in `frontend/` reads snapshot, history, statistics, and diff data through the HTTP API.

The API package currently owns process-wide in-memory snapshot slices and maps. Startup loads persisted snapshots into those indexes. This is a known boundary to improve: snapshot lifecycle and indexing belong in a service layer, not in HTTP handlers or package-level mutable state.

## Package responsibilities

| Path | Responsibility |
|---|---|
| `main.go` | Application startup |
| `internal/api` | HTTP routes, handlers, and current snapshot indexes |
| `internal/models` | Snapshot data model |
| `internal/storage` | SQLite persistence and content hashing |
| `internal/vfs` | Filesystem event monitoring |
| `internal/diff` | Snapshot content comparison |
| `frontend` | Browser dashboard |

## Data and runtime files

SQLite is the persistence layer for the active application. The database file is named `rewindfs.db` and is created relative to the process working directory. It is local runtime data and must stay out of version control.

The current schema stores file path, content, hash, and creation time in the same snapshot row. The earlier blob prototype is not wired into this schema. Moving to content-addressed blobs will require an explicit migration and lifecycle design, including how references are counted and how unreferenced blobs are collected.

## Decisions for upcoming work

- Keep one buildable Go module until a genuinely separate executable is needed.
- Keep SQLite as the metadata store.
- Put snapshot lifecycle rules in an application/service layer.
- Keep HTTP handlers focused on transport, validation, and response mapping.
- Treat paths received through the API as untrusted input. Restore and recovery must be confined to a configured root.
- Keep database files, SQLite sidecars, and generated blobs out of commits.
- Add new storage formats only when they are integrated with the active application and covered by tests.

## Known limitations

The current implementation is a working prototype, not yet a hardened filesystem service. Among the limitations to address are:

- Snapshot indexes are mutable package-level state and are not protected against concurrent access.
- Snapshot persistence errors are currently ignored by the snapshot creation path.
- The watcher monitors only the path passed to it, does not recurse into subdirectories, and identifies files by basename.
- Restore and recovery do not yet enforce a configured filesystem root.
- The current diff implementation reports changed content as whole old/new strings.
- The dashboard interpolates API data into HTML and needs safe DOM rendering.

These limitations are tracked in the project roadmap and should be removed in the relevant implementation sections.
