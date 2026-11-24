// Package servicecli runs a service package from the command line.
//
// It is the command-line consumer of the Core service contract: packages can
// be installed or upgraded from local `.zip` archives, inspected for requested
// and denied permissions (`network`/`storage`/`secrets`), described, and
// invoked with isolated per-service accounts.
package servicecli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/laincat/veneracat-core/service"

	"github.com/laincat/nekophage/internal/services"
)

// maxFetchBytes bounds one fetch a service makes.
const maxFetchBytes = 4 << 20

// requestTimeout bounds one request a service makes.
const requestTimeout = 20 * time.Second

// RunCommand runs one service action and writes its result to out.
func RunCommand(ctx context.Context, action string, args []string, dataDir string, out io.Writer, userAgent string) error {
	switch action {
	case "install":
		if len(args) < 1 || len(args) > 2 {
			return errors.New("服务命令 install 需要 <服务包 zip 路径> [期望服务 ID]")
		}
		expectedID := ""
		if len(args) == 2 {
			expectedID = strings.TrimSpace(args[1])
		}
		rec, report, err := services.InstallZipFile(ctx, dataDir, args[0], expectedID)
		if err != nil {
			return err
		}
		return writeJSON(out, map[string]any{
			"installed":   rec,
			"permissions": report,
		})
	case "permissions":
		if len(args) < 1 || len(args) > 2 {
			return errors.New("服务命令 permissions 需要 <zip/包目录/服务 ID> [允许权限列表]")
		}
		return permissionsCmd(dataDir, args, false, out)
	case "check-permissions":
		if len(args) < 1 || len(args) > 2 {
			return errors.New("服务命令 check-permissions 需要 <zip/包目录/服务 ID> [允许权限列表]")
		}
		return permissionsCmd(dataDir, args, true, out)
	case "describe":
		if len(args) != 1 {
			return errors.New("服务命令 describe 需要 <包目录或服务 ID>")
		}
		return describe(ctx, args[0], dataDir, out, userAgent)
	case "call":
		if len(args) < 2 || len(args) > 3 {
			return errors.New("服务命令 call 需要 <包目录或服务 ID> <方法> [负载 JSON]")
		}
		return call(ctx, args[0], args[1], payloadArg(args), dataDir, out, userAgent)
	case "info":
		if len(args) != 1 {
			return errors.New("服务命令 info 需要 <zip/包目录/服务 ID>")
		}
		return info(dataDir, args[0], out)
	case "list":
		if len(args) != 0 {
			return errors.New("服务命令 list 不需要参数")
		}
		return list(dataDir, out)
	case "remove":
		if len(args) != 1 {
			return errors.New("服务命令 remove 需要 <服务 ID>")
		}
		if err := services.Remove(dataDir, args[0]); err != nil {
			return err
		}
		return writeJSON(out, map[string]string{"removed": args[0]})
	case "account-set":
		if len(args) != 3 {
			return errors.New("服务命令 account-set 需要 <服务 ID> <键> <值>")
		}
		accounts, err := services.OpenAccounts(dataDir, args[0])
		if err != nil {
			return err
		}
		if err := accounts.Set(ctx, args[1], args[2]); err != nil {
			return err
		}
		return writeJSON(out, map[string]any{"service": args[0], "key": args[1], "stored": true})
	case "account-list":
		if len(args) != 1 {
			return errors.New("服务命令 account-list 需要 <服务 ID>")
		}
		accounts, err := services.OpenAccounts(dataDir, args[0])
		if err != nil {
			return err
		}
		keys, err := accounts.Keys()
		if err != nil {
			return err
		}
		return writeJSON(out, map[string]any{"service": args[0], "keys": keys})
	case "account-remove":
		if len(args) != 2 {
			return errors.New("服务命令 account-remove 需要 <服务 ID> <键>")
		}
		accounts, err := services.OpenAccounts(dataDir, args[0])
		if err != nil {
			return err
		}
		if err := accounts.Remove(ctx, args[1]); err != nil {
			return err
		}
		return writeJSON(out, map[string]any{"service": args[0], "key": args[1], "removed": true})
	default:
		return errors.New("服务命令须为 install、permissions、check-permissions、describe、call、info、list、remove、account-set、account-list 或 account-remove")
	}
}

