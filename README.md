# rewindfs

🚧 Work in Progress

rewindfs is a Git-inspired virtual filesystem project focused on snapshotting, rollback, file recovery, and versioned file history.

The project explores how modern version-control systems and snapshot-based filesystems manage file state, recovery, and historical versions through a simplified implementation built in Go.

## Goals

* Track file changes over time
* Create snapshots automatically
* Restore previous versions of files
* Recover deleted files
* Compare changes between snapshots
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
* [x] SHA256 hashing utility
* [ ] Snapshot diff engine
* [ ] Metadata indexing
* [ ] CLI interface
* [ ] Deduplication
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
* Snapshot deletion

### Recovery

* Restore a file from a specific snapshot
* Recover deleted files using the most recent snapshot

### API Endpoints

* `GET /health`
* `GET /stats`
* `GET /snapshots`
* `GET /snapshot/:id`
* `GET /latest/:file`
* `POST /snapshot`
* `POST /restore/:id`
* `POST /recover/:file`
* `DELETE /snapshot/:id`

### Storage

* JSON-based snapshot persistence
* SHA256 hashing utility for future deduplication support

## Planned Features

* Snapshot diff viewer
* Snapshot-to-snapshot comparison
* Metadata indexing
* Content deduplication
* Command-line interface
* Folder rollback
* Recovery logging
* Storage optimization

## Project Focus

This project is being built to gain practical experience with:

* Filesystem architecture
* Version control concepts
* Snapshot-based storage systems
* Recovery and rollback mechanisms
* Metadata management
* Hashing and deduplication
* Backend API development
* Systems programming in Go

## Tech Stack

* Go
* Gin
* SQLite (planned)
* Linux
* File I/O
* Git

## Status

Currently under active development.

### Recent Milestones

* Persistent snapshot storage
* Snapshot restore functionality
* Snapshot statistics endpoint
* Snapshot deletion endpoint
* Deleted file recovery endpoint
* Latest snapshot lookup endpoint
* SHA256 hashing utility

## Contributors

* Siddharth S Menon

  * Snapshot management APIs
  * Recovery APIs
  * Statistics and lookup endpoints

* Tanisha Khoria

  * Snapshot engine
  * Persistence layer
  * Hashing utilities
  * Core snapshot infrastructure
