package main

// Runs a source through Core's JavaScript engine against the live site, with no
// interface and no client: load the script, call search, print what came back.
//
//	go run ./cmd/pixivprobe PATH/TO/plugin.js [KEYWORD]
//
// The interface tests use fixtures so that a failure means "the client broke". This
// is the other direction: it asks whether the source still works on the site it was
// written for. Run it when a source stops answering, before blaming the client.
//
// It needs the network; there is no offline mode on purpose, since an offline copy
// would be another fixture pretending to be the site.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/laincat/veneracat-core/comic/js"
	"github.com/laincat/veneracat-core/host"
)

type httpTransport struct{ client *http.Client }

func (t httpTransport) Do(ctx context.Context, req *host.Request) (*host.Response, error) {
	r, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return nil, err
	}
	for name, values := range req.Header {
		for _, v := range values {
			r.Header.Add(name, v)
		}
	}
	resp, err := t.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	return &host.Response{Status: resp.StatusCode, Header: resp.Header,
		Body: body, FinalURL: resp.Request.URL.String()}, nil
}

type memStore struct {
	data     map[string]json.RawMessage
	settings map[string]json.RawMessage
}

func (m *memStore) LoadData(key, dataKey string) (json.RawMessage, bool, error) {
	v, ok := m.data[key+"/"+dataKey]
	return v, ok, nil
}
func (m *memStore) SaveData(key, dataKey string, data json.RawMessage) error {
	m.data[key+"/"+dataKey] = data
	return nil
}
func (m *memStore) DeleteData(key, dataKey string) error {
	delete(m.data, key+"/"+dataKey)
	return nil
}
func (m *memStore) LoadSetting(key, settingKey string) (json.RawMessage, bool, error) {
	v, ok := m.settings[key+"/"+settingKey]
	return v, ok, nil
}
func (m *memStore) IsLogged(string) bool { return false }

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: pixivprobe PATH/TO/plugin.js [KEYWORD]")
		os.Exit(2)
	}
	src := os.Args[1]
	keyword := "オリジナル"
	if len(os.Args) > 2 {
		keyword = os.Args[2]
	}
	body, err := os.ReadFile(filepath.Clean(src))
	if err != nil {
		fmt.Println("read source:", err)
		os.Exit(1)
	}

	r, err := js.New(js.Options{}, js.Services{
		Transport: httpTransport{client: &http.Client{Timeout: 60 * time.Second}},
		Storage:   &memStore{data: map[string]json.RawMessage{}, settings: map[string]json.RawMessage{}},
		Env:       js.Environment{AppVersion: "1.6.0"},
		Log:       func(level, title, content string) { fmt.Printf("  [%s] %s: %s\n", level, title, content) },
	})
	if err != nil {
		fmt.Println("runtime:", err)
		os.Exit(1)
	}
	defer r.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	meta, err := r.LoadSource(ctx, string(body))
	if err != nil {
		fmt.Println("load:", err)
		os.Exit(1)
	}
	fmt.Printf("source: %s (%s) v%s\n", meta.Name, meta.Key, meta.Version)

	for _, call := range []struct{ path, args string }{
		{"search.load", fmt.Sprintf(`[%q,[],1]`, keyword)},
	} {
		start := time.Now()
		out, err := r.Call(ctx, meta.Key, call.path, rawArgs(call.args)...)
		if err != nil {
			fmt.Printf("%s %s -> error: %v\n", call.path, call.args, err)
			continue
		}
		fmt.Printf("%s %s -> %s in %s\n", call.path, call.args, summarise(out), time.Since(start).Round(time.Millisecond))
	}
}

// rawArgs turns a JSON array of arguments into the values the runtime takes.
func rawArgs(text string) []any {
	var list []any
	if err := json.Unmarshal([]byte(text), &list); err != nil {
		return nil
	}
	return list
}

func summarise(raw json.RawMessage) string {
	var page struct {
		Comics []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"comics"`
		MaxPage int `json:"maxPage"`
	}
	if err := json.Unmarshal(raw, &page); err != nil {
		if len(raw) > 160 {
			raw = raw[:160]
		}
		return "not a page: " + string(raw)
	}
	first := ""
	if len(page.Comics) > 0 {
		first = page.Comics[0].Title
	}
	return fmt.Sprintf("%d comics, maxPage %d, first %q", len(page.Comics), page.MaxPage, first)
}
