# rewindfs

🚧 Work in Progress

rewindfs is a Git-inspired virtual filesystem project focused on snapshotting, rollback, file recovery, file history, and versioned storage.

The project explores how modern version-control systems and snapshot-based filesystems manage file state, recovery, historical versions, content hashing, and storage efficiency through a simplified implementation built in Go.

## Goals

* Track file changes over time
* Create snapshots automatically
* Restore previous versions of files
* Recover deleted files
* Compare changes between snapshots
* Reduce duplicate storage using content hashing
* Explore filesystem and storage design concepts

## Current Progress

* [x] Repository setup
* [x] Filesystem watcher
* [x] Automatic snapshot generation
* [x] Snapshot persistence (JSON storage)
* [x] Snapshot timestamps
* [x] Snapshot lookup API
* [x] Snapshot restore API
* [x] Snapshot delete API
* [x] Snapshot statistics API
* [x] Deleted file recovery API
* [x] Latest snapshot lookup API
* [x] Oldest snapshot lookup API
* [x] File history API
* [x] Full snapshot history API
* [x] Snapshot count API
* [x] File existence API
* [x] Snapshot comparison API
* [x] SHA256 content hashing
* [x] Duplicate snapshot prevention
* [ ] Metadata indexing
* [ ] CLI interface
* [ ] Frontend dashboard
* [ ] Recovery testing

## Implemented Features

### Snapshot Management

* Automatic file monitoring
* Snapshot creation on file changes
* Persistent snapshot storage
* Timestamped snapshots
* Snapshot listing
* Snapshot lookup by ID
* Latest snapshot retrieval
* Oldest snapshot retrieval
* Snapshot deletion
* Snapshot counting per file
* File existence checks
* File history tracking

### Recovery

* Restore a file from a specific snapshot
* Recover deleted files using the most recent snapshot

### Diff and History

* Compare two snapshots
* View snapshot history metadata
* View complete snapshot history
* Track historical versions of files

### Storage Optimization

* SHA256 content hashing
* Hash-based snapshot identification
* Duplicate snapshot prevention
* Unique hash statistics

### API Endpoints

* `GET /health`
* `GET /stats`
* `GET /snapshots`
* `GET /files`
* `GET /snapshot/:id`
* `GET /latest/:file`
* `GET /snapshot-oldest/:file`
* `GET /snapshot-count/:file`
* `GET /snapshots/file/:name`
* `GET /history/:file`
* `GET /history-full/:file`
* `GET /files/:file/exists`
* `GET /snapshot-latest-id/:file`
* `GET /snapshot-first-id/:file`
* `GET /diff/:id1/:id2`
* `POST /snapshot`
* `POST /restore/:id`
* `POST /recover/:file`
* `DELETE /snapshot/:id`

### Storage

* JSON-based snapshot persistence
* SHA256 content hashing
* In-memory snapshot indexing
* Duplicate snapshot filtering

## Planned Features

* Metadata indexing
* Command-line interface
* Frontend dashboard
* Folder rollback
* Recovery logging
* Storage optimization metrics
* SQLite-backed storage
* Snapshot export/import

## Project Focus

This project is being built to gain practical experience with:

* Filesystem architecture
* Version control concepts
* Snapshot-based storage systems
* Recovery and rollback mechanisms
* Metadata management
* Content hashing
* Deduplication techniques
* Backend API development
* Systems programming in Go

## Tech Stack

* Go
* Gin
* JSON Storage
* SQLite (planned)
* Linux
* File I/O
* Git

## Status

Currently under active development.

### Recent Milestones

* Persistent snapshot storage
* Snapshot restore functionality
* Snapshot deletion functionality
* Deleted file recovery
* Snapshot statistics endpoint
* File history tracking
* Snapshot comparison API
* SHA256 content hashing
* Duplicate snapshot prevention
* Storage efficiency statistics

## Contributors

* Siddharth S Menon

  * Snapshot management APIs
  * Recovery APIs
  * Statistics endpoints
  * History and lookup endpoints
  * Snapshot comparison APIs
  * Deduplication implementation

* Tanisha Khoria

  * Snapshot engine
  * Filesystem watcher
  * Persistence layer
  * Core snapshot infrastructure
  * Storage architecture
  * Hashing integration
