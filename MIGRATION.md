# Migration Guide: Go to TypeScript/Bun

This guide helps you migrate from the Go version of Gote to the TypeScript/Bun version.

## Prerequisites

1. **Install Bun**: Download and install from https://bun.sh/
2. **Backup your data**: Before migrating, backup your existing notes and configuration

## Migration Steps

### 1. Install Dependencies

```bash
bun install
```

### 2. Configuration Migration

Your existing configuration should work automatically. The TypeScript version looks for config in the same locations:

- **Windows**: `%APPDATA%/gote/config.json`
- **macOS/Linux**: `~/.config/gote/config.json`

If you have a custom configuration, it will be loaded automatically.

### 3. Data Migration

Your existing encrypted notes are fully compatible! The TypeScript version uses the same:

- Encryption algorithm (AES-256-GCM)
- File format (JSON with encrypted data)
- Password hashing (SHA-256)

**Important**: Use the same password you used with the Go version.

### 4. Start the New Server

```bash
# Development mode (with hot reload)
bun run dev

# Or production mode
bun run build
bun run start
```

### 5. Verify Migration

1. Open http://localhost:8080
2. Enter your existing password
3. Verify all your notes are visible and searchable
4. Test creating, editing, and deleting notes

## Differences from Go Version

### Similarities

- ✅ Same encryption and security model
- ✅ Same file format and data storage
- ✅ Same web interface and features
- ✅ Same configuration options
- ✅ Same cross-platform support

### New Features in TypeScript Version

- 🚀 Faster startup time with Bun
- 🔥 Hot reload during development
- 📦 Modern JavaScript ecosystem integration
- 🛠️ Better development tools and debugging

### Performance

- The TypeScript version should perform similarly to the Go version
- File I/O and encryption operations have comparable performance
- Memory usage may be slightly higher due to the JavaScript runtime

## Troubleshooting

### Problem: Can't decrypt notes

**Solution**: Make sure you're using the exact same password as the Go version.

### Problem: Configuration not found

**Solution**: Check that your config file is in the expected location. You can also set custom paths in the config.

### Problem: Permission errors on Windows

**Solution**: Make sure the application has write permissions to the data and config directories.

### Problem: Port already in use

**Solution**: If port 8080 is in use, you can modify the port in `src/index.ts`.

## Rollback Plan

If you need to go back to the Go version:

1. Stop the TypeScript server
2. Your data files are unchanged and compatible
3. Start the Go version with the same configuration
4. Everything should work exactly as before

## Development

If you want to modify the TypeScript version:

```bash
# Start development server with hot reload
bun run dev

# Run type checking
bun run typecheck

# Build for production
bun run build
```

## Getting Help

If you encounter issues during migration:

1. Check that Bun is properly installed: `bun --version`
2. Verify your data directory has the expected `.json` files
3. Check the terminal output for specific error messages
4. Make sure you're using the same password as the Go version
