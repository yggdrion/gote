import "./style.css";
import {
  UI_CONSTANTS,
  CSS_CLASSES,
  ELEMENT_IDS,
  MESSAGES,
} from "./constants.js";

// Import performance utilities
import {
  Debouncer,
  Throttler,
  SearchOptimizer,
  DOMOptimizer,
  PerformanceMonitor,
} from "./performance.js";

// Import Wails runtime
import {
  IsPasswordSet,
  SetPassword,
  VerifyPassword,
  GetAllNotes,
  GetNote,
  CreateNote,
  UpdateNote,
  DeleteNote,
  SearchNotes,
  SyncFromDisk,
  GetSettings,
  UpdateSettings,
  ResetApplication,
  Logout,
  CreateBackup,
  IsConfigured,
  CompleteInitialSetup,
} from "../wailsjs/go/main/App.js";

// State management
let currentUser = null;
let currentNote = null;
let allNotes = [];
let filteredNotes = [];
let searchQuery = "";

// Performance optimization instances
let searchDebouncer = new Debouncer(300); // 300ms debounce for search
let syncThrottler = new Throttler(2000); // Max 1 sync every 2 seconds
let searchOptimizer = new SearchOptimizer();
let domOptimizer = new DOMOptimizer();
let performanceMonitor = new PerformanceMonitor();

// Markdown instance
let markedInstance = null;

// Initialize markdown with syntax highlighting
async function initializeMarked() {
  try {
    console.log("Loading markdown libraries...");

    // Dynamic imports for offline support
    const { marked } = await import("marked");
    const { markedHighlight } = await import("marked-highlight");
    const hljsModule = await import("highlight.js");
    const hljs = hljsModule.default || hljsModule;

    // Import CSS for syntax highlighting
    await import("highlight.js/styles/github.css");

    console.log("All markdown libraries loaded successfully");

    // Configure highlight.js
    hljs.configure({
      ignoreUnescapedHTML: true,
      languages: [
        "javascript",
        "python",
        "html",
        "css",
        "json",
        "bash",
        "go",
        "java",
        "cpp",
        "sql",
        "typescript",
        "markdown",
        "yaml",
        "xml",
      ],
    });

    // Configure marked with highlight extension
    marked.use(
      markedHighlight({
        langPrefix: "hljs language-",
        highlight(code, lang) {
          try {
            const language = hljs.getLanguage(lang) ? lang : "plaintext";
            return hljs.highlight(code, { language }).value;
          } catch (e) {
            console.warn("Highlight error:", e);
            return code;
          }
        },
      })
    );

    // Configure additional options
    marked.setOptions({
      breaks: true,
      gfm: true,
      headerIds: false,
      mangle: false,
    });

    // Store the marked function as our instance
    markedInstance = marked;

    console.log("Markdown initialized successfully with syntax highlighting");
    return true;
  } catch (error) {
    console.error("Error initializing markdown:", error);
    markedInstance = null;
    return false;
  }
}

// Auto-logout functionality
let inactivityTimer = null;
let lastActivityTime = Date.now();
const INACTIVITY_TIMEOUT = 30 * 60 * 1000; // 30 minutes in milliseconds

// DOM elements
let authScreen, mainApp, settingsScreen, passwordSetup, passwordLogin;
let setupPasswordInput, confirmPasswordInput, setupBtn;
let loginPasswordInput, loginBtn, loginError, resetPasswordBtn;
let newNoteBtn, searchInput, searchBtn, clearSearchBtn;
let syncBtn, settingsBtn, notesGrid, noteEditor;
let noteContent, searchResultsHeader, emptyState;
let saveNoteBtn, cancelEditorBtn, createFirstNoteBtn;
let backFromSettings, syncFromSettings;
let createBackupBtn, logoutBtn;
let notesPathInput, passwordHashPathInput, saveSettingsBtn;

// Setup screen elements
let initialSetupScreen, setupNotesPath, setupPasswordHashPath;
let setupMasterPassword, setupConfirmPassword, completeSetupBtn;

