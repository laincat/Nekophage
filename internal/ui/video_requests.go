package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/player"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/veneracat-core/video"
)

func (a *App) videoKey() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.videoKeyLocked()
}

func (a *App) videoKeyLocked() string {
	sources := a.videoSources()
	for _, source := range sources {
		if source.Key == a.video.key {
			return source.Key
		}
	}
	if len(sources) > 0 {
		return sources[0].Key
	}
	return ""
}

// Source changes invalidate all pending reads, including a resolved play URL
// waiting for the UI thread. An already running player owns its own session.
func (a *App) selectVideoSourceLocked(key string) {
	if a.video.key == key {
		return
	}
	a.videoSearchGen.next()
	a.invalidateVideoDetailsLocked()
	a.video.key = key
	a.video.results, a.video.next, a.video.resultsKey = nil, "", ""
	a.video.searchQuery = ""
	a.video.busy, a.video.errText, a.video.notice = "", "", ""
}

func (a *App) invalidateVideoDetailsLocked() {
	a.videoDetailGen.next()
	a.videoPlayGen.next()
	a.video.selected, a.video.details, a.video.line = nil, nil, 0
	a.video.detailKey, a.video.detailBusy, a.video.detailErr = "", "", ""
}

func (a *App) closeVideoDetails() {
	a.mu.Lock()
	a.invalidateVideoDetailsLocked()
	a.mu.Unlock()
}

func (a *App) closeVideoList() {
	a.mu.Lock()
	a.videoSearchGen.next()
	a.video.busy = ""
	a.mu.Unlock()
}

// Network work returns to the main window before committing state or opening
// another window. The nil case permits actual Core/HTTP tests without a GUI.
func (a *App) dispatchVideo(fn func()) {
	if a.win != nil {
		a.win.Update(fn)
	} else {
		fn()
	}
}

func (a *App) refreshVideoViews() {
	a.mu.Lock()
	windows := []*mygo.Window{a.videoWin, a.detailWin}
	a.mu.Unlock()
	for _, window := range windows {
		if window != nil && !window.IsDestroyed() {
			window.Update(func() {})
		}
	}
}

func (a *App) videoSearch() { a.startVideoPage(false) }
func (a *App) videoMore()   { a.startVideoPage(true) }

func (a *App) startVideoPage(more bool) {
	a.mu.Lock()
	key := a.videoKeyLocked()
	if key == "" {
		a.mu.Unlock()
		return
	}
	query, cursor := strings.TrimSpace(a.video.query), ""
	var ticket int
	if more {
		if a.video.busy != "" || a.video.next == "" || a.video.resultsKey != key {
			a.mu.Unlock()
			return
		}
		query, cursor = a.video.searchQuery, a.video.next
		ticket = a.videoSearchGen.peek()
		a.video.busy = "正在加载下一页…"
	} else {
		a.selectVideoSourceLocked(key)
		a.invalidateVideoDetailsLocked()
		ticket = a.videoSearchGen.next()
		a.video.results, a.video.next = nil, ""
		a.video.searchQuery, a.video.resultsKey = query, key
		a.video.busy = "正在搜索…"
	}
	a.video.errText, a.video.notice = "", ""
	a.mu.Unlock()
	a.refreshVideoViews()
	go func() {
		ctx, cancel := context.WithTimeout(a.videoSearchGen.requestContext(ticket), 45*time.Second)
		defer cancel()
		page, err := a.core.VideoSearch(ctx, key, query, cursor)
		a.dispatchVideo(func() {
			if a.finishVideoPage(ticket, key, more, page, err) {
				a.refreshVideoViews()
			}
		})
	}()
}

func (a *App) finishVideoPage(ticket int, key string, more bool, page *video.Page, err error) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.videoSearchGen.current(ticket) || a.video.key != key {
		return false
	}
	a.video.busy = ""
	if err == nil && page == nil {
		err = fmt.Errorf("video: source returned no search page")
	}
	if err != nil {
		a.video.errText = explain(err)
		return true
	}
	if more {
		a.video.results = append(a.video.results, page.Items...)
	} else {
		a.video.results = page.Items
	}
	a.video.next = page.Next
	a.video.notice = strconv.Itoa(len(a.video.results)) + " 个结果"
	return true
}

func (a *App) openVideoDetails(item video.Item) {
	a.startVideoDetails(a.videoKey(), item, nil)
}

func (a *App) videoResume(entry shelf.VideoHistory) {
	a.startVideoDetails(entry.Source, video.Item{ID: entry.ItemID, Title: entry.Title}, &entry)
}

