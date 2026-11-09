# Architecture

This document describes the application that currently lives on `main`. It is specific to the implementation in the repository; planned features are called out as limitations rather than presented as completed work.

## Application boundary

The repository has one active Go application, defined by the root `go.mod`. Run commands such as `go test ./...` and `go run .` from the repository root.

An earlier `backend/` directory contained a separate Go module, a second SQLite schema, sample files, and a small blob-storage experiment. It was not imported by the root application, so it was removed from the active tree. Content-addressed storage now lives in the active `internal/storage` package, with an explicit v1-to-v2 migration and reference lifecycle.

## Runtime flow

1. `main.go` calls `api.Initialize`, which opens the configured SQLite database and loads persisted snapshots.
2. Only after initialization succeeds, the recursive filesystem watcher starts. It watches directories under the selected root and skips Git metadata, Node dependencies, and common SQLite/runtime files.
3. The watcher debounces create/write/rename events, reads the settled file, and submits it using a slash-separated path relative to the watch root.
4. `internal/api` maps HTTP requests to the snapshot service. `internal/snapshot` validates the file key, calculates the SHA-256 hash, checks the latest version for duplicates, and coordinates persistence.
5. `internal/storage` owns a `Store` instance. It applies schema migrations transactionally, writes snapshot metadata and shared content blobs, and returns IDs assigned by SQLite. Single writes and archive imports use the same blob/reference rules; an import batch commits as one transaction.
6. The API exposes history and lookup routes, workspace-safe restore/recovery, file-path search, a local maintenance route for blob collection, and versioned history archive import/export.
7. The dashboard in `frontend/` reads snapshot, history, statistics, search, and diff data through the HTTP API.

Snapshot creation and deletion are serialized by the service. Read operations use a read lock and return copies of stored values so callers cannot mutate the service's indexes.

## Package responsibilities

| Path | Responsibility |
|---|---|
| `main.go` | Initialize the application, start the watcher, and run the API server |
| `internal/api` | HTTP routes, request/response mapping, application initialization, and the SQLite repository adapter |
| `internal/archive` | Versioned ZIP manifest, deduplicated blob entries, archive validation, and size limits |
| `internal/snapshot` | Snapshot lifecycle, hashing, duplicate checks, indexes, and concurrency control |
| `internal/models` | Snapshot data model |
| `internal/storage` | SQLite connection lifecycle, schema migrations, snapshot metadata, content blobs, and blob collection |
| `internal/workspace` | Workspace-root path validation and atomic restore/recovery writes |
| `internal/vfs` | Filesystem event monitoring |
| `internal/diff` | Bounded line-oriented snapshot comparison |
| `frontend` | Browser dashboard |
| `internal/client` | HTTP client used by the CLI |
| `cmd/rewindfs` | Command-line interface over the local API |

The snapshot service depends on a small repository interface. The API package supplies the SQLite-backed implementation, while service tests use an in-memory repository. Storage tests open independent temporary database files, so test runs do not share or overwrite a developer's runtime database.

## Snapshot lifecycle

Creating a snapshot follows this order:

1. Reject an empty file key.
2. Hash the supplied content.
3. Check whether the latest snapshot for that file already has the same hash.
4. Insert or reuse the SHA-256 content blob and increment its reference count.
5. Insert snapshot metadata pointing to that blob. SQLite's `AUTOINCREMENT` sequence assigns the ID.
6. Add the persisted snapshot to the service's ordered slice and lookup indexes.

If persistence fails, the service leaves its in-memory state unchanged. Deleting a snapshot follows the same rule in reverse: the metadata row is deleted and its blob reference decremented in one transaction. The stored bytes are removed when the final snapshot reference disappears. The maintenance collector can recalculate reference counts from metadata and remove orphaned blobs.

Using the database-assigned ID matters when the highest-numbered snapshot is deleted. Computing the next ID from the largest row still present could reuse a deleted ID after a restart; SQLite's sequence retains the high-water mark.

## SQLite lifecycle and migrations

The API owns one storage `Store` for the lifetime of the application. The store opens the configured path, applies connection settings, checks that SQLite is reachable, and completes migrations before it is made available to the snapshot service.

The default database path remains `./rewindfs.db` so existing installations continue to use their current database. Set `REWINDFS_DB_PATH` to select another location. Parent directories are created when needed.

The schema version is tracked through SQLite's `PRAGMA user_version`. Migrations run in a transaction, so a failed schema change is rolled back rather than leaving a partially initialized database. Version 0 databases receive the initial snapshot schema; version 1 databases are migrated to version 2 by moving their content into `content_blobs` and replacing the content column with a `blob_hash` reference. Existing snapshot IDs and the AUTOINCREMENT high-water mark are preserved. The application refuses to open a database whose schema version is newer than it supports.

SQLite connections use WAL journaling, a five-second busy timeout, foreign-key enforcement, and a single pooled connection per store. The single-connection pool keeps per-connection settings consistent and avoids opening independent in-process connections to the same local database; the busy timeout gives other processes time to release SQLite locks.

## Data and runtime files

SQLite is the persistence layer for the active application. The database file defaults to `rewindfs.db` in the process working directory and can be relocated with `REWINDFS_DB_PATH`. It is local runtime data and must stay out of version control.

The v2 schema separates `snapshots` metadata from `content_blobs`. Each metadata row references a blob by SHA-256 hash; the blob table stores the bytes and a reference count. Two files with identical content can therefore share one stored byte sequence. Snapshot insertion and reference updates are transactional. Deleting the final reference removes the blob, while `Store.GarbageCollect` recalculates counts from metadata and removes orphaned content. The local API exposes this as `POST /maintenance/gc`, and the CLI exposes it as `rewindfs gc`.

## Decisions for upcoming work

- Keep one buildable Go module until a genuinely separate executable is needed.
- Keep SQLite as the metadata store and source of snapshot IDs.
- Keep snapshot lifecycle rules in the snapshot service.
- Keep HTTP handlers focused on transport, validation, and response mapping.
- Treat paths received through the API as untrusted input. Restore and recovery must be confined to a configured root.
- Keep database files, SQLite sidecars, and generated blobs out of commits.
- Add new storage formats only when they are integrated with the active application and covered by tests.

## Current boundaries

- The service watches one workspace root per process.
- Snapshot archives use a versioned ZIP manifest and SHA-256 blob entries. Import assigns new local snapshot IDs and restores history only; it does not write files into the workspace.
- The line diff uses a bounded LCS table and falls back to a full replacement for very large line combinations. Archive import/export currently has fixed 64 MiB compressed and 128 MiB uncompressed limits.
- Workspace path checks reject existing symlinks and traversal, but path-based checks cannot fully prevent a hostile local process from swapping directory entries between checks.
- The API is intentionally local-only. Remote access needs authentication and a separate deployment model. The CLI is an API client and does not open SQLite or write directly to the workspace.

These are deliberate boundaries, not hidden capabilities. New storage formats, remote access, and additional executables should be added with independent migrations and tests.
