# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

`CLAUDE.md` mirrors this file for Claude Code — keep both in sync when editing.

## What this is

`gote` is a secure, cross-platform desktop note-taking app built with [Tauri v2](https://tauri.app): a Rust backend (crate `gote`, lib `gote_lib`, in `src-tauri/`) paired with a vanilla-JS frontend, packaged as a native webview app. Notes are AES-GCM encrypted at rest with a PBKDF2-derived key from a single app password; no framework (React/Vue/etc.) is used on the frontend by design — keep it that way (see Conventions below).

The on-disk format is the one the original Go implementation wrote, and must stay byte-compatible in both directions: users sync one notes folder across machines that may run different versions. `testdata/go-fixtures/` holds a notes folder produced by the real Go code (password `testpass123`); tests copy it into temp dirs and never modify it.

## Commands

Run from the repo root unless noted. The Tauri CLI is a devDependency of `frontend/` (bun is the package manager; npm is not used).

```sh
bun install --cwd frontend                 # install frontend deps (incl. the Tauri CLI)
frontend/node_modules/.bin/tauri dev       # run the app in dev mode (vite on :1420 + cargo run)
frontend/node_modules/.bin/tauri build     # production build + bundles (src-tauri/target/release/)
frontend/node_modules/.bin/tauri build --no-bundle   # just the binary

cd src-tauri
cargo test                                  # all backend tests (no Tauri runtime needed)
cargo clippy --all-targets -- -D warnings   # lint (CI enforces)
cargo fmt --check                           # formatting (CI enforces)
```

Frontend-only (run from `frontend/`): `bun run dev` (vite dev server), `bun run build` (vite build; also invoked automatically by `tauri build`/`tauri dev`), `bun run preview`.

The Tauri CLI's launcher script needs `node` on `PATH`; on a machine with only bun, put a `node` -> `bun` symlink on `PATH`.

CI workflows:
- Run automatically on push/PR: `typos-spelling.yml` (crate-ci/typos), `pull-request-title.yml`, `rust-check.yml` (PRs: build frontend, `cargo fmt --check`, `clippy -D warnings`, `cargo test`).
- Manual (`workflow_dispatch`) only: `build-release.yml` (releases are triggered by hand).
  - Each release ships a Windows NSIS installer (`gote-windows-amd64-setup.exe`), macOS `.dmg`s, the portable `gote-windows-amd64.exe`, the `.app.zip`s (used by the Homebrew cask in `yggdrion/homebrew-tap`) and a signed `latest.json` for the in-app updater. The build injects the semantic-release version with `tauri build --config '{"version":"…"}'`, so the `1.7.0` in `tauri.conf.json`/`Cargo.toml` is only the local dev version.
  - Updater artifacts are signed with the `TAURI_SIGNING_PRIVATE_KEY` (+ `_PASSWORD`, empty) repo secrets; the matching public key is `plugins.updater.pubkey` in `src-tauri/tauri.conf.json`. Losing the private key means existing installs can never auto-update again, so it is backed up outside the repo. Frontend side: `frontend/src/updater.js` (silent check on startup, "Check for Updates" in Settings).
  - The Tauri CLI aborts if `@tauri-apps/*` npm packages and the `tauri*` crates differ in major/minor; npm is pinned with `~` to the crate minors (crates are held back by `rust-version = "1.85"`) and Renovate groups both sides as `tauri`.

PR titles are enforced by CI and drive semantic-release versioning — they must match:
```
^(feat|fix|chore|docs|refactor|test|perf|build|ci)(\([^)]+\))?:\ 
```

## Architecture

**Layering**: `commands.rs` (Tauri commands) -> `app.rs` (`AppCore` facade) -> `note_store.rs` / `image_store.rs` / `backup.rs` -> `crypto.rs`. Everything except `commands.rs` and `lib.rs` is plain Rust with no Tauri dependency, so it is unit-tested with `cargo test`.

- `src-tauri/src/lib.rs` — `run()`: logger, plugins (clipboard-manager, opener), `Mutex<AppCore>` managed state, command registration, and two background threads (session cleanup every 5 min; auto-backup check at startup once configured, then hourly). `RunEvent::Exit` calls `AppCore::shutdown()` to drop the file watcher.
- `commands.rs` — one `#[tauri::command(async)]` per frontend call (snake_case names, camelCase JS args). Thin wrappers that lock the `AppCore` mutex; `create_backup` releases it before running the backup.
- `app.rs` (`AppCore`) — the semantics of every command: setup/login/logout, `require_auth` (valid sliding session or the error `session expired - re-authentication required`, which also clears the key and cached notes), note/image/settings/backup operations, orphan-image cleanup. After logout or expiry the list commands return `[]` and the decrypted cache is cleared. The derived AES key is held only in memory.
- `note_store.rs` (`NoteStore`) — persistence + in-memory cache. Each note is one encrypted pretty-JSON file `<8-hex-id>.json` (`{id, encrypted_data, created_at, updated_at}`); decrypted payload is JSON `{content, category, original_category?, images?}` with a legacy fallback where the whole plaintext is the content. A `notify` watcher on the notes directory live-reloads externally changed notes (cloud sync) with "newest `updated_at` wins" and mtime tracking so the app's own writes are not reprocessed; only `^[0-9a-fA-F]{8}\.json$` files are notes. Writes go through a temp file + rename. Unreadable files are skipped; `move_to_corrupted` moves one to `corrupted/`.
- `image_store.rs` (`ImageStore`) — same encrypted-JSON-per-item approach for pasted images (`images/<id>.json`, bytes encrypted directly). Also `extract_image_ids` for `![alt](image:<id>)` references.
- `backup.rs` — zips root `*.json` files (incl. `.gote_config.json`) + `images/` into `<notesDir>/backups/backup-<YYYYMMDD-HHMM>.zip`, every entry prefixed `backup-<stamp>/`.
- `auth.rs` (`AuthManager`) — PBKDF2-derives both a password-verification hash (`password + "verification"`) and the actual AES key from the same salt. The salt can live locally (`~/.config/gote/password_hash`) or inside the notes directory (`.gote_config.json`), so a second machine syncing the same folder bootstraps by verifying the password against the existing encrypted notes (`encrypted_data`) instead of needing the local hash file. Sessions are in-memory with a 30-minute sliding timeout.
- `crypto.rs` — AES-256-GCM (12-byte nonce prefix, base64) + PBKDF2-HMAC-SHA256 (100k iterations, 32-byte key/salt).
- `models.rs` — `Category` (`private`/`work`/`trash`), `Note`, `Image`, `EncryptedNote`, `NotePayload` (encode/decode incl. legacy), `NoteDto` (timestamps pre-formatted as RFC3339 strings, second precision, local offset).
- `config.rs` — default paths (`~/Documents/Gote/Notes` for notes, `~/.config/gote/` for password hash + app config, on all platforms including Windows) and the small JSON `Config{notesPath, passwordHashPath}`.
- `ids.rs` — short ids and validation. **Every id that reaches a filesystem path must pass `ensure_valid_id`** (path-traversal guard, error `invalid id`).
- `error.rs` — `AppError` serializes as its message string; `SESSION_EXPIRED` is the string the frontend matches on.

**Tauri config**: `src-tauri/tauri.conf.json` (window 1200x1000, CSP, bundle icons) and `src-tauri/capabilities/default.json` (permissions for core, clipboard read/write text, opener). Identifier `com.gote.app`.

**Frontend**: `frontend/src/main.js` (~1900 lines) is a single vanilla-JS module that drives the entire UI — auth screens, note list/editor, markdown preview (`marked` + `highlight.js`), settings. It imports the PascalCase backend functions from `frontend/src/api.js` (thin `invoke` wrappers that turn Tauri's string rejections into `Error`s, because `main.js` reads `error.message`) and uses `@tauri-apps/plugin-clipboard-manager` / `@tauri-apps/plugin-opener` directly. When adding a backend command: add the Rust command + `generate_handler!` entry, then an `api.js` wrapper. `frontend/src/constants.js` centralizes shared UI constants/CSS class names/element IDs/messages. `frontend/src/style.css` is the dark theme. Built with Vite; no component framework.

## Conventions

From `.github/copilot-instructions.md`:
- This is a Tauri app with vanilla JavaScript and no framework — use `cargo test`, `cargo clippy --all-targets -- -D warnings` and `tauri build --no-bundle` to verify Rust code and the build.
- Focus on readability and simplicity. Use modern JavaScript features but avoid complex patterns.
- Keep `cargo fmt` and clippy clean; tests that touch disk use temp dirs (never the real `~/.config/gote` or `~/Documents/Gote`).
- Never modify `testdata/go-fixtures/`; copy it into a temp dir first.

## Stale docs, don't trust these

- `CHANGELOG.md` and `docs/superpowers/plans/` describe the original Go implementation that this Rust port replaced.