func (a *App) startVideoDetails(key string, item video.Item, resume *shelf.VideoHistory) {
	if key == "" {
		return
	}
	a.mu.Lock()
	a.selectVideoSourceLocked(key)
	a.invalidateVideoDetailsLocked()
	ticket := a.videoDetailGen.peek()
	a.video.selected, a.video.detailKey = &item, key
	a.video.detailBusy = "正在读取详情…"
	a.mu.Unlock()
	a.showVideoDetailsWindow(item.Title)
	a.refreshVideoViews()
	go func() {
		ctx, cancel := context.WithTimeout(a.videoDetailGen.requestContext(ticket), 45*time.Second)
		defer cancel()
		details, err := a.core.VideoDetails(ctx, key, item.ID)
		a.dispatchVideo(func() {
			if !a.finishVideoDetails(ticket, key, item.ID, details, err) {
				return
			}
			a.refreshVideoViews()
			if err != nil || details == nil || resume == nil {
				return
			}
			episode := -1
			a.mu.Lock()
			if !a.videoDetailGen.current(ticket) || a.video.detailKey != key {
				a.mu.Unlock()
				return
			}
			for i, line := range details.Lines {
				if line.ID != resume.LineID {
					continue
				}
				for j, ep := range line.Episodes {
					if ep.ID == resume.EpisodeID {
						a.video.line, episode = i, j
					}
				}
			}
			if episode < 0 {
				a.video.detailErr = "这个视频源已经没有这一集了"
			}
			a.mu.Unlock()
			if episode >= 0 {
				a.playVideo(episode)
			} else {
				a.refreshVideoViews()
			}
		})
	}()
}

func (a *App) finishVideoDetails(ticket int, key, id string, details *video.Details, err error) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.videoDetailGen.current(ticket) || a.video.detailKey != key || a.video.selected == nil || a.video.selected.ID != id {
		return false
	}
	a.video.detailBusy = ""
	if err == nil && details == nil {
		err = fmt.Errorf("video: source returned no details")
	}
	if err != nil {
		a.video.detailErr = explain(err)
	} else {
		a.video.details = details
	}
	return true
}

func (a *App) showVideoDetailsWindow(title string) {
	if a.win == nil {
		return
	}
	a.mu.Lock()
	window := a.detailWin
	a.mu.Unlock()
	if window == nil || window.IsDestroyed() {
		window = mygo.NewWindow(mygo.WindowOptions{
			Title: "视频详情", Width: 760, Height: 680, Content: ui.View(a.videoDetailView),
		})
		a.mu.Lock()
		a.detailWin = window
		a.mu.Unlock()
		window.OnClosed(func() {
			a.mu.Lock()
			if a.detailWin == window {
				a.invalidateVideoDetailsLocked()
			}
			a.mu.Unlock()
		})
	}
	window.SetTitle(title)
	window.Show()
	window.Focus()
}

func (a *App) playVideo(episodeIndex int) {
	a.mu.Lock()
	state := a.video
	if state.selected == nil || state.details == nil || len(state.details.Lines) == 0 {
		a.mu.Unlock()
		return
	}
	line := state.details.Lines[clampIndex(state.line, len(state.details.Lines))]
	if episodeIndex < 0 || episodeIndex >= len(line.Episodes) {
		a.mu.Unlock()
		return
	}
	episode := line.Episodes[episodeIndex]
	if episode.Title == "" {
		episode.Title = "第 " + strconv.Itoa(episodeIndex+1) + " 集"
	}
	ticket := a.videoPlayGen.next()
	a.video.detailBusy, a.video.detailErr = "正在准备播放…", ""
	a.mu.Unlock()
	a.refreshVideoViews()
	key, item := state.detailKey, *state.selected
	go func() {
		ctx, cancel := context.WithTimeout(a.videoPlayGen.requestContext(ticket), 45*time.Second)
		defer cancel()
		play, err := a.core.VideoPlay(ctx, key, item.ID, line.ID, episode.ID)
		a.dispatchVideo(func() {
			a.mu.Lock()
			current := a.videoPlayGen.current(ticket) && a.video.detailKey == key && a.video.selected != nil && a.video.selected.ID == item.ID
			if !current {
				a.mu.Unlock()
				return
			}
			a.video.detailBusy = ""
			if err != nil {
				a.video.detailErr = explain(err)
			}
			a.mu.Unlock()
			if err != nil {
				a.refreshVideoViews()
				return
			}
			meta := player.Meta{Source: key, ItemID: item.ID, Title: item.Title,
				LineID: line.ID, LineName: line.Name, EpisodeID: episode.ID, EpisodeName: episode.Title}
			var start float64
			if a.shelf != nil {
				if saved, ok := a.shelf.VideoProgress(key, item.ID, episode.ID); ok && !videoFinished(saved.Position, saved.Duration) {
					start = saved.Position
				}
			}
			if a.win != nil {
				if err := a.openPlayer(play, meta, item.Title+" · "+episode.Title, start); err != nil {
					a.mu.Lock()
					a.video.detailErr = explain(err)
					a.mu.Unlock()
					a.refreshVideoViews()
					return
				}
			}
			a.mu.Lock()
			if a.videoPlayGen.current(ticket) {
				a.video.playing = episode.Title
			}
			a.mu.Unlock()
			a.refreshVideoViews()
		})
	}()
}
