package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"

	"rewindfs/internal/client"
	"rewindfs/internal/models"
)

const defaultAPI = "http://127.0.0.1:8080"

func main() {
	baseURL := strings.TrimSpace(os.Getenv("REWINDFS_API"))
	if baseURL == "" {
		baseURL = defaultAPI
	}
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr, baseURL))
}

func run(ctx context.Context, args []string, input io.Reader, output, errorsOut io.Writer, baseURL string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printUsage(output)
		return 0
	}

	api, err := client.New(baseURL, nil)
	if err != nil {
		fmt.Fprintln(errorsOut, err)
		return 2
	}

	fail := func(err error) int {
		fmt.Fprintln(errorsOut, err)
		return 1
	}

	switch args[0] {
	case "status":
		if len(args) != 1 {
			printUsage(errorsOut)
			return 2
		}
		var health struct {
			Status string `json:"status"`
		}
		var stats struct {
			TotalSnapshots int `json:"total_snapshots"`
			TrackedFiles   int `json:"tracked_files"`
			UniqueHashes   int `json:"unique_hashes"`
		}
		if err := api.Get(ctx, "/health", &health); err != nil {
			return fail(err)
		}
		if err := api.Get(ctx, "/stats", &stats); err != nil {
			return fail(err)
		}
		fmt.Fprintf(output, "rewindfs: %s\nsnapshots: %d\ntracked files: %d\nunique hashes: %d\n",
			health.Status, stats.TotalSnapshots, stats.TrackedFiles, stats.UniqueHashes)
		return 0

	case "files":
		if len(args) != 1 {
			printUsage(errorsOut)
			return 2
		}
		var result struct {
			Files []string `json:"files"`
		}
		if err := api.Get(ctx, "/files", &result); err != nil {
			return fail(err)
		}
		if len(result.Files) == 0 {
			fmt.Fprintln(output, "no tracked files")
			return 0
		}
		for _, file := range result.Files {
			fmt.Fprintln(output, file)
		}
		return 0

	case "snapshots":
		if len(args) != 1 {
			printUsage(errorsOut)
			return 2
		}
		var items []models.Snapshot
		if err := api.Get(ctx, "/snapshots", &items); err != nil {
			return fail(err)
		}
		return printJSON(output, items, errorsOut)

	case "history":
		if len(args) != 2 {
			printUsage(errorsOut)
			return 2
		}
		path := "/history?file=" + url.QueryEscape(args[1])
		var history []map[string]any
		if err := api.Get(ctx, path, &history); err != nil {
			return fail(err)
		}
		return printJSON(output, history, errorsOut)

	case "diff":
		if len(args) != 3 || !positiveInteger(args[1]) || !positiveInteger(args[2]) {
			printUsage(errorsOut)
			return 2
		}
		var result struct {
			Snapshot1 int    `json:"snapshot1"`
			Snapshot2 int    `json:"snapshot2"`
			File1     string `json:"file1"`
			File2     string `json:"file2"`
			Diff      string `json:"diff"`
		}
		if err := api.Get(ctx, "/diff/"+args[1]+"/"+args[2], &result); err != nil {
			return fail(err)
		}
		fmt.Fprintf(output, "snapshot %d (%s) -> snapshot %d (%s)\n",
			result.Snapshot1, result.File1, result.Snapshot2, result.File2)
		fmt.Fprint(output, result.Diff)
		return 0

	case "restore":
		if len(args) < 2 || len(args) > 3 || !positiveInteger(args[1]) ||
			(len(args) == 3 && args[2] != "--yes") {
			printUsage(errorsOut)
			return 2
		}
		if len(args) != 3 && !confirm(input, output, "restore this snapshot and overwrite the current file? [y/N] ") {
			fmt.Fprintln(output, "cancelled")
			return 0
		}
		if err := api.Post(ctx, "/restore/"+args[1], nil, nil); err != nil {
			return fail(err)
		}
		fmt.Fprintln(output, "snapshot restored")
		return 0

	case "recover":
		if len(args) < 2 || len(args) > 3 ||
			(len(args) == 3 && args[2] != "--yes") {
			printUsage(errorsOut)
			return 2
		}
		if len(args) != 3 && !confirm(input, output, "recover the latest snapshot and write it to the workspace? [y/N] ") {
			fmt.Fprintln(output, "cancelled")
			return 0
		}
		path := "/recover?file=" + url.QueryEscape(args[1])
		if err := api.Post(ctx, path, nil, nil); err != nil {
			return fail(err)
		}
		fmt.Fprintln(output, "file recovered")
		return 0

	case "delete":
		if len(args) != 2 || !positiveInteger(args[1]) {
			printUsage(errorsOut)
			return 2
		}
		if !confirm(input, output, "delete this snapshot permanently? [y/N] ") {
			fmt.Fprintln(output, "cancelled")
			return 0
		}
		if err := api.Delete(ctx, "/snapshot/"+args[1]); err != nil {
			return fail(err)
		}
		fmt.Fprintln(output, "snapshot deleted")
		return 0

	default:
		fmt.Fprintf(errorsOut, "unknown command: %s\n", args[0])
		printUsage(errorsOut)
		return 2
	}
}

func positiveInteger(value string) bool {
	number, err := strconv.Atoi(value)
	return err == nil && number > 0
}

func confirm(input io.Reader, output io.Writer, prompt string) bool {
	fmt.Fprint(output, prompt)
	answer, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && answer == "" {
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func printJSON(output io.Writer, value any, errorsOut io.Writer) int {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(errorsOut, err)
		return 1
	}
	return 0
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, `rewindfs - local filesystem snapshots

usage:
  rewindfs status
  rewindfs files
  rewindfs snapshots
  rewindfs history <file>
  rewindfs diff <snapshot-id-1> <snapshot-id-2>
  rewindfs restore <snapshot-id> [--yes]
  rewindfs recover <file> [--yes]
  rewindfs delete <snapshot-id>

set REWINDFS_API to use another local API address.
restore, recover, and delete ask before changing workspace data.`)
}
