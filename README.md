# Gote - TypeScript Note-Taking App

A simple, modern note-taking application built with TypeScript and Bun, featuring AES-256-GCM encryption for secure note storage.

## Features

- 📝 Create, edit, and delete notes with **GitHub-flavored markdown support**
- 🎨 **Syntax highlighting** for code blocks in 10+ languages
- 🔍 Search through notes by title and content
- 💾 **Encrypted JSON file storage** with AES-256-GCM encryption
- 🌐 **Offline-first design** - works without internet connection
- 🎨 Clean, responsive web interface
- ⌨️ Keyboard shortcuts (Ctrl/Cmd+S to save, Ctrl/Cmd+N for new note)
- 📱 Mobile-friendly design
- 🔒 Built-in password-based authentication and encryption
- ⚙️ Configurable settings (notes directory, password hash location)

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

### Prerequisites

- [Bun](https://bun.sh/) runtime installed on your system

### Installation & Setup

1. **Clone the repository:**

   ```bash
   git clone <repository-url>
   cd gote
   ```

2. **Install dependencies:**

   ```bash
   bun install
   ```

3. **Run the application:**

   ```bash
   # Development mode (with hot reload)
   bun run dev

   # Production mode
   bun run build
   bun run start
   ```

4. **Open your browser:**
   Navigate to `http://localhost:8080`

5. **First-time setup:**
   - Create your master password (minimum 6 characters)
   - This password encrypts all your notes with AES-256-GCM
   - **Important**: Remember this password - there's no recovery option!

## Project Structure

```
gote/
├── src/
│   ├── index.ts         # Main application entry point
│   ├── router.ts        # HTTP routing and handlers
│   ├── auth.ts          # Authentication and session management
│   ├── crypto.ts        # AES-256-GCM encryption/decryption
│   ├── noteStore.ts     # Note storage and retrieval
│   ├── templates.ts     # Go-template compatible renderer
│   ├── config.ts        # Configuration management
│   └── types.ts         # TypeScript type definitions
├── static/              # Static web assets
│   ├── index.html       # Main application UI
│   ├── login.html       # Authentication page
│   ├── style.css        # Responsive CSS styling
│   ├── script.js        # Client-side JavaScript
│   └── vendor/          # Local vendor libraries
├── data/                # Encrypted JSON files (created automatically)
├── scripts/             # Development and test scripts
├── package.json         # Dependencies and scripts
├── tsconfig.json        # TypeScript configuration
└── README.md           # This file
```

## API Endpoints

The application provides both a web interface and a REST API:

### Web Routes

- `GET /` - Main application (requires authentication)
- `GET /login` - Authentication page
- `POST /auth` - Handle login/setup
- `POST /logout` - Handle logout

### API Routes (Protected)

- `GET /api/notes` - Get all notes
- `POST /api/notes` - Create new note
- `GET /api/notes/{id}` - Get specific note
- `PUT /api/notes/{id}` - Update note
- `DELETE /api/notes/{id}` - Delete note
- `GET /api/settings` - Get configuration
- `POST /api/settings` - Update configuration

### Security

All notes are encrypted with AES-256-GCM using a key derived from your password. Session-based authentication protects all API endpoints.

## Data Storage & Security

Notes are stored as individual encrypted JSON files in the configured data directory (default: `./data`). Each note is encrypted with AES-256-GCM using a key derived from your master password.

### Encryption Details

- **Algorithm**: AES-256-GCM (Galois/Counter Mode)
- **Key Derivation**: SHA-256 hash of master password
- **Authentication**: Built-in authentication tag prevents tampering
- **File Format**: Base64-encoded encrypted data

### Example encrypted note file:

```json
{
  "id": "a1b2c3d4",
  "encryptedData": "base64-encoded-encrypted-content",
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-15T10:30:00Z"
}
```

### Configuration

The application stores configuration in your system's standard config directory:

- **Windows**: `%APPDATA%\gote\`
- **Linux/macOS**: `~/.config/gote/`

Files:

- `gote_password_hash`: Verification hash for your master password
- `config.json`: Application settings (data directory, etc.)

## Development

### Available Scripts

```bash
# Development with hot reload
bun run dev

# Build for production
bun run build

# Run production build
bun run start

# Update vendor libraries
bun run update-vendor

# Check vendor library health
bun run check-vendor
```

### Project Architecture

The application follows a modular TypeScript architecture:

1. **Entry Point** (`src/index.ts`): Server initialization and configuration
2. **Router** (`src/router.ts`): HTTP request routing and handlers
3. **Authentication** (`src/auth.ts`): Session management and password verification
4. **Encryption** (`src/crypto.ts`): AES-256-GCM encryption/decryption
5. **Storage** (`src/noteStore.ts`): Note persistence and retrieval
6. **Templates** (`src/templates.ts`): Go-template compatible rendering
7. **Types** (`src/types.ts`): TypeScript interfaces and type definitions

### Adding Features

1. **New API endpoints**: Add handlers in `src/router.ts`
2. **UI changes**: Modify templates in `static/` directory
3. **Styling**: Update `static/style.css`
4. **Client-side functionality**: Update `static/script.js`
5. **New encryption**: Extend `src/crypto.ts`

### Building for Production

```bash
# Install dependencies
bun install

# Build the application
bun run build

# The built application can be run with:
bun run start
```

## Key Features

### Security First

- **AES-256-GCM encryption**: Military-grade encryption for all notes
- **Password-based authentication**: No account registration required
- **Local storage**: Your data never leaves your machine
- **Session management**: Secure authentication with configurable timeouts

### Developer Experience

- **TypeScript**: Full type safety and modern JavaScript features
- **Bun runtime**: Fast JavaScript runtime with built-in bundling
- **Hot reload**: Instant development feedback
- **Modular architecture**: Clean separation of concerns

### User Experience

- **Markdown support**: GitHub-flavored markdown with syntax highlighting
- **Responsive design**: Works on desktop and mobile devices
- **Keyboard shortcuts**: Efficient note management
- **Search functionality**: Find notes quickly by content
- **Settings panel**: Customize data directory and other options

## CI/CD and Automation

### GitHub Actions Workflows

The project includes several automated workflows:

#### 🔄 **Continuous Integration** (`.github/workflows/ci.yml`)

- **Triggers**: Push/PR to main branches
- **Jobs**:
  - **Test & Build**: TypeScript compilation, crypto tests, template rendering tests
  - **Security Audit**: Dependency vulnerability scanning with `bun audit`
  - **Code Quality**: Type checking and dependency health checks

#### 📦 **Vendor File Sync** (`.github/workflows/sync-vendor.yml`)

- **Triggers**: Changes to `package.json` or `bun.lock`
- **Purpose**: Automatically updates offline vendor libraries
- **Actions**: Downloads updated JavaScript libraries, commits changes

#### 🚀 **Automated Releases** (`.github/workflows/release.yml`)

- **Triggers**: Git tags (e.g., `v1.0.0`)
- **Actions**: Builds application, creates release archives, publishes GitHub releases

### Dependabot Configuration

Automated dependency management via `.github/dependabot.yml`:

```yaml
# Weekly dependency updates
- package-ecosystem: "npm"
  schedule:
    interval: "weekly"
    day: "monday"
    time: "09:00"
```

**Benefits**:

- 🔒 **Security updates**: Automatic security patches
- 📈 **Stay current**: Weekly updates to latest stable versions
- 🏷️ **Organized PRs**: Labeled and categorized for easy review
- 🤖 **Zero maintenance**: Fully automated dependency management

### Available Scripts

```bash
# Development
bun run dev              # Hot reload development server
bun run build            # Production build
bun run start            # Run production server

# Testing
bun run test             # Run all tests
bun run test:crypto      # Test encryption/decryption
bun run test:templates   # Test template rendering

# Maintenance
bun run update-vendor    # Update offline vendor libraries
bun run check-vendor     # Verify vendor library integrity
```

## Migration from Go Version

This TypeScript version maintains full compatibility with data created by the original Go version. The encryption format and file structure are identical, so you can seamlessly switch between versions.

## License

This project is open source and available under the MIT License.
