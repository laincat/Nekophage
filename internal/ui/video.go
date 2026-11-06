package ui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/player"
	"github.com/laincat/nekophage/internal/shelf"
	"github.com/laincat/veneracat-core/plugin"
	"github.com/laincat/veneracat-core/video"
)

// The video window is the desktop side of Core's video providers. Core parses
// the source and hands back a play declaration; this window only chooses what
// to play and where to resume, and the player window shows it through the
// local media proxy.
type videoState struct {
	// key is the chosen provider; empty means the first enabled one.
	key         string
	query       string
	searchQuery string
	resultsKey  string
	results     []video.Item
	next        string
	busy        string // list search/pagination only
	errText     string
	notice      string
	detailKey   string
	detailBusy  string // details/play resolution only
	detailErr   string
	selected    *video.Item
	details     *video.Details
	line        int
	// playing names the episode in the player window.
	playing  string
	probeErr error
}

// videoSources lists enabled providers. A comic source and a video source can
// share a key, so the list is filtered by type rather than by name.
func (a *App) videoSources() []plugin.Record {
	var out []plugin.Record
	for _, rec := range a.core.Plugins() {
		if rec.Enabled && (rec.Type == plugin.TypeVideoJS || rec.Type == plugin.TypeKazumi || rec.Type == plugin.TypeBuiltin) {
			out = append(out, rec)
		}
	}
	return out
}

// openVideo shows the video window, creating it once.
func (a *App) openVideo() {
	if a.videoWin == nil || a.videoWin.IsDestroyed() {
		window := mygo.NewWindow(mygo.WindowOptions{
			Title: "视频", Width: 860, Height: 720, Content: ui.View(a.videoView),
		})
		a.mu.Lock()
		a.videoWin = window
		a.mu.Unlock()
		window.OnClosed(func() { a.closeVideoList() })
		window.Show()
		return
	}
	a.videoWin.Show()
	a.videoWin.Focus()
}

func (a *App) videoView(c *ui.Context) {
	t := c.Theme()
	a.mu.Lock()
	state := a.video
	sources := a.videoSources()
	a.mu.Unlock()

	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(16).Gap(8).Children(func() {
			ui.Text(c, "视频").FontSize(16).Bold()
			ui.Text(c, "选择视频源，搜索后打开详情，选择线路和剧集播放。").FontSize(12).TextColor(t.TextMuted)
			if len(sources) == 0 {
				ui.Text(c, "还没有可用的视频源。在「源管理」选择视频源或 Kazumi 规则,从链接或文件安装。").FontSize(12).TextColor(t.TextMuted)
			}
			labels := make([]string, len(sources))
			for i, s := range sources {
				labels[i] = s.Name
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if len(sources) > 0 {
					sel := labels[0]
					if state.key != "" {
						for i, s := range sources {
							if s.Key == state.key {
								sel = labels[i]
							}
						}
					}
					if ui.Select(c, &sel, labels).Width(220).Label("视频源").Changed() {
						if indexOfOption(labels, sel) >= 0 {
							a.mu.Lock()
							a.selectVideoSourceLocked(sources[indexOfOption(labels, sel)].Key)
							a.mu.Unlock()
							a.refreshVideoViews()
						}
					}
				}
				if ui.SearchField(c, &a.video.query).Width(280).Submitted() {
					a.videoSearch()
				}
				if ui.Button(c, "搜索").Disabled(len(sources) == 0).Clicked() {
					a.videoSearch()
				}
				if ui.Button(c, "更多").Disabled(state.busy != "" || state.next == "").Clicked() {
					a.videoMore()
				}
			})
			if state.busy != "" {
				ui.Text(c, state.busy).FontSize(12).TextColor(t.TextMuted)
			}
			if state.errText != "" {
				ui.Text(c, state.errText).FontSize(12).TextColor(ui.RGB(224, 49, 49)).Selectable()
			}
			if state.notice != "" {
				ui.Text(c, state.notice).FontSize(12).TextColor(t.TextMuted)
			}
			if a.shelf != nil && len(a.shelf.VideoHistory(5)) > 0 {
				ui.Text(c, "继续观看").FontSize(13).Bold()
				for _, entry := range a.shelf.VideoHistory(5) {
					entry := entry
					label := entry.Title
					if entry.EpisodeName != "" {
						label += " · " + entry.EpisodeName
					}
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if ui.Button(c, label).Clicked() {
							a.videoResume(entry)
						}
						ui.Text(c, progressText(entry.Position, entry.Duration)).FontSize(11).TextColor(t.TextMuted)
					})
				}
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Padding(12, 16).Children(func() {
			for _, item := range state.results {
				item := item
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					label := item.Title
					if item.Subtitle != "" {
						label += "  " + item.Subtitle
					}
					if ui.Button(c, label).Clicked() {
						a.openVideoDetails(item)
					}
				})
			}
			if len(state.results) == 0 && state.busy == "" && state.errText == "" {
				ui.Text(c, "搜索结果会显示在这里。").FontSize(12).TextColor(t.TextMuted)
			}
		})
	})
}

