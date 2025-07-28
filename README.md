# Meemo - Go Note-Taking App

A simple, modern note-taking application built with Go, inspired by the original Meemo but rebuilt with JSON file storage instead of MongoDB.

## Features

- 📝 Create, edit, and delete notes
- 🔍 Search through notes by title and content
- 💾 JSON file-based storage (no database required)
- 🎨 Clean, responsive web interface
- ⌨️ Keyboard shortcuts (Ctrl/Cmd+S to save, Ctrl/Cmd+N for new note)
- 📱 Mobile-friendly design

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
