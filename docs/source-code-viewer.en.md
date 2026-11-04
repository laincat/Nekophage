# Source code viewer

[简体中文](source-code-viewer.md) | English

Each installed JavaScript source and Kazumi rule in the source manager has a "查看源码" (view source) button. It opens a read-only copy with MyGo v0.7.3's own `ui/editor`: JavaScript sources use the JavaScript language configuration, Kazumi rules use JSON. MyGo is not upgraded and no dependency is added.

This is not an editor:

- The text can be selected, copied, searched and folded, but not changed. `ReadOnly` blocks typing, pasting and accessibility writes.
- Closing the window releases that viewer's editor state; a background load that finishes late is discarded.
- It never re-runs, installs or saves a source. Reopening reads the current installed record, so an updated source shows its new body.

The implementation is `internal/ui/sourcecode.go`, reached from "查看源码" in `internal/ui/manage.go`. `internal/ui/app.go` keeps the open viewer so the source manager does not rebuild the editor every frame. The background read returns through `Window.Update`.

Known limits:

- The snapshot is taken when the window opens and does not follow outside updates during that opening.
- Closing the source manager does not close the source window; each closes on its own.
- Prior Mac development-build results retain their earlier Core scope. Windows native-window results for the current combination follow below. System clipboard, real operating-system composition cancellation and installed-build source viewing remain unverified.

## Windows native-window validation (2026-11-04)

The Lead drove actual Windows windows through MyGo automation in an isolated2b38049/Core313951e e2e build. Comic JavaScript, video JavaScript and Kazumi JSON bodies exactly matched installed records. Explicit setValue, text input and Backspace could not modify the read-only content. Switching sources closed the previous viewer; closing and reopening source management retained and reused the same viewer.

The actual source-manager URL-install path updated the comic from1.0.0 to2.0.0 through one loopback request. The open viewer retained1.0.0; closing and reopening showed2.0.0. Native WM_CLOSE shut down only task-owned windows, and both independent processes exited0. The second process restored the2.0.0 body, with other source bodies/types/enabled states preserved. Logs/screenshots: `E:/Temp/nk-source-viewer-windows/`. This used an e2e test build; release builds do not import the automation plugin. System clipboard, real IME and macOS windows with the new Core combination remain unverified; the Orca Worker's prior Mac results still belong to the earlierccb candidate.
