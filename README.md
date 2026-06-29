# rewindfs

🚧 Work in Progress

rewindfs is a Git-inspired virtual filesystem project focused on snapshot-based file versioning, rollback, recovery, history tracking, deduplication, and persistent storage.

The project explores how modern version-control systems and snapshot-oriented filesystems manage historical file state through automatic snapshot creation, content hashing, recovery workflows, REST APIs, and persistent metadata storage.

---

## Goals

* Track file changes over time
* Create snapshots automatically
* Persist snapshots across application restarts
* Restore previous versions of files
* Recover deleted files
* Compare changes between snapshots
* Reduce duplicate storage using content hashing
* Explore filesystem and storage design concepts
* Build maintainable backend architecture
* Implement automated testing and CI/CD
* Build a web dashboard for snapshot visualization

---

## Current Progress

* [x] Repository setup
* [x] Filesystem watcher
* [x] Automatic snapshot generation
* [x] Snapshot persistence
* [x] Migration from JSON storage to SQLite
* [x] SQLite database initialization
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
* [x] SQLite persistence tests
* [x] Frontend dashboard
* [x] Dashboard statistics panel
* [x] Dashboard file listing
* [ ] Snapshot history dashboard
* [ ] Snapshot restore dashboard actions
* [ ] Snapshot delete dashboard actions
* [ ] Metadata indexing
* [ ] CLI interface
* [ ] Recovery testing
* [ ] Folder rollback
* [ ] Snapshot search
* [ ] Multi-directory monitoring

---

## Implemented Features

### Snapshot Management

* Automatic file monitoring using fsnotify
* Snapshot creation on file modification
* Persistent snapshot storage using SQLite
* Timestamped snapshots
* Snapshot listing
* Snapshot lookup by ID
* Latest snapshot retrieval
* Oldest snapshot retrieval
* Snapshot deletion
* Snapshot counting per file
* File existence checks
* File history tracking

---

### Recovery

* Restore a file from a specific snapshot
* Recover deleted files using the latest snapshot
* File rollback infrastructure

---

### Diff and History

* Compare two snapshots
* View snapshot history metadata
* View complete snapshot history
* Track historical versions of files
* Retrieve first and latest snapshot IDs

---

### Storage Optimization

* SHA256 content hashing
* Hash-based snapshot identification
* Duplicate snapshot prevention
* Unique hash statistics
* Content-based deduplication
* SQLite indexes for faster lookup

---

### Testing & Quality

* Unit tests for hashing
* Unit tests for snapshot management
* Unit tests for deduplication
* Unit tests for diff functionality
* Unit tests for SQLite persistence
* GitHub Actions automated test execution
* Automated validation on every push

---

### API Architecture

* Route modularization
* Dedicated stats handler module
* Dedicated history handler module
* Dedicated lookup handler module
* Reduced server.go complexity
* Incremental backend refactoring
* RESTful API organization

---

### Dashboard

* Statistics dashboard
* Total snapshot count
* Tracked file count
* Unique hash count
* File listing panel
* Automatic data refresh

---

## API Endpoints

### System

* `GET /`
* `GET /health`
* `GET /stats`

---

### Snapshot APIs

* `GET /snapshots`
* `GET /snapshot/:id`
* `GET /snapshot-count/:file`
* `GET /snapshots/file/:name`
* `POST /snapshot`
* `DELETE /snapshot/:id`

---

### History APIs

* `GET /latest/:file`
* `GET /snapshot-oldest/:file`
* `GET /history/:file`
* `GET /history-full/:file`
* `GET /snapshot-latest-id/:file`
* `GET /snapshot-first-id/:file`

---

### Recovery APIs

* `POST /restore/:id`
* `POST /recover/:file`

---

### Utility APIs

* `GET /files`
* `GET /files/:name/exists`
* `GET /diff/:id1/:id2`

---

## Storage

### SQLite Persistence

Snapshots are stored in SQLite instead of JSON files.

#### Snapshot Schema

```sql
CREATE TABLE snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file TEXT NOT NULL,
    content TEXT NOT NULL,
    hash TEXT NOT NULL,
    created_at DATETIME NOT NULL
);
```

#### Indexes

```sql
CREATE INDEX idx_snapshots_file
ON snapshots(file);

CREATE INDEX idx_snapshots_hash
ON snapshots(hash);
```

---

### Snapshot Fields

* id
* file
* content
* hash
* created_at

---

## Example Usage

### Create Snapshot

```bash
curl -X POST http://localhost:8080/snapshot \
-H "Content-Type: application/json" \
-d '{
  "file":"test.txt",
  "content":"hello world"
}'
```

---

### Get All Snapshots

```bash
curl http://localhost:8080/snapshots
```

---

### Restore Snapshot

```bash
curl -X POST http://localhost:8080/restore/1
```

---

### Recover Deleted File

```bash
curl -X POST http://localhost:8080/recover/test.txt
```

---

## Project Structure

```text
rewindfs/
├── frontend/
│   └── index.html
├── internal/
│   ├── api/
│   │   ├── server.go
│   │   ├── history_handlers.go
│   │   ├── lookup_handlers.go
│   │   ├── snapshots.go
│   │   └── stats_handlers.go
│   ├── diff/
│   ├── models/
│   ├── storage/
│   │   ├── sqlite.go
│   │   ├── sqlite_test.go
│   │   ├── hash.go
│   │   └── hash_test.go
│   └── vfs/
│       └── watcher.go
├── rewindfs.db
├── main.go
├── go.mod
├── go.sum
└── README.md
```

---

## Planned Features

* Snapshot history dashboard
* Snapshot restore buttons
* Snapshot delete buttons
* Metadata indexing
* Command-line interface
* Folder rollback
* Recovery logging
* Storage optimization metrics
* Snapshot export/import
* Snapshot search
* Multi-directory monitoring

---

## Project Focus

This project is being built to gain practical experience with:

* Filesystem architecture
* Version control concepts
* Snapshot-based storage systems
* Database-backed persistence
* Recovery and rollback mechanisms
* Metadata management
* Content hashing
* Deduplication techniques
* Backend API development
* REST API design
* Automated testing
* CI/CD workflows
* Systems programming in Go

---

## Tech Stack

* Go
* Gin
* SQLite
* fsnotify
* SHA256
* REST APIs
* HTML
* CSS
* JavaScript
* GitHub Actions
* Linux
* File I/O
* Git

---

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
* Migration from JSON storage to SQLite
* SQLite persistence testing
* Dashboard statistics and file listing

---

## Contributors

### Siddharth S Menon

* Snapshot management APIs
* Recovery APIs
* Statistics endpoints
* History and lookup endpoints
* Snapshot comparison APIs
* Deduplication implementation
* SQLite integration
* SQLite persistence testing
* Frontend dashboard
* Automated testing
* GitHub Actions integration
* API refactoring and modularization

### Tanisha Khoria

* Snapshot engine
* Filesystem watcher
* Core snapshot infrastructure
* Persistence architecture
* Storage design
* Hashing integration
* Snapshot automation
* File monitoring workflows
