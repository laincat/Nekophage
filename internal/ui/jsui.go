package ui

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/laincat/veneracat-core/comic/js"
)

// jsUI gives a source the UI it asks for: a message, a dialog with buttons, an
// input, a select, a loading spinner.
//
// Without this every source that calls UI.showMessage or UI.showDialog failed
// with "No JavaScript UI handler provided" — ccc's login, wnacg's domain
// refresh, happy's cache wipe. It is not cosmetic: several sources do their work
// from inside a dialog callback, so the button press is what starts the work.
type JSHandler struct {
	app *App
	// autoConfirm is set by the capture tools, which cannot press a button in a
	// window they are photographing. Nothing in a normal run sets it.
	autoConfirm bool

	mu      sync.Mutex
	loading map[int]func()
	nextID  int
	// last is the most recent dialog window, so a capture tool can photograph
	// what a source put on screen.
	last *mygo.Window
}

// NewJSHandler builds the handler before the App exists, because Core needs it
// when it starts: a source may call UI during init. Attach hands it the App once
// the window is there.
func NewJSHandler() *JSHandler {
	return &JSHandler{loading: map[int]func(){}}
}

// Attach points the handler at the window it reports through.
func (h *JSHandler) Attach(app *App) {
	h.app = app
	app.SetJSUI(h)
}

// LastWindow is the most recent dialog window, for the capture tools: a source's
// dialog is otherwise invisible to a test.
func (h *JSHandler) LastWindow() *mygo.Window {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.last
}

// LoadingCount is how many spinners the source is waiting on; the window shows
// one line while any is open.
func (h *JSHandler) LoadingCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.loading)
}

// SetAutoConfirm makes the capture tools take the first dialog action, since a
// tool cannot press a button in the window it is photographing.
func (h *JSHandler) SetAutoConfirm(on bool) {
	h.autoConfirm = on
}

// ShowMessage puts a line in the main window; a source's "done" message is not
// worth a window of its own.
func (h *JSHandler) ShowMessage(message string) {
	// A handler can exist before the window and outlive it; a message with
	// nowhere to go is dropped rather than crashing the source's call.
	if h.app == nil {
		return
	}
	h.app.setNotice(message)
	if h.app.win != nil {
		h.app.win.Update(func() {})
	}
}

// LaunchURL opens a link in the system browser.
func (h *JSHandler) LaunchURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("nekophage: opening %s: %v", url, err)
	}
}

// ShowDialog shows buttons and waits for one to be pressed. The action's own
// Invoke runs the script callback, which is where the source does its work.
func (h *JSHandler) ShowDialog(title, content string, actions []js.DialogAction) error {
	if len(actions) == 0 {
		// Nothing to press: show it and return, rather than open a window that
		// can never be dismissed.
		h.ShowMessage(content)
		return nil
	}
	if h.autoConfirm {
		// The capture tools take the first action, which is the confirmation.
		// It runs off this goroutine: Invoke calls back into the script, and the
		// script is the caller sitting in this very call.
		done := make(chan error, 1)
		go func() { done <- invokeSafely(actions[0]) }()
		return <-done
	}
	done := make(chan error, 1)
	var once sync.Once
	finish := func(err error) { once.Do(func() { done <- err }) }
	h.openWindow(title, 460, 240, func(c *ui.Context, close func()) {
		ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
			if content != "" {
				ui.Scroll(c).Grow(1).Children(func() {
					ui.Text(c, content).Selectable()
				})
			}
			ui.Row(c).Gap(8).Children(func() {
				// Buttons sit at the trailing edge, the way a desktop dialog
				// does, so the eye finds the confirmation where it expects it.
				ui.Spacer(c)
				for _, action := range actions {
					action := action
					label := action.Text
					if label == "" {
						label = "确定"
					}
					var clicked bool
					if action.Style == "filled" {
						clicked = ui.PrimaryButton(c, label).Clicked()
					} else {
						clicked = ui.Button(c, label).Clicked()
					}
					if !clicked {
						continue
					}
					close()
					go func() { finish(invokeSafely(action)) }()
					return
				}
			})
		})
	}, func() { finish(nil) })
	return <-done
}

// ShowLoading returns an id the source cancels with; the main window shows it.
func (h *JSHandler) ShowLoading(onCancel func()) int {
	h.mu.Lock()
	id := h.nextID
	h.nextID++
	h.loading[id] = onCancel
	h.mu.Unlock()
	if h.app != nil && h.app.win != nil {
		h.app.win.Update(func() {})
	}
	return id
}

// CancelLoading drops a spinner the source no longer needs.
func (h *JSHandler) CancelLoading(id int) {
	h.mu.Lock()
	delete(h.loading, id)
	h.mu.Unlock()
	if h.app != nil && h.app.win != nil {
		h.app.win.Update(func() {})
	}
}

