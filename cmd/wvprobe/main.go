package main

// Reads the browser-login declaration of real sources and asks each one whether a
// page counts as logged in, which is the part the client cannot decide.
//
//	wvprobe SCRIPT...

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/laincat/nekophage/internal/core"
	"github.com/laincat/nekophage/internal/sourceapi"
)

func main() {
	dir, _ := os.MkdirTemp("", "wv")
	defer os.RemoveAll(dir)
	c, err := core.Open(core.Options{DataDir: dir, AppVersion: sourceapi.Level, Locale: "zh_CN"})
	if err != nil {
		panic(err)
	}
	defer c.Close()
	ctx := context.Background()
	for _, path := range os.Args[1:] {
		body, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		rec, err := c.Install(ctx, string(body), "file", "")
		if err != nil {
			fmt.Printf("%-22s install failed: %v\n", filepath.Base(path), err)
			continue
		}
		styles := c.AccountSupport(ctx, rec.Key)
		decl, err := c.AccountWebviewLogin(ctx, rec.Key)
		if err != nil {
			fmt.Printf("%-22s %s: %v\n", filepath.Base(path), rec.Key, err)
			continue
		}
		fmt.Printf("%-22s %s webview=%v url=%q handle=%q\n", filepath.Base(path), rec.Key, styles.Webview, decl.URL, decl.Handle)
		if decl.Handle == "" {
			continue
		}
		// The login page itself must not count as logged in, or the window would
		// close before the user types anything. mycomic is the exception and the
		// source means it: its "login" is passing the Cloudflare challenge, so a
		// real page means success and the window closing is the point.
		onLogin, err := c.WebviewStatus(ctx, rec.Key, decl.Handle, decl.URL, "登录")
		fmt.Printf("    login page counts as done: %v (err %v)\n", onLogin, err)
		// A page the source does not recognise is ordinary, not a failure.
		unknown, err := c.WebviewStatus(ctx, rec.Key, decl.Handle, "https://example.invalid/", "")
		fmt.Printf("    unknown page counts as done: %v (err %v)\n", unknown, err)
	}
}
