import "./style.css";
import {
  UI_CONSTANTS,
  CSS_CLASSES,
  ELEMENT_IDS,
  MESSAGES,
} from "./constants.js";

// Import Wails runtime
import {
  IsPasswordSet,
  SetPassword,
  VerifyPassword,
  GetAllNotes,
  GetNote,
  CreateNote,
  CreateNoteWithCategory,
  UpdateNote,
  UpdateNoteCategory,
  DeleteNote,
  GetNotesByCategory,
  MoveToTrash,
  RestoreFromTrash,
  PermanentlyDeleteNote,
  SearchNotes,
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
  CleanupOrphanedImages,
  GetImageStats,
} from "../wailsjs/go/main/App.js";

// Import Wails runtime for browser functionality
import {
  BrowserOpenURL,
  ClipboardGetText,
} from "../wailsjs/runtime/runtime.js";

// State management
let currentUser = null;
let currentNote = null;
let originalNoteContent = ""; // Track original content to detect changes
let allNotes = [];
let filteredNotes = [];
let searchQuery = "";
let currentCategory = "private"; // Track currently selected category

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
let newNoteBtn, newNoteFromClipboardBtn, searchInput, searchBtn, clearSearchBtn;
let settingsBtn, trashBtn, notesGrid, noteEditor;
let noteContent, searchResultsHeader, emptyState;
let saveNoteBtn, cancelEditorBtn, createFirstNoteBtn;
let backFromSettings;
let createBackupBtn, logoutBtn;
let notesPathInput, passwordHashPathInput, saveSettingsBtn;

// Category filter elements
let filterPrivateBtn, filterWorkBtn;

// Editor category filter elements
let editorFilterPrivateBtn, editorFilterWorkBtn;

// Setup screen elements
let initialSetupScreen, setupNotesPath, setupPasswordHashPath;
let setupMasterPassword, setupConfirmPassword, completeSetupBtn;

// Delete confirmation modal elements
let deleteModal, confirmDeleteBtn, cancelDeleteBtn;
let closeEditorModal, saveAndCloseBtn, discardChangesBtn, cancelCloseBtn;
let noteToDelete = null;

// Image modal elements
let imageModal, imageModalImage, imageModalInfo;

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
  newNoteFromClipboardBtn = document.getElementById(
    "new-note-from-clipboard-btn"
  );
  searchInput = document.getElementById("search-input");
  searchBtn = document.getElementById("search-btn");
  clearSearchBtn = document.getElementById("clear-search-btn");
  trashBtn = document.getElementById("trash-btn");
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
  createBackupBtn = document.getElementById("create-backup-btn");
  logoutBtn = document.getElementById("logout-btn");

  // Category filter elements
  filterPrivateBtn = document.getElementById("filter-private");
  filterWorkBtn = document.getElementById("filter-work");

  // Editor category filter elements
  editorFilterPrivateBtn = document.getElementById("editor-filter-private");
  editorFilterWorkBtn = document.getElementById("editor-filter-work");

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

  // Delete confirmation modal elements
  deleteModal = document.getElementById("delete-confirmation-modal");
  confirmDeleteBtn = document.getElementById("confirm-delete-btn");
  cancelDeleteBtn = document.getElementById("cancel-delete-btn");

  // Close editor confirmation modal elements
  closeEditorModal = document.getElementById("close-editor-modal");
  saveAndCloseBtn = document.getElementById("save-and-close-btn");
  discardChangesBtn = document.getElementById("discard-changes-btn");
  cancelCloseBtn = document.getElementById("cancel-close-btn");

  // Image modal elements
  imageModal = document.getElementById("image-modal");
  imageModalImage = document.getElementById("image-modal-image");
  imageModalInfo = document.getElementById("image-modal-info");
}

