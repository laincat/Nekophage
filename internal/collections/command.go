package collections

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/store"
)

// RunCommand is used by the desktop binary without opening a window. Following
// a source only caches its catalogue; it does not run or install its entries.
func RunCommand(ctx context.Context, dataDir, action string, args []string, out io.Writer, userAgent string) error {
	want := 1
	switch action {
	case "list":
		want = 0
	case "follow", "refresh", "unfollow":
	case "install":
		want = 2
	default:
		return errors.New("合集命令须为 list、follow、refresh、unfollow 或 install")
	}
	if len(args) != want {
		return fmt.Errorf("合集命令 %s 需要 %d 个参数", action, want)
	}
	s, err := store.Open(dataDir)
	if err != nil {
		return err
	}
	tr := core.NewTransport(nil, userAgent, 0)
	manager, err := Open(s, tr)
	if err != nil {
		return err
	}
	defer manager.Close()
	if action == "install" {
		return installEntry(ctx, manager, args[0], args[1], dataDir, userAgent, out)
	}
	switch action {
	case "follow":
		err = manager.Follow(ctx, args[0])
	case "refresh":
		err = manager.Refresh(ctx, args[0])
	case "unfollow":
		err = manager.Unfollow(args[0])
	}
	if err != nil {
		return err
	}
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	// These script hosts already exist. Theme/module/service and embedded
	// Magpie-provider runtimes are not advertised by the desktop client.
	views, err := manager.List(platform, []string{"comic-source.js", "video-source.js"})
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(views)
}
