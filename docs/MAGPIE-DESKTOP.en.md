# Magpie on the desktop: an evaluation

English | [简体中文](MAGPIE-DESKTOP.md)

Date: 2026-10-31. **This is an evaluation only; no code was changed.** "Magpie" here means the Magpie plugin system (provider plugins), which is unrelated to comic source plugins. The desktop client has neither today.

## Conclusion

**Not recommended right now.** The desktop client is a reader, and the cost of Magpie sits in three places — credential storage, a conversation UI, and the install/sign-in flow — none of which the reader needs. If AI is wanted on the desktop, take **route B (template packages only)**: OpenAI-compatible and Anthropic Messages. Route C, matching iOS feature for feature, costs roughly twice as much, and what it adds is community-package support — of which exactly two packages currently run.

## What already exists

| Capability | State |
|---|---|
| Package loading, relative modules, refusing Node dependencies | In Core; compiles into the desktop binary as-is |
| Authorisation (device code / API key), refresh, usage, check-in, options | In Core |
| Three inbound dialects (chat/responses/messages), streaming chunks, cancellation, retry rules | In Core |
| Per-source cookie partitions and the transport | Already in the desktop client (`internal/cookies`, `internal/core`) |
| Package store, secret store, conversation UI, install entry point | **Nothing exists on the desktop** |

Core's entry point is small: `magpieplugin.Load(bundle)` yields a `Loader`, `Loader.NewInstance(opts)` yields one `Instance` per account, and the host calls `Authorize`/`Poll`/`Chat`/`Usage`. The host implements two interfaces — `Host.Do(ctx, req)` to send one request, and `AuthStore.Get/Set` to keep credentials per provider. iOS's `mobile/magpie.go` is ~500 lines because it crosses the gomobile boundary with JSON; **the desktop calls the Go API directly and does not need that layer.**

## Files that would change

| File | Action | Content |
|---|---|---|
| `internal/magpie/providers.go` | new | Provider records, package directory scan, enable/disable |
| `internal/magpie/package.go` | new | zip/dir → `Bundle` → cached `Loader` (unpack limits come from Core) |
| `internal/magpie/runner.go` | new | `Instance` lifetime, `Chat` events onto a channel for the UI, cancellation |
| `internal/magpie/signin.go` | new | Device-code two-phase and immediate API-key sign-in, matching iOS |
| `internal/magpie/usage.go` | new | Normalised usage (`null` means unreported, not zero) |
| `internal/magpie/options.go` | new | `provider|option` switches; untouched keys keep the package defaults |
| `internal/magpie/auth_windows.go` | new | DPAPI-encrypted file |
| `internal/magpie/auth_unix.go` | new | 0600 file; prefer secret-service on Linux when present |
| `internal/ui/magpie.go` | new | Provider list, install entry point, usage and options panels |
| `internal/ui/magpie_chat.go` | new | Conversation window (MyGo has `TextInput` and `Scroll`) |
| `internal/ui/app.go` | modify | Entry point for the Magpie window and its lifetime |
| `main.go` | modify | Wire store/auth/host, reusing `internal/core`'s transport |
| `docs/STATUS.md`, `README.md` | modify | Record the decision either way |

## Effort

| Part | Estimate | Basis |
|---|---|---|
| Logic (providers/package/runner/signin/usage/options) | 700–900 lines of Go | ~930 lines of Swift on iOS for the same modules, minus the cross-language encoding |
| Credential storage (with platform branches) | 150–250 lines | Two platform implementations plus migration and fallback notes |
| UI (list, install, sign-in, chat, usage) | 900–1200 lines of MyGo | ~785 lines of SwiftUI across five views on iOS |
| Tests | 400–600 lines | Package refusal, authorisation state machine, streaming and cancellation, credential storage |

Roughly **2200–3000 lines** in total, documentation aside. No Core changes are needed.

## Three routes and their trade-offs

| Route | Scope | Cost |
|---|---|---|
| A: do not integrate | The desktop stays a reader | No AI on the desktop; zero maintenance |
| B: template packages only | OpenAI-compatible plus Anthropic Messages; users install their own equivalents | Narrow coverage, but the dialect is declared by the package rather than hard-coded in the host |
| C: match iOS | Community packages and capability detection included | Nearly twice the cost, while most community packages still cannot run because of Node dependencies |

## Risks

| Risk | Detail |
|---|---|
| Credentials at rest | Windows has DPAPI; on Linux without secret-service only 0600 permissions remain, and the UI has to say so |
| Streaming UI | MyGo is immediate-mode, so `Chat` events must be pulled onto the UI thread — the same "pull, do not push" lesson iOS learned |
| Community expectations | Users will expect "install and it works", while most packages are refused at load time; the error must name the missing module |
| Licensing | Magpie itself is MIT, but third-party packages carry their own licences; not bundling them keeps the risk lowest |
| Maintenance surface | A second UI and state machine parallel to iOS: every Magpie behaviour change has to be made twice |

## Recommendation

Choose between **A and B** first, rather than starting from C:

- If the desktop has no near-term need for AI — take A, write "the desktop does not integrate Magpie" into `docs/OPEN-QUESTIONS.md`, and revisit later;
- If the desktop should talk to the user's own gateway — take B, build `providers.go` + `runner.go` + one conversation window, and get "install → API-key sign-in → streaming reply" working with the two template packages before adding usage and options.

Either way, **whether reference packages ship inside the client** is a separate decision (Core's `docs/OPEN-QUESTIONS.md`, item 4): bundling makes the feature work out of the box and takes on the responsibility of distributing third-party code with every release.
