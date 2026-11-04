package collections

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/nekophage/internal/services"
	"github.com/laincat/veneracat-core/plugin"
)

// installEntry installs one collection artifact after checking that this client
// supports the entry and that the downloaded bytes match the declared digest.
//
// The entry's declared key is an identity claim: a script that registers a
// different key is uninstalled again rather than silently kept, because the
// collection and the installed source would then disagree about what was run.
func installEntry(ctx context.Context, m *Manager, collectionID, entryID, dataDir, userAgent string, out io.Writer) error {
	c, err := core.Open(core.Options{DataDir: dataDir, AppVersion: "1.6.3", Locale: "zh_CN"})
	if err != nil {
		return err
	}
	defer c.Close()
	lib := library.New(c)
	if _, err := lib.Restore(ctx); err != nil {
		return err
	}
	result, err := InstallEntry(ctx, m, collectionID, entryID, dataDir, c, lib)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(result)
}

// InstallEntry installs one compatible entry of a collection through the
// caller's Core and library.
//
// It is the single install path: the command line prints the result and the
// window shows it, but the checks are the same. A comic source whose script
// registers a different key is uninstalled again, and a service package is
// loaded once before anything is written.
func InstallEntry(ctx context.Context, m *Manager, collectionID, entryID, dataDir string, c *core.Core, lib *library.Library) (any, error) {
	entry, err := m.Entry(collectionID, entryID, Platform(), Capabilities())
	if err != nil {
		return nil, err
	}
	if !entry.Compatible {
		return nil, fmt.Errorf("合集条目 %s/%s 不支持当前平台,缺少 %v", collectionID, entryID, entry.Missing)
	}
	body, err := m.Artifact(ctx, entry)
	if err != nil {
		return nil, err
	}
	origin := "collection:" + collectionID + "/" + entryID
	if entry.Kind == "service.js" {
		digest := ""
		if entry.Artifact != nil {
			digest = entry.Artifact.SHA256
		}
		return services.Install(ctx, dataDir, origin, digest, body)
	}
	if entry.Kind != "comic-source.js" {
		return nil, fmt.Errorf("合集条目类型 %s 暂不支持", entry.Kind)
	}
	rec, err := lib.InstallTypedText(ctx, plugin.TypeComic, string(body), origin)
	if err != nil {
		return nil, err
	}
	if entry.Key != "" && rec.Key != entry.Key {
		removeErr := lib.Remove(ctx, rec.Key)
		return nil, fmt.Errorf("合集条目声明的 key %q 与脚本注册的 %q 不一致,已回滚(%v)", entry.Key, rec.Key, removeErr)
	}
	return rec, nil
}

// Platform is this client's platform in a collection's vocabulary.
func Platform() string {
	if runtime.GOOS == "darwin" {
		return "macos"
	}
	return runtime.GOOS
}

// Capabilities is what this client can install from a collection.
func Capabilities() []string {
	return []string{"comic-source.js", "video-source.js", "service.js"}
}
