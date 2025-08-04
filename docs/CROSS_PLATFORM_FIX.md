# Cross-Platform Decryption Fix

## Problem

When using Gote across multiple devices (Windows, macOS, Linux), you may encounter:

```
Error decrypting note from path/file.json cipher: message authentication failed
```

## Root Cause

The encryption salt was stored locally on each device in different locations:

- **Windows**: `%APPDATA%\.config\gote\password_hash`
- **macOS**: `~/.config/gote/password_hash`
- **Linux**: `~/.config/gote/password_hash`

When syncing notes between devices, each device generates its own salt, leading to different encryption keys and decryption failures.

## Automatic Solution (Recommended)

The latest version of Gote (v1.2+) automatically handles cross-platform compatibility:

### ✅ For New Installations:

- Gote will automatically create a `.gote_config.json` file in your notes directory
- This file contains the encryption salt and is synced across all devices
- Just install Gote on each device and use the same password

### ✅ For Existing Installations:

1. **Update to latest version** of Gote
2. **First login on any device** will automatically create the cross-platform config
3. **Other devices** will automatically detect and use the synced configuration

## Manual Migration (If Needed)

If you're still seeing decryption errors, run the migration tool:

### Windows:

```powershell
cd c:\Users\rapha\workspace\gote\tools\migrate
go build -o migrate.exe
.\migrate.exe "C:\Users\rapha\sync\gote"
```

### macOS/Linux:

```bash
cd ~/workspace/gote/tools/migrate
go build -o migrate
./migrate "/path/to/your/synced/notes"
```

## What This Fix Does

### ✅ **Automatic Cross-Platform Support**

- Salt is now stored in your synced notes directory (`.gote_config.json`)
- Same salt = same encryption key = notes work everywhere

### ✅ **Backward Compatible**

- Existing notes continue to work
- Falls back to local salt if cross-platform config is missing

### ✅ **Secure**

- Uses same PBKDF2 encryption (100,000 iterations)
- Salt is still random and unique per installation
- No security compromises

### ✅ **Zero Configuration**

- Works automatically once updated
- No user intervention required
- Seamless experience across devices

## File Structure After Fix

Your synced notes directory will contain:

```
notes/
├── .gote_config.json         # Cross-platform encryption config
├── abc123def.json            # Your encrypted notes
├── def456ghi.json
└── ...
```

The `.gote_config.json` file contains:

```json
{
  "salt": "base64-encoded-salt-here",
  "createdAt": "2025-08-04T10:30:00Z",
  "version": "1.0"
}
```

## Verification

After the fix:

1. ✅ Notes decrypt successfully on all devices
2. ✅ New notes can be created on any device
3. ✅ File sync continues to work normally
4. ✅ Same password works everywhere

## Troubleshooting

### Still getting decryption errors?

1. Check that `.gote_config.json` exists in your notes directory
2. Verify all devices are using the same password
3. Ensure the notes directory is properly synced
4. Try the manual migration tool

### Config file missing?

- Login once on the device where you originally created notes
- The config will be automatically created and synced

### Different passwords on different devices?

- Gote now uses the same salt everywhere, so passwords must match exactly
- Reset password if needed: delete `.gote_config.json` and local password files

---

**This fix enables seamless note access across all your devices while maintaining enterprise-grade encryption security.**
