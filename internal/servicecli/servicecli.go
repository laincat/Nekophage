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

	"github.com/laincat/veneracat-core/service"

	"github.com/laincat/nekophage/internal/core"
)

// maxFetchBytes bounds one fetch a service makes.
const maxFetchBytes = 4 << 20

// RunCommand runs one service action and writes its result to out.
func RunCommand(ctx context.Context, action string, args []string, dataDir string, out io.Writer, userAgent string) error {
	switch action {
	case "call":
		if len(args) < 2 || len(args) > 3 {
			return errors.New("服务命令 call 需要 <包目录> <方法> [负载 JSON]")
		}
		return call(ctx, args[0], args[1], payloadArg(args), dataDir, out, userAgent)
	case "info":
		if len(args) != 1 {
			return errors.New("服务命令 info 需要 <包目录>")
		}
		return info(args[0], out)
	default:
		return errors.New("服务命令须为 call 或 info")
	}
}

// info prints the manifest of a package without running it.
func info(dir string, out io.Writer) error {
	m, _, err := service.ReadPackage(dir)
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
	m, source, err := service.ReadPackage(dir)
	if err != nil {
		return err
	}
	store, err := openStorage(dataDir, m.ID)
	if err != nil {
		return err
	}
	host := service.Host{
		Fetch: func(ctx context.Context, rawURL string) (string, error) {
			body, _, err := transport(userAgent).Fetch(ctx, rawURL, maxFetchBytes)
			return string(body), err
		},
		StorageGet: store.Get,
		StorageSet: store.Set,
		Log: func(level, message string) {
			fmt.Fprintf(os.Stderr, "%s: %s %s\n", m.ID, level, message)
		},
	}
	rt := service.NewRuntime(host, service.PermissionNetwork, service.PermissionStorage)
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

// payloadArg returns the request payload, or an empty JSON document.
func payloadArg(args []string) []byte {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return nil
	}
	return json.RawMessage(args[2])
}

// transport builds the HTTP transport the service uses for the network.
func transport(userAgent string) *core.Transport {
	return core.NewTransport(nil, userAgent, 0)
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
