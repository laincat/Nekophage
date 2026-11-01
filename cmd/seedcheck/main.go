package main

// Restores a seed file the way a client does at launch and reports what came
// back, so a seed that no client can read is caught before a simulator run.
//
//	seedcheck SEED.json

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/plugin"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: seedcheck SEED.json")
		os.Exit(2)
	}
	body, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var recs []plugin.Record
	if err := json.Unmarshal(body, &recs); err != nil {
		fmt.Fprintf(os.Stderr, "seed is not a record list: %v\n", err)
		os.Exit(1)
	}
	dir, err := os.MkdirTemp("", "seedcheck")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	c, err := core.Open(core.Options{DataDir: dir, AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		panic(err)
	}
	defer c.Close()
	failed := c.Restore(context.Background(), recs)
	for _, r := range c.Plugins() {
		fmt.Printf("restored %s (%s) enabled=%v body=%d\n", r.Key, r.Name, r.Enabled, len(r.Body))
	}
	if len(failed) > 0 {
		for k, err := range failed {
			fmt.Printf("FAILED %s: %v\n", k, err)
		}
		os.Exit(1)
	}
	if len(c.Plugins()) == 0 {
		fmt.Fprintln(os.Stderr, "the seed restored nothing")
		os.Exit(1)
	}
}