// Image modal functions
function showImageModal(imageSrc, imageAlt) {
  const modal = document.getElementById("image-modal");
  const modalImage = document.getElementById("image-modal-image");
  const modalInfo = document.getElementById("image-modal-info");

  modalImage.src = imageSrc;
  modalImage.alt = imageAlt;
  modalInfo.textContent = imageAlt || "Image";

  modal.classList.add("show");

  // Close on click outside image
  modal.onclick = function (e) {
    if (e.target === modal) {
      hideImageModal();
    }
  };

  // Close on escape key
  document.addEventListener("keydown", handleImageModalKeydown);
}

function hideImageModal() {
  const modal = document.getElementById("image-modal");
  modal.classList.remove("show");
  document.removeEventListener("keydown", handleImageModalKeydown);
}

function handleImageModalKeydown(e) {
  if (e.key === "Escape") {
    hideImageModal();
  }
}

// Make functions globally available for inline event handlers
window.hideImageModal = hideImageModal;

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
  newNoteFromClipboardBtn.addEventListener("click", createNoteFromClipboard);
  searchBtn.addEventListener("click", handleSearch);
  clearSearchBtn.addEventListener("click", clearSearch);
  trashBtn.addEventListener("click", () => switchCategory("trash"));
  settingsBtn.addEventListener("click", openSettings);
  createFirstNoteBtn.addEventListener("click", createNewNote);

  // Category filter listeners
  filterPrivateBtn.addEventListener("click", () => switchCategory("private"));
  filterWorkBtn.addEventListener("click", () => switchCategory("work"));

  // Editor category filter listeners
  editorFilterPrivateBtn.addEventListener("click", () =>
    switchEditorCategory("private")
  );
  editorFilterWorkBtn.addEventListener("click", () =>
    switchEditorCategory("work")
  );

  // Search input listener with debouncing
  searchInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") handleSearch();
  });
  searchInput.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      e.preventDefault();
      clearSearch();
      searchInput.blur(); // Remove focus from search input
    }
  });
  searchInput.addEventListener("input", (e) => {
    if (e.target.value === "") {
      clearSearch();
    } else {
      handleSearchInput();
    }
  });

  // Editor listeners
  saveNoteBtn.addEventListener("click", saveCurrentNote);
  cancelEditorBtn.addEventListener("click", attemptCloseEditor);

  // Settings listeners
  backFromSettings.addEventListener("click", closeSettings);
  createBackupBtn.addEventListener("click", handleCreateBackup);
  saveSettingsBtn.addEventListener("click", handleSaveSettings);
  logoutBtn.addEventListener("click", handleLogout);

  // Delete confirmation modal listeners
  confirmDeleteBtn.addEventListener("click", confirmDeleteNote);
  cancelDeleteBtn.addEventListener("click", cancelDeleteNote);

  // Close editor confirmation modal listeners
  saveAndCloseBtn.addEventListener("click", handleSaveAndCloseEditor);
  discardChangesBtn.addEventListener(
    "click",
    handleDiscardChangesAndCloseEditor
  );
  cancelCloseBtn.addEventListener("click", hideCloseEditorModal);

  // Close modal when clicking outside
  deleteModal.addEventListener("click", (e) => {
    if (e.target === deleteModal) {
      cancelDeleteNote();
    }
  });

  // Close editor modal when clicking outside
  closeEditorModal.addEventListener("click", (e) => {
    if (e.target === closeEditorModal) {
      hideCloseEditorModal();
    }
  });

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

  // Update category buttons to show active state
  updateCategoryButtons();

  // Load notes
  await loadNotes();
}

