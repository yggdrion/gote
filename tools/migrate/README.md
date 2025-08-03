# Gote Migration Tool

This tool helps migrate your Gote notes from the legacy SHA-256 encryption to the new secure PBKDF2 encryption method.

## When to Use This Tool

You should use this migration tool if you're getting errors like:

```
Error decrypting note from ...: cipher: message authentication failed
```

This error occurs when your notes were encrypted with the legacy SHA-256 method but the application is trying to decrypt them with the new PBKDF2 method.

## How to Use

1. **Close the Gote application** before running the migration tool.

2. **Build the migration tool**:

   ```powershell
   cd c:\Users\rapha\workspace\gote\tools\migrate
   go build -o migrate.exe
   ```

3. **Run the migration**:

   ```powershell
   .\migrate.exe "C:\Users\rapha\sync\gote"
   ```

   Replace the path with your actual Gote data directory.

4. **Enter your password** when prompted.

## What the Tool Does

1. **Detection**: Detects whether your notes are using legacy SHA-256 or modern PBKDF2 encryption.

2. **Validation**: If already using PBKDF2, validates that all notes can be decrypted with your password.

3. **Migration**: If using legacy encryption:

   - Creates a backup of all notes in `backup_before_migration/`
   - Decrypts each note with the old method
   - Re-encrypts each note with the new PBKDF2 method
   - Saves the new encryption configuration

4. **Safety**: Always creates backups before making any changes.

## Security Improvements

The new PBKDF2 encryption provides:

- **Salt-based key derivation**: Each installation uses a unique salt
- **100,000 iterations**: Meets OWASP security recommendations
- **Resistance to rainbow table attacks**: Salt prevents precomputed attacks
- **Future-proof configuration**: Easily upgradeable encryption parameters

## Troubleshooting

### "Failed to decrypt notes with provided password"

- Double-check your password
- Make sure you're using the same password that was used to encrypt the notes

### "No notes found to migrate"

- Verify the data directory path is correct
- Check that the directory contains `.json` files

### "Migration failed"

- Check the error message for details
- Your original notes are safe - backups are created before any changes
- You can restore from the `backup_before_migration/` folder if needed

## Safety Features

- **Automatic backups**: Original notes are backed up before migration
- **Validation**: Tests decryption before proceeding
- **Atomic operations**: Each note is migrated individually
- **Rollback capability**: Original backups can be restored if needed

## After Migration

After successful migration:

1. Start the Gote application normally
2. Enter your password as usual
3. All notes should now decrypt successfully
4. The new PBKDF2 encryption is automatically used for all new notes

## Support

If you encounter issues:

1. Check that you're using the correct password
2. Verify the data directory path
3. Look for detailed error messages in the console output
4. Your original notes are always preserved in the backup directory
