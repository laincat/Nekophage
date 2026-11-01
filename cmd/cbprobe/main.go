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
	"github.com/laincat/nekophage/internal/ui"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("usage: cbprobe SCRIPT.js KEY SETTING|--all [SECONDS]")
		os.Exit(2)
	}
	script, key, setting := os.Args[1], os.Args[2], os.Args[3]
	bound := 120
	if len(os.Args) > 4 {
		if n, err := strconv.Atoi(os.Args[4]); err == nil {
			bound = n
		}
	}
	all := setting == "--all"

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
	// A callback that asks the source's own dialog cannot finish without someone
	// answering it, so this takes the first action, the way a person pressing the
	// only button would.
	handler := ui.NewJSHandler()
	handler.SetAutoConfirm(true)
	c, err := core.Open(core.Options{DataDir: dir, AppVersion: "1.6.3", Locale: "zh_CN", UI: handler})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	ctx := context.Background()
	rec, err := c.Install(ctx, string(body), "file", filepath.Base(script))
	if err != nil {
		fmt.Println("FAIL install:", err)
		os.Exit(1)
	}
	if key == "-" {
		// The script's own key, so a caller need not repeat what the file says.
		key = rec.Key
	}

	if all {
		// The source itself says which settings are buttons; guessing from the
		// file text picks up lookalikes that are not callbacks at all.
		list, err := c.Settings(ctx, key)
		if err != nil {
			fmt.Println("FAIL settings:", err)
			os.Exit(1)
		}
		found := false
		for _, item := range list {
			if item.Type != "callback" {
				continue
			}
			found = true
			runOne(c, key, item.Key, bound)
		}
		if !found {
			fmt.Println("NONE: the source declares no callback")
		}
		return
	}

	runOne(c, key, setting, bound)
}

func runOne(c *core.Core, key, setting string, bound int) {
	ctx := context.Background()
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(bound)*time.Second)
	defer cancel()
	start := time.Now()
	answer, err := c.CallSettingCallback(runCtx, key, setting)
	elapsed := time.Since(start)

	// The setting is part of every line: one source declares several callbacks and
	// the output has to say which one answered.
	switch {
	case err == nil:
		fmt.Printf("ANSWER  %s.%s  %s  %q\n", key, setting, elapsed.Round(time.Millisecond), answer)
	case errors.Is(err, context.DeadlineExceeded):
		fmt.Printf("TIMEOUT %s.%s  %s  never answered\n", key, setting, elapsed.Round(time.Millisecond))
		os.Exit(3)
	default:
		fmt.Printf("ERROR   %s.%s  %s  %v\n", key, setting, elapsed.Round(time.Millisecond), err)
	}
}
