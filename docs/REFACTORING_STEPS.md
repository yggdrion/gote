# Immediate Refactoring Action Plan

## Step 1: Remove Unused HTTP Handler Code (30 minutes)

### Files to Delete:

```bash
# Run these commands to remove unused code:
rm pkg/handlers/api.go
rm pkg/handlers/auth.go
rm pkg/handlers/web.go
rm pkg/middleware/auth.go
rmdir pkg/handlers
rmdir pkg/middleware
```

### Update go.mod:

```bash
# Remove unused dependencies
go mod edit -droprequire github.com/go-chi/chi/v5
go mod edit -droprequire github.com/labstack/echo/v4
go mod tidy
```

### Verify build still works:

```bash
go build -o build/gote.exe
```

## Step 2: Create Service Layer (2 hours)

### Already Created:

- ✅ `pkg/services/auth_service.go`
- ✅ `pkg/services/note_service.go`

### Next: Update app.go to use services

```go
// In app.go, replace direct component usage with services:
type App struct {
    ctx         context.Context
    authService *services.AuthService
    noteService *services.NoteService
    config      *config.Config
    currentKey  []byte
}

func (a *App) startup(ctx context.Context) {
    a.ctx = ctx
    // Initialize services instead of direct components
    authManager := auth.NewManager(cfg.PasswordHashPath)
    a.authService = services.NewAuthService(authManager, cfg)
    // ... etc
}
```

## Step 3: Frontend Consolidation (3 hours)

### Create New Structure:

```bash
mkdir frontend/src/modules
```

### Extract Common Functions:

1. Move duplicate markdown rendering to `modules/markdown.js`
2. Consolidate DOM utilities in `modules/dom-utils.js`
3. Combine authentication logic in `modules/auth-manager.js`

### Remove Duplicated Code:

- Decide which file to keep as main entry point
- Move unique functionality to modules
- Remove the other file

## Step 4: CSS Cleanup (1 hour)

### Audit Current CSS:

```bash
# Check what's actually used in app.css
grep -r "input-box\|logo\|result" frontend/
```

### If app.css is unused:

- Remove `app.css` import from main.js
- Delete `app.css`
- Consolidate remaining CSS files

## Quick Wins (15 minutes each)

### Add Missing go.mod dependency:

```bash
go get golang.org/x/crypto
```

### Fix Minor Issues:

1. Add error handling to `backup.go` defer statements
2. Add constants for magic numbers in JavaScript
3. Standardize error message format

## Verification Steps

After each step:

1. ✅ Build succeeds: `go build`
2. ✅ Frontend loads: Check in browser/app
3. ✅ Core functions work: Create/edit/delete notes
4. ✅ Authentication works: Login/logout
5. ✅ File operations work: Save/load notes

## Risk Mitigation

### Before Starting:

- ✅ Commit current working state
- ✅ Create backup: `git branch refactor-backup`

### During Refactoring:

- Test after each major change
- Keep changes atomic (one logical change per commit)
- If issues arise, rollback individual changes

## Time Estimate

- **Total time**: 6-7 hours
- **Can be done incrementally**: Yes
- **Breaking changes**: Minimal (mostly removing unused code)
- **User impact**: None (internal code organization)

## Success Criteria

1. ✅ Application builds and runs without errors
2. ✅ All existing functionality works as before
3. ✅ Codebase is more organized and maintainable
4. ✅ Reduced file count and complexity
5. ✅ No unused dependencies in go.mod
