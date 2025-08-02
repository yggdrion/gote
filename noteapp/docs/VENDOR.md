# Vendor Library Management

This document explains how the offline vendor library system works in this Go note-taking application using **Dependabot** for automated dependency management.

## Overview

The application uses local copies of JavaScript libraries instead of CDN links to ensure it works completely offline. All vendor libraries are stored in `static/vendor/` and managed through **Dependabot**, npm scripts, and GitHub Actions.

## How It Works

1. **Dependencies defined in `package.json`** - Standard npm package management
2. **Dependabot monitors for updates** - Automatically creates PRs for new versions  
3. **GitHub Actions syncs vendor files** - Downloads updated files when package.json changes
4. **Local scripts for manual management** - Node.js scripts for health checks and updates

## Directory Structure

```
static/vendor/
├── marked.min.js          # GitHub-flavored markdown parser
├── highlight.min.js       # Syntax highlighting library
├── github.min.css         # GitHub-style highlighting theme
└── versions.txt          # Version tracking file (auto-generated)
```

## Dependency Management with Dependabot

### Configuration (`.github/dependabot.yml`)

```yaml
version: 2
updates:
  - package-ecosystem: "npm"
    directory: "/"
    schedule:
      interval: "weekly"
      day: "monday"
      time: "09:00"
```

### Benefits of Using Dependabot

- ✅ **Automatic updates**: Weekly PRs for new versions
- ✅ **Security alerts**: Notifies about vulnerable dependencies
- ✅ **Semantic versioning**: Respects version constraints in package.json
- ✅ **Release notes**: Links to changelogs in PR descriptions
- ✅ **Native GitHub integration**: No custom scripts to maintain

## Directory Structure

```
static/vendor/
├── marked.min.js          # GitHub-flavored markdown parser
├── highlight.min.js       # Syntax highlighting library
├── github.min.css         # GitHub-style highlighting theme
└── versions.txt          # Version tracking file
```

## Version Tracking

The `versions.txt` file tracks:
- Current versions of all libraries
- Download URLs with version placeholders
- Last update check date

Example:
```
marked.js=9.1.6
highlight.js=11.9.0
last_checked=2025-07-28
```

## Management Scripts

### Node.js Scripts

#### `npm run check-vendor` 
Health check script that:
- ✅ Verifies all required files exist
- ✅ Validates JavaScript syntax
- ✅ Checks file sizes for sanity
- 📋 Shows current versions

Usage:
```bash
npm run check-vendor
# or
make vendor-check
```

#### `npm run update-vendor`
Manual update script that:
- 🔍 Reads versions from package.json  
- ⬇️ Downloads libraries from CDN
- 🧪 Validates downloaded files
- 📝 Updates version tracking
- 🔄 Creates backups during update

Usage:
```bash
npm run update-vendor
# or  
make vendor-update
```

## GitHub Actions Automation

### Automatic Vendor File Sync (`.github/workflows/sync-vendor.yml`)

**Triggers**: 
- When `package.json` or `package-lock.json` changes
- On Dependabot PRs and after merge to main

**Process**:
1. � Install npm dependencies 
2. 🔄 Run `npm run update-vendor` to sync files
3. 🧪 Validate with `npm run check-vendor`
4. 📝 Commit vendor file changes automatically

**Features**:
- ✅ Automatic sync when dependencies change
- 🤖 Works seamlessly with Dependabot PRs
- 🧪 Validates all downloads before committing
- � No manual intervention required

### Dependabot Integration

**Weekly Schedule**: Dependabot creates PRs every Monday at 9 AM UTC

**Workflow**:
1. 🤖 Dependabot detects new versions
2. � Creates PR updating `package.json`
3. � GitHub Actions automatically syncs vendor files
4. 🧪 All files validated before merge
5. ✅ Ready for review and merge

## Supported Libraries

### marked.js
- **Purpose**: GitHub-flavored markdown parsing
- **Source**: https://github.com/markedjs/marked
- **CDN**: https://cdn.jsdelivr.net/npm/marked@{version}/marked.min.js

### highlight.js  
- **Purpose**: Syntax highlighting for code blocks
- **Source**: https://github.com/highlightjs/highlight.js
- **CDN**: https://cdnjs.cloudflare.com/ajax/libs/highlight.js/{version}/highlight.min.js
- **CSS**: https://cdnjs.cloudflare.com/ajax/libs/highlight.js/{version}/styles/github.min.css

## Adding New Libraries

To add a new vendor library:

1. **Download the library**:
   ```bash
   cd static/vendor
   curl -o newlib.min.js "https://cdn.example.com/newlib@1.0.0/newlib.min.js"
   ```

2. **Update versions.txt**:
   ```
   newlib.js=1.0.0
   newlib.js.url=https://cdn.example.com/newlib@{version}/newlib.min.js
   ```

3. **Update HTML**:
   ```html
   <script src="/static/vendor/newlib.min.js"></script>
   ```

4. **Update scripts**:
   - Add validation to `scripts/check-vendor.js`
   - Add update logic to `scripts/update-vendor.js`
   - Add GitHub API check to Dependabot configuration

## Troubleshooting

### Library not loading
1. Check browser network tab for 404 errors
2. Run `make vendor-check` to validate files
3. Verify HTML references correct paths

### Update script fails
1. Check internet connection
2. Verify GitHub APIs are accessible
3. Check file permissions on scripts
4. Ensure Node.js is available for validation

### GitHub Actions fails
1. Check workflow logs in GitHub Actions tab
2. Verify repository has proper permissions
3. Check API rate limits
4. Ensure jq and curl are available in runner

## Benefits

- 🌐 **Offline-first**: Works without internet
- 🚀 **Fast loading**: No external network requests
- 🔒 **Security**: No CDN dependency or supply chain risks
- 🔄 **Auto-updates**: Automated version management
- 🧪 **Validated**: All updates tested before deployment
- 📦 **Self-contained**: Everything needed is included

## Best Practices

1. **Always test after updates**: Run the application and test markdown/highlighting
2. **Review PRs carefully**: Check changelog links before merging
3. **Keep versions.txt updated**: Manually update if adding libraries
4. **Monitor file sizes**: Large increases might indicate issues
5. **Backup before manual updates**: Scripts create backups automatically
