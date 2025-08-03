# Fix for Decryption Errors

## Problem

You are seeing errors like:

```
2025/08/03 13:33:14 Error decrypting note from C:\Users\rapha\sync\gote\b882055e.json: cipher: message authentication failed
```

## Root Cause

During the security refactoring (Phase 4), we upgraded the encryption method from SHA-256 to PBKDF2 for better security. However, your existing notes were encrypted with the old method, and the application is now trying to decrypt them with the new method.

## Solution

I've created a migration tool that will fix this issue by converting your notes to use the new encryption method.

### Quick Fix Steps:

1. **Close Gote application** if it's running

2. **Open PowerShell and navigate to the migration tool**:

   ```powershell
   cd c:\Users\rapha\workspace\gote\tools\migrate
   ```

3. **Build the migration tool**:

   ```powershell
   go build -o migrate.exe
   ```

4. **Run the migration**:

   ```powershell
   .\migrate.exe "C:\Users\rapha\sync\gote"
   ```

5. **Enter your password** when prompted

6. **Restart Gote** - the errors should be gone!

### What the Migration Does:

- ✅ **Safe**: Creates backups before making any changes
- ✅ **Automatic**: Detects your current encryption method
- ✅ **Smart**: Only migrates if needed
- ✅ **Secure**: Upgrades to PBKDF2 with salt (OWASP compliant)

## Technical Details

The migration tool:

1. Checks if you're already using the new PBKDF2 method
2. If not, decrypts all notes with the old SHA-256 method
3. Re-encrypts them with the new PBKDF2 method
4. Creates a configuration file to remember the new method
5. Makes backups of everything before starting

## After Migration

- Your notes will decrypt correctly
- New notes will automatically use the secure PBKDF2 encryption
- You get better security with salt-based key derivation
- The application will remember your encryption method

## Rollback (if needed)

If something goes wrong, your original notes are backed up in:

```
C:\Users\rapha\sync\gote\backup_before_migration\
```

You can restore them by copying the files back to the main directory.

---

**This fix resolves the "cipher: message authentication failed" errors while upgrading your notes to enterprise-grade encryption security.**
