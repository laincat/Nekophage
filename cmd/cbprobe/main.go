package main

// Runs a source's setting callback the way the settings screen does, so a callback
// that never answers can be told apart from a screen that never asks.
//
//	go run ./cmd/cbprobe SCRIPT.js KEY SETTING [SECONDS]
//
// It prints how long the call took and what came back; a callback that reaches an
// outside host depends on that host, and this separates that from a client fault.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: cbprobe SCRIPT.js KEY SETTING [SECONDS]")
		os.Exit(2)
	}
	script, key, setting := os.Args[1], os.Args[2], os.Args[3]
	bound := 120
	if len(os.Args) > 4 {
		if n, err := strconv.Atoi(os.Args[4]); err == nil {
			bound = n
		}
	}

	dir, err := os.MkdirTemp("", "cbprobe")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	body, err := os.ReadFile(script)
	if err != nil {
		fmt.Println("FAIL read:", err)
		os.Exit(1)
	}
	c, err := core.Open(core.Options{DataDir: dir, AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	ctx := context.Background()
	if _, err := c.Install(ctx, string(body), "file", filepath.Base(script)); err != nil {
		fmt.Println("FAIL install:", err)
		os.Exit(1)
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(bound)*time.Second)
	defer cancel()
	start := time.Now()
	answer, err := c.CallSettingCallback(runCtx, key, setting)
	elapsed := time.Since(start)

	switch {
	case err == nil:
		fmt.Printf("ANSWER after %s: %q\n", elapsed.Round(time.Millisecond), answer)
	case errors.Is(err, context.DeadlineExceeded):
		fmt.Printf("TIMEOUT after %s: the callback never answered\n", elapsed.Round(time.Millisecond))
		os.Exit(3)
	default:
		fmt.Printf("ERROR after %s: %v\n", elapsed.Round(time.Millisecond), err)
	}
}
