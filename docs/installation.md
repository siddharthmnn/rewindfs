# Installing and running RewindFS

RewindFS currently runs as a local application. The HTTP API binds to
`127.0.0.1:8080` and checks local Host and Origin values. It is not designed
to be exposed directly to a network.

## Requirements

- Go 1.22 or newer (CI and release builds currently use Go 1.24).
- A C compiler and CGO for the server, because SQLite uses
  `github.com/mattn/go-sqlite3`.
- Git, if building from source.

On Ubuntu or WSL:

```bash
sudo apt update
sudo apt install -y golang-go gcc build-essential git
```

Use a Go toolchain matching the version used by the repository if the distro
package is older than the module's supported toolchain.

## Build from source

Clone the repository and run the test suite:

```bash
git clone https://github.com/siddharthmnn/rewindfs.git
cd rewindfs
go test ./...
go test -race ./...
go vet ./...
```

Build and start the server:

```bash
CGO_ENABLED=1 go build -trimpath -o rewindfs-server .
./rewindfs-server
```

The server watches the current directory by default. It creates
`rewindfs.db` there unless a separate database path is configured.

## Configuration

| Variable | Default | Description |
|---|---|---|
| `REWINDFS_ROOT` | `.` | Workspace directory to watch and restore into |
| `REWINDFS_DB_PATH` | `rewindfs.db` | SQLite database path |
| `REWINDFS_IGNORE` | empty | Comma-separated ignore patterns, in addition to built-in exclusions |
| `REWINDFS_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error` |
| `REWINDFS_LOG_FORMAT` | `text` | `text` or `json` structured log output |

Example:

```bash
REWINDFS_ROOT="$HOME/projects/demo" \
REWINDFS_DB_PATH="$HOME/.local/share/rewindfs/demo.db" \
REWINDFS_IGNORE="*.log,build/,vendor/" \
REWINDFS_LOG_LEVEL=debug \
REWINDFS_LOG_FORMAT=json \
./rewindfs-server
```

Use a database location outside the watched workspace when practical. The
watcher already excludes SQLite files, but keeping application state separate
makes the workspace easier to reason about and back up.

## CLI

The CLI is a separate binary and talks to the running local API. It does not
start the watcher or open the SQLite database itself.

Build it:

```bash
CGO_ENABLED=0 go build -trimpath -o rewindfs ./cmd/rewindfs
```

With the server running in another terminal:

```bash
./rewindfs status
./rewindfs files
./rewindfs history src/main.go
./rewindfs diff 1 2
./rewindfs restore 1
```

Set `REWINDFS_API` if the API uses a different local address. The default is
`http://127.0.0.1:8080`.

## Health and readiness

- `GET /health` is a liveness check for the HTTP process.
- `GET /ready` checks that SQLite responds and the configured workspace is
  still available.

A readiness failure returns HTTP 503. Neither endpoint exposes database
credentials, local paths, or underlying OS errors.

## Backups and recovery

Use the CLI archive commands to export and import snapshot history:

```bash
./rewindfs export ./rewindfs-backup.zip
./rewindfs import ./rewindfs-backup.zip
```

An import restores history into SQLite; it does not write imported file contents into
the workspace. Use `restore` or `recover` when you intend to write a specific
snapshot back to disk. These operations ask for confirmation unless an
explicit supported `--yes` option is supplied.

Keep archive backups outside the workspace being watched. Treat them as
sensitive: snapshots can contain the full contents of files that were tracked.

## Release artifacts

Tagged releases publish:

- Linux amd64 server and CLI binaries.
- macOS arm64 server and CLI binaries.
- Windows amd64 CLI binary.
- SHA-256 checksums for the published assets.

The server currently requires CGO. The Windows release is CLI-only because the
SQLite server build needs a Windows C toolchain; the CLI itself does not need
CGO.
