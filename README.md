# gote

A secure, cross-platform desktop note-taking app built with [Tauri v2](https://tauri.app): a Rust backend and a vanilla JavaScript frontend (no framework).

## Features

- **Secure notes**: AES-256-GCM encrypted at rest, key derived from your password with PBKDF2
- **Markdown preview** with syntax highlighting and code-block copy buttons
- **Search** across all notes, plus `private` / `work` / `trash` categories
- **Pasted images**, stored encrypted next to your notes
- **Folder sync friendly**: live-reloads notes changed on disk (Dropbox, iCloud, ...) and can be set up on a second machine from the synced folder
- **Automatic daily zip backups** (`<notes>/backups/`)
- **Cross-platform**: Windows, macOS, Linux

## Install

Download from the [latest release](https://github.com/yggdrion/gote/releases/latest):

- **Windows:** `gote-windows-amd64-setup.exe` (installer, no admin rights needed) or the portable `gote-windows-amd64.exe`
- **macOS:** `gote-macos-arm64.dmg` (Apple Silicon) or `gote-macos-amd64.dmg` (Intel), or `brew install --cask yggdrion/tap/gote`

The builds are not code-signed, so Windows SmartScreen and macOS Gatekeeper warn on first launch.

Installed copies update themselves: gote checks for a new release on startup and offers to install it, and Settings has a "Check for Updates" button. Updates are verified against a signing key built into the app. The portable `.exe` can't update itself, so use the installer if you want auto-updates.

## Architecture

### Backend (Rust, `src-tauri/`)

- `src/commands.rs`: Tauri commands called by the frontend
- `src/app.rs`: `AppCore`, the application logic behind every command
- `src/auth.rs`: password verification and sessions
- `src/crypto.rs`: AES-GCM and PBKDF2
- `src/note_store.rs`, `src/image_store.rs`, `src/backup.rs`: encrypted storage, file watcher, zip backups
- `src/models.rs`, `src/config.rs`, `src/ids.rs`, `src/error.rs`: models, configuration, id validation, errors

### Frontend (`frontend/`)

- `index.html`: single-page app structure with auth and main screens
- `src/main.js`: application logic and event handling
- `src/api.js`: wrappers that call the Tauri commands
- `src/style.css`: dark theme

## Development

Prerequisites: a Rust toolchain, [bun](https://bun.sh) and the [Tauri system dependencies](https://tauri.app/start/prerequisites/) (on Linux: webkit2gtk 4.1, libsoup 3).

```bash
bun install --cwd frontend
frontend/node_modules/.bin/tauri dev                # run in development mode
frontend/node_modules/.bin/tauri build              # production build and bundles
frontend/node_modules/.bin/tauri build --no-bundle  # only the binary: src-tauri/target/release/gote

cd src-tauri
cargo test
cargo clippy --all-targets -- -D warnings
cargo fmt --check
```

## Data storage

- **Notes**: encrypted `<id>.json` files in the notes folder (default `~/Documents/Gote/Notes`)
- **Shared salt**: `<notes folder>/.gote_config.json`, so another machine can unlock the same folder
- **Local password hash**: `~/.config/gote/password_hash`
- **App config**: `~/.config/gote/config`

## Security

- Notes and images are encrypted with a key derived from your password (PBKDF2-HMAC-SHA256, 100k iterations)
- The derived key lives only in memory and is dropped on logout or after 30 minutes of inactivity
- The password itself is never stored