// progressText is the "watched minutes / total" line under a resume entry.
func progressText(position, duration float64) string {
	if duration <= 0 {
		return ""
	}
	pct := int(position / duration * 100)
	return clockText(position) + " / " + clockText(duration) + " · " + strconv.Itoa(pct) + "%"
}

func clockText(seconds float64) string {
	total := int(seconds)
	if total < 0 {
		total = 0
	}
	h, m, s := total/3600, total%3600/60, total%60
	if h > 0 {
		return strconv.Itoa(h) + ":" + twoDigits(m) + ":" + twoDigits(s)
	}
	return strconv.Itoa(m) + ":" + twoDigits(s)
}

func twoDigits(v int) string {
	if v < 10 {
		return "0" + strconv.Itoa(v)
	}
	return strconv.Itoa(v)
}

func (a *App) videoDetailView(c *ui.Context) {
	t := c.Theme()
	a.mu.Lock()
	state := a.video
	a.mu.Unlock()
	item, details := state.selected, state.details

	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(16).Gap(8).Children(func() {
			title := "视频详情"
			if item != nil {
				title = item.Title
			}
			ui.Text(c, title).FontSize(16).Bold()
			if details != nil && details.Description != "" {
				ui.Text(c, details.Description).FontSize(12).TextColor(t.TextMuted).MaxLines(3)
			}
			if state.detailBusy != "" {
				ui.Text(c, state.detailBusy).FontSize(12).TextColor(t.TextMuted)
			}
			if state.detailErr != "" {
				ui.Text(c, state.detailErr).FontSize(12).TextColor(ui.RGB(224, 49, 49)).Selectable()
			}
			if details == nil || len(details.Lines) == 0 {
				if state.detailBusy == "" && state.detailErr == "" {
					ui.Text(c, "这个视频没有可播放的线路。").FontSize(12).TextColor(t.TextMuted)
				}
				return
			}
			lines := details.Lines
			names := make([]string, len(lines))
			for i, line := range lines {
				names[i] = line.Name
				if names[i] == "" {
					names[i] = "线路 " + strconv.Itoa(i+1)
				}
			}
			if len(names) > 1 {
				sel := names[0]
				if state.line >= 0 && state.line < len(names) {
					sel = names[state.line]
				}
				if ui.Select(c, &sel, names).Width(220).Label("线路").Changed() {
					if index := indexOfOption(names, sel); index >= 0 {
						a.mu.Lock()
						a.videoPlayGen.next()
						a.video.line, a.video.detailBusy = index, ""
						a.mu.Unlock()
						a.detailWin.Update(func() {})
					}
				}
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Padding(12, 16).Children(func() {
			if details == nil || len(details.Lines) == 0 {
				return
			}
			line := details.Lines[clampIndex(state.line, len(details.Lines))]
			for i, episode := range line.Episodes {
				i, episode := i, episode
				name := episode.Title
				if name == "" {
					name = "第 " + strconv.Itoa(i+1) + " 集"
				}
				// The saved position is shown next to the episode, so a
				// resumed episode reads as resumed before it is clicked.
				label := name
				if item != nil && a.shelf != nil {
					if saved, ok := a.shelf.VideoProgress(state.detailKey, item.ID, episode.ID); ok && saved.Position > 0 {
						label += "  · " + clockText(saved.Position)
					}
				}
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if ui.Button(c, label).Clicked() {
						a.playVideo(i)
					}
				})
			}
		})
	})
}

