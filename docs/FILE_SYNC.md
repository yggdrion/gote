# File Synchronization in Gote

## Overview

Gote now includes built-in file system watching and synchronization capabilities to handle external file changes from tools like Syncthing, Git, or direct file system modifications.

## Features

### 1. Automatic File System Watching
- Monitors the notes directory for file changes (create, modify, delete, rename)
- Automatically loads new or modified notes into memory
- Removes deleted notes from the in-memory store
- Only processes `.json` files to ignore temporary files

### 2. Thread-Safe Operations
- All note operations are protected with read/write mutexes
- Prevents race conditions between web requests and file system events
- Safe concurrent access to the notes store

### 3. Smart Conflict Resolution
- Compares modification timestamps to determine which version is newer
- Prefers external file changes over in-memory cache when external file is newer
- Prevents infinite loops by tracking file modification times from our own writes

### 4. Manual Sync Capability
- Sync button in the header for quick refresh
- Sync option in settings modal with explanation
- API endpoint `/api/sync` for programmatic refresh
- Full disk rescan to resolve any inconsistencies

## How It Works

### File System Events
```
File Created/Modified → Decrypt & Load → Update In-Memory Store (if newer)
File Deleted → Remove from In-Memory Store
File Renamed → Treat as Delete + Create
```

### Conflict Resolution Logic
1. External file change detected
2. Check if we have the file modification time recorded
3. If external file is newer than our record, update in-memory note
4. If in-memory note is newer, keep current version (external change ignored)
5. Update our modification time record

### API Endpoints
- `POST /api/sync` - Force full refresh from disk

## Usage with Syncthing

1. **Setup**: Configure Syncthing to sync your notes directory across devices
2. **Automatic**: File changes from other devices are automatically detected and loaded
3. **Manual**: Use the sync button if you notice any inconsistencies
4. **Conflicts**: Gote uses "last writer wins" based on file modification time

## Testing File Sync

### Test External File Changes
1. Start Gote and create a few notes
2. While Gote is running, use another text editor to modify a note file in the data directory
3. The change should be automatically reflected in Gote
4. Or click the sync button to force a refresh

### Test File Addition
1. Copy a valid encrypted note JSON file into the data directory
2. The new note should automatically appear in Gote
3. Or click sync to force a refresh

### Test File Deletion
1. Delete a note file from the data directory while Gote is running
2. The note should disappear from Gote automatically

## Configuration

The file watching is enabled by default. To disable it or handle errors gracefully:
- Check the logs for any file watcher warnings
- File watching failures don't prevent normal operation
- Manual sync is always available as a fallback

## Limitations

1. **Encryption Key**: All devices must use the same password for encrypted notes to sync properly
2. **File Format**: Only properly formatted encrypted JSON files are processed
3. **Large Files**: Very large note files may cause brief delays during sync
4. **Network Sync**: Syncthing conflicts (conflicted-copy files) are ignored but logged

## Recommendations

1. **Use the same password** on all devices
2. **Don't edit raw JSON files** unless you understand the encryption format
3. **Use the sync button** if you notice any inconsistencies
4. **Check logs** if you encounter sync issues
5. **Backup your data** before making manual file changes
