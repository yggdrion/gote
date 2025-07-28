# Gote - Note Taking App (TypeScript/Bun Version)

A secure, encrypted note-taking application built with Bun and TypeScript. This is a port of the original Go version.

## Features

- **Encrypted Notes**: All notes are encrypted using AES-256-GCM before being stored
- **Password Protection**: First-time setup creates a password-protected session
- **Search**: Full-text search across all your notes
- **Markdown Support**: Basic markdown formatting for bold, italic, code blocks, and headings
- **Offline First**: Works completely offline with local file storage
- **Cross-Platform**: Runs on Windows, macOS, and Linux

## Installation

### Prerequisites

- [Bun](https://bun.sh/) - JavaScript runtime and package manager

### Setup

1. Install dependencies:

```bash
bun install
```

2. Start the development server:

```bash
bun run dev
```

3. Open your browser and navigate to `http://localhost:8080`

4. On first run, you'll be prompted to create a password. This password encrypts all your notes.

## Production Build

```bash
# Build the application
bun run build

# Run the production build
bun run start
```

## Configuration

The application stores its configuration in:

- **Windows**: `%APPDATA%/gote/config.json`
- **macOS/Linux**: `~/.config/gote/config.json`

Configuration options:

- `notesPath`: Directory where encrypted notes are stored (default: `./data`)
- `passwordHashPath`: Location of the password hash file

## Data Storage

- **Notes**: Stored as encrypted JSON files in the configured data directory
- **Password**: Hashed and stored separately from notes for security
- **Config**: Stored in the user's config directory

## Security

- All notes are encrypted using AES-256-GCM encryption
- Passwords are hashed using SHA-256
- Sessions expire after 30 minutes of inactivity
- No data is sent to external servers - everything is stored locally

## Development

### Project Structure

```
src/
├── index.ts        # Main application entry point
├── types.ts        # TypeScript type definitions
├── config.ts       # Configuration management
├── auth.ts         # Authentication and session management
├── crypto.ts       # Encryption/decryption utilities
├── noteStore.ts    # Note storage and management
├── templates.ts    # HTML template rendering
└── router.ts       # HTTP request routing
```

### Scripts

- `bun run dev` - Start development server with hot reload
- `bun run build` - Build for production
- `bun run start` - Start production server
- `bun run update-vendor` - Update frontend dependencies
- `bun run check-vendor` - Check frontend dependency versions

## Migration from Go Version

If you have an existing Go installation of gote, your data should be compatible as long as you use the same password and data directory paths. The encryption format and file structure are identical.

## License

This project is open source. See the original Go version for license details.