async function loadNotes() {
  try {
    // Load notes filtered by current category
    allNotes = (await GetNotesByCategory(currentCategory)) || [];
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
  requestAnimationFrame(() => {
    notesGrid.innerHTML = "";
    searchResultsHeader.classList.toggle("hidden", !searchQuery);
    emptyState.classList.toggle("hidden", filteredNotes.length > 0);

    if (searchQuery) {
      const resultsText = document.getElementById("search-results-text");
      resultsText.textContent = `Search results for "${escapeHtml(
        searchQuery
      )}" (${filteredNotes.length} found)`;
    }

    if (filteredNotes.length === 0) {
      const emptyText = document.getElementById("empty-state-text");
      const message = searchQuery
        ? `No notes found for "${escapeHtml(
            searchQuery
          )}". <button class="link-button" onclick="clearSearch()">Show all notes</button>`
        : `No notes found. <button class="link-button" onclick="createNewNote()">Create your first note</button>`;
      emptyText.innerHTML = message;
      return;
    }

    // Sort notes by updated date (most recent first)
    const sortedNotes = [...filteredNotes].sort(
      (a, b) => new Date(b.updated_at) - new Date(a.updated_at)
    );

    // Use document fragment for efficient DOM manipulation
    const fragment = document.createDocumentFragment();

    sortedNotes.forEach((note) => {
      const noteCard = document.createElement("div");
      noteCard.className = "note-card";
      noteCard.dataset.noteId = note.id;

      const noteActions = document.createElement("div");
      noteActions.className = "note-actions";

      // Show different actions based on category
      if (currentCategory === "trash") {
        noteActions.innerHTML = `
            <button class="restore-btn" data-note-id="${note.id}">Restore</button>
            <button class="delete-permanent-btn" data-note-id="${note.id}">Delete Permanently</button>
          `;
      } else {
        noteActions.innerHTML = `
            <button class="edit-btn" data-note-id="${note.id}">Edit</button>
            <button class="delete-btn" data-note-id="${note.id}">Delete</button>
          `;
      }

      const noteContentDiv = document.createElement("div");
      noteContentDiv.className = "note-content markdown-content";
      noteContentDiv.innerHTML = renderMarkdown(note.content);

      noteCard.appendChild(noteActions);
      noteCard.appendChild(noteContentDiv);

      // Load images after HTML is inserted into DOM
      loadImagesInDOM(noteContentDiv);

      // Add external link handlers after HTML is inserted into DOM
      addExternalLinkHandlersToContainer(noteContentDiv);

      // Add event listeners based on button type
      if (currentCategory === "trash") {
        noteCard
          .querySelector(".restore-btn")
          .addEventListener("click", (e) => {
            e.stopPropagation();
            restoreNote(note.id);
          });

        noteCard
          .querySelector(".delete-permanent-btn")
          .addEventListener("click", (e) => {
            e.stopPropagation();
            permanentlyDeleteNote(note.id);
          });
      } else {
        noteCard.querySelector(".edit-btn").addEventListener("click", (e) => {
          e.stopPropagation();
          editNote(note.id);
        });

        noteCard.querySelector(".delete-btn").addEventListener("click", (e) => {
          e.stopPropagation();
          deleteNote(note.id);
        });
      }

      fragment.appendChild(noteCard);
    });

    notesGrid.appendChild(fragment);
  });
}

// Make some functions globally available for inline event handlers
window.clearSearch = clearSearch;
window.createNewNote = createNewNote;

// Image handling functions
function loadImagesInDOM(element) {
  const images = element.querySelectorAll('img[src^="image:"]');
  console.log(`Found ${images.length} images with image: scheme`);

  images.forEach(async (img, index) => {
    const imageId = img.src.replace("image:", "");
    console.log(`Loading image ${index + 1}/${images.length}: ${imageId}`);

    try {
      const dataUrl = await GetImageAsDataURL(imageId);
      console.log(`Successfully loaded image ${imageId}, setting data URL`);
      img.src = dataUrl;

      // Add click handler for image modal
      img.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();
        showImageModal(dataUrl, img.alt || "Image");
      });

      // Add hover effect class
      img.classList.add("note-image");
    } catch (error) {
      console.error(`Failed to load image ${imageId}:`, error);
      // Set a broken image placeholder
      img.src =
        "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAwIiBoZWlnaHQ9IjEwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iMTAwJSIgaGVpZ2h0PSIxMDAlIiBmaWxsPSIjZjBmMGYwIiBzdHJva2U9IiNjY2MiLz48dGV4dCB4PSI1MCUiIHk9IjUwJSIgZG9taW5hbnQtYmFzZWxpbmU9Im1pZGRsZSIgdGV4dC1hbmNob3I9Im1pZGRsZSIgZmlsbD0iIzk5OSI+SW1hZ2UgTm90IEZvdW5kPC90ZXh0Pjwvc3ZnPg==";
      img.alt = "Image not found";
      img.title = `Failed to load image: ${imageId}`;
    }
  });
}