// Initialize app when DOM is loaded
document.addEventListener("DOMContentLoaded", async () => {
  console.log("DOM loaded, initializing markdown...");
  await initializeMarked();
  console.log("Markdown initialization complete, initializing DOM...");
  initializeDOM();
  setupEventListeners();
  await checkInitialState();
});

function initializeDOM() {
  // Get DOM elements
  authScreen = document.getElementById("auth-screen");
  mainApp = document.getElementById("main-app");
  settingsScreen = document.getElementById("settings-screen");
  passwordSetup = document.getElementById("password-setup");
  passwordLogin = document.getElementById("password-login");
  setupPasswordInput = document.getElementById("setup-password");
  confirmPasswordInput = document.getElementById("confirm-password");
  setupBtn = document.getElementById("setup-btn");
  loginPasswordInput = document.getElementById("login-password");
  loginBtn = document.getElementById("login-btn");
  loginError = document.getElementById("login-error");
  resetPasswordBtn = document.getElementById("reset-password-btn");
  newNoteBtn = document.getElementById("new-note-btn");
  searchInput = document.getElementById("search-input");
  searchBtn = document.getElementById("search-btn");
  clearSearchBtn = document.getElementById("clear-search-btn");
  syncBtn = document.getElementById("sync-btn");
  settingsBtn = document.getElementById("settings-btn");
  notesGrid = document.getElementById("notes-grid");
  noteEditor = document.getElementById("note-editor");
  noteContent = document.getElementById("note-content");
  searchResultsHeader = document.getElementById("search-results-header");
  emptyState = document.getElementById("empty-state");
  saveNoteBtn = document.getElementById("save-note-btn");
  cancelEditorBtn = document.getElementById("cancel-editor-btn");
  createFirstNoteBtn = document.getElementById("create-first-note");
  backFromSettings = document.getElementById("back-from-settings");
  syncFromSettings = document.getElementById("sync-from-settings");
  createBackupBtn = document.getElementById("create-backup-btn");
  logoutBtn = document.getElementById("logout-btn");

  // Settings input elements
  notesPathInput = document.getElementById("notes-path-input");
  passwordHashPathInput = document.getElementById("password-hash-path-input");
  saveSettingsBtn = document.getElementById("save-settings-btn");

  // Initial setup elements
  initialSetupScreen = document.getElementById("initial-setup-screen");
  setupNotesPath = document.getElementById("setup-notes-path");
  setupPasswordHashPath = document.getElementById("setup-password-hash-path");
  setupMasterPassword = document.getElementById("setup-master-password");
  setupConfirmPassword = document.getElementById("setup-confirm-password");
  completeSetupBtn = document.getElementById("complete-setup-btn");
}

function setupEventListeners() {
  // Initial setup listener
  completeSetupBtn.addEventListener("click", handleCompleteSetup);

  // Authentication listeners
  setupBtn.addEventListener("click", handlePasswordSetup);
  loginBtn.addEventListener("click", handleLogin);
  resetPasswordBtn.addEventListener("click", handlePasswordReset);

  // Enter key listeners for auth
  setupPasswordInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") handlePasswordSetup();
  });
  confirmPasswordInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") handlePasswordSetup();
  });
  loginPasswordInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") handleLogin();
  });

  // Main app listeners
  newNoteBtn.addEventListener("click", createNewNote);
  searchBtn.addEventListener("click", handleSearch);
  clearSearchBtn.addEventListener("click", clearSearch);
  syncBtn.addEventListener("click", handleSync);
  settingsBtn.addEventListener("click", openSettings);
  createFirstNoteBtn.addEventListener("click", createNewNote);

  // Search input listener with debouncing
  searchInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") handleSearch();
  });
  searchInput.addEventListener("input", (e) => {
    if (e.target.value === "") {
      clearSearch();
    } else {
      // Use debounced search for real-time search
      handleSearchInput();
    }
  });

  // Editor listeners
  saveNoteBtn.addEventListener("click", saveCurrentNote);
  cancelEditorBtn.addEventListener("click", closeEditor);

  // Settings listeners
  backFromSettings.addEventListener("click", closeSettings);
  syncFromSettings.addEventListener("click", handleSync);
  createBackupBtn.addEventListener("click", handleCreateBackup);
  saveSettingsBtn.addEventListener("click", handleSaveSettings);
  logoutBtn.addEventListener("click", handleLogout);

  // Global keyboard shortcuts
  document.addEventListener("keydown", handleGlobalKeyboard);
}

