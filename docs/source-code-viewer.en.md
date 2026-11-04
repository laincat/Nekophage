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
- Only the local Mac development build and an isolated fixture source were verified. Installers, Windows, and composition-cancel with a real operating-system input method were not.
