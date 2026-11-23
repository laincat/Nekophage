package archive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/laincat/nekophage/internal/searches"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/veneracat-core/backup"
	"github.com/laincat/veneracat-core/comic/hostapi"
	"github.com/laincat/veneracat-core/gohost/cookies"
	"github.com/laincat/veneracat-core/gohost/library"
	"github.com/laincat/veneracat-core/plugin"
)

type Applied struct {
	Sources, Fields, History, Favourites, Searches, Pages, Cookies int
}

func Apply(ctx context.Context, b *Backup, lib *library.Library, sh *shelf.Shelf, history *searches.Store) (Applied, error) {
	var result Applied
	var failures error
	failedSources := make(map[string]bool)
	for _, source := range b.Plan.Sources {
		if err := ctx.Err(); err != nil {
			return result, errors.Join(failures, err)
		}
		if source.Script == "" {
			continue
		}
		_, err := lib.InstallExpecting(ctx, source.Key, plugin.TypeComic, source.Script, "backup:"+source.Key, "")
		if err != nil {
			failedSources[source.Key] = true
			failures = errors.Join(failures, fmt.Errorf("%s: %w", source.Key, err))
			continue
		}
		result.Sources++
	}
	for _, position := range b.Positions() {
		if err := ctx.Err(); err != nil {
			return result, errors.Join(failures, err)
		}
		chapter := ""
		if position.Episode > 0 {
			chapter = "第 " + strconv.Itoa(position.Episode) + " 话"
		}
		merged, err := sh.Merge(shelf.History{
			Comic:   shelf.Comic{Source: position.Source, ID: position.ID, Title: position.Title, Author: position.Author, Cover: position.Cover},
			Chapter: chapter, ChapterIndex: position.Episode - 1, Page: position.Page, Time: position.Time,
		})
		failures = errors.Join(failures, err)
		if merged && err == nil {
			result.History++
		}
	}
	for _, folder := range b.Favourites() {
		for _, item := range folder.Items {
			if err := ctx.Err(); err != nil {
				return result, errors.Join(failures, err)
			}
			added, err := sh.Add(folder.Name, shelf.Comic{Source: item.Source, ID: item.ID, Title: item.Title, Author: item.Author, Cover: item.Cover})
			failures = errors.Join(failures, err)
			if added && err == nil {
				result.Favourites++
			}
		}
	}
	pages, err := b.Pages()
	failures = errors.Join(failures, err)
	for _, page := range pages {
		if err := ctx.Err(); err != nil {
			return result, errors.Join(failures, err)
		}
		err := sh.KeepPage(shelf.ImageFavourite{Source: page.Source, ComicID: page.ComicID, ComicTitle: page.ComicTitle,
			ChapterID: page.ChapterID, ChapterIndex: page.ChapterIndex, ChapterName: page.ChapterName, Page: page.Page, URL: page.URL, Time: page.Time})
		failures = errors.Join(failures, err)
		if err == nil {
			result.Pages++
		}
	}
	if err := ctx.Err(); err != nil {
		return result, errors.Join(failures, err)
	}
	installed := []string{}
	for _, source := range lib.ComicSources() {
		if !failedSources[source.Key] {
			installed = append(installed, source.Key)
		}
	}
	result.Cookies, err = restoreCookies(b.Plan, lib.Jar(), installed, time.Now())
	failures = errors.Join(failures, err)
	if history != nil && len(b.Content.SearchHistory) > 0 {
		result.Searches, err = history.Merge(b.Content.SearchHistory)
		failures = errors.Join(failures, err)
	}
	data, err := b.SourceData()
	failures = errors.Join(failures, err)
	for _, source := range data {
		if err := ctx.Err(); err != nil {
			return result, errors.Join(failures, err)
		}
		if failedSources[source.Key] {
			continue
		}
		values := make(map[string]json.RawMessage, len(source.Fields))
		for _, field := range source.Fields {
			values[field.Name] = field.Value
		}
		fields, settings, err := lib.ImportSourceValues(ctx, source.Key, values)
		result.Fields += fields + settings
		failures = errors.Join(failures, err)
	}
	return result, failures
}

func restoreCookies(plan *backup.Plan, jar *cookies.Jar, installed []string, now time.Time) (int, error) {
	if plan == nil || len(plan.Cookies) == 0 || len(plan.CookieTargets) == 0 {
		return 0, nil
	}
	present := make(map[string]bool, len(installed))
	for _, key := range installed {
		present[key] = true
	}
	rows := toNetCookies(plan.Cookies, now)
	if len(rows) == 0 {
		return 0, nil
	}
	var failures error
	written := 0
	for _, target := range plan.CookieTargets {
		if !present[target] {
			continue
		}
		if err := jar.Import(hostapi.Partition(target), rows); err != nil {
			failures = errors.Join(failures, fmt.Errorf("%s: %w", target, err))
			continue
		}
		written += len(rows)
	}
	return written, failures
}

// toNetCookies turns a backup's cookie rows into the shape the jar imports.
//
// Expiry is milliseconds since the epoch in the backup and a time here; a row
// without one is a session cookie and stays without one.
func toNetCookies(rows []backup.Cookie, now time.Time) []*http.Cookie {
	out := make([]*http.Cookie, 0, len(rows))
	for _, row := range rows {
		c := &http.Cookie{Name: row.Name, Value: row.Value, Domain: row.Domain, Path: row.Path,
			Secure: row.Secure, HttpOnly: row.HTTPOnly}
		if row.Expires != nil && *row.Expires > 0 {
			when := time.UnixMilli(*row.Expires)
			// Already expired: the jar would never send it, yet the summary
			// would count it as a restored sign-in. Skip it instead.
			if !when.After(now) {
				continue
			}
			c.Expires = when
		}
		out = append(out, c)
	}
	return out
}
