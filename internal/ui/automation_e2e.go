//go:build e2e

package ui

// Explicit test builds only. The release binary does not expose an automation
// endpoint. MYGO_AUTOMATION must name a private socket to enable this plugin.
import _ "github.com/egoist/mygo/plugins/automation"