async function checkAuthState() {
  try {
    const passwordSet = await IsPasswordSet();
    authScreen.style.display = "flex";
    mainApp.style.display = "none";

    if (passwordSet) {
      passwordSetup.style.display = "none";
      passwordLogin.style.display = "block";
      loginPasswordInput.focus();
    } else {
      passwordSetup.style.display = "block";
      passwordLogin.style.display = "none";
      setupPasswordInput.focus();
    }
  } catch (error) {
    console.error("Error checking auth state:", error);
  }
}

async function handlePasswordSetup() {
  const password = setupPasswordInput.value;
  const confirm = confirmPasswordInput.value;

  if (!password || password.length < UI_CONSTANTS.MIN_PASSWORD_LENGTH) {
    alert("Password must be at least 6 characters long");
    return;
  }

  if (password !== confirm) {
    alert("Passwords do not match");
    return;
  }

  try {
    await SetPassword(password);
    initializeApp();
  } catch (error) {
    console.error("Error setting password:", error);
    alert("Failed to set password");
  }
}

async function handleLogin() {
  const password = loginPasswordInput.value;

  if (!password) {
    showLoginError("Please enter your password");
    return;
  }

  try {
    const isValid = await VerifyPassword(password);
    if (isValid) {
      initializeApp();
    } else {
      showLoginError("Invalid password");
      loginPasswordInput.value = "";
      loginPasswordInput.focus();
    }
  } catch (error) {
    console.error("Error verifying password:", error);
    showLoginError("Authentication failed");
  }
}

function showLoginError(message) {
  loginError.textContent = message;
  loginError.style.display = "block";
  setTimeout(() => {
    loginError.style.display = "none";
  }, UI_CONSTANTS.FADE_DURATION_MS * 10); // Show error for longer
}

async function checkInitialState() {
  try {
    const isConfigured = await IsConfigured();

    if (!isConfigured) {
      // Show initial setup screen
      showInitialSetup();
    } else {
      // Continue with normal auth flow
      checkAuthState();
    }
  } catch (error) {
    console.error("Error checking initial state:", error);
    // Fall back to normal auth flow
    checkAuthState();
  }
}

function showInitialSetup() {
  // Hide other screens
  authScreen.style.display = "none";
  mainApp.style.display = "none";

  // Show setup screen
  initialSetupScreen.style.display = "flex";

  // Pre-populate the paths with defaults
  GetSettings()
    .then((settings) => {
      setupPasswordHashPath.value = settings.passwordHashPath || "";
      // Pre-fill the notes path with default location
      setupNotesPath.value = settings.notesPath || "";
      setupNotesPath.placeholder =
        "Default: " + (settings.notesPath || "Documents/Gote/Notes");
    })
    .catch((err) => {
      console.error("Error loading default settings:", err);
      setupNotesPath.placeholder = "Default: Documents/Gote/Notes";
    });

  // Focus on the first input
  setupNotesPath.focus();
}

async function handleCompleteSetup() {
  const notesPath = setupNotesPath.value.trim();
  const passwordHashPath = setupPasswordHashPath.value.trim();
  const password = setupMasterPassword.value;
  const confirmPassword = setupConfirmPassword.value;

  // Disable button during setup
  completeSetupBtn.disabled = true;
  completeSetupBtn.textContent = "Setting up...";

  try {
    await CompleteInitialSetup(
      notesPath,
      passwordHashPath,
      password,
      confirmPassword
    );

    // Setup completed successfully, initialize the app
    initialSetupScreen.style.display = "none";
    initializeApp();
  } catch (error) {
    console.error("Error completing setup:", error);
    alert("Setup failed: " + error.message);
  } finally {
    completeSetupBtn.disabled = false;
    completeSetupBtn.textContent = "🚀 Complete Setup";
  }
}