function addExternalLinkHandlersToContainer(element) {
  const links = element.querySelectorAll('a[href^="http"]');
  links.forEach((link) => {
    link.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      BrowserOpenURL(link.href);
    });
  });
}

// Process custom image syntax in rendered HTML
function processCustomImages(html) {
  // Replace custom image syntax ![alt](image:id) with proper img tags
  // This handles cases where marked might not process custom schemes correctly
  const originalHtml = html;
  const processedHtml = html.replace(
    /!\[([^\]]*)\]\(image:([^)]+)\)/g,
    '<img src="image:$2" alt="$1" />'
  );

  if (originalHtml !== processedHtml) {
    console.log("processCustomImages: Found and processed image syntax");
    console.log("Original:", originalHtml.substring(0, 200) + "...");
    console.log("Processed:", processedHtml.substring(0, 200) + "...");
  }

  return processedHtml;
}

// Additional missing functions that may be called

function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

function renderMarkdown(content) {
  if (!content || content.trim() === "") {
    return '<em class="empty-note-text">Empty note...</em>';
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

async function createNewNote() {
  try {
    // Don't create notes in trash category - switch to private instead
    const category = currentCategory === "trash" ? "private" : currentCategory;
    const newNote = await CreateNoteWithCategory("", category);

    // If we created in a different category, reload notes
    if (category !== currentCategory) {
      switchCategory(category);
    } else {
      allNotes.push(newNote);
      filteredNotes = searchQuery ? filteredNotes : [...allNotes];
      renderNotesList();
    }

    editNote(newNote.id);
  } catch (error) {
    console.error("Error creating note:", error);
    alert("Failed to create note");
  }
}

async function createNoteFromClipboard() {
  try {
    // First try to get image from clipboard
    let clipboardContent = "";
    let hasImage = false;

    // Check if clipboard has image data
    try {
      const clipboardData = await navigator.clipboard.read();
      for (const item of clipboardData) {
        if (item.types.some((type) => type.startsWith("image/"))) {
          // Get the first image type
          const imageType = item.types.find((type) =>
            type.startsWith("image/")
          );
          const blob = await item.getType(imageType);

          // Convert blob to base64
          const base64Data = await blobToBase64(blob);
          const base64String = base64Data.split(",")[1]; // Remove data:image/xxx;base64, prefix

          // Save image using the backend
          const imageResult = await SaveImageFromClipboard(
            base64String,
            imageType
          );

          // Create markdown content with the image
          clipboardContent = `![Clipboard Image](image:${imageResult.id})`;
          hasImage = true;
          break;
        }
      }
    } catch (error) {
      console.log("No image in clipboard or clipboard access failed:", error);
    }

    // If no image found, try to get text content
    if (!hasImage) {
      const clipboardText = await ClipboardGetText();

      if (!clipboardText || clipboardText.trim() === "") {
        alert("Clipboard is empty");
        return;
      }

      // Wrap clipboard text content in a code block
      clipboardContent = "```\n" + clipboardText + "\n```";
    }

    // Don't create notes in trash category - switch to private instead
    const category = currentCategory === "trash" ? "private" : currentCategory;

    // Create note with the clipboard content
    const newNote = await CreateNoteWithCategory(clipboardContent, category);

    // If we created in a different category, reload notes
    if (category !== currentCategory) {
      switchCategory(category);
    } else {
      allNotes.push(newNote);
      filteredNotes = searchQuery ? filteredNotes : [...allNotes];
      renderNotesList();
    }

    // Don't open editor - note is created and saved directly
    const contentType = hasImage ? "image" : "text";
    console.log(`Note created from clipboard ${contentType} content`);
  } catch (error) {
    console.error("Error creating note from clipboard:", error);
    alert("Failed to create note from clipboard");
  }
}

// Helper function to convert blob to base64
function blobToBase64(blob) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(blob);
    reader.onload = () => resolve(reader.result);
    reader.onerror = (error) => reject(error);
  });
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
    originalNoteContent = note.content; // Track original content for change detection

    // Set the editor category buttons to reflect the note's category
    updateEditorCategoryButtons(note.category);

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
  originalNoteContent = ""; // Reset original content tracking
}

