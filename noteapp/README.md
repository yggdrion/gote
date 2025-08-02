# Meemo - Go Note-Taking App

A simple, modern note-taking application built with Go, inspired by the original Meemo but rebuilt with JSON file storage instead of MongoDB.

## Features

- 📝 Create, edit, and delete notes with **GitHub-flavored markdown support**
- 🎨 **Syntax highlighting** for code blocks in 10+ languages
- 🔍 Search through notes by title and content
- 💾 JSON file-based storage (no database required)
- 🔄 **Real-time file synchronization** - Works with Syncthing, Dropbox, etc.
- 🌐 **Offline-first design** - works without internet connection
- 🎨 Clean, responsive web interface
- ⌨️ Keyboard shortcuts (Ctrl/Cmd+S to save, Ctrl/Cmd+N for new note)
- 📱 Mobile-friendly design
- 🔒 Built-in authentication system

## Offline Capabilities

This application is designed to work completely offline:

- **Local vendor libraries**: All JavaScript dependencies (marked.js, highlight.js) are stored locally in `static/vendor/`
- **No CDN dependencies**: Zero external network requests required for functionality  
- **Dependabot integration**: Automated dependency updates via standard npm workflow
- **Manual updates**: Use `npm run update-vendor` to sync vendor files manually

### Vendor Libraries

Dependencies are managed via `package.json` and **Dependabot**:
- **marked.js**: GitHub-flavored markdown parser
- **highlight.js**: Syntax highlighting for code blocks
- **CSS themes**: GitHub-style syntax highlighting theme

### Updating Vendor Libraries

**Automatic (recommended):**
- Dependabot monitors `package.json` for updates weekly
- Creates PRs automatically when new versions are available
- GitHub Actions syncs vendor files when package.json changes
- All updates validated before deployment

**Manual:**
```bash
# Update package.json first, then:
npm run update-vendor

# Or use make target:
make vendor-update
```

**Health Check:**
```bash
npm run check-vendor
# or
make vendor-check
```

## Quick Start

1. **Clone or create the project:**
   ```bash
   mkdir meemo-go && cd meemo-go
   ```

2. **Initialize Go module:**
   ```bash
   go mod init meemo
   go mod tidy
   ```

3. **Run the application:**
   ```bash
   go run main.go
   ```

4. **Open your browser:**
   Navigate to `http://localhost:8080`

## Project Structure

```
meemo-go/
├── main.go          # Main application with HTTP server and note management
├── go.mod           # Go module dependencies
├── data/            # JSON files for note storage (created automatically)
│   ├── a1b2c3d4.json
│   ├── e5f6g7h8.json
│   └── ...
└── static/          # Static web assets
    ├── style.css    # Responsive CSS styling
    └── script.js    # Client-side JavaScript
```

## API Endpoints

The application provides both a web interface and a REST API:

### Web Routes
- `GET /` - Home page with note list and search
- `GET /note/{id}` - View individual note
- `GET /new` - Create new note form
- `GET /edit/{id}` - Edit note form

### API Routes
- `GET /api/notes` - Get all notes
- `POST /api/notes` - Create new note
- `GET /api/notes/{id}` - Get specific note
- `PUT /api/notes/{id}` - Update note
- `DELETE /api/notes/{id}` - Delete note
- `GET /api/search?q={query}` - Search notes
- `GET /api/settings` - Get current settings
- `POST /api/settings` - Update settings
- `POST /api/sync` - Force refresh from disk

## Data Storage

Notes are stored as individual JSON files in the `./data` directory. Each note file is named with a short UUID (e.g., `a1b2c3d4.json`, `e5f6g7h8.json`).

Example note structure:
```json
{
  "id": "a1b2c3d4",
  "title": "My First Note",
  "content": "This is the content of my note...",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

## Configuration

The application uses these default settings:
- **Port:** 8080
- **Data directory:** `./data`
- **Static files:** `./static`

You can modify these in the `main.go` file if needed.

## File Synchronization

Gote includes built-in file system watching for seamless integration with sync tools like **Syncthing**, **Dropbox**, **Google Drive**, or manual file editing:

### Features
- 🔄 **Automatic file watching** - Detects changes to note files while the app is running
- 🚀 **Real-time sync** - New, modified, or deleted files are immediately reflected in the UI
- 🔒 **Thread-safe operations** - Safe concurrent access during sync operations
- 🧠 **Smart conflict resolution** - Uses modification timestamps to determine which version to keep
- 🔘 **Manual sync button** - Force refresh from disk when needed

### Usage with Syncthing
1. Configure Syncthing to sync your `./data` directory across devices
2. Use the same password on all devices for encrypted notes
3. Changes from other devices appear automatically
4. Use the 🔄 Sync button if you notice any inconsistencies

### Manual Sync
- **Header sync button**: Quick access to refresh from disk
- **Settings modal**: Detailed sync options with explanations
- **API endpoint**: `POST /api/sync` for programmatic refresh

For detailed information, see [docs/FILE_SYNC.md](docs/FILE_SYNC.md).

## Development

### Adding Features

The application is structured to make it easy to add new features:

1. **New API endpoints:** Add handlers in `main.go`
2. **UI changes:** Modify the HTML templates in the handler functions
3. **Styling:** Update `static/style.css`
4. **Client-side functionality:** Update `static/script.js`

### Building for Production

```bash
# Build binary
go build -o meemo main.go

# Run the binary
./meemo
```

## Differences from Original Meemo

This rebuild maintains the simplicity of the original while modernizing the stack:

- **Go instead of Node.js** - Better performance and simpler deployment
- **JSON files instead of MongoDB** - No database setup required
- **Modern responsive design** - Better mobile experience
- **REST API** - Easy integration with other tools
- **Keyboard shortcuts** - Better user experience

## License

This project is open source and available under the MIT License.
