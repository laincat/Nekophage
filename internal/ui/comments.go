package ui

import (
	"context"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/nekophage/internal/core"
)

// The comment thread of a comic.
//
// Twenty of the official sources declare `loadComments` and thirteen declare
// `sendComment`; this client called neither, so for those sources the thread was
// simply unreachable. A source that keeps no comments gets no button rather than
// an empty window: "no comments yet" and "this source has no comments" are
// different facts.

// commentsState is one comic's thread.
type commentsState struct {
	win  *mygo.Window
	open bool
	key  string
	id   string
	sub  string

	busy    bool
	errText string
	notice  string
	page    int
	maxPage int
	list    []core.Comment
	draft   string
	// replyTo is the comment being answered; "" for a new thread entry.
	replyTo  string
	replyWho string
	// canSend and canLike are read once when the window opens: asking Core on
	// every repaint would run a source call inside the render path.
	canSend bool
	canLike bool
}

// openComments shows the thread, or focuses it when it is already open for this
// comic.
func (a *App) openComments() {
	key, comic := a.currentKey(), a.detailComic
	c := &a.comments
	if c.win != nil && c.open && c.key == key && c.id == comic.ID {
		c.win.Show()
		c.win.Focus()
		return
	}
	*c = commentsState{key: key, id: comic.ID, sub: a.detail.SubID, page: 1,
		canSend: a.core.CanSendComment(context.Background(), key),
		canLike: a.core.CanLikeComment(context.Background(), key)}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:   "评论",
		Width:   720,
		Height:  620,
		Content: ui.View(func(ctx *ui.Context) { a.commentsView(ctx) }),
	})
	c.win, c.open = win, true
	win.OnClosed(func() { c.open = false })
	win.Show()
	win.Focus()
	a.loadComments(1)
}

// CommentsWindow is the open thread window, or nil.
func (a *App) CommentsWindow() *mygo.Window { return a.comments.win }

// OpenCommentsForCheck opens the thread from the current details, for the
// screenshot tool.
func (a *App) OpenCommentsForCheck() { a.openComments() }

// loadComments reads one page in the background.
func (a *App) loadComments(page int) {
	c := &a.comments
	if page < 1 {
		page = 1
	}
	a.mu.Lock()
	c.busy, c.errText = true, ""
	key, id, sub := c.key, c.id, c.sub
	a.mu.Unlock()
	a.comments.win.Update(func() {})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		res, err := a.core.Comments(ctx, key, id, sub, page, "")
		a.win.Update(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			c.busy = false
			if err != nil {
				c.errText = explain(err)
				return
			}
			c.list, c.page, c.maxPage = res.Comments, page, res.MaxPage
			if c.maxPage > 0 && c.page > c.maxPage {
				c.page = c.maxPage
			}
		})
		a.comments.win.Update(func() {})
	}()
}

// likeComment votes on one comment and reads the page again, because a source
// may report the new score only on the next read.
func (a *App) likeComment(one core.Comment, isLike bool) {
	c := &a.comments
	a.mu.Lock()
	c.busy, c.errText, c.notice = true, "", ""
	key, id, sub := c.key, c.id, c.sub
	a.mu.Unlock()
	a.comments.win.Update(func() {})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		err := a.core.LikeComment(ctx, key, id, sub, one.ID, isLike)
		a.win.Update(func() {
			a.mu.Lock()
			c.busy = false
			if err != nil {
				c.errText = explain(err)
			}
		})
		a.comments.win.Update(func() {})
		if err == nil {
			a.loadComments(c.page)
		}
	}()
}

// sendComment posts the draft and reloads the thread, because sources differ in
// whether they keep the new comment in the response.
func (a *App) sendComment() {
	c := &a.comments
	text := c.draft
	if text == "" {
		return
	}
	a.mu.Lock()
	c.busy, c.errText, c.notice = true, "", ""
	key, id, sub, replyTo := c.key, c.id, c.sub, c.replyTo
	a.mu.Unlock()
	a.comments.win.Update(func() {})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		err := a.core.SendComment(ctx, key, id, sub, text, replyTo)
		a.win.Update(func() {
			a.mu.Lock()
			c.busy = false
			if err != nil {
				c.errText = explain(err)
				return
			}
			c.draft, c.replyTo, c.replyWho = "", "", ""
			c.notice = "已发送"
		})
		a.comments.win.Update(func() {})
		if err == nil {
			a.loadComments(1)
		}
	}()
}

