package collections

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"slices"
	"strings"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/services"
	"github.com/laincat/nekophage/internal/sourceapi"
	"github.com/laincat/veneracat-core/gohost/library"
	"github.com/laincat/veneracat-core/plugin"
	"github.com/laincat/veneracat-core/subscription"
)

// installEntry installs one collection artifact after checking that this client
// supports the entry and that the downloaded bytes match the declared digest.
func installEntry(ctx context.Context, m *Manager, collectionID, entryID, dataDir, userAgent string, out io.Writer) error {
	c, err := core.Open(core.Options{DataDir: dataDir, AppVersion: sourceapi.Level, Locale: "zh_CN", UserAgent: userAgent})
	if err != nil {
		return err
	}
	defer c.Close()
	lib := library.New(c)
	if _, err := lib.Restore(ctx); err != nil {
		return err
	}
	result, err := InstallEntry(ctx, m, collectionID, entryID, dataDir, lib)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(result)
}

// InstallEntry installs one compatible entry of a collection through the
// caller's Core and library.
//
// It is the single install path: the command line prints the result and the
// window shows it. Both verify the declared identity before publishing.
func InstallEntry(ctx context.Context, m *Manager, collectionID, entryID, dataDir string, lib *library.Library) (any, error) {
	entry, err := m.Entry(collectionID, entryID, Platform(), Capabilities())
	if err != nil {
		return nil, err
	}
	if !entry.Compatible {
		return nil, fmt.Errorf("合集条目 %s/%s 不支持当前平台,缺少 %v", collectionID, entryID, entry.Missing)
	}
	kind, ok := installKind(entry)
	if !ok {
		return nil, fmt.Errorf("合集条目类型 %s 暂不支持", entry.Kind)
	}
	body, err := m.Artifact(ctx, entry)
	if err != nil {
		return nil, err
	}
	origin := "collection:" + collectionID + "/" + entryID
	digest := strings.ToLower(entry.Artifact.SHA256)
	switch kind {
	case "service.js":
		return services.InstallExpecting(ctx, dataDir, entry.Key, origin, digest, body)
	case "comic-source.js":
		return lib.InstallExpecting(ctx, entry.Key, plugin.TypeComic, string(body), origin, digest)
	case subscription.CapabilityNovelJS:
		return lib.InstallExpecting(ctx, entry.Key, plugin.TypeNovelJS, string(body), origin, digest)
	default:
		return nil, fmt.Errorf("合集条目类型 %s 暂不支持", entry.Kind)
	}
}

func installKind(entry Entry) (string, bool) {
	switch entry.Kind {
	case "comic-source.js", "service.js", subscription.CapabilityNovelJS:
		return entry.Kind, true
	case "novel-source":
		return subscription.CapabilityNovelJS, slices.Contains(entry.Supports.RequiresCapabilities, subscription.CapabilityNovelJS)
	default:
		return "", false
	}
}

// CanInstall reports whether this client can download and install entry.
func CanInstall(entry Entry) bool {
	if !entry.Compatible || entry.Artifact == nil {
		return false
	}
	_, ok := installKind(entry)
	return ok
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
	return []string{"comic-source.js", subscription.CapabilityNovelJS, "service.js"}
}