async function initializeApp() {
  authScreen.style.display = "none";
  mainApp.style.display = "block";
  currentUser = { authenticated: true };

  // Start activity tracking for auto-logout
  startActivityTracking();

  // Load notes
  await loadNotes();
}

async function loadNotes() {
  performanceMonitor.startTiming("loadNotes");

  try {
    allNotes = (await GetAllNotes()) || [];
    filteredNotes = [...allNotes];

    // Build search index for better performance
    searchOptimizer.buildIndex(allNotes);

    renderNotesList();

    const loadTime = performanceMonitor.endTiming("loadNotes");
    console.log(`Notes loaded in ${loadTime.toFixed(2)}ms`);
  } catch (error) {
    performanceMonitor.endTiming("loadNotes");
    console.error("Error loading notes:", error);
    allNotes = [];
    filteredNotes = [];
    renderNotesList();
  }
}

function renderNotesList() {
  performanceMonitor.startTiming("renderNotes");

  // Use batch DOM operations for better performance
  domOptimizer.batchUpdate(() => {
    notesGrid.innerHTML = "";
    domOptimizer.toggleClass(searchResultsHeader, "hidden", !searchQuery);
    domOptimizer.toggleClass(emptyState, "hidden", filteredNotes.length > 0);

    if (searchQuery) {
      const resultsText = document.getElementById("search-results-text");
      domOptimizer.updateTextContent(
        resultsText,
        `Search results for "${escapeHtml(searchQuery)}" (${
          filteredNotes.length
        } found)`
      );
    }

    if (filteredNotes.length === 0) {
      const emptyText = document.getElementById("empty-state-text");
      const message = searchQuery
        ? `No notes found for "${escapeHtml(
            searchQuery
          )}". <button class="link-button" onclick="clearSearch()">Show all notes</button>`
        : `No notes found. <button class="link-button" onclick="createNewNote()">Create your first note</button>`;
      emptyText.innerHTML = message;
      performanceMonitor.endTiming("renderNotes");
      return;
    }

    // Sort notes by updated date (most recent first)
    const sortedNotes = [...filteredNotes].sort(
      (a, b) => new Date(b.updated_at) - new Date(a.updated_at)
    );

    // Use document fragment for efficient DOM manipulation
    const fragment = document.createDocumentFragment();

    sortedNotes.forEach((note) => {
      const noteCard = domOptimizer.getElement("div", "note-card");
      noteCard.dataset.noteId = note.id;

      const noteActions = domOptimizer.getElement("div", "note-actions");
      noteActions.innerHTML = `
        <button class="edit-btn" data-note-id="${note.id}">Edit</button>
        <button class="delete-btn" data-note-id="${note.id}">Delete</button>
      `;

      const noteContentDiv = domOptimizer.getElement(
        "div",
        "note-content markdown-content"
      );
      noteContentDiv.innerHTML = renderMarkdown(note.content);

      noteCard.appendChild(noteActions);
      noteCard.appendChild(noteContentDiv);

      // Add event listeners
      noteCard.querySelector(".edit-btn").addEventListener("click", (e) => {
        e.stopPropagation();
        editNote(note.id);
      });

      noteCard.querySelector(".delete-btn").addEventListener("click", (e) => {
        e.stopPropagation();
        deleteNote(note.id);
      });

      fragment.appendChild(noteCard);
    });

    notesGrid.appendChild(fragment);

    const renderTime = performanceMonitor.endTiming("renderNotes");
    console.log(`Notes rendered in ${renderTime.toFixed(2)}ms`);
  });
}