func permissionsCmd(dataDir string, args []string, strict bool, out io.Writer) error {
	m, err := readManifest(dataDir, args[0])
	if err != nil {
		return err
	}
	var allowed []service.Permission
	if len(args) == 2 {
		parsed, err := parsePermissionCSV(args[1])
		if err != nil {
			return err
		}
		allowed = parsed
	} else {
		// Default host grants: network + storage, plus secrets only if the
		// service already has stored isolated account values.
		allowed = []service.Permission{service.PermissionNetwork, service.PermissionStorage}
		if accounts, err := services.OpenAccounts(dataDir, m.ID); err == nil {
			if count, err := accounts.Count(); err == nil && count > 0 {
				allowed = append(allowed, service.PermissionSecrets)
			}
		}
	}
	var report services.PermissionReport
	if strict {
		report, err = services.RequirePermissions(m, allowed)
		if err != nil {
			return err
		}
	} else {
		report, err = services.EvaluatePermissions(m, allowed)
		if err != nil {
			return err
		}
	}
	return writeJSON(out, report)
}

func parsePermissionCSV(raw string) ([]service.Permission, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.EqualFold(trimmed, "none") {
		return []service.Permission{}, nil
	}
	parts := strings.Split(trimmed, ",")
	out := make([]service.Permission, 0, len(parts))
	for _, p := range parts {
		token := strings.TrimSpace(p)
		if token == "" {
			continue
		}
		out = append(out, service.Permission(token))
	}
	return out, nil
}

// info prints the manifest of a package without running it.
func info(dataDir, target string, out io.Writer) error {
	m, err := readManifest(dataDir, target)
	if err != nil {
		return err
	}
	return writeJSON(out, map[string]any{
		"id":          m.ID,
		"name":        m.Name,
		"version":     m.Version,
		"kind":        m.Kind,
		"entry":       m.Entry,
		"permissions": m.Permissions,
	})
}

// describe loads a service and prints the methods exported by describe().
func describe(ctx context.Context, target, dataDir string, out io.Writer, userAgent string) error {
	rt, _, err := openTarget(ctx, dataDir, target, userAgent)
	if err != nil {
		return err
	}
	defer rt.Close()
	methods, err := rt.Describe(ctx)
	if err != nil {
		return err
	}
	if methods == nil {
		methods = []service.Method{}
	}
	return writeJSON(out, map[string]any{"methods": methods})
}

// call runs one method of a package and prints its result.
func call(ctx context.Context, dir, method string, payload []byte, dataDir string, out io.Writer, userAgent string) error {
	rt, manifest, err := openTarget(ctx, dataDir, dir, userAgent)
	if err != nil {
		return err
	}
	defer rt.Close()
	result, err := rt.Call(ctx, method, payload)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, strings.TrimSpace(string(result)))
	_ = manifest
	return nil
}

func openTarget(ctx context.Context, dataDir, dir, userAgent string) (*service.Runtime, *service.Manifest, error) {
	target := strings.TrimSpace(dir)
	if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
		manifest, source, err := service.ReadPackage(target)
		if err != nil {
			return nil, nil, err
		}
		rt, err := services.HostPackageWithOptions(ctx, dataDir, manifest, source, services.HostOptions{UserAgent: userAgent})
		return rt, manifest, err
	}
	return services.Open(ctx, dataDir, target, userAgent)
}

// list prints the installed services.
func list(dataDir string, out io.Writer) error {
	all, err := services.List(dataDir)
	if err != nil {
		return err
	}
	if all == nil {
		all = []services.Record{}
	}
	return writeJSON(out, all)
}

// readManifest resolves a package manifest from a `.zip` file, directory, or
// installed service ID.
func readManifest(dataDir, target string) (*service.Manifest, error) {
	if stat, err := os.Stat(target); err == nil {
		if stat.IsDir() {
			m, _, err := service.ReadPackage(target)
			return m, err
		}
		if strings.HasSuffix(strings.ToLower(target), ".zip") {
			m, _, _, err := services.InspectZipFile(target)
			return m, err
		}
	}
	m, _, err := services.Load(context.Background(), dataDir, target)
	return m, err
}

// payloadArg returns the request payload, or an empty JSON document.
func payloadArg(args []string) []byte {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return nil
	}
	return json.RawMessage(args[2])
}

// writeJSON prints one JSON document.
func writeJSON(out io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(data))
	return nil
}
