# rewindfs

🚧 Work in Progress

rewindfs is a Git-inspired snapshotting filesystem project focused on file versioning, rollback, recovery, history tracking, deduplication, and storage efficiency.

The project explores how modern version-control systems and snapshot-based filesystems manage historical file state through automatic snapshot creation, content hashing, recovery workflows, and version-aware storage.

## Goals

* Track file changes over time
* Create snapshots automatically
* Restore previous versions of files
* Recover deleted files
* Compare changes between snapshots
* Reduce duplicate storage using content hashing
* Explore filesystem and storage design concepts
* Build maintainable backend architecture
* Implement automated testing and CI/CD

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
* [x] Automated unit tests
* [x] GitHub Actions CI pipeline
* [x] API route modularization
* [ ] Metadata indexing
* [ ] CLI interface
* [ ] Frontend dashboard
* [ ] SQLite storage backend
* [ ] Recovery testing

## Implemented Features

### Snapshot Management

* Automatic file monitoring using fsnotify
* Snapshot creation on file modification
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
* Recover deleted files using the latest snapshot

### Diff and History

* Compare two snapshots
* View snapshot history metadata
* View complete snapshot history
* Track historical versions of files
* Retrieve first and latest snapshot IDs

### Storage Optimization

* SHA256 content hashing
* Hash-based snapshot identification
* Duplicate snapshot prevention
* Unique hash statistics
* Content-based deduplication

### Testing & Quality

* Unit tests for hashing
* Unit tests for snapshot management
* Unit tests for deduplication
* Unit tests for diff functionality
* GitHub Actions automated test execution
* Automated validation on every push

### API Architecture

* Route modularization
* Dedicated stats handler module
* Dedicated history handler module
* Reduced server.go complexity
* Incremental backend refactoring

### API Endpoints

#### System

* `GET /health`
* `GET /stats`

#### Snapshot APIs

* `GET /snapshots`
* `GET /snapshot/:id`
* `GET /snapshot-count/:file`
* `GET /snapshots/file/:name`
* `POST /snapshot`
* `DELETE /snapshot/:id`

#### History APIs

* `GET /latest/:file`
* `GET /snapshot-oldest/:file`
* `GET /history/:file`
* `GET /history-full/:file`
* `GET /snapshot-latest-id/:file`
* `GET /snapshot-first-id/:file`

#### Recovery APIs

* `POST /restore/:id`
* `POST /recover/:file`

#### Utility APIs

* `GET /files`
* `GET /files/:name/exists`
* `GET /diff/:id1/:id2`

## Storage

* JSON-based snapshot persistence
* SHA256 content hashing
* In-memory snapshot indexing
* Duplicate snapshot filtering
* Content-based version tracking

## Project Structure

```text
internal/
├── api/
│   ├── server.go
│   ├── stats_handlers.go
│   └── history_handlers.go
├── diff/
├── models/
├── storage/
└── vfs/
```

## Planned Features

* Metadata indexing
* Command-line interface
* Frontend dashboard
* Folder rollback
* Recovery logging
* Storage optimization metrics
* SQLite-backed storage
* Snapshot export/import
* Snapshot search
* Multi-directory monitoring

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
* Automated testing
* CI/CD workflows
* Systems programming in Go

## Tech Stack

* Go
* Gin
* fsnotify
* JSON Storage
* GitHub Actions
* SQLite (planned)
* Linux
* File I/O
* Git

## Status

Currently under active development.

### Recent Milestones

* Automatic snapshot watcher integration
* SHA256 hashing implementation
* Duplicate snapshot prevention
* Snapshot comparison API
* Recovery and restore functionality
* Automated test suite
* GitHub Actions CI pipeline
* API route modularization
* Server architecture cleanup

## Contributors

### Siddharth S Menon

* Snapshot management APIs
* Recovery APIs
* Statistics endpoints
* History and lookup endpoints
* Snapshot comparison APIs
* Deduplication implementation
* Automated testing
* GitHub Actions integration
* API refactoring and modularization

### Tanisha Khoria

* Snapshot engine
* Filesystem watcher
* Persistence layer
* Core snapshot infrastructure
* Storage architecture
* Hashing integration
* Snapshot automation