// commentsView draws the thread, its pages and the box to write in.
func (a *App) commentsView(ctx *ui.Context) {
	t := ctx.Theme()
	a.mu.Lock()
	c := &a.comments
	busy, canLike, errText, notice := c.busy, c.canLike, c.errText, c.notice
	list, page, maxPage := c.list, c.page, c.maxPage
	replyTo, replyWho := c.replyTo, c.replyWho
	draft := c.draft
	a.mu.Unlock()

	ui.Column(ctx).Fill().Children(func() {
		ui.Row(ctx).Padding(8, 12).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Textf(ctx, "第 %d 页", page).FontSize(12).TextColor(t.TextMuted)
			if maxPage > 0 {
				ui.Textf(ctx, "/ %d", maxPage).FontSize(12).TextColor(t.TextMuted)
			}
			ui.Spacer(ctx)
			if busy {
				ui.Spinner(ctx)
				ui.Text(ctx, "正在读取…").FontSize(12).TextColor(t.TextMuted)
			}
			if ui.Button(ctx, "上一页").Disabled(busy || page <= 1).Clicked() {
				a.loadComments(page - 1)
			}
			if ui.Button(ctx, "下一页").Disabled(busy || maxPage == 0 || page >= maxPage).Clicked() {
				a.loadComments(page + 1)
			}
			if ui.Button(ctx, "刷新").Disabled(busy).Clicked() {
				a.loadComments(page)
			}
		})
		ui.Divider(ctx)
		if errText != "" {
			ui.Text(ctx, errText).Padding(8, 12).FontSize(12).TextColor(t.Danger).Selectable()
		}
		switch {
		case busy && len(list) == 0:
			ui.Column(ctx).Fill().Center().Children(func() { ui.Spinner(ctx) })
		case len(list) == 0:
			ui.Column(ctx).Fill().Center().Children(func() {
				ui.Text(ctx, "这一页没有评论。").TextColor(t.TextMuted)
			})
		default:
			ui.List(ctx, nil, len(list), func(i int) {
				one := list[i]
				ui.Column(ctx).Padding(8, 12).Gap(2).Children(func() {
					ui.Row(ctx).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(ctx, one.UserName).FontSize(12).Bold()
						if one.Time != "" {
							ui.Text(ctx, one.Time).FontSize(11).TextColor(t.TextMuted)
						}
						if one.Score != "" {
							ui.Textf(ctx, "· %s", one.Score).FontSize(11).TextColor(t.TextMuted)
						}
						if one.IsLiked {
							ui.Text(ctx, "· 已赞").FontSize(11).TextColor(t.Accent)
						}
					})
					ui.Text(ctx, one.Content).FontSize(13).Selectable()
					ui.Row(ctx).Gap(8).Children(func() {
						if one.ReplyCount > 0 {
							ui.Textf(ctx, "%d 条回复", one.ReplyCount).FontSize(11).TextColor(t.TextMuted)
						}
						// A source that votes gets a control; sending false is how
						// the same button takes the vote back.
						if canLike && one.ID != "" {
							label := "赞"
							if one.IsLiked {
								label = "取消赞"
							}
							if ui.Button(ctx, label).Disabled(busy).Clicked() {
								a.likeComment(one, !one.IsLiked)
							}
						}
						if c.canSend {
							if ui.Button(ctx, "回复").Clicked() {
								a.mu.Lock()
								c.replyTo, c.replyWho = one.ID, one.UserName
								a.mu.Unlock()
								a.comments.win.Update(func() {})
							}
						}
					})
				})
			}).Grow(1)
		}
		ui.Divider(ctx)
		ui.Column(ctx).Padding(8, 12).Gap(6).Children(func() {
			if notice != "" {
				ui.Text(ctx, notice).FontSize(11).TextColor(t.Accent)
			}
			if replyTo != "" {
				ui.Row(ctx).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Textf(ctx, "回复 %s", replyWho).FontSize(11).TextColor(t.TextMuted)
					if ui.Button(ctx, "取消").Clicked() {
						a.mu.Lock()
						c.replyTo, c.replyWho = "", ""
						a.mu.Unlock()
						a.comments.win.Update(func() {})
					}
				})
			}
			ui.Row(ctx).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(ctx, &draft).Placeholder("写点什么…").Label("评论内容").Grow(1)
				if busy {
					ui.Spinner(ctx)
				} else if ui.PrimaryButton(ctx, "发送").Disabled(draft == "").Clicked() {
					a.mu.Lock()
					c.draft = draft
					a.mu.Unlock()
					a.sendComment()
				}
			})
			// The draft lives in the state so a repaint does not lose it.
			a.mu.Lock()
			c.draft = draft
			a.mu.Unlock()
		})
	})
}
