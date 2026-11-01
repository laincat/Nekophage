package main

// Runs a source through the reading path without any interface: install, search,
// open one comic, read its tags, prove a tag tap reaches the source's own rule.
//
//	go run ./cmd/comicprobe SCRIPT.js [KEYWORD]
//
// It exists because the interface tests need a source they control: the ones they
// seed are other people's scripts on other people's sites, so a failure there says
// "the site changed" as often as it says "the client broke". This checks the same
// contract the client uses — the same Core call, the same JSON — before that source
// is trusted to stand in for one.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/laincat/nekophage/internal/core"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: comicprobe SCRIPT.js [KEYWORD]")
		os.Exit(2)
	}
	script := os.Args[1]
	keyword := "probe"
	if len(os.Args) > 2 {
		keyword = os.Args[2]
	}

	body, err := os.ReadFile(script)
	if err != nil {
		fmt.Println("FAIL read:", err)
		os.Exit(1)
	}
	dir, err := os.MkdirTemp("", "comicprobe")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	c, err := core.Open(core.Options{DataDir: dir, AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		panic(err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	rec, err := c.Install(ctx, string(body), "file", filepath.Base(script))
	if err != nil {
		fmt.Println("FAIL install:", err)
		os.Exit(1)
	}

	page, err := c.Search(ctx, rec.Key, keyword, nil, 1)
	if err != nil {
		fmt.Println("FAIL search:", err)
		os.Exit(1)
	}
	if len(page.Comics) == 0 {
		fmt.Println("FAIL search returned no comics, so nothing after this is reachable")
		os.Exit(1)
	}
	first := page.Comics[0]
	fmt.Printf("search: %d comic(s), first id %q title %q\n", len(page.Comics), first.ID, first.Title)

	details, err := c.Details(ctx, rec.Key, first.ID)
	if err != nil {
		fmt.Println("FAIL info:", err)
		os.Exit(1)
	}
	fmt.Printf("info: title %q, tags %d, chapters %d\n", details.Title, len(details.Tags), len(details.Chapters))

	if len(details.Tags) == 0 {
		fmt.Println("FAIL no tags, so the tag row the interface draws is empty")
		os.Exit(1)
	}

	// Every group, not just the first: a source answers null for the namespaces it
	// does not handle, so pressing one group says nothing about the others. A
	// source may map one namespace to a search and another to a category, and both
	// branches of the client's tag handling deserve a run.
	handled := 0
	for _, tagGroup := range details.Tags {
		if len(tagGroup.Values) == 0 {
			continue
		}
		value := tagGroup.Values[0]
		action, err := c.ClickTag(ctx, rec.Key, tagGroup.Name, value)
		switch {
		case err != nil:
			fmt.Printf("  %-10s %-14q -> %v\n", tagGroup.Name, value, err)
		case action.Action == "":
			fmt.Printf("  %-10s %-14q -> not handled by the source\n", tagGroup.Name, value)
		default:
			handled++
			fmt.Printf("  %-10s %-14q -> %s %q\n", tagGroup.Name, value, action.Action, action.Keyword)
		}
	}
	if handled == 0 {
		fmt.Println("FAIL no tag group was handled, so a tap would do nothing anywhere")
		os.Exit(1)
	}
	fmt.Println("PASS tag taps reach the source's rule:", handled, "of", len(details.Tags), "group(s)")
}