function renderMarkdown(content) {
  if (!content || content.trim() === "") {
    return '<em style="color: #999;">Empty note...</em>';
  }

  try {
    if (markedInstance && typeof markedInstance === "function") {
      return markedInstance(content);
    } else if (markedInstance && typeof markedInstance.parse === "function") {
      return markedInstance.parse(content);
    }
  } catch (error) {
    console.error("Error rendering markdown:", error);
  }

  // Simple fallback - just escape HTML and preserve line breaks
  return escapeHtml(content).replace(/\n/g, "<br>");
}

function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

async function createNewNote() {
  try {
    const newNote = await CreateNote("");
    allNotes.push(newNote);
    filteredNotes = searchQuery ? filteredNotes : [...allNotes];
    renderNotesList();
    editNote(newNote.id);
  } catch (error) {
    console.error("Error creating note:", error);
    alert("Failed to create note");
  }
}

async function editNote(noteId) {
  try {
    const note = await GetNote(noteId);
    if (!note) {
      alert("Note not found");
      return;
    }

    currentNote = note;
    noteContent.value = note.content;
    showEditor();
  } catch (error) {
    console.error("Error loading note:", error);
    alert("Failed to load note");
  }
}

function showEditor() {
  noteEditor.classList.remove("hidden");
  noteContent.focus();
}

function closeEditor() {
  noteEditor.classList.add("hidden");
  currentNote = null;
  noteContent.value = "";
}

async function saveCurrentNote() {
  if (!currentNote) return;

  try {
    const updatedNote = await UpdateNote(currentNote.id, noteContent.value);
    currentNote = updatedNote;

    // Update notes list
    const index = allNotes.findIndex((n) => n.id === currentNote.id);
    if (index !== -1) {
      allNotes[index] = updatedNote;
      filteredNotes = searchQuery
        ? filteredNotes.map((n) => (n.id === updatedNote.id ? updatedNote : n))
        : [...allNotes];
      renderNotesList();
    }

    // Show save feedback briefly, then close editor
    const originalText = saveNoteBtn.textContent;
    saveNoteBtn.textContent = "Saved!";
    saveNoteBtn.disabled = true;

    setTimeout(() => {
      saveNoteBtn.textContent = originalText;
      saveNoteBtn.disabled = false;
      closeEditor(); // Close the editor after saving
    }, 500); // Shorter feedback time since we're closing
  } catch (error) {
    console.error("Error saving note:", error);
    alert("Failed to save note");
  }
}

async function deleteNote(noteId) {
  if (!confirm("Are you sure you want to delete this note?")) {
    return;
  }

  try {
    await DeleteNote(noteId);

    // Remove from arrays
    allNotes = allNotes.filter((n) => n.id !== noteId);
    filteredNotes = filteredNotes.filter((n) => n.id !== noteId);

    renderNotesList();

    // Close editor if this note was being edited
    if (currentNote && currentNote.id === noteId) {
      closeEditor();
    }
  } catch (error) {
    console.error("Error deleting note:", error);
    alert("Failed to delete note");
  }
}

async function handleSearch() {
  const query = searchInput.value.trim();
  if (!query) {
    clearSearch();
    return;
  }

  performanceMonitor.startTiming("search");

  try {
    searchQuery = query;

    // Use optimized search with caching and indexing
    filteredNotes = searchOptimizer.search(query, allNotes);

    renderNotesList();
    clearSearchBtn.style.display = "block";

    const searchTime = performanceMonitor.endTiming("search");
    console.log(`Search completed in ${searchTime.toFixed(2)}ms`);
  } catch (error) {
    performanceMonitor.endTiming("search");
    console.error("Error searching notes:", error);
    alert("Search failed");
  }
}

// Optimized search with debouncing for real-time search
function handleSearchInput() {
  searchDebouncer.debounce("search", () => {
    handleSearch();
  });
}

function clearSearch() {
  searchQuery = "";
  searchInput.value = "";
  filteredNotes = [...allNotes];
  renderNotesList();
  clearSearchBtn.style.display = "none";
}