func clampIndex(index, length int) int {
	if index < 0 || index >= length {
		return 0
	}
	return index
}

// openPlayer shows the player window for one play declaration. Only one
// player is open at a time: a second episode replaces the first, and the old
// session stops so its media requests do not keep running.
func (a *App) openPlayer(play *video.Play, meta player.Meta, title string, start float64) error {
	if a.playerSrv == nil {
		server, err := player.NewServer()
		if err != nil {
			return err
		}
		server.OnProgress = a.saveVideoProgress
		// The subtitle choice is remembered per episode next to its position,
		// so reopening the episode restores the track the viewer picked.
		server.Subtitle = a.savedVideoSubtitle
		server.OnSubtitle = a.saveVideoSubtitle
		// With nothing saved for an episode, its subtitle follows the
		// system's language order, the way the iOS client chooses.
		server.Languages = mygo.App.PreferredLanguages
		a.mu.Lock()
		a.playerSrv = server
		a.mu.Unlock()
		debugLog("video: player server started")
	}
	session, err := a.playerSrv.Open(a.core, meta.Source, play, meta)
	if err != nil {
		return err
	}
	if a.playerWin == nil {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title: "播放", Width: 1000, Height: 640, Content: ui.View(a.playerView_),
		})
		view, err := win.NewWebView(mygo.WebViewOptions{
			URL: "about:blank", Page: mygo.PageOptions{Session: a.dev.WebSession},
		})
		if err != nil {
			session.Close()
			win.Destroy()
			return err
		}
		a.mu.Lock()
		a.playerWin, a.playerView = win, view
		a.mu.Unlock()
		win.OnClosed(func() {
			a.mu.Lock()
			a.videoPlayGen.next()
			a.video.detailBusy = ""
			if a.playerSess != nil {
				a.playerSess.Close()
				a.playerSess = nil
			}
			a.playerView = nil
			a.playerWin = nil
			a.mu.Unlock()
		})
	}
	if old := a.playerSess; old != nil {
		old.Close()
	}
	a.mu.Lock()
	a.playerSess = session
	a.mu.Unlock()
	a.playerWin.SetTitle(title)
	a.playerWin.Show()
	a.playerWin.Focus()
	pageURL := session.URL(title, start)
	if a.dev.VideoForceHLS {
		pageURL += "&mode=hls"
	}
	if a.dev.VideoForceNative {
		pageURL += "&mode=native"
	}
	if err := a.playerView.Page().LoadURL(pageURL); err != nil {
		return err
	}
	return nil
}

// playerView_ draws the player window's toolbar; the WebView fills the rest.
func (a *App) playerView_(c *ui.Context) {
	t := c.Theme()
	a.mu.Lock()
	title := a.video.playing
	hasSession := a.playerSess != nil
	a.mu.Unlock()
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Padding(8, 12).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, title).FontSize(13).TextColor(t.TextMuted).SingleLine().Grow(1)
			if ui.Button(c, "关闭播放").Disabled(!hasSession).Clicked() {
				if session := a.playerSess; session != nil {
					session.Close()
					a.playerSess = nil
				}
				if a.playerView != nil {
					_, _ = a.playerView.Page().Eval("document.getElementById('v').pause(); 0")
				}
				if a.playerWin != nil {
					a.playerWin.Close()
				}
			}
		})
		ui.Divider(c)
		if a.playerView != nil {
			ui.WebView(c, a.playerView).Grow(1)
		}
	})
}

// saveVideoProgress stores a position report. It runs on the player server's
// goroutine, so it touches only the shelf, which has its own lock.
func (a *App) saveVideoProgress(meta player.Meta, progress player.Progress) {
	if a.shelf == nil || meta.ItemID == "" || progress.Time <= 0 {
		debugLog("video progress skipped: shelf=%v meta=%+v progress=%+v", a.shelf != nil, meta, progress)
		return
	}
	// Finished means the credits, not "a short clip is nearly over": the last
	// two percent within ten seconds of the end. Anything earlier is a
	// position worth resuming, including on a ten-second fixture.
	if progress.Paused && videoFinished(progress.Time, progress.Duration) {
		_ = a.shelf.ForgetVideo(meta.Source, meta.ItemID, meta.EpisodeID)
		return
	}
	if err := a.shelf.RecordVideo(shelf.VideoHistory{
		Source: meta.Source, ItemID: meta.ItemID, Title: meta.Title,
		LineID: meta.LineID, LineName: meta.LineName,
		EpisodeID: meta.EpisodeID, EpisodeName: meta.EpisodeName,
		Position: progress.Time, Duration: progress.Duration,
	}); err != nil {
		debugLog("video progress write failed: %v", err)
	}
}

