import "./app.css";
import "./style.css";

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
  ChangePassword,
  ResetApplication,
  Logout,
} from "../wailsjs/go/main/App.js";

// State management
let currentUser = null;
let currentNote = null;
let allNotes = [];
let filteredNotes = [];
let searchQuery = "";

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
let backFromSettings,
  syncFromSettings,
  currentPassword,
  newPassword,
  confirmNewPassword;
let changePasswordBtn, createBackupBtn, notesPath, passwordHashPath, logoutBtn;

// Initialize app when DOM is loaded
document.addEventListener("DOMContentLoaded", async () => {
  console.log("DOM loaded, initializing markdown...");
  await initializeMarked();
  console.log("Markdown initialization complete, initializing DOM...");
  initializeDOM();
  setupEventListeners();
  checkAuthState();
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
  currentPassword = document.getElementById("current-password");
  newPassword = document.getElementById("new-password");
  confirmNewPassword = document.getElementById("confirm-new-password");
  changePasswordBtn = document.getElementById("change-password-btn");
  createBackupBtn = document.getElementById("create-backup-btn");
  notesPath = document.getElementById("notes-path");
  passwordHashPath = document.getElementById("password-hash-path");
  logoutBtn = document.getElementById("logout-btn");
}

function setupEventListeners() {
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

  // Search input listener
  searchInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") handleSearch();
  });
  searchInput.addEventListener("input", (e) => {
    if (e.target.value === "") {
      clearSearch();
    }
  });

  // Editor listeners
  saveNoteBtn.addEventListener("click", saveCurrentNote);
  cancelEditorBtn.addEventListener("click", closeEditor);

  // Settings listeners
  backFromSettings.addEventListener("click", closeSettings);
  syncFromSettings.addEventListener("click", handleSync);
  changePasswordBtn.addEventListener("click", handleChangePassword);
  createBackupBtn.addEventListener("click", handleCreateBackup);
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

  if (!password || password.length < 6) {
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
  }, 3000);
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
  try {
    allNotes = (await GetAllNotes()) || [];
    filteredNotes = [...allNotes];
    renderNotesList();
  } catch (error) {
    console.error("Error loading notes:", error);
    allNotes = [];
    filteredNotes = [];
    renderNotesList();
  }
}

function renderNotesList() {
  notesGrid.innerHTML = "";
  searchResultsHeader.style.display = "none";
  emptyState.style.display = "none";

  if (searchQuery) {
    searchResultsHeader.style.display = "block";
    const resultsText = document.getElementById("search-results-text");
    resultsText.innerHTML = `Search results for "<strong>${escapeHtml(
      searchQuery
    )}</strong>" (${filteredNotes.length} found)`;
  }

  if (filteredNotes.length === 0) {
    emptyState.style.display = "block";
    const emptyText = document.getElementById("empty-state-text");
    if (searchQuery) {
      emptyText.innerHTML = `No notes found for "${escapeHtml(
        searchQuery
      )}". <button class="link-button" onclick="clearSearch()">Show all notes</button>`;
    } else {
      emptyText.innerHTML = `No notes found. <button class="link-button" onclick="createNewNote()">Create your first note</button>`;
    }
    return;
  }

  // Sort notes by updated date (most recent first)
  const sortedNotes = [...filteredNotes].sort(
    (a, b) => new Date(b.updated_at) - new Date(a.updated_at)
  );

  sortedNotes.forEach((note) => {
    const noteCard = document.createElement("div");
    noteCard.className = "note-card";
    noteCard.dataset.noteId = note.id;

    const noteActions = document.createElement("div");
    noteActions.className = "note-actions";
    noteActions.innerHTML = `
      <button class="edit-btn" data-note-id="${note.id}">Edit</button>
      <button class="delete-btn" data-note-id="${note.id}">Delete</button>
    `;

    const noteContentDiv = document.createElement("div");
    noteContentDiv.className = "note-content markdown-content";
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

    notesGrid.appendChild(noteCard);
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

  try {
    searchQuery = query;
    filteredNotes = (await SearchNotes(query)) || [];
    renderNotesList();
    clearSearchBtn.style.display = "block";
  } catch (error) {
    console.error("Error searching notes:", error);
    alert("Search failed");
  }
}

function clearSearch() {
  searchQuery = "";
  searchInput.value = "";
  filteredNotes = [...allNotes];
  renderNotesList();
  clearSearchBtn.style.display = "none";
}

async function handleSync() {
  try {
    await SyncFromDisk();
    await loadNotes();

    // Show sync feedback
    const originalText = syncBtn.textContent;
    syncBtn.textContent = "✓";
    setTimeout(() => {
      syncBtn.textContent = originalText;
    }, 1000);
  } catch (error) {
    console.error("Error syncing:", error);
    alert("Sync failed");
  }
}

async function openSettings() {
  try {
    const settings = await GetSettings();
    notesPath.textContent = settings.notesPath || "Not set";
    passwordHashPath.textContent = settings.passwordHashPath || "Not set";

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

  // Clear password fields
  currentPassword.value = "";
  newPassword.value = "";
  confirmNewPassword.value = "";
}

async function handleChangePassword() {
  const current = currentPassword.value;
  const newPass = newPassword.value;
  const confirm = confirmNewPassword.value;

  if (!current || !newPass || !confirm) {
    alert("Please fill in all password fields");
    return;
  }

  if (newPass.length < 6) {
    alert("New password must be at least 6 characters long");
    return;
  }

  if (newPass !== confirm) {
    alert("New passwords do not match");
    return;
  }

  try {
    await ChangePassword(current, newPass);
    alert(
      "Password changed successfully. You will be logged out and need to log in with your new password."
    );

    // Password change clears the session, so perform logout cleanup
    await performLogoutCleanup();
  } catch (error) {
    console.error("Error changing password:", error);
    alert("Failed to change password: " + error.message);
  }
}

async function handleCreateBackup() {
  try {
    // Show loading state
    const originalText = createBackupBtn.textContent;
    createBackupBtn.textContent = "Creating backup...";
    createBackupBtn.disabled = true;

    // Note: This would need to be implemented in the Go backend
    // For now just show a placeholder message
    alert("Backup functionality will be implemented soon");

    // Restore button state
    createBackupBtn.textContent = originalText;
    createBackupBtn.disabled = false;
  } catch (error) {
    console.error("Error creating backup:", error);
    alert("Failed to create backup");

    // Restore button state
    createBackupBtn.textContent = "🗄️ Create Backup Snapshot";
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
  currentPassword.value = "";
  newPassword.value = "";
  confirmNewPassword.value = "";

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
