package main

// Closes a source's own dialog the way a person would, and checks the caller is
// released.
//
//	go run ./cmd/dialogprobe
//
// A dialog the source opens is a call that cannot return until it is answered or
// closed. If closing it left the caller waiting, the source would sit there until
// some unrelated deadline, which is the same failure as a closed login window
// still polling — invisible until it happens.

import (
	"fmt"
	"os"
	"time"

	"github.com/egoist/mygo"
	"github.com/laincat/nekophage/internal/ui"
	"github.com/laincat/veneracat-core/comic/js"
)

func main() {
	handler := ui.NewJSHandler()
	// No auto-confirm: the point is that closing the window, not pressing a
	// button, ends the wait.
	handler.SetAutoConfirm(false)

	result := make(chan error, 1)
	var answered bool
	mygo.App.WhenReady(func() {
		go func() {
			err := handler.ShowDialog("Refresh Domain List", "Update Failed\n\nUsing Custom:", []js.DialogAction{
				{Text: "OK", Style: "filled"},
			})
			result <- err
			mygo.App.Quit()
		}()
		go func() {
			// Closing the window stands in for the person who decided the dialog
			// was not worth reading. One fixed sleep was not enough: on a machine
			// already running other windows the dialog sometimes took longer to
			// appear, and the probe reported "a dialog window was never opened"
			// for a dialog that was simply still coming up.
			var win = handler.LastWindow()
			for i := 0; i < 100 && win == nil; i++ {
				time.Sleep(100 * time.Millisecond)
				win = handler.LastWindow()
			}
			if win == nil {
				result <- fmt.Errorf("no dialog window was opened within 10s")
				mygo.App.Quit()
				return
			}
			win.Close()
			answered = true
		}()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Println("FAIL app:", err)
		os.Exit(1)
	}

	if !answered {
		fmt.Println("FAIL a dialog window was never opened")
		os.Exit(1)
	}
	select {
	case err := <-result:
		if err != nil {
			fmt.Println("FAIL closing the dialog reported an error:", err)
			os.Exit(1)
		}
		fmt.Println("PASS the source dialog was released when its window closed")
	case <-time.After(5 * time.Second):
		fmt.Println("FAIL the caller was still waiting after the dialog closed")
		os.Exit(1)
	}
}
