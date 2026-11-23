package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/veneracat-core/comic/js"
)

type probeUI struct{}

func (probeUI) ShowMessage(text string)                                              { fmt.Println(text) }
func (probeUI) LaunchURL(string)                                                     {}
func (probeUI) ShowLoading(func()) int                                               { return 0 }
func (probeUI) CancelLoading(int)                                                    {}
func (probeUI) ShowInputDialog(string, string, func(string) string) (*string, error) { return nil, nil }
func (probeUI) ShowSelectDialog(string, []string, *int) (*int, error)                { return nil, nil }
func (probeUI) ShowDialog(_, _ string, actions []js.DialogAction) error {
	if len(actions) == 0 {
		return nil
	}
	return actions[0].Invoke()
}

func main() {
	if len(os.Args) < 3 || (os.Args[1] != "callbacks" && os.Args[1] != "tags") {
		fmt.Fprintln(os.Stderr, "usage: sourcecheck callbacks SCRIPT.js [SETTING|--all] [SECONDS] | tags SCRIPT.js [KEYWORD]")
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2], os.Args[3:]); err != nil {
		fmt.Println("FAIL", err)
		if errors.Is(err, context.DeadlineExceeded) {
			os.Exit(3)
		}
		os.Exit(1)
	}
}

func run(mode, script string, args []string) error {
	body, err := os.ReadFile(script)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	dir, err := os.MkdirTemp("", "nekophage-sourcecheck-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	c, err := core.Open(core.Options{DataDir: dir, UI: probeUI{}})
	if err != nil {
		return err
	}
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	rec, err := c.Install(ctx, string(body), script, "")
	if err != nil {
		return fmt.Errorf("install: %w", err)
	}
	if mode == "callbacks" {
		return callbacks(c, rec.Key, args)
	}
	query := "probe"
	if len(args) > 0 {
		query = args[0]
	}
	return tags(ctx, c, rec.Key, query)
}

func callbacks(c *core.Core, key string, args []string) error {
	setting, bound := "--all", 30*time.Second
	if len(args) > 0 {
		setting = args[0]
	}
	if len(args) > 1 {
		seconds, err := strconv.Atoi(args[1])
		if err != nil || seconds <= 0 {
			return fmt.Errorf("invalid callback timeout %q", args[1])
		}
		bound = time.Duration(seconds) * time.Second
	}
	names := []string{setting}
	if setting == "--all" {
		ctx, cancel := context.WithTimeout(context.Background(), bound)
		defer cancel()
		declared, err := c.Settings(ctx, key)
		if err != nil {
			return fmt.Errorf("settings: %w", err)
		}
		names = nil
		for _, entry := range declared {
			if entry.Type == "callback" {
				names = append(names, entry.Key)
			}
		}
	}
	if len(names) == 0 {
		fmt.Println("NONE: the source declares no callback")
	}
	for _, name := range names {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), bound)
		answer, err := c.CallSettingCallback(ctx, key, name)
		cancel()
		elapsed := time.Since(start).Round(time.Millisecond)
		switch {
		case err == nil:
			fmt.Printf("ANSWER %s.%s %s %q\n", key, name, elapsed, answer)
		case errors.Is(err, context.DeadlineExceeded):
			return fmt.Errorf("TIMEOUT %s.%s %s: %w", key, name, elapsed, err)
		default:
			fmt.Printf("ERROR %s.%s %s %v\n", key, name, elapsed, err)
		}
	}
	return nil
}

func tags(ctx context.Context, c *core.Core, key, query string) error {
	options, err := c.DefaultOptions(ctx, key)
	if err != nil {
		return fmt.Errorf("search options: %w", err)
	}
	page, err := c.Search(ctx, key, query, options, 1)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	if len(page.Comics) == 0 {
		return errors.New("search returned no comics")
	}
	first := page.Comics[0]
	fmt.Printf("search: %d comic(s), first id %q title %q\n", len(page.Comics), first.ID, first.Title)
	details, err := c.Details(ctx, key, first.ID)
	if err != nil {
		return fmt.Errorf("info: %w", err)
	}
	fmt.Printf("info: title %q, tags %d, chapters %d\n", details.Title, len(details.Tags), len(details.Chapters))
	if len(details.Tags) == 0 {
		return errors.New("no tag groups")
	}
	handled := 0
	for _, group := range details.Tags {
		if len(group.Values) == 0 {
			continue
		}
		value := group.Values[0]
		action, err := c.ClickTag(ctx, key, group.Name, value)
		switch {
		case errors.Is(err, core.ErrNoTagAction) || (err == nil && action.Action == ""):
			fmt.Printf("  %s %q: not handled\n", group.Name, value)
		case err != nil:
			fmt.Printf("  %s %q: %v\n", group.Name, value, err)
		default:
			handled++
			fmt.Printf("  %s %q -> %s %q\n", group.Name, value, action.Action, action.Keyword)
		}
	}
	if handled == 0 {
		return errors.New("no tag group was handled")
	}
	fmt.Printf("PASS tags handled %d/%d\n", handled, len(details.Tags))
	return nil
}
