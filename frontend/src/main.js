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
let authScreen,
  mainApp,
  passwordSetup,
  passwordLogin,
  setupPasswordInput,
  confirmPasswordInput,
  setupBtn;
let loginPasswordInput,
  loginBtn,
  loginError,
  newNoteBtn,
  searchInput,
  searchBtn,
  clearSearchBtn;
let syncBtn,
  settingsBtn,
  notesList,
  noteEditor,
  welcomeScreen,
  noteContent,
  notePreview;
let saveBtn,
  deleteBtn,
  noteInfo,
  settingsModal,
  closeSettings,
  currentPassword,
  newPassword,
  confirmNewPassword;
let changePasswordBtn, notesPath;

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
  notesList = document.getElementById("notes-list");
  noteEditor = document.getElementById("note-editor");
  welcomeScreen = document.getElementById("welcome-screen");
  noteContent = document.getElementById("note-content");
  notePreview = document.getElementById("note-preview");
  saveBtn = document.getElementById("save-btn");
  deleteBtn = document.getElementById("delete-btn");
  noteInfo = document.getElementById("note-info");
  settingsModal = document.getElementById("settings-modal");
  closeSettings = document.getElementById("close-settings");
  currentPassword = document.getElementById("current-password");
  newPassword = document.getElementById("new-password");
  confirmNewPassword = document.getElementById("confirm-new-password");
  changePasswordBtn = document.getElementById("change-password-btn");
  notesPath = document.getElementById("notes-path");
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
  saveBtn.addEventListener("click", saveCurrentNote);
  deleteBtn.addEventListener("click", deleteCurrentNote);
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

  // Show welcome screen initially
  showWelcomeScreen();
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
  }
}

function renderNotesList() {
  notesList.innerHTML = "";

  if (filteredNotes.length === 0) {
    notesList.innerHTML =
      '<div style="padding: 1rem; text-align: center; color: #a0a0a0;">No notes found</div>';
    return;
  }

  // Sort notes by updated date (most recent first)
  const sortedNotes = [...filteredNotes].sort(
    (a, b) => new Date(b.updated_at) - new Date(a.updated_at)
  );

  sortedNotes.forEach((note) => {
    const noteItem = document.createElement("div");
    noteItem.className = "note-item";
    noteItem.dataset.noteId = note.id;

    const title = extractTitle(note.content) || "Untitled";
    const excerpt = extractExcerpt(note.content);
    const date = formatDate(note.updated_at);

    noteItem.innerHTML = `
            <div class="note-title">${escapeHtml(title)}</div>
            <div class="note-excerpt">${escapeHtml(excerpt)}</div>
            <div class="note-date">${date}</div>
        `;

    noteItem.addEventListener("click", () => openNote(note.id));
    notesList.appendChild(noteItem);
  });
}

function extractTitle(content) {
  if (!content) return "";
  const lines = content.split("\n");
  const firstLine = lines[0].trim();

  // Remove markdown heading syntax
  if (firstLine.startsWith("#")) {
    return firstLine.replace(/^#+\s*/, "");
  }

  return firstLine || "Untitled";
}

function extractExcerpt(content) {
  if (!content) return "";

  // Remove title line and get first few lines
  let lines = content.split("\n");
  if (lines[0].startsWith("#")) {
    lines = lines.slice(1);
  }

  const text = lines.join(" ").trim();
  return text.length > 100 ? text.substring(0, 100) + "..." : text;
}

function formatDate(dateString) {
  const date = new Date(dateString);
  const now = new Date();
  const diffTime = Math.abs(now - date);
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

  if (diffDays === 1) return "Today";
  if (diffDays === 2) return "Yesterday";
  if (diffDays <= 7) return `${diffDays - 1} days ago`;

  return date.toLocaleDateString();
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
    filteredNotes = [...allNotes];
    renderNotesList();
    openNote(newNote.id);
  } catch (error) {
    console.error("Error creating note:", error);
    alert("Failed to create note");
  }
}

async function openNote(noteId) {
  try {
    const note = await GetNote(noteId);
    if (!note) {
      alert("Note not found");
      return;
    }

    currentNote = note;
    showNoteEditor();

    // Update UI
    noteContent.value = note.content;
    updatePreview();
    updateNoteInfo();

    // Highlight active note in list
    document.querySelectorAll(".note-item").forEach((item) => {
      item.classList.remove("active");
      if (item.dataset.noteId === noteId) {
        item.classList.add("active");
      }
    });

    // Focus editor
    noteContent.focus();
  } catch (error) {
    console.error("Error opening note:", error);
    alert("Failed to open note");
  }
}

function showWelcomeScreen() {
  noteEditor.style.display = "none";
  welcomeScreen.style.display = "flex";
  currentNote = null;
}

function showNoteEditor() {
  welcomeScreen.style.display = "none";
  noteEditor.style.display = "flex";

  // Create editor content div if it doesn't exist
  if (!noteEditor.querySelector(".editor-content")) {
    const editorContent = document.createElement("div");
    editorContent.className = "editor-content";
    editorContent.appendChild(noteContent);
    editorContent.appendChild(document.querySelector(".preview-container"));
    noteEditor.appendChild(editorContent);
  }
}

function updatePreview() {
  if (!noteContent.value) {
    notePreview.innerHTML =
      '<p style="color: #a0a0a0; font-style: italic;">Preview will appear here...</p>';
    return;
  }

  // Simple markdown rendering (you might want to use a proper markdown library)
  let html = noteContent.value;

  // Headers
  html = html.replace(/^### (.*$)/gim, "<h3>$1</h3>");
  html = html.replace(/^## (.*$)/gim, "<h2>$1</h2>");
  html = html.replace(/^# (.*$)/gim, "<h1>$1</h1>");

  // Bold
  html = html.replace(/\*\*(.*)\*\*/gim, "<strong>$1</strong>");

  // Italic
  html = html.replace(/\*(.*)\*/gim, "<em>$1</em>");

  // Code blocks
  html = html.replace(/```([^`]+)```/gim, "<pre><code>$1</code></pre>");

  // Inline code
  html = html.replace(/`([^`]+)`/gim, "<code>$1</code>");

  // Line breaks
  html = html.replace(/\n/gim, "<br>");

  notePreview.innerHTML = html;
}

function updateNoteInfo() {
  if (!currentNote) return;

  const title = extractTitle(currentNote.content) || "Untitled";
  const date = formatDate(currentNote.updated_at);
  noteInfo.textContent = `${title} - ${date}`;
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

    updateNoteInfo();

    // Show save feedback
    const originalText = saveBtn.textContent;
    saveBtn.textContent = "Saved!";
    setTimeout(() => {
      saveBtn.textContent = originalText;
    }, 1000);
  } catch (error) {
    console.error("Error saving note:", error);
    alert("Failed to save note");
  }
}

async function deleteCurrentNote() {
  if (!currentNote) return;

  if (!confirm("Are you sure you want to delete this note?")) {
    return;
  }

  try {
    await DeleteNote(currentNote.id);

    // Remove from arrays
    allNotes = allNotes.filter((n) => n.id !== currentNote.id);
    filteredNotes = filteredNotes.filter((n) => n.id !== currentNote.id);

    renderNotesList();
    showWelcomeScreen();
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
    clearSearchBtn.style.display = "inline-block";
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