async function handleSync() {
  syncThrottler.throttle("sync", async () => {
    performanceMonitor.startTiming("sync");

    try {
      await SyncFromDisk();
      await loadNotes();

      // Show sync feedback
      const originalText = syncBtn.textContent;
      syncBtn.textContent = "✓";
      setTimeout(() => {
        syncBtn.textContent = originalText;
      }, 1000);

      const syncTime = performanceMonitor.endTiming("sync");
      console.log(`Sync completed in ${syncTime.toFixed(2)}ms`);
    } catch (error) {
      performanceMonitor.endTiming("sync");
      console.error("Error syncing:", error);
      alert("Sync failed");
    }
  });
}

async function openSettings() {
  try {
    const settings = await GetSettings();

    // Update input fields for editing
    notesPathInput.value = settings.notesPath || "";
    passwordHashPathInput.value = settings.passwordHashPath || "";

    // Hide main app and show settings screen
    mainApp.style.display = "none";
    settingsScreen.style.display = "flex";
  } catch (error) {
    console.error("Error loading settings:", error);
  }
}

function closeSettings() {
  // Hide settings screen and show main app
  settingsScreen.style.display = "none";
  mainApp.style.display = "block";

  // Reset activity timer when returning to main app
  resetInactivityTimer();
}

async function handleSaveSettings() {
  const notesPathValue = notesPathInput.value.trim();
  const passwordHashPathValue = passwordHashPathInput.value.trim();

  // Disable the button to prevent multiple clicks
  saveSettingsBtn.disabled = true;
  saveSettingsBtn.textContent = "Saving...";

  try {
    await UpdateSettings(notesPathValue, passwordHashPathValue);

    // Update the display values
    const settings = await GetSettings();
    notesPath.textContent = settings.notesPath || "Not set";
    passwordHashPath.textContent = settings.passwordHashPath || "Not set";

    // Update the input values with the actual saved values
    notesPathInput.value = settings.notesPath || "";
    passwordHashPathInput.value = settings.passwordHashPath || "";

    alert(
      "Settings saved successfully!\n\nYou have been logged out for security reasons. Please log in again to access your notes from the new location."
    );

    // User has been logged out on the backend, so redirect to auth screen
    settingsScreen.style.display = "none";
    authScreen.style.display = "flex";
    mainApp.style.display = "none";

    // Clear any cached data
    allNotes = [];
    filteredNotes = [];
    currentNote = null;

    // Check auth state to show appropriate login screen
    checkAuthState();
  } catch (error) {
    console.error("Error saving settings:", error);
    alert("Failed to save settings: " + error.message);
  } finally {
    // Re-enable the button
    saveSettingsBtn.disabled = false;
    saveSettingsBtn.textContent = "💾 Save Settings";
  }
}

async function handleCreateBackup() {
  try {
    // Show loading state
    const originalText = createBackupBtn.textContent;
    createBackupBtn.textContent = "Creating backup...";
    createBackupBtn.disabled = true;

    // Call the backend to create backup
    const backupPath = await CreateBackup();

    // Show success message with the backup path
    alert(`Backup created successfully!\n\nSaved to: ${backupPath}`);

    // Restore button state
    createBackupBtn.textContent = originalText;
    createBackupBtn.disabled = false;
  } catch (error) {
    console.error("Error creating backup:", error);
    alert("Failed to create backup: " + error.message);

    // Restore button state
    createBackupBtn.textContent = "🗄️ Create Backup";
    createBackupBtn.disabled = false;
  }
}

async function performLogoutCleanup() {
  // Stop activity tracking
  stopActivityTracking();

  // Clear any sensitive data from memory
  allNotes = [];
  filteredNotes = [];
  searchQuery = "";

  // Clear input fields
  loginPasswordInput.value = "";
  setupPasswordInput.value = "";
  confirmPasswordInput.value = "";

  // Clear any displayed errors
  loginError.style.display = "none";
  loginError.textContent = "";

  // Return to auth screen
  authScreen.style.display = "flex";
  mainApp.style.display = "none";
  settingsScreen.style.display = "none";

  // Check auth state to show appropriate login/setup screen
  checkAuthState();
}

