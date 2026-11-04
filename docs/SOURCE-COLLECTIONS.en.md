# Shared-source collection subscriptions

[简体中文](SOURCE-COLLECTIONS.md) | English

The desktop client reuses Core ccb6268 for collection parsing and platform/capability matching. The client owns cookie-free downloads, cancellation and profile snapshots. Each publisher provides one cross-platform collection; subscribing tracks its catalogue without automatically downloading, installing or executing plugins, themes, modules or services.

## Application entry point

These commands use the actual Nekophage executable, print JSON and exit without opening the main window. `follow` takes a collection URL; `refresh` and `unfollow` take the collection's declared ID.

```sh
Nekophage -data-dir ./profile -collection follow https://example.org/source.json
Nekophage -data-dir ./profile -collection list
Nekophage -data-dir ./profile -collection refresh publisher.collection
Nekophage -data-dir ./profile -collection unfollow publisher.collection
```

`list` reads the local snapshot offline and retains subscriptions across processes. All entries remain visible; Core produces `compatible` and `missing`, so unsupported platforms, unknown capabilities and design entries are not discarded. macOS maps to platform `macos`; the desktop currently advertises only its existing `comic-source.js` and `video-source.js` hosts. Compatibility is neither authorization nor installation.

## Storage and request behavior

The profile's `collections.json` is separate from `plugins.json`; unfollowing never removes installed sources. Downloads are bounded to 4 MiB each and combined persisted snapshots to 32 MiB. Existing atomic writes preserve both memory and the old file on persistence failure. Storage does not HTML-escape descriptions, preventing valid documents from expanding beyond the download bound and failing to reopen.

Refresh retains the subscribed URL and collection ID. A different ID, invalid JSON, excessive content or HTTP failure preserves the old snapshot. Another URL cannot silently occupy an existing ID. Corrupt or future storage formats return errors without clearing data. Downloads reuse the cookie-free transport and do not read source sign-ins.

Within one Manager, a new refresh cancels the older request for the same URL. Unfollowing and closing the Manager also cancel owned requests. Late responses cannot overwrite newer snapshots or resurrect forgotten subscriptions. The command connects interrupt signals to its request context; the system Ctrl+C gesture was not separately accepted in this round.

As with other desktop profile data, this store currently has one process owner. Serialize commands against the same profile; merging concurrent writes from independent processes is not guaranteed. Different profile directories remain isolated.

## Validation and next steps

Tests use the real Core parser, client Transport and loopback HTTP for restoration, compatibility, preservation of installed records, profile isolation, failed refreshes, write-failure retries, connection cancellation/replacement, unfollow lifecycle, invalid snapshots and HTML-description round trips. The actual desktop build also exercises commands across processes against the shared-source project's seven-entry collection.

This round adds no catalogue GUI, entry installer, module executor or service-plugin host. Existing source installation remains separate. A future catalogue GUI can reuse this Manager, with networking and parsing off the UI thread. Execution results and log locations are recorded in [STATUS](STATUS.en.md).
