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
  SaveImageFromClipboard,
  GetImage,
  DeleteImage,
  ListImages,
  GetImageAsDataURL,
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

  // Clipboard handling for images
  document.addEventListener("paste", handleClipboardPaste);
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

      // Load images after HTML is inserted into DOM
      loadImagesInDOM(noteContentDiv);

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
    let html;
    if (markedInstance && typeof markedInstance === "function") {
      html = markedInstance(content);
    } else if (markedInstance && typeof markedInstance.parse === "function") {
      html = markedInstance.parse(content);
    } else {
      // Simple fallback - just escape HTML and preserve line breaks
      html = escapeHtml(content).replace(/\n/g, "<br>");
    }

    // Post-process to handle custom image syntax
    html = processCustomImages(html);

    return html;
  } catch (error) {
    console.error("Error rendering markdown:", error);
    // Simple fallback - just escape HTML and preserve line breaks
    return escapeHtml(content).replace(/\n/g, "<br>");
  }
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

// Image handling functions

// Handle clipboard paste events for images
async function handleClipboardPaste(event) {
  // Only handle paste when editing a note
  if (!currentNote || !noteContent) {
    return;
  }

  const items = event.clipboardData?.items;
  if (!items) return;

  for (let i = 0; i < items.length; i++) {
    const item = items[i];

    // Check if the item is an image
    if (item.type.startsWith("image/")) {
      event.preventDefault(); // Prevent default paste behavior

      try {
        const file = item.getAsFile();
        if (!file) continue;

        // Show loading indicator
        showImageUploadProgress("Uploading image...");

        // Convert file to base64
        const base64Data = await fileToBase64(file);

        // Save image via backend
        const image = await SaveImageFromClipboard(base64Data, item.type);

        // Insert image reference into the note content
        insertImageIntoNote(image.id, image.filename);

        showImageUploadProgress("Image uploaded successfully!", "success");
        setTimeout(() => hideImageUploadProgress(), 2000);
      } catch (error) {
        console.error("Error uploading image:", error);
        showImageUploadProgress("Failed to upload image", "error");
        setTimeout(() => hideImageUploadProgress(), 3000);
      }
      break; // Only handle the first image
    }
  }
}

// Convert file to base64 string
function fileToBase64(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      // Remove the data:... prefix to get just the base64 data
      const base64 = reader.result.split(",")[1];
      resolve(base64);
    };
    reader.onerror = reject;
    reader.readAsDataURL(file);
  });
}

// Insert image reference into note content at cursor position
function insertImageIntoNote(imageId, filename) {
  const imageMarkdown = `![${filename}](image:${imageId})`;

  // Get current cursor position
  const cursorPos = noteContent.selectionStart;
  const textBefore = noteContent.value.substring(0, cursorPos);
  const textAfter = noteContent.value.substring(noteContent.selectionEnd);

  // Insert image markdown at cursor position
  noteContent.value = textBefore + imageMarkdown + textAfter;

  // Update cursor position
  const newCursorPos = cursorPos + imageMarkdown.length;
  noteContent.setSelectionRange(newCursorPos, newCursorPos);

  // Trigger input event to update preview
  noteContent.dispatchEvent(new Event("input", { bubbles: true }));
}

// Show image upload progress
function showImageUploadProgress(message, type = "info") {
  let progressDiv = document.getElementById("image-upload-progress");

  if (!progressDiv) {
    progressDiv = document.createElement("div");
    progressDiv.id = "image-upload-progress";
    progressDiv.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      padding: 12px 20px;
      border-radius: 4px;
      color: white;
      font-weight: 500;
      z-index: 1000;
      transition: all 0.3s ease;
    `;
    document.body.appendChild(progressDiv);
  }

  // Set background color based on type
  const colors = {
    info: "#2196F3",
    success: "#4CAF50",
    error: "#f44336",
  };

  progressDiv.style.backgroundColor = colors[type] || colors.info;
  progressDiv.textContent = message;
  progressDiv.style.display = "block";
}

// Hide image upload progress
function hideImageUploadProgress() {
  const progressDiv = document.getElementById("image-upload-progress");
  if (progressDiv) {
    progressDiv.style.display = "none";
  }
}

// Load images in DOM element
async function loadImagesInDOM(element) {
  // Find all images with data-image-id attribute within the element
  const images = element.querySelectorAll("img[data-image-id]");

  for (const img of images) {
    const imageId = img.getAttribute("data-image-id");
    try {
      console.log(`Loading image ${imageId}...`);
      const dataUrl = await GetImageAsDataURL(imageId);
      img.src = dataUrl;
      img.removeAttribute("data-image-id"); // Remove the temporary attribute
      console.log(`Image ${imageId} loaded successfully`);
    } catch (error) {
      console.error(`Failed to load image ${imageId}:`, error);
      img.src =
        "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTIxIDlWN0MxOSA1IDIwIDMgMTggM0g2QzQgMyAzIDUgMyA3VjE3QzMgMTkgNCAyMSA2IDIxSDE4QzIwIDIxIDIxIDE5IDIxIDE3VjE1IiBzdHJva2U9IiNmZjAwMDAiIHN0cm9rZS13aWR0aD0iMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIi8+CjxwYXRoIGQ9Ik0xIDFMMjMgMjMiIHN0cm9rZT0iI2ZmMDAwMCIgc3Ryb2tlLXdpZHRoPSIyIiBzdHJva2UtbGluZWNhcD0icm91bmQiIHN0cm9rZS1saW5lam9pbj0icm91bmQiLz4KPC9zdmc+"; // Error icon
      img.alt = `Failed to load image: ${img.alt}`;
      img.title = `Image not found: ${imageId}`;
    }
  }
}

// Process custom image syntax in rendered HTML
function processCustomImages(html) {
  // Replace our custom image syntax with proper img tags
  // Look for <img src="image:imageId" ...> patterns that marked.js created
  const imageRegex = /<img([^>]*?)src="image:([^"]+)"([^>]*?)>/g;

  return html.replace(imageRegex, (match, beforeSrc, imageId, afterSrc) => {
    console.log(`Processing custom image: ${imageId}`);
    return `<img${beforeSrc}src="data:image/png;base64,loading..." data-image-id="${imageId}" class="note-image"${afterSrc} style="max-width: 100%; height: auto; border-radius: 4px; margin: 8px 0;">`;
  });
}

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