// ShowInputDialog asks for one line of text. A nil result means the window was
// closed, which the source sees as null.
func (h *JSHandler) ShowInputDialog(title, image string, validate func(string) string) (*string, error) {
	if h.autoConfirm {
		return nil, nil
	}
	text := ""
	// A source may hand over an image — a captcha, a QR code — and the field is
	// useless without it. It is fetched through the source's own thumbnail rules,
	// so its headers and any image rewriting apply here too.
	// A source may hand over an image — a captcha, a QR code — and the field is
	// useless without it. The bytes are fetched through the source's own
	// thumbnail rules, so its headers and any image rewriting apply here too.
	picture := &dialogPicture{}
	if image != "" && h.app != nil {
		picture.url = image
	}
	result := make(chan *string, 1)
	var once sync.Once
	finish := func(value *string) { once.Do(func() { result <- value }) }
	win := h.openWindow(title, 420, 220, func(c *ui.Context, close func()) {
		ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
			if title != "" {
				ui.Text(c, title).FontSize(15).Bold()
			}
			if b := picture.bitmap(); b != nil {
				// An explicit width: inside a filled column a height alone can
				// work out to no width at all, and the image then vanishes.
				ui.Image(c, b).Fit(ui.Contain).Width(200).Height(96).Radius(4)
			}
			ui.TextInput(c, &text).Grow(1)
			// The source's validator decides; its message goes under the field.
			if hint := safeValidate(validate, text); hint != "" {
				ui.Text(c, hint).FontSize(11).TextColor(ui.RGB(224, 49, 49))
			}
			ui.Row(c).Gap(8).Children(func() {
				if ui.PrimaryButton(c, "确定").Clicked() {
					if hint := safeValidate(validate, text); hint == "" {
						value := text
						close()
						finish(&value)
					}
				}
				if ui.Button(c, "取消").Clicked() {
					close()
					finish(nil)
				}
			})
		})
	}, func() { finish(nil) })
	// The picture arrives after the window is up, and the window has to be told
	// to paint again: the fetch does not touch this window.
	if picture.url != "" {
		go h.fetchDialogImage(picture, win)
	}
	return <-result, nil
}

// fetchDialogImage gets the picture a source asked to show and repaints the
// dialog it belongs to.
//
// The bytes are kept by the handler rather than in the main window's image
// cache: that cache is read by the main window's thread, and this one is
// written here, so sharing it would be a race. The dialog decodes its own copy.
func (h *JSHandler) fetchDialogImage(picture *dialogPicture, win *mygo.Window) {
	if h.app == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	data, _, err := h.app.core.FetchThumbnail(ctx, h.app.currentKey(), picture.url, "jsinput")
	if err == nil {
		picture.set(data)
	}
	if win != nil && !win.IsDestroyed() {
		win.Update(func() {})
	}
}

// dialogPicture holds the image a source asked to show beside a field, and
// decodes it once.
type dialogPicture struct {
	url string

	mu      sync.Mutex
	data    []byte
	decoded ui.ImageSource
}

// set stores the downloaded bytes.
func (p *dialogPicture) set(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.data = data
	p.decoded = nil
}

// bitmap decodes the bytes once, on the window's thread. It returns nil while
// the download is still running or after a failure.
func (p *dialogPicture) bitmap() ui.ImageSource {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.decoded != nil {
		return p.decoded
	}
	if len(p.data) == 0 {
		return nil
	}
	b, err := ui.DecodeBitmap(p.data)
	if err != nil {
		// A corrupt picture is remembered as a failure instead of being decoded
		// again on every frame.
		p.data = nil
		return nil
	}
	p.decoded = b
	return b
}

// ShowSelectDialog asks for one of a list and returns its index.
func (h *JSHandler) ShowSelectDialog(title string, options []string, initialIndex *int) (*int, error) {
	if len(options) == 0 {
		return nil, nil
	}
	start := 0
	if initialIndex != nil && *initialIndex >= 0 && *initialIndex < len(options) {
		start = *initialIndex
	}
	if h.autoConfirm {
		index := start
		return &index, nil
	}
	pick := options[start]
	result := make(chan *int, 1)
	var once sync.Once
	finish := func(value *int) { once.Do(func() { result <- value }) }
	h.openWindow(title, 420, 220, func(c *ui.Context, close func()) {
		ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
			if title != "" {
				ui.Text(c, title).FontSize(15).Bold()
			}
			ui.Select(c, &pick, options).Grow(1).Changed()
			ui.Row(c).Gap(8).Children(func() {
				if ui.PrimaryButton(c, "确定").Clicked() {
					index := start
					for i, option := range options {
						if option == pick {
							index = i
							break
						}
					}
					close()
					finish(&index)
				}
				if ui.Button(c, "取消").Clicked() {
					close()
					finish(nil)
				}
			})
		})
	}, func() { finish(nil) })
	return <-result, nil
}

// openWindow shows a window that draws through draw and closes through close.
// onClosed runs when the user closes it without pressing anything, so the
// source's promise settles instead of hanging.
func (h *JSHandler) openWindow(title string, width, height int, draw func(c *ui.Context, close func()), onClosed func()) *mygo.Window {
	closed := atomic.Bool{}
	var win *mygo.Window
	closeFn := func() {
		if closed.Swap(true) {
			return
		}
		if win != nil {
			win.Close()
		}
	}
	win = mygo.NewWindow(mygo.WindowOptions{
		Title:   title,
		Width:   width,
		Height:  height,
		Content: ui.View(func(c *ui.Context) { draw(c, closeFn) }),
	})
	h.mu.Lock()
	h.last = win
	h.mu.Unlock()
	win.OnClosed(func() {
		if !closed.Swap(true) {
			onClosed()
		}
	})
	win.Show()
	win.Focus()
	return win
}

// invokeSafely runs a dialog action's script callback.
//
// The callback is source code: it can throw, and a panic here would leave the
// caller waiting on a dialog nobody can answer, because the goroutine that owed
// it a result is gone. A panic becomes an error instead.
func invokeSafely(action js.DialogAction) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("源内的操作失败:%v", r)
		}
	}()
	if action.Invoke == nil {
		return nil
	}
	return action.Invoke()
}

// safeValidate runs the source's validator, treating a panic as invalid input
// rather than taking the window down with it.
func safeValidate(validate func(string) string, text string) (hint string) {
	if validate == nil {
		return ""
	}
	defer func() {
		if recover() != nil {
			hint = "输入无效"
		}
	}()
	return validate(text)
}