// savedVideoSubtitle answers with the track already chosen for this episode.
func (a *App) savedVideoSubtitle(meta player.Meta) (player.Subtitle, bool) {
	if a.shelf == nil || meta.ItemID == "" {
		return player.Subtitle{}, false
	}
	choice, ok := a.shelf.VideoSubtitle(meta.Source, meta.ItemID, meta.EpisodeID)
	if !ok {
		return player.Subtitle{}, false
	}
	return player.Subtitle{Off: choice.Off, Name: choice.Name, Language: choice.Language}, true
}

// saveVideoSubtitle stores a choice without adding an episode to history.
func (a *App) saveVideoSubtitle(meta player.Meta, choice player.Subtitle) {
	if a.shelf == nil || meta.ItemID == "" {
		return
	}
	stored := &shelf.VideoSubtitle{Off: choice.Off, Name: choice.Name, Language: choice.Language}
	if ok, err := a.shelf.SetVideoSubtitle(meta.Source, meta.ItemID, meta.EpisodeID, stored); err != nil {
		debugLog("video subtitle write failed: %v", err)
	} else if !ok {
		debugLog("video subtitle skipped: episode not in history yet")
	}
}

// videoFinished is shared by the save and resume paths; otherwise a short
// clip can keep a valid position on disk but still start from zero.
func videoFinished(position, duration float64) bool {
	return duration > 0 && duration-position <= 10 && position/duration >= 0.98
}

