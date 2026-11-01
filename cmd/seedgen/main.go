package main

// Writes the store file a client restores its sources from, so a simulator can be
// seeded without touching the interface: the client reads this file at launch and
// the sources are there, which is what a UI test needs to reach a settings screen
// or a detail page.
//
//	seedgen OUT.json SCRIPT...
//
// The record format is Core's own, so the file is what `Records()` returns on any
// platform: the same bytes a client would have written after installing by hand.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/plugin"
)

// normaliseNewlines makes CRLF and lone CR into LF, so the bytes a seed carries
// depend on the fixture and not on where the seed was built.
func normaliseNewlines(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

// pluginType reads the kind from the script, the way Core's loader does: the
// seed builder is given files, not types.
func pluginType(script string) plugin.Type {
	if strings.Contains(script, "extends VideoSource") {
		return plugin.TypeVideoJS
	}
	return plugin.TypeComic
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: seedgen OUT.json SCRIPT...")
		os.Exit(2)
	}
	dir, err := os.MkdirTemp("", "seed")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	c, err := core.Open(core.Options{DataDir: dir, AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		panic(err)
	}
	defer c.Close()
	ctx := context.Background()
	for _, path := range os.Args[2:] {
		body, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		// Line endings are normalised so a seed built on one platform has the same
		// bytes as one built on another. Without this the same fixture produced a
		// different md5 per host, which turns "the seed matches the tree" into a
		// question nobody can answer from a hash.
		text := normaliseNewlines(string(body))
		rec, err := c.InstallTyped(ctx, pluginType(text), text, "file", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", filepath.Base(path), err)
			os.Exit(1)
		}
		fmt.Printf("installed %s (%s) %d bytes\n", rec.Key, rec.Name, len(rec.Body))
	}
	out, err := json.Marshal(c.Records())
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[1], out, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %s (%d bytes)\n", os.Args[1], len(out))
}
