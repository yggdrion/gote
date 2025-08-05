// Constants for the Gote Notes Application

export const UI_CONSTANTS = {
  // Timing
  SEARCH_DEBOUNCE_MS: 300,
  AUTO_SAVE_DELAY_MS: 1000,
  SESSION_TIMEOUT_MS: 30 * 60 * 1000, // 30 minutes
  AUTO_LOGOUT_WARNING_MS: 5 * 60 * 1000, // 5 minutes before logout

  // UI Limits
  MAX_NOTE_PREVIEW_LENGTH: 200,
  MIN_PASSWORD_LENGTH: 6,

  // File Extensions
  NOTE_FILE_EXTENSION: ".json",
  BACKUP_FILE_PREFIX: "backup-",

  // Animation Durations
  FADE_DURATION_MS: 300,
  SLIDE_DURATION_MS: 300,
};

export const CSS_CLASSES = {
  // UI States
  HIDDEN: "hidden",
  SELECTED: "selected",
  MODAL_OPEN: "modal-open",

  // Components
  NOTE_CARD: "note-card",
  NOTE_EDITOR: "note-editor",
  AUTH_SCREEN: "auth-screen",
  MAIN_APP: "main-app",
  SETTINGS_SCREEN: "settings-screen",

  // Buttons
  BTN_PRIMARY: "btn-primary",
  BTN_SECONDARY: "btn-secondary",
  BTN_DANGER: "btn-danger",
  BTN_WARNING: "btn-warning",

  // Form States
  ERROR: "error",
  SUCCESS: "success",
  LOADING: "loading",
};

export const ELEMENT_IDS = {
  // Screens
  INITIAL_SETUP_SCREEN: "initial-setup-screen",
  AUTH_SCREEN: "auth-screen",
  MAIN_APP: "main-app",
  SETTINGS_SCREEN: "settings-screen",

  // Authentication
  SETUP_PASSWORD: "setup-password",
  CONFIRM_PASSWORD: "confirm-password",
  LOGIN_PASSWORD: "login-password",
  LOGIN_ERROR: "login-error",

  // Notes
  NOTES_GRID: "notes-grid",
  NOTE_EDITOR: "note-editor",
  NOTE_CONTENT: "note-content",
  SEARCH_INPUT: "search-input",
  SEARCH_RESULTS_HEADER: "search-results-header",
  EMPTY_STATE: "empty-state",

  // Buttons
  NEW_NOTE_BTN: "new-note-btn",
  SAVE_NOTE_BTN: "save-note-btn",
  CANCEL_EDITOR_BTN: "cancel-editor-btn",
  SETTINGS_BTN: "settings-btn",
  SEARCH_BTN: "search-btn",
  CLEAR_SEARCH_BTN: "clear-search-btn",
  LOGIN_BTN: "login-btn",
  SETUP_BTN: "setup-btn",
  LOGOUT_BTN: "logout-btn",
  CHANGE_PASSWORD_BTN: "change-password-btn",
  CREATE_BACKUP_BTN: "create-backup-btn",
};

export const MESSAGES = {
  // Success Messages
  NOTE_SAVED: "Note saved successfully",
  NOTE_DELETED: "Note deleted successfully",
  PASSWORD_CHANGED: "Password changed successfully",
  BACKUP_CREATED: "Backup created successfully",

  // Error Messages
  LOGIN_FAILED: "Invalid password. Please try again.",
  SAVE_FAILED: "Failed to save note. Please try again.",
  DELETE_FAILED: "Failed to delete note. Please try again.",
  PASSWORD_MISMATCH: "Passwords do not match.",
  PASSWORD_TOO_SHORT: `Password must be at least ${UI_CONSTANTS.MIN_PASSWORD_LENGTH} characters long.`,
  NETWORK_ERROR: "Network error. Please check your connection.",
  UNKNOWN_ERROR: "An unexpected error occurred. Please try again.",

  // Confirmation Messages
  DELETE_CONFIRM: "Are you sure you want to delete this note?",
  RESET_CONFIRM:
    "Are you sure you want to reset the application? This will remove your password but keep your encrypted notes.",
  LOGOUT_CONFIRM: "Are you sure you want to logout?",

  // Info Messages
  EMPTY_NOTES: "No notes found. Create your first note to get started.",
  SEARCH_NO_RESULTS: "No notes found matching your search.",
  FIRST_TIME_SETUP:
    "Welcome! Please complete the initial setup to get started.",
};

export const KEYBOARD_SHORTCUTS = {
  NEW_NOTE: "Ctrl+N",
  SAVE_NOTE: "Ctrl+S",
  SEARCH: "Ctrl+F",
  SETTINGS: "Ctrl+,",
  CLOSE_EDITOR: "Escape",
};

export const STORAGE_KEYS = {
  // Local storage keys if needed for frontend preferences
  THEME: "gote_theme",
  LAST_SEARCH: "gote_last_search",
  WINDOW_SIZE: "gote_window_size",
};
