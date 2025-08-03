# Gote Refactoring Analysis & Recommendations

## Overview

This document outlines comprehensive refactoring recommendations for the Gote notes application based on code analysis performed on August 3, 2025.

## Critical Issues (Immediate Action Required)

### 1. Remove Unused HTTP Handler Code

**Impact**: High - Reduces bundle size and eliminates confusion
**Files to Remove**:

- `pkg/handlers/api.go` (386 lines)
- `pkg/handlers/auth.go`
- `pkg/handlers/web.go`
- `pkg/middleware/auth.go`

**Dependencies to Remove**:

```go
// From go.mod - verify these are unused:
github.com/go-chi/chi/v5 v5.2.2
github.com/labstack/echo/v4 v4.13.3 // if unused elsewhere
```

**Reasoning**: The application uses Wails for frontend-backend communication, making HTTP handlers redundant.

## High Priority Refactoring

### 2. Restructure Business Logic (app.go is 372 lines)

#### Current Issues:

- Mixed responsibilities in app.go
- Business logic tightly coupled to Wails bindings
- Difficult to test individual components

#### Proposed Structure:

```
pkg/services/
├── auth_service.go     # Authentication business logic
├── note_service.go     # Note management business logic
├── config_service.go   # Configuration management
└── backup_service.go   # Backup operations
```

#### Benefits:

- Easier unit testing
- Better separation of concerns
- Improved maintainability
- Cleaner Wails bindings

### 3. Consolidate Frontend Code

#### Current Issues:

- Code duplication between `main.js` (960 lines) and `noteapp-main.js`
- Similar DOM initialization patterns
- Duplicated markdown rendering functions
- Redundant event handling

#### Proposed Structure:

```
frontend/src/
├── main.js              # Entry point and initialization
├── modules/
│   ├── dom-utils.js     # DOM manipulation utilities
│   ├── markdown.js      # Markdown rendering
│   ├── note-manager.js  # Note CRUD operations
│   ├── auth-manager.js  # Authentication handling
│   └── ui-components.js # Reusable UI components
└── constants.js         # Shared constants
```

## Medium Priority Improvements

### 4. CSS Organization & Optimization

#### Current Issues:

- 4 separate CSS files with overlapping styles
- Unused styles in `app.css`
- Inconsistent naming conventions
- No CSS custom properties for theming

#### Proposed Structure:

```
frontend/src/styles/
├── main.css          # Base styles, typography, layout
├── components.css    # Component-specific styles
└── themes.css        # Color schemes and themes
```

#### CSS Custom Properties Example:

```css
:root {
  --primary-color: #667eea;
  --secondary-color: #764ba2;
  --background-gradient: linear-gradient(
    135deg,
    var(--primary-color) 0%,
    var(--secondary-color) 100%
  );
  --border-radius: 15px;
  --box-shadow: 0 4px 15px rgba(0, 0, 0, 0.08);
}
```

### 5. Error Handling Standardization

#### Current Issues:

- Inconsistent error handling patterns
- Some errors only logged, not shown to users
- Missing error recovery mechanisms

#### Proposed Improvements:

```javascript
// Centralized error handler
class ErrorHandler {
  static handle(error, context = "") {
    console.error(`[${context}]`, error);
    this.showUserNotification(this.getUserFriendlyMessage(error));
  }

  static getUserFriendlyMessage(error) {
    // Map technical errors to user-friendly messages
  }

  static showUserNotification(message) {
    // Show toast notification or modal
  }
}
```

### 6. Constants Management

#### Current Issues:

- Magic numbers scattered throughout code
- Hardcoded strings
- No centralized configuration

#### Proposed Constants File:

```javascript
// frontend/src/constants.js
export const UI_CONSTANTS = {
  SEARCH_DEBOUNCE_MS: 300,
  AUTO_SAVE_DELAY_MS: 1000,
  SESSION_TIMEOUT_MS: 30 * 60 * 1000, // 30 minutes
  MAX_NOTE_PREVIEW_LENGTH: 200,
};

export const API_ENDPOINTS = {
  // If needed for future HTTP API
};

export const CSS_CLASSES = {
  NOTE_CARD: "note-card",
  NOTE_SELECTED: "selected",
  MODAL_HIDDEN: "hidden",
};
```

## Security Improvements

### 7. Enhanced Key Derivation

#### Current Issue:

Simple SHA-256 key derivation is vulnerable to rainbow table attacks.

#### Recommendation:

Implement PBKDF2 with proper salt:

```go
// pkg/crypto/crypto.go - Enhanced version
import "golang.org/x/crypto/pbkdf2"

const (
    KeySize = 32
    SaltSize = 16
    Iterations = 100000 // OWASP recommended minimum
)

func DeriveKeySecure(password string, salt []byte) []byte {
    return pbkdf2.Key([]byte(password), salt, Iterations, KeySize, sha256.New)
}
```

**Migration Strategy**: Keep existing `DeriveKey` for backwards compatibility, gradually migrate to secure version.

## Low Priority Optimizations

### 8. File System Watching Optimization

#### Current Implementation:

File system watcher processes all events immediately.

#### Proposed Improvement:

Add debouncing to reduce redundant file operations:

```go
type DebouncedWatcher struct {
    watcher *fsnotify.Watcher
    debounceTime time.Duration
    events map[string]*time.Timer
    mutex sync.RWMutex
}
```

### 9. Memory Usage Optimization

#### Current Issue:

All notes loaded into memory simultaneously.

#### For Large Note Collections:

Consider lazy loading or pagination for collections > 1000 notes.

### 10. TypeScript Migration

#### Benefits:

- Better IDE support
- Compile-time error detection
- Improved refactoring capabilities
- Better documentation through types

#### Migration Strategy:

1. Start with utility functions
2. Gradually convert components
3. Add proper type definitions for Wails bindings

## Performance Monitoring

### Recommended Metrics:

- App startup time
- Note loading time
- Search performance
- Memory usage
- File I/O operations

## Implementation Priority

1. **Week 1**: Remove unused HTTP handlers
2. **Week 2**: Extract service layer from app.go
3. **Week 3**: Consolidate frontend JavaScript
4. **Week 4**: CSS reorganization and constants
5. **Week 5**: Error handling improvements
6. **Week 6**: Security enhancements (key derivation)

## Testing Strategy

### Current State:

No visible test files in the codebase.

### Recommendations:

```
tests/
├── unit/
│   ├── services/     # Test service layer
│   ├── crypto/       # Test encryption/decryption
│   └── storage/      # Test file operations
├── integration/
│   └── app_test.go   # Test Wails bindings
└── e2e/
    └── frontend/     # Test UI workflows
```

## Conclusion

This refactoring plan addresses immediate technical debt while establishing a foundation for future development. The critical issues should be addressed first, followed by the structural improvements that will make the codebase more maintainable and testable.

Estimated effort: 6 weeks of focused development work.
Risk level: Low to Medium (most changes are additive or removal of unused code).
