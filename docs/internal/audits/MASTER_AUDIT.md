# Documentation Audit Summary (October 20, 2025)

## 1. Overview

The documentation set is comprehensive but several high-visibility guides are out of sync with the current CLI/API behaviour. The most urgent fixes are correcting CLI flag coverage, removing instructions for non-existent --ai flags, and aligning safety-level descriptions with the actual consolidation rules. Internal audit notes also point contributors toward the wrong fixes, so they need a refresh alongside the public docs.

## 2. Top Documentation Gaps & Mismatches

- **`docs/user docs/cli-reference.md:14`**: Still lists `--tui` as a global flag even though only `analyze` and `squash` register it (`internal/cli/root.go:231-243`). The CLI reference also lacks any entry for the dedicated `tui` command hierarchy (`internal/cli/tui.go:12-58`).
- **`docs/user docs/ai-features.md:342-427`**: Teaches users to run `pgsquash … --ai` (including `--ai --dry-run` and redaction switches), but no CLI flag named `--ai` exists (`internal/cli/root.go:231-323`), so those workflows fail.
- **`docs/user docs/environment-variables.md:14-25`**: Introduces `PGSQUASH_DOCKER_NETWORK` and `PGSQUASH_CONFIG_PATH` as generally supported variables, yet the Go code never reads them—only Docker helper scripts do—and `DefaultConfig` only honours `PROD_DB_DSN` from the environment (`internal/config/config.go:1-137`). The same page claims env vars override config (`docs/user docs/environment-variables.md:328-342`), but there is no CLI path that reads `PGSQUASH_SAFETY_LEVEL`; it appears solely in container entrypoints.
- **`docs/user docs/safety-levels.md:49-75`**: Describes Paranoid mode as “minimal changes / no DROP→CREATE removal,” but the rule engine applies the same aggressive rules plus dead-code removal for Paranoid (`internal/squasher/engine.go:320-368`), so the guide misrepresents risk.
- **`docs/user docs/advanced-features.md:44-156`**: Shows helper APIs such as `builder.Build()`, `builder.GetErrors()`, `tracker.ProcessMigration(ctx, …)` and `tracker.GetAllLifecycles()`, yet the real APIs expose `String()`, `Errors()` and `ProcessMigration(*types.Migration, int)` with no context or error return (`internal/builder/sql.go:66-188`, `internal/tracking/unified_tracker.go:659-720`).
- **`docs/user docs/error-reference.md:48-49`**: References a `--strict` mode that the CLI does not implement (no mention anywhere under `internal/cli`).
- **`docs/user docs/github-webhooks.md:25-28`**: Promises “Blocks merging – failed validation = no merge,” but the webhook path only posts PR comments and never sets commit statuses or required checks (`internal/github/webhook.go:200-366`, `internal/github/client.go:1-200`).
- **`docs/internal/audits/ACTIONABLE_CHECKLIST.md:16-74`**: Directs contributors to add `--tui` to global options and references non-existent files like `internal/tracking/tracker.go`, so the internal action plan is now misleading.

## 3. Missing Coverage / Cleanup Opportunities

- The environment-variable guide should call out required API server secrets such as `JWT_SECRET` and `DATABASE_URL` (`cmd/api-server/main.go:73-105`), ideally separating CLI variables from API-server-only settings.
- Multiple audit artefacts under `docs/internal/audits/` overlap (Actionable Checklist, INDEX, EXECUTIVE_SUMMARY). Consider folding them into a single maintained status doc to avoid divergent instructions.
- Quick-reference docs could link readers to the existing `cmd/api-server/README.md` for full REST docs; that isn’t discoverable from the `docs/` tree.
- Safety-level documentation needs a truth table derived from `NewSquasherRuleEngine` so that contributor updates stay aligned with changes in consolidation rules.
- Advanced feature examples should be regenerated using the actual Go APIs (e.g., call `builder.String()` and `tracker.GetObjects()`), preventing future drift.

## 4. Recommended Next Steps

1.  **Update the CLI reference**: drop `--tui` from “Global Options,” add a full `tui` command section, and mention the real global switches (`internal/cli/root.go:216-275`).
2.  **Rewrite AI usage docs** to focus on `ai-test`, `ai-demo`, `ai-fix`, config flags, and workflow presets instead of non-existent CLI flags.
3.  **Split the environment-variable guide** into “CLI” vs “API server” sections, document the required server secrets, and remove unsupported variables.
4.  **Refresh safety-levels.md** with a matrix generated from `internal/squasher/engine.go` so Paranoid/Aggressive differences are accurate.
5.  **Replace the advanced code snippets** with compilable examples and correct API names (`internal/builder/sql.go`, `internal/tracking/unified_tracker.go`).
6.  **Delete or rewrite stale guidance** in `docs/internal/audits/ACTIONABLE_CHECKLIST.md` so it no longer prescribes incorrect fixes.
