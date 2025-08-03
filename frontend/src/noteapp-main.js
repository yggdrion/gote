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
} from "../wailsjs/go/main/App.js";

// State management
let currentUser = null;
let currentNote = null;
let allNotes = [];
let filteredNotes = [];
let searchQuery = "";

// DOM elements
let authScreen, mainApp, passwordSetup, passwordLogin;
let setupPasswordInput, confirmPasswordInput, setupBtn;
let loginPasswordInput, loginBtn, loginError;
let newNoteBtn, searchInput, searchBtn, clearSearchBtn;
let syncBtn, settingsBtn, notesGrid, noteEditor;
let noteContent, notePreview, searchResultsHeader, emptyState;
let saveNoteBtn, closeEditorBtn, cancelEditorBtn, createFirstNoteBtn;
let settingsModal,
  closeSettings,
  currentPassword,
  newPassword,
  confirmNewPassword;
let changePasswordBtn, notesPath, passwordHashPath;

// Initialize app when DOM is loaded
document.addEventListener("DOMContentLoaded", () => {
  initializeDOM();
  setupEventListeners();
  checkAuthState();
});

function initializeDOM() {
  // Get DOM elements
  authScreen = document.getElementById("auth-screen");
  mainApp = document.getElementById("main-app");
  passwordSetup = document.getElementById("password-setup");
  passwordLogin = document.getElementById("password-login");
  setupPasswordInput = document.getElementById("setup-password");
  confirmPasswordInput = document.getElementById("confirm-password");
  setupBtn = document.getElementById("setup-btn");
  loginPasswordInput = document.getElementById("login-password");
  loginBtn = document.getElementById("login-btn");
  loginError = document.getElementById("login-error");
  newNoteBtn = document.getElementById("new-note-btn");
  searchInput = document.getElementById("search-input");
  searchBtn = document.getElementById("search-btn");
  clearSearchBtn = document.getElementById("clear-search-btn");
  syncBtn = document.getElementById("sync-btn");
  settingsBtn = document.getElementById("settings-btn");
  notesGrid = document.getElementById("notes-grid");
  noteEditor = document.getElementById("note-editor");
  noteContent = document.getElementById("note-content");
  notePreview = document.getElementById("note-preview");
  searchResultsHeader = document.getElementById("search-results-header");
  emptyState = document.getElementById("empty-state");
  saveNoteBtn = document.getElementById("save-note-btn");
  closeEditorBtn = document.getElementById("close-editor-btn");
  cancelEditorBtn = document.getElementById("cancel-editor-btn");
  createFirstNoteBtn = document.getElementById("create-first-note");
  settingsModal = document.getElementById("settings-modal");
  closeSettings = document.getElementById("close-settings");
  currentPassword = document.getElementById("current-password");
  newPassword = document.getElementById("new-password");
  confirmNewPassword = document.getElementById("confirm-new-password");
  changePasswordBtn = document.getElementById("change-password-btn");
  notesPath = document.getElementById("notes-path");
  passwordHashPath = document.getElementById("password-hash-path");
}

function setupEventListeners() {
  // Authentication listeners
  setupBtn.addEventListener("click", handlePasswordSetup);
  loginBtn.addEventListener("click", handleLogin);

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
  closeEditorBtn.addEventListener("click", closeEditor);
  cancelEditorBtn.addEventListener("click", closeEditor);
  noteContent.addEventListener("input", updatePreview);

  // Settings listeners
  closeSettings.addEventListener("click", closeSettingsModal);
  changePasswordBtn.addEventListener("click", handleChangePassword);

  // Modal click outside to close
  settingsModal.addEventListener("click", (e) => {
    if (e.target === settingsModal) {
      closeSettingsModal();
    }
  });
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
  if (!content) return "";

  let html = escapeHtml(content);

  // Headers
  html = html.replace(/^### (.*$)/gim, "<h3>$1</h3>");
  html = html.replace(/^## (.*$)/gim, "<h2>$1</h2>");
  html = html.replace(/^# (.*$)/gim, "<h1>$1</h1>");

  // Bold
  html = html.replace(/\*\*(.*?)\*\*/gim, "<strong>$1</strong>");

  // Italic
  html = html.replace(/\*(.*?)\*/gim, "<em>$1</em>");

  // Code blocks
  html = html.replace(/```([^`]*?)```/gims, "<pre><code>$1</code></pre>");

  // Inline code
  html = html.replace(/`([^`]+)`/gim, "<code>$1</code>");

  // Task lists
  html = html.replace(
    /^\s*- \[x\] (.*)$/gim,
    '<input type="checkbox" checked disabled> $1'
  );
  html = html.replace(
    /^\s*- \[ \] (.*)$/gim,
    '<input type="checkbox" disabled> $1'
  );

  // Regular lists
  html = html.replace(/^\s*- (.*)$/gim, "<li>$1</li>");
  html = html.replace(/(<li>.*<\/li>)/gims, "<ul>$1</ul>");

  // Links
  html = html.replace(
    /\[([^\]]+)\]\(([^)]+)\)/gim,
    '<a href="$2" target="_blank">$1</a>'
  );

  // Blockquotes
  html = html.replace(/^> (.*)$/gim, "<blockquote>$1</blockquote>");

  // Line breaks
  html = html.replace(/\n/gim, "<br>");

  return html;
}

function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

async function createNewNote() {
  try {
    const newNote = await CreateNote(
      "# New Note\n\nStart writing your note here..."
    );
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
    updatePreview();
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
  notePreview.innerHTML = "";
}

function updatePreview() {
  if (!noteContent.value) {
    notePreview.innerHTML =
      '<p style="color: #999; font-style: italic;">Preview will appear here...</p>';
    return;
  }

  notePreview.innerHTML = renderMarkdown(noteContent.value);
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

    // Show save feedback
    const originalText = saveNoteBtn.textContent;
    saveNoteBtn.textContent = "Saved!";
    saveNoteBtn.disabled = true;
    setTimeout(() => {
      saveNoteBtn.textContent = originalText;
      saveNoteBtn.disabled = false;
    }, 1000);
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
    settingsModal.style.display = "flex";
  } catch (error) {
    console.error("Error loading settings:", error);
  }
}

function closeSettingsModal() {
  settingsModal.style.display = "none";
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
    alert("Password changed successfully");
    closeSettingsModal();
  } catch (error) {
    console.error("Error changing password:", error);
    alert("Failed to change password: " + error.message);
  }
}

// Make some functions globally available for inline event handlers
window.clearSearch = clearSearch;
window.createNewNote = createNewNote;