async function saveCurrentNote() {
  if (!currentNote) return;

  try {
    const updatedNote = await UpdateNote(currentNote.id, noteContent.value);
    currentNote = updatedNote;

    // Check if the note's category matches the current view
    const noteCategoryMatchesView = updatedNote.category === currentCategory;

    if (noteCategoryMatchesView) {
      // Update notes list if the note is still in the current category view
      const index = allNotes.findIndex((n) => n.id === currentNote.id);
      if (index !== -1) {
        allNotes[index] = updatedNote;
        filteredNotes = searchQuery
          ? filteredNotes.map((n) =>
              n.id === updatedNote.id ? updatedNote : n
            )
          : [...allNotes];

        renderNotesList();
      }
    } else {
      // Reload notes if the category changed and note is no longer in current view
      await loadNotes();
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
  // Store the note ID for the confirmation
  noteToDelete = noteId;

  // Show the custom delete confirmation modal
  showDeleteModal();
}

function showDeleteModal() {
  deleteModal.style.display = "flex";
  // Focus the cancel button for better accessibility
  cancelDeleteBtn.focus();
}

function hideDeleteModal() {
  deleteModal.style.display = "none";
  noteToDelete = null;
}

function cancelDeleteNote() {
  hideDeleteModal();
}

async function confirmDeleteNote() {
  if (!noteToDelete) {
    hideDeleteModal();
    return;
  }

  try {
    // Use smart deletion logic: move to trash first, then permanent delete
    await MoveToTrash(noteToDelete);

    // Remove from current arrays (since we're viewing private/work)
    allNotes = allNotes.filter((n) => n.id !== noteToDelete);
    filteredNotes = filteredNotes.filter((n) => n.id !== noteToDelete);

    renderNotesList();

    // Close editor if this note was being edited
    if (currentNote && currentNote.id === noteToDelete) {
      closeEditor();
    }

    hideDeleteModal();
    console.log("Note moved to trash");
  } catch (error) {
    console.error("Error deleting note:", error);
    alert("Failed to delete note");
    hideDeleteModal();
  }
}

// Note action functions
async function restoreNote(noteId) {
  try {
    // Use the new RestoreFromTrash API that restores to original category
    await RestoreFromTrash(noteId);

    // Reload current view
    await loadNotes();

    console.log("Note restored to original category");
  } catch (error) {
    console.error("Error restoring note:", error);
    alert("Failed to restore note");
  }
}

async function permanentlyDeleteNote(noteId) {
  // Show confirmation dialog
  if (!confirm("Permanently delete this note? This action cannot be undone.")) {
    return;
  }

  try {
    await PermanentlyDeleteNote(noteId);

    // Remove from local arrays
    allNotes = allNotes.filter((n) => n.id !== noteId);
    filteredNotes = filteredNotes.filter((n) => n.id !== noteId);

    renderNotesList();

    // Close editor if this note was being edited
    if (currentNote && currentNote.id === noteId) {
      closeEditor();
    }

    console.log("Note permanently deleted");
  } catch (error) {
    console.error("Error permanently deleting note:", error);
    alert("Failed to permanently delete note");
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

async function handleSearch() {
  const query = searchInput.value.trim().toLowerCase();
  searchQuery = query;

  if (query === "") {
    // Clear search - show all notes
    filteredNotes = [...allNotes];
  } else {
    // Filter notes by search query
    filteredNotes = allNotes.filter((note) =>
      note.content.toLowerCase().includes(query)
    );
  }

  renderNotesList();
}

function handleSearchInput() {
  // Real-time search as user types
  const query = searchInput.value.trim().toLowerCase();
  searchQuery = query;

  if (query === "") {
    clearSearch();
  } else {
    // Filter notes by search query
    filteredNotes = allNotes.filter((note) =>
      note.content.toLowerCase().includes(query)
    );
    renderNotesList();
  }
}

function clearSearch() {
  searchQuery = "";
  searchInput.value = "";
  filteredNotes = [...allNotes];
  renderNotesList();
}

function attemptCloseEditor() {
  // Check if there are unsaved changes
  if (currentNote && noteContent.value !== originalNoteContent) {
    if (confirm("You have unsaved changes. Save before closing?")) {
      saveCurrentNote().then(() => closeEditor());
    } else {
      closeEditor();
    }
  } else {
    closeEditor();
  }
}

function switchCategory(category) {
  // Update current category
  currentCategory = category;

  // Update UI to reflect active category
  updateCategoryButtons();

  // Load notes for the new category
  loadNotes();
}

function updateCategoryButtons() {
  // Remove active class from all category filter buttons
  filterPrivateBtn.classList.remove("active");
  filterWorkBtn.classList.remove("active");

  // Remove active class from trash button
  trashBtn.classList.remove("active");

  // Add active class to current category button
  switch (currentCategory) {
    case "private":
      filterPrivateBtn.classList.add("active");
      break;
    case "work":
      filterWorkBtn.classList.add("active");
      break;
    case "trash":
      trashBtn.classList.add("active");
      break;
  }
}

function updateEditorCategoryButtons(category) {
  // Remove active class from all editor category buttons
  editorFilterPrivateBtn.classList.remove("active");
  editorFilterWorkBtn.classList.remove("active");

  // Add active class to the specified category button
  switch (category) {
    case "private":
      editorFilterPrivateBtn.classList.add("active");
      break;
    case "work":
      editorFilterWorkBtn.classList.add("active");
      break;
  }
}

async function switchEditorCategory(category) {
  if (!currentNote) return;

  try {
    // Update the note's category
    const updatedNote = await UpdateNoteCategory(currentNote.id, category);
    currentNote = updatedNote;

    // Update the editor UI
    updateEditorCategoryButtons(category);

    console.log(`Note moved to ${category} category`);
  } catch (error) {
    console.error("Error updating note category:", error);
    alert("Failed to update note category");
  }
}

function openSettings() {
  // Hide main app and show settings
  mainApp.style.display = "none";
  settingsScreen.style.display = "block";

  // Load current settings
  loadSettings();
}

function closeSettings() {
  // Hide settings and show main app
  settingsScreen.style.display = "none";
  mainApp.style.display = "block";
}

async function loadSettings() {
  try {
    const settings = await GetSettings();
    notesPathInput.value = settings.notesPath || "";
    passwordHashPathInput.value = settings.passwordHashPath || "";
  } catch (error) {
    console.error("Error loading settings:", error);
  }
}

function handleDiscardChangesAndCloseEditor() {
  // Close editor without saving
  closeEditor();
  hideCloseEditorModal();
}

function hideCloseEditorModal() {
  closeEditorModal.style.display = "none";
}

function startActivityTracking() {
  // Track user activity for auto-logout
  const activities = [
    "mousedown",
    "mousemove",
    "keypress",
    "scroll",
    "touchstart",
  ];

  const resetTimer = () => {
    lastActivityTime = Date.now();

    if (inactivityTimer) {
      clearTimeout(inactivityTimer);
    }

    inactivityTimer = setTimeout(() => {
      if (currentUser && Date.now() - lastActivityTime >= INACTIVITY_TIMEOUT) {
        console.log("Auto-logout due to inactivity");
        handleLogout();
      }
    }, INACTIVITY_TIMEOUT);
  };

  activities.forEach((activity) => {
    document.addEventListener(activity, resetTimer, true);
  });

  // Start the timer
  resetTimer();
}

async function handleCreateBackup() {
  try {
    await CreateBackup();
    alert("Backup created successfully");
  } catch (error) {
    console.error("Error creating backup:", error);
    alert("Failed to create backup: " + error.message);
  }
}

async function handleSaveSettings() {
  // Placeholder for save settings functionality
  console.log("Save settings functionality not implemented yet");
  alert("Settings saved");
}

async function handleLogout() {
  if (confirm("Are you sure you want to logout?")) {
    try {
      await Logout();
      currentUser = null;
      checkAuthState();
    } catch (error) {
      console.error("Error during logout:", error);
      alert("Failed to logout: " + error.message);
    }
  }
}

async function handleSaveAndCloseEditor() {
  try {
    await saveCurrentNote();
    closeEditor();
  } catch (error) {
    console.error("Error saving note:", error);
    alert("Failed to save note");
  }
}

function handleGlobalKeyboard(event) {
  // Handle global keyboard shortcuts
  if (event.ctrlKey || event.metaKey) {
    switch (event.key) {
      case "n":
        if (event.shiftKey) {
          // Ctrl/Cmd+Shift+N - Create note from clipboard
          event.preventDefault();
          createNoteFromClipboard();
        } else {
          // Ctrl/Cmd+N - Create new note
          event.preventDefault();
          createNewNote();
        }
        break;
      case "s":
        if (currentNote) {
          event.preventDefault();
          saveCurrentNote();
        }
        break;
      case "f":
        event.preventDefault();
        searchInput.focus();
        break;
    }
  }

  // Handle escape key
  if (event.key === "Escape") {
    if (noteEditor && !noteEditor.classList.contains("hidden")) {
      closeEditor();
    }
  }
}

async function handleClipboardPaste(event) {
  // Handle clipboard paste events
  if (event.target === noteContent && noteContent) {
    // Handle paste in the note editor
    const clipboardData = event.clipboardData || window.clipboardData;
    if (!clipboardData) return;

    // Check if there are any image files in the clipboard
    const items = Array.from(clipboardData.items);
    const imageItems = items.filter((item) => item.type.startsWith("image/"));

    if (imageItems.length > 0) {
      event.preventDefault(); // Prevent default paste behavior for images

      for (const item of imageItems) {
        const file = item.getAsFile();
        if (file) {
          try {
            // Convert file to base64
            const base64Data = await fileToBase64(file);
            const base64String = base64Data.split(",")[1]; // Remove data:image/xxx;base64, prefix

            // Save image using the backend
            const imageResult = await SaveImageFromClipboard(
              base64String,
              file.type
            );

            // Insert markdown image syntax at cursor position
            const imageMarkdown = `![Image](image:${imageResult.id})`;
            insertTextAtCursor(noteContent, imageMarkdown);

            console.log(`Image pasted and saved with ID: ${imageResult.id}`);
          } catch (error) {
            console.error("Failed to save pasted image:", error);
            alert("Failed to save pasted image");
          }
        }
      }
    }
    // For text content, let the default paste behavior work
    return;
  }

  // For other paste events, could implement custom behavior
  console.log("Clipboard paste detected outside editor");
}

// Helper function to convert file to base64
function fileToBase64(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result);
    reader.onerror = (error) => reject(error);
  });
}

// Helper function to insert text at cursor position
function insertTextAtCursor(textArea, text) {
  const start = textArea.selectionStart;
  const end = textArea.selectionEnd;
  const value = textArea.value;

  textArea.value = value.slice(0, start) + text + value.slice(end);

  // Set cursor position after the inserted text
  const newCursorPos = start + text.length;
  textArea.setSelectionRange(newCursorPos, newCursorPos);
  textArea.focus();
}
