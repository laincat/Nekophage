package main

// Checks that updating a window after the user closed it is safe.
//
//	go run ./cmd/updateprobe
//
// The sources panel keeps a window handle and background work (a source install,
// a callback) calls Update on it when it finishes. If the user closed the panel
// first, that call happens on a window nobody can see; it must return rather than
// panic or block, because the work that made it has no other way out.

import (
	"fmt"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	done := make(chan string, 1)
	mygo.App.WhenReady(func() {
		go func() {
			win := mygo.NewWindow(mygo.WindowOptions{
				Title: "update probe", Width: 360, Height: 200,
				Content: ui.View(func(c *ui.Context) { ui.Text(c, "probe") }),
			})
			win.Show()
			time.Sleep(time.Second)
			win.Close()
			time.Sleep(time.Second)

			finished := make(chan struct{})
			go func() {
				defer close(finished)
				win.Update(func() {})
				win.Update(func() {})
			}()
			select {
			case <-finished:
				done <- "PASS Update returned after the window closed"
			case <-time.After(5 * time.Second):
				done <- "FAIL Update blocked after the window closed"
			}
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Println("FAIL app:", err)
		os.Exit(1)
	}
	msg := <-done
	fmt.Println(msg)
	if msg[:4] == "FAIL" {
		os.Exit(1)
	}
}
