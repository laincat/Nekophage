package archive

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/laincat/nekophage/internal/searches"
	"github.com/laincat/veneracat-core/backup"
	"github.com/laincat/veneracat-core/gohost"
	"github.com/laincat/veneracat-core/plugin"
)

func Export(ctx context.Context, path string, core *gohost.Core, searches *searches.Store) error {
	records := core.Records()
	sources := make([]backup.WriteSource, 0, len(records))
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return err
		}
		// A rule or a video source is not a comic source: the official app's
		// comic_source/ entries are comic scripts only.
		if record.Type != plugin.TypeComic || len(record.Body) == 0 {
			continue
		}
		// The source's own stored object travels beside its script: without it a
		// restore brings back code that has lost its tokens and its chosen domain.
		//
		// This client keeps data and settings in two buckets while the archive has
		// one object per source, so they are merged here and split again on the way
		// in. Data wins a name clash: it is what the source wrote for itself.
		data, err := core.Store().AllData(record.Key)
		if err != nil {
			return err
		}
		settings, err := core.Store().AllSettings(record.Key)
		if err != nil {
			return err
		}
		merged := make(map[string]json.RawMessage, len(data)+len(settings))
		for name, value := range settings {
			merged[name] = value
		}
		for name, value := range data {
			merged[name] = value
		}
		sources = append(sources, backup.WriteSource{
			Name: record.Key + ".js",
			Body: []byte(record.Body),
			Data: merged,
		})
	}

	// The store keeps the newest first; the archive lists them oldest first.
	history := []string{}
	if searches != nil {
		items := searches.Items(200)
		for i := len(items) - 1; i >= 0; i-- {
			history = append(history, items[i])
		}
	}

	// Settings are kept per source in this client, so there is no app-level
	// document to put here; an empty object is honest about that.
	var buf bytes.Buffer
	if err := backup.Write(&buf, backup.WriteOptions{
		Sources:       sources,
		Settings:      map[string]json.RawMessage{},
		SearchHistory: history,
	}); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".archive-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(buf.Bytes()); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
