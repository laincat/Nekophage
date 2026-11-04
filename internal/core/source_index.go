package core

import (
	"context"

	"github.com/laincat/veneracat-core/plugin"
	"github.com/laincat/veneracat-core/subscription"
)

// InstallComicEntry keeps the index's declared identity authoritative. Core
// checks the downloaded script against that identity before publishing it.
func (c *Core) InstallComicEntry(ctx context.Context, entry subscription.Entry, body, indexURL string) (*plugin.Record, error) {
	return c.reg.InstallFromComicIndex(ctx, indexURL, entry, body)
}
