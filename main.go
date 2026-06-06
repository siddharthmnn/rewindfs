package main

import (
        "rewindfs/internal/api"
        "rewindfs/internal/vfs"
)

func main() {
        go vfs.StartWatcher(".")
        api.StartServer()
}
