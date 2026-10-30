// Nekophage is the desktop companion of Nekularity: read comics and watch
// video from Venera-compatible sources.
//
// The main window is native UI drawn by MyGo from Go; the reader is a web
// page in a WebView whose pages Go streams. Business logic lives in
// VeneraCat Core, linked into this binary.
//
//	go build . && ./Nekophage
//
// Building needs the Core checkout scripts/fetch-core.sh makes, or a
// go.work that points at one.
package main

import (
	"context"
	"log"

	"github.com/egoist/mygo"
	"github.com/laincat/nekophage/internal/browser"
	"github.com/laincat/nekophage/internal/cookies"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/nekophage/internal/reader"
	"github.com/laincat/nekophage/internal/ui"
	"github.com/laincat/veneracat-core/host"

	// MyGo's own control strings (placeholders, dialog buttons) in the same
	// language as the rest of the interface.
	_ "github.com/egoist/mygo/locales/zh-Hans"
)

func main() {
	// Sources, settings and cookies live in the per-user data directory.
	dataDir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		log.Fatal(err)
	}

	c, err := core.Open(core.Options{
		DataDir:    dataDir,
		Locale:     "zh_CN",
		AppVersion: sourceVersion,
		// Challenges open in a window of their own; video sources that need
		// a stream sniffed would use the same browser.
		VerifierFactory: func(jar *cookies.Jar) host.Verifier { return browser.NewVerifier(jar) },
	})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// The interface is Chinese throughout, so MyGo's built-in texts follow.
	mygo.App.SetLanguages("zh-Hans")

	lib := library.New(c)
	srv := reader.New(c)
	// The scheme must be registered before the reader's window is made.
	if err := mygo.Protocol.Handle(reader.Scheme, srv); err != nil {
		log.Fatal(err)
	}
	// Sources are restored here rather than in WhenReady: Core compiles a
	// script per source, which belongs off the main thread.
	if failed, err := lib.Restore(context.Background()); err != nil {
		log.Printf("nekophage: restoring sources: %v", err)
	} else {
		for key, err := range failed {
			log.Printf("nekophage: source %s did not restore: %v", key, err)
		}
	}

	if err := ui.New(c, lib, srv).Run(); err != nil {
		log.Fatal(err)
	}
}

// version is the app's own version, shown to the user.
const version = "0.1.0"

// sourceVersion is what sources see through getAppVersion() and what
// minAppVersion is compared with. It is the Venera source API level this
// client implements, not the app's own version: a source that asks for
// minAppVersion 1.3.4 is asking for that API, and reporting 0.1.0 would make
// every such source refuse to install.
const sourceVersion = "1.6.0"