// runVideoProbe drives the whole video path with no human clicking: search,
// details, play, then asks the player page what it decoded and saves a frame
// read back from the video element. It is only used by the development tools
// (cmd/uicheck), never by the app's own flows.
func (a *App) runVideoProbe(query, out string, quit bool) {
	report := func(ok bool, detail string) {
		a.mu.Lock()
		if !ok {
			a.video.probeErr = fmt.Errorf("video probe: %s", detail)
		}
		a.mu.Unlock()
		fmt.Printf("video-probe: ok=%v %s\n", ok, detail)
		if quit {
			mygo.App.Quit()
		}
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				report(false, fmt.Sprintf("panic: %v", r))
			}
		}()
		time.Sleep(700 * time.Millisecond)
		a.openVideo()
		a.mu.Lock()
		a.video.query = query
		a.mu.Unlock()
		a.videoWin.Update(func() {})
		a.videoSearch()
		if !a.waitFor(60, func() bool {
			a.mu.Lock()
			defer a.mu.Unlock()
			return len(a.video.results) > 0 || a.video.errText != ""
		}) {
			report(false, "the video search never answered")
			return
		}
		a.mu.Lock()
		first, searchErr := video.Item{}, a.video.errText
		if len(a.video.results) > 0 {
			first = a.video.results[0]
		}
		a.mu.Unlock()
		if searchErr != "" || first.ID == "" {
			report(false, "the video search failed: "+searchErr)
			return
		}
		a.openVideoDetails(first)
		if !a.waitFor(60, func() bool {
			a.mu.Lock()
			defer a.mu.Unlock()
			return a.video.details != nil || a.video.detailErr != ""
		}) {
			report(false, "the video details never answered")
			return
		}
		a.mu.Lock()
		details, detailErr := a.video.details, a.video.detailErr
		a.mu.Unlock()
		if detailErr != "" || details == nil || len(details.Lines) == 0 || len(details.Lines[0].Episodes) == 0 {
			report(false, "the video details had no episode: "+detailErr)
			return
		}
		a.playVideo(0)
		if !a.waitFor(80, func() bool {
			a.mu.Lock()
			defer a.mu.Unlock()
			return a.playerView != nil || a.video.detailErr != ""
		}) {
			report(false, "the player window never opened")
			return
		}
		a.mu.Lock()
		playErr := a.video.detailErr
		view := a.playerView
		a.mu.Unlock()
		if playErr != "" || view == nil {
			report(false, "playback failed: "+playErr)
			return
		}
		// The page is the only thing that knows whether pixels decoded, so ask
		// it, twice, a second apart: a still frame and a running clock.
		var firstState, secondState string
		for i := 0; i < 40; i++ {
			time.Sleep(time.Second)
			raw, err := view.Page().Eval(`(() => {
				const v = document.getElementById('v');
				const failure = document.getElementById('err');
				if (!v || !failure) return JSON.stringify({ready: 0, w: 0, h: 0,
					duration: 0, time: 0, err: '', phase: 'loading-document'});
				return JSON.stringify({ready: v.readyState, w: v.videoWidth, h: v.videoHeight,
					duration: v.duration, time: v.currentTime, err: failure.textContent});
			})()`)
			if err != nil {
				report(false, "reading the player page failed: "+err.Error())
				return
			}
			state, _ := raw.(string)
			firstState = state
			var observation struct {
				Err string `json:"err"`
			}
			if err := json.Unmarshal([]byte(state), &observation); err != nil {
				report(false, "the page returned no playback state")
				return
			}
			if observation.Err != "" {
				report(false, "the player reported: "+observation.Err)
				return
			}
			if strings.Contains(state, `"w":0`) {
				continue
			}
			_, _ = view.Page().Eval("document.getElementById('v').play(); 0")
			time.Sleep(2 * time.Second)
			raw2, _ := view.Page().Eval(`(() => {
				const v = document.getElementById('v');
				return JSON.stringify({ready: v.readyState, w: v.videoWidth, h: v.videoHeight,
					duration: v.duration, time: v.currentTime, err: document.getElementById('err').textContent});
			})()`)
			secondState, _ = raw2.(string)
			break
		}
		if secondState == "" {
			report(false, "the player never decoded the media: "+firstState)
			return
		}
		decoder, _ := view.Page().Eval("document.getElementById('v').dataset.decoder || ''")
		audioDeclared, _ := view.Page().Eval("document.getElementById('v').dataset.audioDeclared || 'unknown'")
		// Inspect active cues on the showing track. Cached cues on a disabled
		// track cannot prove the right subtitles are selected at this time.
		// Video frame readback excludes the browser's text-track overlay.
		subtitleEval, _ := view.Page().Eval(`(function(v){
			var cues = 0, text = '', modes = [];
			for (var i = 0; i < v.textTracks.length; i++) {
				var t = v.textTracks[i];
				modes.push(t.mode);
				if (t.mode !== 'showing') continue;
				var c = t.activeCues;
				if (!c) continue;
				cues += c.length;
				if (!text && c.length) text = String(c[0].text || '').replace(/\s+/g, ' ').trim();
			}
			return [cues, text, modes.join(',')];
		})(document.getElementById('v'))`)
		subtitleCues, subtitleText, subtitleModes := 0.0, "", ""
		if parts, ok := subtitleEval.([]interface{}); ok && len(parts) == 3 {
			subtitleCues, _ = parts[0].(float64)
			subtitleText, _ = parts[1].(string)
			subtitleModes, _ = parts[2].(string)
		}
		audioCounter, _ := view.Page().Eval("typeof document.getElementById('v').webkitAudioDecodedByteCount === 'number' ? document.getElementById('v').webkitAudioDecodedByteCount : -1")
		audioBytes, counterAvailable := audioCounter.(float64)
		if a.dev.VideoRequireSubtitles {
			if subtitleCues <= 0 || subtitleText == "" {
				report(false, fmt.Sprintf("no active subtitle text: cues=%v modes=%v", subtitleCues, subtitleModes))
				return
			}
			if want := a.dev.VideoSubtitleText; want != "" && !strings.Contains(subtitleText, want) {
				report(false, fmt.Sprintf("wrong subtitle: want=%q got=%q", want, subtitleText))
				return
			}
		}
		if a.dev.VideoRequireAudio && (!counterAvailable || audioBytes <= 0) {
			report(false, fmt.Sprintf("audio decoding was not observed: declared=%v decodedBytes=%v", audioDeclared, audioCounter))
			return
		}
		if a.dev.VideoForceHLS && decoder != "hls.js" {
			report(false, fmt.Sprintf("the HLS.js probe used a different decoder: %v", decoder))
			return
		}
		var before, after struct {
			Ready    float64 `json:"ready"`
			W        float64 `json:"w"`
			H        float64 `json:"h"`
			Duration float64 `json:"duration"`
			Time     float64 `json:"time"`
			Err      string  `json:"err"`
		}
		_ = json.Unmarshal([]byte(firstState), &before)
		_ = json.Unmarshal([]byte(secondState), &after)
		if after.Err != "" {
			report(false, "the player reported: "+after.Err)
			return
		}
		if after.W <= 0 || after.Ready < 2 || after.Time <= before.Time {
			report(false, "the media did not play: before="+firstState+" after="+secondState)
			return
		}
		// A frame read back from the video element is the difference between
		// "the element has a size" and "pixels were decoded".
		if out != "" {
			raw, err := view.Page().Eval(`(() => {
				const v = document.getElementById('v');
				const c = document.createElement('canvas');
				c.width = v.videoWidth; c.height = v.videoHeight;
				c.getContext('2d').drawImage(v, 0, 0);
				return c.toDataURL('image/png');
			})()`)
			if err != nil {
				report(false, "reading a frame failed: "+err.Error())
				return
			}
			data, _ := raw.(string)
			if !strings.HasPrefix(data, "data:image/png;base64,") {
				report(false, "the video element produced no frame")
				return
			}
			png, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, "data:image/png;base64,"))
			if err != nil || len(png) < 1000 {
				report(false, "the frame was empty")
				return
			}
			if err := os.WriteFile(out, png, 0o644); err != nil {
				report(false, "writing the frame failed: "+err.Error())
				return
			}
		}
		// Pausing makes the page report its position; the shelf is what the
		// window would resume from next time.
		_, _ = view.Page().Eval("document.getElementById('v').pause(); 0")
		key := a.videoKey()
		episodeID := details.Lines[0].Episodes[0].ID
		var saved shelf.VideoHistory
		if !a.waitFor(60, func() bool {
			entry, ok := a.shelf.VideoProgress(key, first.ID, episodeID)
			if ok && entry.Position >= after.Time-0.4 && entry.Position > 0 {
				saved = entry
				return true
			}
			return false
		}) {
			report(false, "the pause event never saved the actual position")
			return
		}
		// Read a fresh shelf from disk, then use the real Continue Watching
		// action. Never synthesize a progress POST or set currentTime here.
		reopened, err := shelf.Open(a.core.DataDir())
		if err != nil {
			report(false, "reopening persisted history failed: "+err.Error())
			return
		}
		disk, ok := reopened.VideoProgress(key, first.ID, episodeID)
		if !ok || disk.Position < after.Time-0.4 {
			report(false, fmt.Sprintf("the disk position did not match the page: %+v", disk))
			return
		}
		oldURL, _ := view.Page().Eval("location.href")
		a.videoResume(saved)
		var resumed float64
		if !a.waitFor(100, func() bool {
			currentURL, err := view.Page().Eval("location.href")
			if err != nil || currentURL == oldURL {
				return false
			}
			position, err := view.Page().Eval("document.getElementById('v') ? document.getElementById('v').currentTime : 0")
			resumed, _ = position.(float64)
			return err == nil && resumed >= disk.Position-0.3
		}) {
			report(false, fmt.Sprintf("resume failed: saved=%.3f observed=%.3f", disk.Position, resumed))
			return
		}
		report(true, fmt.Sprintf("frame=%s decoder=%v audioDeclared=%v audioDecodedBytes=%v activeSubtitleCues=%v subtitle=%q subtitleModes=%v size=%vx%v duration=%.1f time=%.2f->%.2f disk=%s@%.3fs reopened=%.3fs",
			out, decoder, audioDeclared, audioCounter, subtitleCues, subtitleText, subtitleModes, after.W, after.H, after.Duration, before.Time, after.Time,
			disk.EpisodeName, disk.Position, resumed))
	}()
}

// VideoProbeError lets the development command return a failing exit status
// when its actual playback/progress/resume observation failed.
func (a *App) VideoProbeError() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.video.probeErr
}

// waitFor polls a condition for up to n tenths of a second.
func (a *App) waitFor(tenths int, done func() bool) bool {
	for i := 0; i < tenths; i++ {
		if done() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return done()
}
