# Gote - Wails Notes App

This Wails application has been transformed from a basic template into a full-featured note-taking application by integrating the functionality from the `noteapp` directory.

## Features

- **Secure Notes**: Password-protected encrypted note storage
- **Real-time Preview**: Markdown preview with syntax highlighting
- **Search**: Full-text search across all notes
- **File Sync**: Automatic synchronization from disk
- **Modern UI**: Dark theme with responsive design
- **Cross-platform**: Built with Wails for native performance

## Architecture

### Backend (Go)

- **app.go**: Main application logic with Wails bindings
- **pkg/auth**: Password authentication and session management
- **pkg/config**: Configuration management
- **pkg/crypto**: Encryption utilities for secure note storage
- **pkg/models**: Data models for notes and sessions
- **pkg/storage**: File-based note storage with encryption
- **pkg/utils**: Utility functions

### Frontend (JavaScript/HTML/CSS)

- **index.html**: Single-page app structure with auth and main screens
- **main.js**: Application logic, event handling, and Wails bindings
- **style.css**: Modern dark theme styling
- **wailsjs/**: Auto-generated Wails bindings

## Key Transformations Made

1. **Converted HTTP Server to Wails App**:

   - Removed Chi router and HTTP handlers
   - Integrated functionality directly into Wails App struct
   - Created frontend bindings for all note operations

2. **Authentication Flow**:

   - Password setup screen for first-time users
   - Login screen for returning users
   - Session management through Wails context

3. **Note Management**:

   - Create, read, update, delete operations
   - Full-text search functionality
   - Markdown preview with live updates
   - File synchronization

4. **UI/UX Improvements**:
   - Responsive sidebar with note list
   - Split-pane editor with live preview
   - Settings modal for password changes
   - Modern dark theme

## Running the App

### Development

```bash
wails dev
```

### Production Build

```bash
wails build
```

### Built Application

The built executable will be available at `build/bin/gote.exe`

## Data Storage

- **Notes**: Stored in `./data/notes/` as encrypted files
- **Password**: Hash stored in `./data/password_hash`
- **Config**: User-specific config in system directories

## Security

- Notes are encrypted using the user's password
- Password is hashed before storage
- All note content is encrypted at rest

## Dependencies

### Go Dependencies

- `github.com/wailsapp/wails/v2`: Wails framework
- `github.com/fsnotify/fsnotify`: File system watching
- `github.com/google/uuid`: UUID generation
- `golang.org/x/term`: Terminal input utilities

### Frontend Dependencies

- Vite for build tooling
- Native JavaScript with Wails bindings
- CSS Grid and Flexbox for responsive layout

## Future Enhancements

- Rich text editor with WYSIWYG mode
- Export functionality (PDF, HTML)
- Tag-based organization
- Note templates
- Cloud synchronization
- Plugin system for extensions
