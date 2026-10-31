# Magpie on desktop: reuse the local installation

English | [简体中文](MAGPIE-DESKTOP.md)

This is an evaluation, with no implementation changes. The user has chosen to use the locally installed Magpie initially. Windows/macOS are the primary platforms; Linux/Web retain interfaces only. The goal remains provider and other plugin reuse and routing across providers/models by requirements, difficulty, subscription expiry and allowance.

## Initial architecture and responsibilities

Nekophage acts as a client of Magpie's existing local gateway. Magpie continues to own plugins, runtime, provider accounts, login/refresh, the model catalog and routing. The initial integration does not rebuild a plugin host or duplicate provider secrets in Nekophage.

| Part | Responsibility |
|---|---|
| Nekophage | Connection settings, connectivity/capability checks, model or group selection, tasks, streamed results/cancellation and Chinese diagnostics |
| Local Magpie | Original plugin execution/management, provider login/refresh, account selection, supported strategies/groups/fallback, usage and route records |
| Operating system | Normal user installation and startup of Magpie; Nekophage initially does not manage its installation, updates or process lifetime |

Keep the MyGo main UI. Plugin pages, panels and menu interactions initially stay in Magpie's own UI. Calling plugin-backed capabilities through the gateway does not embed those interfaces in Nekophage. Plugin tools require checking the exposed MCP/tool APIs and permissions separately; not every plugin is a model endpoint.

iOS still needs in-app Magpie plugins, accounts and routing across providers, running independently rather than becoming a desktop-gateway client; full routing and compatibility across plugin categories remain unfinished. Desktop reuse does not reduce that iOS goal or require a desktop Magpie. Linux/Web may later implement the same connection interfaces; browser cross-origin behaviour, deployment and credentials are outside initial acceptance.

## Checked interfaces and unverified details

The following comes from local reference source, not a connection test against the user's installed version. Actual interfaces and capabilities must follow probing results.

| Item | Reference implementation | Integration |
|---|---|---|
| Address | Default `127.0.0.1:3425`, changeable through settings or `MAGPIE_ADDR` | Default to `http://127.0.0.1:3425`, allow a custom port and normalise paths to avoid duplicate `/v1` |
| Catalog | `GET /v1/models` | Read exposed model/group IDs and preserve them; do not turn groups back into individual providers |
| Inference | `POST /v1/chat/completions`, `/v1/responses`, `/v1/messages` | Select and verify one protocol for initial tasks; separately test tool, reasoning and image fidelity |
| Local identity | Reference source has local tokens `magpie` and `magpie-<agent>` | Verify against the actual version; a local identifier is neither a provider secret nor a credential for secure remote access |
| Status/records | Reference has `/v1/magpie/*` usage/routing APIs | Optional, version-probed capabilities; initially use Magpie's UI rather than depending on private GUI `/api/*` endpoints |

Connect locally by default. Remote addresses require sharing settings, a real gateway key and transport security and are not enabled implicitly. Only if a gateway key is configured should Windows DPAPI/macOS Keychain store it; provider credentials remain Magpie's. Do not read private Magpie configuration or secret files to bypass public interfaces.

## Connecting requirements to routing

Nekophage sends tasks to the configured model or group; Magpie performs its supported selection and failure handling. Avoid independent retries/provider switching on both sides that duplicate requests or charges.

| Need | Initial approach | Boundary to verify |
|---|---|---|
| Task requirements | Map task types to Magpie models/groups or select explicitly | Actual catalog and requests establish capabilities, context and tool support |
| Difficulty | Map simple/normal/complex tiers to existing groups; call a group already configured with classifier rules | Do not assume every version automatically detects difficulty; complex classification stays managed in Magpie |
| Quota/availability | Reuse Magpie's existing allowance, cooldown, fallback and account policies | Unknown quota is not unlimited; exact strategies depend on the installation and plugin reports |
| Subscription expiry | Check whether the installation already has the required policy or a suitable strategy plugin | Displayed expiry does not establish expiry-first routing; token expiry, resets, expiring credits and subscription ends differ |
| Plugin extensions | Install/enable in Magpie and use its gateway/MCP or its own UI | Not every plugin feature automatically appears inside Nekophage |

