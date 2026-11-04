// Package servicecli runs a service package from the command line.
//
// It is the first consumer of the Core service contract: the package on disk
// is loaded, its methods are called, and the result is printed as JSON. The
// command line has no account store, so the secrets permission is not granted;
// the network and storage permissions are, because the user asked for this
// package by name.
package servicecli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	case "call":
		if len(args) < 2 || len(args) > 3 {
			return errors.New("服务命令 call 需要 <包目录或服务 ID> <方法> [负载 JSON]")
		}
		return call(ctx, args[0], args[1], payloadArg(args), dataDir, out, userAgent)
	case "info":
		if len(args) != 1 {
			return errors.New("服务命令 info 需要 <包目录或服务 ID>")
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
		return errors.New("服务命令须为 call、info、list、remove、account-set、account-list 或 account-remove")
	}
}

// info prints the manifest of a package without running it.
func info(dataDir, target string, out io.Writer) error {
	m, _, err := read(dataDir, target)
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

// call runs one method of a package and prints its result.
func call(ctx context.Context, dir, method string, payload []byte, dataDir string, out io.Writer, userAgent string) error {
	m, source, err := read(dataDir, dir)
	if err != nil {
		return err
	}
	store, err := openStorage(dataDir, m.ID)
	if err != nil {
		return err
	}
	// Every request a service makes goes through the bridge: scheme, host
	// policy, redirects and response size are enforced there, not by whatever
	// the plugin decided to call.
	bridge := service.NewBridge(service.BridgePolicy{MaxBytes: maxFetchBytes, Timeout: requestTimeout})
	host := service.Host{
		Fetch:      bridge.Fetch,
		StorageGet: store.Get,
		StorageSet: store.Set,
		Secret: func(ctx context.Context, key string) (string, error) {
			accounts, err := services.OpenAccounts(dataDir, m.ID)
			if err != nil {
				return "", err
			}
			return accounts.Get(ctx, key)
		},
		Log: func(level, message string) {
			fmt.Fprintf(os.Stderr, "%s: %s %s\n", m.ID, level, message)
		},
	}
	// Secrets are granted only once the user has entered something for this
	// service: a service that asks for them but was never given an account has
	// nothing to read, and granting the permission would only widen the surface.
	granted := []service.Permission{service.PermissionNetwork, service.PermissionStorage}
	if accounts, err := services.OpenAccounts(dataDir, m.ID); err == nil {
		if count, err := accounts.Count(); err == nil && count > 0 {
			granted = append(granted, service.PermissionSecrets)
		}
	}
	rt := service.NewRuntime(host, granted...)
	defer rt.Close()
	if err := rt.Load(ctx, m, source); err != nil {
		return err
	}
	result, err := rt.Call(ctx, method, payload)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, strings.TrimSpace(string(result)))
	return nil
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

// read resolves a package: a directory when the argument is one, otherwise an
// installed service by id.
func read(dataDir, target string) (*service.Manifest, string, error) {
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return service.ReadPackage(target)
	}
	return services.Load(context.Background(), dataDir, target)
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

// storage keeps the values of one service in the data directory.
type storage struct {
	path string
}

func openStorage(dataDir, id string) (*storage, error) {
	dir := filepath.Join(dataDir, "services")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &storage{path: filepath.Join(dir, id+".json")}, nil
}

// Get returns one stored value.
func (s *storage) Get(ctx context.Context, key string) (string, bool, error) {
	values, err := s.read()
	if err != nil {
		return "", false, err
	}
	value, ok := values[key]
	return value, ok, nil
}

// Set writes one stored value.
func (s *storage) Set(ctx context.Context, key, value string) error {
	values, err := s.read()
	if err != nil {
		return err
	}
	values[key] = value
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// read loads the stored values, treating a missing file as an empty set.
func (s *storage) read() (map[string]string, error) {
	values := map[string]string{}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return values, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return values, nil
	}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("服务存储 %s 损坏: %w", s.path, err)
	}
	return values, nil
}
