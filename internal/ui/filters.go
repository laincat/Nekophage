package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
)

// The filters a source offers are groups of options: search.load receives
// one chosen value per group, so a client that skips them searches with the
// wrong parameters. A group is single-choice ("select", "dropdown") or
// multi-choice ("multi-select"), and a multi-choice value is a JSON array of
// keys written as one string, which is what the official client sends.

// pickLabel names a two-state control.
func pickLabel(on bool, onText, offText string) string {
	if on {
		return onText
	}
	return offText
}

// loadSearchOptions reads the current source's filters and resets the chosen
// values to their defaults.
func (a *App) loadSearchOptions(key string) {
	if key == "" {
		return
	}
	a.mu.Lock()
	ticket := a.searchGen.peek()
	a.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(a.searchGen.requestContext(ticket), 30*time.Second)
		defer cancel()
		groups, err := a.core.SearchOptions(ctx, key)
		if err != nil {
			log.Printf("nekophage: search filters of %s: %v", key, err)
			return
		}
		values := make([]string, len(groups))
		for i, g := range groups {
			values[i] = g.Default
		}
		picked := false
		if spec := a.dev.Filter; spec != "" {
			var index int
			var key string
			if n, err := fmt.Sscanf(spec, "%d=%s", &index, &key); n == 2 && err == nil &&
				index >= 0 && index < len(values) && key != "" {
				values[index] = key
				picked = true
			} else {
				log.Printf("nekophage: filter %q is not index=key", spec)
			}
		}
		a.mu.Lock()
		if !a.searchGen.current(ticket) || a.sourceKey != key {
			a.mu.Unlock()
			return
		}
		a.searchGroups, a.searchValues, a.searchKey = groups, values, key
		query := a.query
		a.mu.Unlock()
		if a.win != nil {
			a.win.Update(func() {})
		}
		if picked && query != "" {
			// The search that ran before the filters arrived used the
			// defaults; run it once more with the choice.
			a.startSearch()
		}
	}()
}

// optionsFor is what search.load should receive for a source: the chosen
// values when they belong to it.
func (a *App) optionsFor(key string) ([]string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.searchKey != key || len(a.searchValues) != len(a.searchGroups) {
		return nil, false
	}
	return append([]string(nil), a.searchValues...), true
}

// multiValues reads a multi-select group's value: a JSON array, or a single
// key as some sources write their default.
func multiValues(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		return out
	}
	// A single key may also arrive JSON-quoted.
	var one string
	if err := json.Unmarshal([]byte(raw), &one); err == nil {
		if one == "" {
			return nil
		}
		return []string{one}
	}
	return []string{raw}
}

// encodeMulti writes a multi-select value the way search.load expects it.
func encodeMulti(values []string) string {
	if len(values) == 0 {
		return ""
	}
	b, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(b)
}

// filtersView draws the source's search filters: one row of chips per group.
func (a *App) filtersView(c *ui.Context, t *ui.Theme) {
	a.mu.Lock()
	groups := append([]core.SearchOptionGroup(nil), a.searchGroups...)
	values := append([]string(nil), a.searchValues...)
	a.mu.Unlock()
	if len(groups) == 0 {
		return
	}
	optionChips(c, t, groups, values, a.chooseOption)
	ui.Divider(c)
}

// optionChips draws one row of chips per option group.
//
// Both a search and a category page take one chosen value per group, so the
// drawing is shared and only what happens on a tap differs: a search reruns
// itself, a category page reloads the entry that is open.
func optionChips(c *ui.Context, t *ui.Theme, groups []core.SearchOptionGroup, values []string,
	pick func(group int, key string, multi bool)) {
	ui.Column(c).Padding(6, 16).Gap(4).Children(func() {
		for i, g := range groups {
			if i >= len(values) {
				break
			}
			i, g := i, g
			label := g.Label
			if label == "" {
				label = "选项 " + itoa(i+1)
			}
			multi := g.Type == "multi-select"
			chosen := values[i]
			selected := func(key string) bool {
				if multi {
					for _, v := range multiValues(chosen) {
						if v == key {
							return true
						}
					}
					return false
				}
				return chosen == key
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, label).FontSize(11).TextColor(t.TextMuted)
				ui.ScrollHorizontal(c).Grow(1).Children(func() {
					ui.Row(c).Gap(4).Children(func() {
						for j, key := range g.Values {
							j, key := j, key
							text := key
							if j < len(g.Labels) && g.Labels[j] != "" {
								text = g.Labels[j]
							}
							if selected(key) {
								if ui.PrimaryButton(c, text).Clicked() {
									pick(i, key, multi)
								}
								continue
							}
							if ui.Button(c, text).Clicked() {
								pick(i, key, multi)
							}
						}
					})
				})
			})
		}
	})
}

func (a *App) chooseOption(group int, key string, multi bool) {
	a.mu.Lock()
	if group < len(a.searchValues) {
		if multi {
			current := multiValues(a.searchValues[group])
			out := make([]string, 0, len(current)+1)
			had := false
			for _, v := range current {
				if v == key {
					had = true
					continue
				}
				out = append(out, v)
			}
			if !had {
				out = append(out, key)
			}
			a.searchValues[group] = encodeMulti(out)
		} else {
			a.searchValues[group] = key
		}
	}
	a.mu.Unlock()
	if a.win != nil {
		a.win.Update(func() {})
	}
}
