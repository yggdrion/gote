# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## What this is

`gote` is a secure, cross-platform desktop note-taking app built with [Wails v2](https://wails.io): a Go backend (module `gote`, Go 1.25) paired with a vanilla-JS frontend, packaged as a native webview app. Notes are AES-GCM encrypted at rest with a PBKDF2-derived key from a single app password; no framework (React/Vue/etc.) is used on the frontend by design — keep it that way (see Conventions below).

## Commands

Run from the repo root unless noted.

```sh
wails dev                          # run the app in dev mode (hot-reloads frontend + backend)
wails build                        # production build -> build/bin/gote.exe (Windows)
golangci-lint run --timeout=5m     # Go lint (matches .golangci.yml, skips frontend/dist)
```

Frontend-only (run from `frontend/`):
```sh
npm run dev        # vite dev server
npm run build       # vite build (also invoked automatically by `wails build`)
npm run preview     # preview a production build
```

There are no tests in this repo (no `*_test.go` files, no frontend test script) — there is no test command to run.

Two of the three lint/check CI workflows (`golangci-lint.yml`, `lint-static.yml`) are `workflow_dispatch`-only, i.e. they don't run automatically on push/PR. `typos-spelling.yml` (crate-ci/typos) and `pull-request-title.yml` do run automatically. `lint-static.yml` lints a `static/**` glob that doesn't exist in the current layout — treat it as stale/non-authoritative.

PR titles are enforced by CI and drive semantic-release versioning — they must match:
```
^(feat|fix|chore|docs|refactor|test|perf|build|ci)(\([^)]+\))?:\ 
```

## Architecture

**Entry point**: `main.go` embeds `frontend/dist` (`//go:embed`), constructs `App` (`app.go`) via `NewApp()`, and calls `wails.Run` binding every exported `App` method directly to the frontend. Wails auto-generates the JS bindings into `frontend/wailsjs/` — never hand-edit that directory, regenerate via `wails dev`/`wails build`.

**Backend layering** (`App` → service → storage → crypto):
- `app.go` (`App`) is the Wails-bound facade: startup/config bootstrap, auth flows, note/category/image CRUD, backups, and background goroutines (session cleanup every 5 min, hourly-checked daily auto-backup, orphaned-image cleanup via regex-matching `![alt](image:<id>)` references in note content). Most methods delegate to `pkg/services.NoteService`, falling back to calling `pkg/storage.NoteStore` directly where the service isn't used.
- `pkg/services` (`NoteService`) — thin validation layer (empty ID/content checks, auth-key-present checks, trash-vs-permanent-delete branching) wrapping `NoteStore`.
- `pkg/storage`:
  - `notestore.go` (`NoteStore`) is the core persistence + in-memory cache. Each note is one encrypted JSON file named `<8-hex-id>.json` (`pkg/utils.GenerateShortUUID`) in the notes directory. Decrypted content is JSON (`{content, category, original_category, images}`) with fallback parsing for a legacy plain-string format. Watches the notes directory with `fsnotify` to live-reload notes changed externally (e.g. by cloud sync), using "newest `UpdatedAt` wins" conflict resolution and modtime tracking to avoid reprocessing its own writes. Corrupt files are moved to `corrupted/`, not deleted.
  - `imagestore.go` (`ImageStore`) — same encrypted-JSON-per-item approach for pasted images (`images/<id>.json`).
  - `backup.go` — zips all notes + `images/` + `.gote_config.json` into `<notesDir>/backups/backup-<YYYYMMDD-HHMM>.zip`.
- `pkg/auth` (`auth.Manager`) — PBKDF2-derives both a password-verification hash and the actual AES key from the same salt. The salt can live locally (`~/.config/gote/password_hash`) or inside the notes directory itself (`.gote_config.json`), so a second machine syncing the same notes folder (Dropbox/iCloud/etc.) can bootstrap by verifying the password against existing encrypted notes instead of needing the local hash file. Sessions are in-memory (`pkg/models.Session`, 30 min timeout); the derived AES key is held only in memory (`App.currentKey`), never persisted.
- `pkg/crypto` — AES-GCM encrypt/decrypt + PBKDF2 key derivation (100k iterations, 32-byte key/salt).
- `pkg/models` — `Note`, `NoteCategory` (`private`/`work`/`trash`), `Image`, `EncryptedNote`, `Session`.
- `pkg/types` (`wails.go`) — `WailsNote` DTO/converters; timestamps are pre-formatted to RFC3339 strings here because Wails serializes Go `time.Time` awkwardly to JS.
- `pkg/config` — resolves default paths (`~/Documents/Gote/Notes` for notes, `~/.config/gote/` for password hash + app config, on all platforms including Windows) and loads/saves the small JSON `Config{NotesPath, PasswordHashPath}`.

**Frontend**: `frontend/src/main.js` (~1900 lines) is a single vanilla-JS module that imports the generated Wails bindings directly (`IsPasswordSet`, `SetPassword`, `GetAllNotes`, `CreateNote`, etc.) and drives the entire UI — auth screens, note list/editor, markdown preview (`marked` + `highlight.js`), settings. `frontend/src/constants.js` centralizes shared UI constants/CSS class names/element IDs/messages. `frontend/src/style.css` is the dark theme. Built with Vite; no component framework, no additional module/bundling structure beyond this.

## Conventions

From `.github/copilot-instructions.md`:
- This is a Wails app with vanilla JavaScript and no framework — use `wails build` to test the build process and verify Go code.
- Focus on readability and simplicity. Use modern JavaScript features but avoid complex patterns.

## Stale docs, don't trust these

- README's "Converted HTTP Server to Wails App" section and its `./data/notes/`-based paths describe an older layout (there was once a top-level `noteapp` dir, since folded into `app.go` + `pkg/`). Trust `pkg/config/config.go` for actual default paths.
- `.typos.toml` excludes `noteapp/static/vendor/**`, which no longer exists.
