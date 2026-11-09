# RewindFS CLI

The command-line interface is a thin client of the running RewindFS API. It does not implement snapshot storage or filesystem watching itself.

Start the main application first:

```bash
go run .
```

Then run commands from another terminal:

```bash
go run ./cmd/rewindfs status
go run ./cmd/rewindfs files
go run ./cmd/rewindfs snapshots
go run ./cmd/rewindfs history src/main.go
go run ./cmd/rewindfs diff 12 18
go run ./cmd/rewindfs restore 12
go run ./cmd/rewindfs recover src/main.go
go run ./cmd/rewindfs delete 12
```

Restore, recovery, and deletion ask for confirmation. Restore and recovery accept `--yes` for deliberate scripted usage:

```bash
go run ./cmd/rewindfs restore 12 --yes
go run ./cmd/rewindfs recover src/main.go --yes
```

Set `REWINDFS_API` to point the CLI at another RewindFS API address. The default is `http://127.0.0.1:8080`.

```bash
REWINDFS_API="http://127.0.0.1:8080" go run ./cmd/rewindfs status
```
