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
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/egoist/mygo"
	"github.com/laincat/nekophage/internal/browser"
	"github.com/laincat/nekophage/internal/collections"
	"github.com/laincat/nekophage/internal/cookies"
	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/library"
	"github.com/laincat/nekophage/internal/portable"
	"github.com/laincat/nekophage/internal/reader"
	"github.com/laincat/nekophage/internal/servicecli"
	"github.com/laincat/nekophage/internal/sourceapi"
	"github.com/laincat/nekophage/internal/ui"
	"github.com/laincat/veneracat-core/host"
	coreversion "github.com/laincat/veneracat-core/version"

	// MyGo's own control strings (placeholders, dialog buttons) in the same
	// language as the rest of the interface.
	_ "github.com/egoist/mygo/locales/zh-Hans"
)

// buildInfo answers "which build is this" without opening a window: the version,
// the source API level sources see, the Go runtime, and the commit when the binary
// was built from a checkout. A report that says "it does not start" is much easier
// to act on with the commit in hand.
func buildInfo() string {
	commit := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				commit = setting.Value
			}
		}
	}
	return fmt.Sprintf("Nekophage %s (Venera source API %s)\nCore %s\ncommit %s\n%s %s/%s",
		version, sourceVersion, coreversion.String(), commit, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func main() {
	showVersion := flag.Bool("version", false, "print the build and exit")
	dataDirOverride := flag.String("data-dir", "", "store application data in this directory")
	collectionAction := flag.String("collection", "", "follow URL, list, refresh ID, or unfollow ID; print JSON without opening a window")
	serviceAction := flag.String("service", "", "call a method of a service package, or print its manifest; print JSON without opening a window")
	flag.Parse()
	if *showVersion {
		fmt.Fprintln(os.Stdout, buildInfo())
		return
	}

	// Set the data folder before any application component resolves its path.
	// An explicit -data-dir wins; otherwise the copy is portable and keeps its
	// data beside the executable, falling back to the per-user profile only
	// when the program folder cannot be written (an install under Program
	// Files, a read-only medium).
	if *dataDirOverride != "" {
		if err := os.MkdirAll(*dataDirOverride, 0o700); err != nil {
			log.Fatal(err)
		}
		mygo.App.SetPath(mygo.PathUserData, *dataDirOverride)
	} else if exe, err := os.Executable(); err == nil {
		root := filepath.Dir(exe)
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			root = resolved
		}
		if dirs, err := portable.Prepare(root); err != nil {
			log.Printf("nekophage: %v; using the per-user profile", err)
		} else {
			// A first portable start copies an installed build's sources, shelf
			// and settings. The old path is worked out by hand: asking MyGo for
			// it would create an empty profile folder on a machine that never
			// had one, which is the trace a portable copy must not leave.
			if config, err := os.UserConfigDir(); err == nil {
				old := filepath.Join(config, mygo.App.Name())
				if n, err := portable.MigrateFrom(old, dirs.Data); err != nil {
					log.Printf("nekophage: copying the earlier profile from %s: %v", old, err)
				} else if n > 0 {
					log.Printf("nekophage: copied %d files from the earlier profile %s", n, old)
				}
			}
			mygo.App.SetPath(mygo.PathUserData, dirs.Data)
			mygo.App.SetPath(mygo.PathCache, dirs.Cache)
			mygo.App.SetPath(mygo.PathLogs, dirs.Logs)
			if err := portable.PointWebViewAt(dirs.WebView, os.Getenv, os.Setenv); err != nil {
				log.Printf("nekophage: keeping the WebView profile in the default place: %v", err)
			}
		}
	}

	// Sources, settings, cookies, plugins and services live here.
	dataDir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		log.Fatal(err)
	}

	if *serviceAction != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		if err := servicecli.RunCommand(ctx, *serviceAction, flag.Args(), dataDir, os.Stdout, "Nekophage/"+version); err != nil {
			log.Fatalf("服务操作失败: %v", err)
		}
		return
	}

	if *collectionAction != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		if err := collections.RunCommand(ctx, dataDir, *collectionAction, flag.Args(), os.Stdout, "Nekophage/"+version); err != nil {
			log.Fatalf("合集操作失败: %v", err)
		}
		return
	}

	// The source UI handler exists before Core, because a source may call UI
	// during init; it is attached to the App once the window exists.
	jsui := ui.NewJSHandler()
	c, err := core.Open(core.Options{
		DataDir:    dataDir,
		Locale:     "zh_CN",
		AppVersion: sourceVersion,
		UI:         jsui,
		// Challenges open in a window of their own; video sources that need
		// a stream sniffed would use the same browser.
		VerifierFactory: func(jar *cookies.Jar) host.Verifier { return browser.NewVerifier(jar) },
	})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// The collection panel tells servers which client this is.
	ui.SetUserAgent("Nekophage/" + version + " (Core " + coreversion.String() + ")")

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

	app := ui.New(c, lib, srv)
	jsui.Attach(app)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// version is the app's own version, shown to the user.
const version = "0.2.31"

// sourceVersion is what sources see through getAppVersion() and what
// minAppVersion is compared with. It is the Venera source API level this
// client implements, not the app's own version; the value and the reasoning
// live in internal/sourceapi so the checking tools report the same level.
const sourceVersion = sourceapi.Level