async function handleLogout() {
  try {
    // Call backend logout to clear session
    await Logout();

    // Perform cleanup
    await performLogoutCleanup();
  } catch (error) {
    console.error("Error logging out:", error);
    alert("Failed to logout: " + error.message);
  }
}

async function handlePasswordReset() {
  const confirmMessage =
    "Reset password file?\n\n" +
    "This will remove the stored password hash so you can set a new password.\n" +
    "Your encrypted notes will remain safe but inaccessible until you set a new password.\n\n" +
    "Continue?";

  if (!confirm(confirmMessage)) {
    return;
  }

  try {
    await ResetApplication();
    alert("Password reset successfully. You can now set a new password.");
    // Refresh the auth state to show password setup
    checkAuthState();
  } catch (error) {
    console.error("Error resetting password:", error);
    alert("Failed to reset password: " + error.message);
  }
}

// Auto-logout functionality
function resetInactivityTimer() {
  lastActivityTime = Date.now();

  // Clear existing timer
  if (inactivityTimer) {
    clearTimeout(inactivityTimer);
  }

  // Only set timer if user is logged in (not on auth screen)
  if (
    authScreen.style.display === "none" &&
    mainApp.style.display === "block"
  ) {
    inactivityTimer = setTimeout(() => {
      autoLogout();
    }, INACTIVITY_TIMEOUT);
  }
}

function autoLogout() {
  console.log("Auto-logout triggered due to inactivity");

  // Call the same logout function but without user confirmation
  handleLogout();
}

function startActivityTracking() {
  // Track various user activities
  const activityEvents = [
    "mousedown",
    "mousemove",
    "keypress",
    "scroll",
    "touchstart",
    "click",
  ];

  activityEvents.forEach((event) => {
    document.addEventListener(event, resetInactivityTimer, true);
  });

  // Start the initial timer
  resetInactivityTimer();
}

function stopActivityTracking() {
  // Clear the timer
  if (inactivityTimer) {
    clearTimeout(inactivityTimer);
    inactivityTimer = null;
  }
}

// Make some functions globally available for inline event handlers
window.clearSearch = clearSearch;
window.createNewNote = createNewNote;

function handleGlobalKeyboard(e) {
  // Check for Ctrl/Cmd key combinations
  const isCtrlOrCmd = e.ctrlKey || e.metaKey;

  if (isCtrlOrCmd && e.key === "s") {
    // Ctrl+S / Cmd+S: Save note and close editor
    e.preventDefault();
    if (currentNote && !noteEditor.classList.contains("hidden")) {
      saveAndCloseNote();
    }
    return;
  }

  if (isCtrlOrCmd && e.key === "f") {
    // Ctrl+F / Cmd+F: Focus search input
    e.preventDefault();
    // Only if we're in the main app (not auth or settings screen)
    if (
      mainApp.style.display !== "none" &&
      authScreen.style.display === "none" &&
      settingsScreen.style.display === "none"
    ) {
      searchInput.focus();
      searchInput.select();
    }
    return;
  }
}

async function saveAndCloseNote() {
  if (!currentNote) return;

  try {
    const updatedNote = await UpdateNote(currentNote.id, noteContent.value);
    currentNote = updatedNote;

    // Update notes list
    const index = allNotes.findIndex((n) => n.id === currentNote.id);
    if (index !== -1) {
      allNotes[index] = updatedNote;
      filteredNotes = searchQuery
        ? filteredNotes.map((n) => (n.id === updatedNote.id ? updatedNote : n))
        : [...allNotes];
      renderNotesList();
    }

    // Close the editor
    closeEditor();
  } catch (error) {
    console.error("Error saving note:", error);
    alert("Failed to save note");
  }
}