For example, map complex tasks to a strong-model group whose ordering is configured in Magpie. If consuming usable allowance from expiring subscriptions first is required but absent in the installation, scope a later strategy plugin/upstream extension instead of silently building another router in the initial adapter. Nekophage must not switch models after output and splice answers. Cancellation closes the current request and must be verified to stop upstream work through Magpie.

## Proposed files

These are planned paths, not implementations created this turn.

| File | Action | Content |
|---|---|---|
| `internal/magpie/client.go` | New | Connection interface, address/version/capability info, replaceable by future platform implementations |
| `internal/magpie/config.go` | New | Local address, protocol, model/group and task-tier mappings, without provider accounts |
| `internal/magpie/models.go` | New | `/v1/models` parsing/refresh, preserving model/group IDs |
| `internal/magpie/request.go`, `stream.go` | New | Requests, SSE, cancellation, timeouts, pre/post-output errors and field fidelity |
| `internal/magpie/errors.go` | New | Chinese diagnostics for unavailable service, wrong address, denied access, missing model and interrupted streams |
| `internal/magpie/secret*.go` | Conditional | DPAPI/Keychain only when a gateway key needs storage |
| `internal/ui/magpie.go` | New | Connection settings/tests, model/group selection, task mappings and management entry |
| `internal/ui/app.go`, `main.go` | Modify | Client wiring, AI feature entry points and request cleanup on exit |
| Related `_test.go`, bilingual README/STATUS | New/modify | Fake-gateway contract tests, actual Windows/macOS installations and usage instructions |

Initially no new Core routing package, Bun IPC, plugin installer, provider login or full routing editor is needed. The previous 28–55-day self-hosted runtime/router estimate no longer applies to this delivery. Such a design remains a separate future evaluation if independence from an external Magpie installation becomes necessary.

## Effort and acceptance

Roughly **4–8 development days** for basic connection support and one actual task entry point, not a delivery commitment. Excludes new AI product features, missing subscription policies, all MCP/plugin-page integration and waits for account authorisation. A protocol unable to preserve required tools/multimodal fields needs a separate estimate extension.

| Stage | Estimate | Acceptance |
|---|---|---|
| Probe local installation | 0.5–1 day | Confirm actual address, version/probeable capabilities, models/groups and authentication behaviour |
| Client and contract tests | 1.5–3 days | Discovery, requests, streaming and cancellation, including unavailable/denied/interrupted cases |
| Settings and task mapping | 1–2 days | Persist settings, select groups and send simple/complex tasks to intended groups without duplicating routing |
| Windows/macOS verification | 1–2 days | Real requests from Nekophage, compared with Magpie's selected providers, route records and cancellation |

Acceptance includes actual selection across providers/accounts and fallback, not merely a successful `/v1/models`. Billable requests use user-selected models/accounts; credentials never enter logs. If installed but stopped, diagnose startup/address requirements instead of reinstalling, upgrading or registering a persistent service.

## Subsequent boundaries

The initial local-installation approach is selected; do not ask again whether to embed Bun or build a host. Before implementation, inspect/probe the actual environment and ask only for connection details that cannot be found. Plugins, accounts and routing rules stay managed in Magpie. After actual validation, consider status panels, MCP adapters, subscription expiry strategies or bundled deployment separately.

Reference source: [gateway routing](https://github.com/yetone/magpie/blob/main/docs/subsystems/gateway-routing.md), [plugin runtime](https://github.com/yetone/magpie/blob/main/docs/subsystems/plugin-runtime.md). This evaluation checks a local reference snapshot, not the installed version.
