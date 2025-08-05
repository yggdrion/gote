import "./style.css";
import {
    UI_CONSTANTS,
    CSS_CLASSES,
    ELEMENT_IDS,
    MESSAGES,
} from "./constants.js";

// Import performance utilities (simplified)
import { Debouncer } from "./performance.js";

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
    CleanupOrphanedImages,
    GetImageStats,
} from "../wailsjs/go/main/App.js";

// Import Wails runtime for browser functionality
import { BrowserOpenURL, ClipboardGetText } from "../wailsjs/runtime/runtime.js";

// State management
let currentUser = null;
let currentNote = null;
let originalNoteContent = ""; // Track original content to detect changes
let allNotes = [];
let filteredNotes = [];
let searchQuery = "";

// Performance optimization instances
let searchDebouncer = new Debouncer(300); // 300ms debounce for search

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
let syncBtn, settingsBtn, notesGrid, noteEditor;
let noteContent, searchResultsHeader, emptyState;
let saveNoteBtn, cancelEditorBtn, createFirstNoteBtn;
let backFromSettings, syncFromSettings;
let createBackupBtn, logoutBtn;
let notesPathInput, passwordHashPathInput, saveSettingsBtn;

// Setup screen elements
let initialSetupScreen, setupNotesPath, setupPasswordHashPath;
let setupMasterPassword, setupConfirmPassword, completeSetupBtn;

// Delete confirmation modal elements
let deleteModal, confirmDeleteBtn, cancelDeleteBtn;
let closeEditorModal, saveAndCloseBtn, discardChangesBtn, cancelCloseBtn;
let noteToDelete = null;

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
    newNoteFromClipboardBtn = document.getElementById("new-note-from-clipboard-btn");
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

    // Delete confirmation modal elements
    deleteModal = document.getElementById("delete-confirmation-modal");
    confirmDeleteBtn = document.getElementById("confirm-delete-btn");
    cancelDeleteBtn = document.getElementById("cancel-delete-btn");

    // Close editor confirmation modal elements
    closeEditorModal = document.getElementById("close-editor-modal");
    saveAndCloseBtn = document.getElementById("save-and-close-btn");
    discardChangesBtn = document.getElementById("discard-changes-btn");
    cancelCloseBtn = document.getElementById("cancel-close-btn");

    // Create image modal for enlarged viewing
    createImageModal();
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
    newNoteFromClipboardBtn.addEventListener("click", createNoteFromClipboard);
    searchBtn.addEventListener("click", handleSearch);
    clearSearchBtn.addEventListener("click", clearSearch);
    syncBtn.addEventListener("click", handleSync);
    settingsBtn.addEventListener("click", openSettings);
    createFirstNoteBtn.addEventListener("click", createNewNote);

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
            // Use debounced search for real-time search
            handleSearchInput();
        }
    });

    // Editor listeners
    saveNoteBtn.addEventListener("click", saveCurrentNote);
    cancelEditorBtn.addEventListener("click", attemptCloseEditor);

    // Settings listeners
    backFromSettings.addEventListener("click", closeSettings);
    syncFromSettings.addEventListener("click", handleSync);
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
    // Use batch DOM operations for better performance
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
            noteActions.innerHTML = `
        <button class="edit-btn" data-note-id="${note.id}">Edit</button>
        <button class="delete-btn" data-note-id="${note.id}">Delete</button>
      `;

            const noteContentDiv = document.createElement("div");
            noteContentDiv.className = "note-content markdown-content";
            noteContentDiv.innerHTML = renderMarkdown(note.content);

            noteCard.appendChild(noteActions);
            noteCard.appendChild(noteContentDiv);

            // Load images after HTML is inserted into DOM
            loadImagesInDOM(noteContentDiv);

            // Add external link handlers after HTML is inserted into DOM
            addExternalLinkHandlersToContainer(noteContentDiv);

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
    });
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

async function createNoteFromClipboard() {
    try {
        // Get clipboard content
        const clipboardText = await ClipboardGetText();

        if (!clipboardText || clipboardText.trim() === "") {
            alert("Clipboard is empty");
            return;
        }

        // Wrap clipboard content in a code block
        const noteContent = "```\n" + clipboardText + "\n```";

        // Create note with the clipboard content
        const newNote = await CreateNote(noteContent);
        allNotes.push(newNote);
        filteredNotes = searchQuery ? filteredNotes : [...allNotes];

        renderNotesList();

        // Don't open editor - note is created and saved directly
        console.log("Note created from clipboard content");
    } catch (error) {
        console.error("Error creating note from clipboard:", error);
        alert("Failed to create note from clipboard");
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
        originalNoteContent = note.content; // Track original content for change detection
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
        await DeleteNote(noteToDelete);

        // Remove from arrays
        allNotes = allNotes.filter((n) => n.id !== noteToDelete);
        filteredNotes = filteredNotes.filter((n) => n.id !== noteToDelete);

        renderNotesList();

        // Close editor if this note was being edited
        if (currentNote && currentNote.id === noteToDelete) {
            closeEditor();
        }

        hideDeleteModal();
    } catch (error) {
        console.error("Error deleting note:", error);
        alert("Failed to delete note");
        hideDeleteModal();
    }
}

// Check if there are unsaved changes in the editor
function hasUnsavedChanges() {
    return currentNote && noteContent.value !== originalNoteContent;
}

// Show the close editor confirmation modal
function showCloseEditorModal() {
    closeEditorModal.style.display = "flex";
    // Focus the cancel button for better accessibility
    cancelCloseBtn.focus();
}

// Hide the close editor confirmation modal
function hideCloseEditorModal() {
    closeEditorModal.style.display = "none";
}

// Handle save and close from the modal
async function handleSaveAndCloseEditor() {
    hideCloseEditorModal();
    await saveAndCloseNote();
}

// Handle discard changes and close from the modal
function handleDiscardChangesAndCloseEditor() {
    hideCloseEditorModal();
    closeEditor();
}

// Attempt to close the editor with confirmation if needed
function attemptCloseEditor() {
    if (hasUnsavedChanges()) {
        showCloseEditorModal();
    } else {
        closeEditor();
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

        // Use backend search - it's simpler and already works with code blocks
        filteredNotes = await SearchNotes(query);

        renderNotesList();
        clearSearchBtn.style.display = "block";
    } catch (error) {
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
        progressDiv.className = "image-upload-progress";
        document.body.appendChild(progressDiv);
    }

    // Remove previous type classes and add new one
    progressDiv.classList.remove("info", "success", "error");
    progressDiv.classList.add(type);

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

            // Add click handler for image enlargement
            addImageClickHandler(img);

            console.log(`Image ${imageId} loaded successfully`);
        } catch (error) {
            console.error(`Failed to load image ${imageId}:`, error);
            img.src =
                "data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTIxIDlWN0MxOSA1IDIwIDMgMTggM0g2QzQgMyAzIDUgMyA3VjE3QzMgMTkgNCAyMSA2IDIxSDE4QzIwIDIxIDIxIDE5IDIxIDE3VjE1IiBzdHJva2U9IiNmZjAwMDAiIHN0cm9rZS13aWR0aD0iMiIgc3Ryb2tlLWxpbmVjYXA9InJvdW5kIiBzdHJva2UtbGluZWpvaW49InJvdW5kIi8+CjxwYXRoIGQ9Ik0xIDFMMjMgMjMiIHN0cm9rZT0iI2ZmMDAwMCIgc3Ryb2tlLXdpZHRoPSIyIiBzdHJva2UtbGluZWNhcD0icm91bmQiIHN0cm9rZS1saW5lam9pbj0icm91bmQiLz4KPC9zdmc+"; // Error icon
            img.alt = `Failed to load image: ${img.alt}`;
            img.title = `Image not found: ${imageId}`;
        }
    }

    // Also add click handlers to any existing loaded images in this element
    addImageClickHandlersToContainer(element);
}

// Process custom image syntax in rendered HTML
function processCustomImages(html) {
    // Replace our custom image syntax with proper img tags
    // Look for <img src="image:imageId" ...> patterns that marked.js created
    const imageRegex = /<img([^>]*?)src="image:([^"]+)"([^>]*?)>/g;

    return html.replace(imageRegex, (match, beforeSrc, imageId, afterSrc) => {
        console.log(`Processing custom image: ${imageId}`);
        return `<img${beforeSrc}src="data:image/png;base64,loading..." data-image-id="${imageId}" class="note-image"${afterSrc}>`;
    });
}

function handleGlobalKeyboard(e) {
    // Handle close editor modal keyboard shortcuts
    if (closeEditorModal.style.display === "flex") {
        if (e.key === "Escape") {
            e.preventDefault();
            hideCloseEditorModal();
            return;
        }
        if (e.key === "Enter") {
            e.preventDefault();
            handleSaveAndCloseEditor();
            return;
        }
    }

    // Handle delete modal keyboard shortcuts
    if (deleteModal.style.display === "flex") {
        if (e.key === "Escape") {
            e.preventDefault();
            cancelDeleteNote();
            return;
        }
        if (e.key === "Enter") {
            e.preventDefault();
            confirmDeleteNote();
            return;
        }
    }

    // Handle Escape key in editor
    if (
        e.key === "Escape" &&
        currentNote &&
        !noteEditor.classList.contains("hidden")
    ) {
        e.preventDefault();
        attemptCloseEditor();
        return;
    }

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

    if (isCtrlOrCmd && e.key === "n") {
        // Ctrl+N / Cmd+N: Create new note
        e.preventDefault();
        // Only if we're in the main app (not auth or settings screen)
        if (
            mainApp.style.display !== "none" &&
            authScreen.style.display === "none" &&
            settingsScreen.style.display === "none"
        ) {
            createNewNote();
        }
        return;
    }

    if (isCtrlOrCmd && e.shiftKey && e.key === "N") {
        // Ctrl+Shift+V / Cmd+Shift+V: Create note from clipboard
        e.preventDefault();
        // Only if we're in the main app (not auth or settings screen)
        if (
            mainApp.style.display !== "none" &&
            authScreen.style.display === "none" &&
            settingsScreen.style.display === "none"
        ) {
            createNoteFromClipboard();
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

// Image management functions for debugging/testing
window.imageDebug = {
    async getStats() {
        try {
            const stats = await GetImageStats();
            console.log("Image Statistics:", stats);
            return stats;
        } catch (error) {
            console.error("Failed to get image stats:", error);
        }
    },

    async cleanup() {
        try {
            const cleaned = await CleanupOrphanedImages();
            console.log(`Cleaned up ${cleaned} orphaned images`);
            return cleaned;
        } catch (error) {
            console.error("Failed to cleanup images:", error);
        }
    },

    async listAll() {
        try {
            const images = await ListImages();
            console.log("All Images:", images);
            return images;
        } catch (error) {
            console.error("Failed to list images:", error);
        }
    },
};

// Image modal functionality

// Create the image modal DOM structure
function createImageModal() {
    const modal = document.createElement("div");
    modal.id = "image-modal";
    modal.className = "image-modal";

    modal.innerHTML = `
    <div class="image-modal-content">
      <button class="image-modal-close" id="image-modal-close">&times;</button>
      <img class="image-modal-image" id="image-modal-image" alt="Enlarged image">
      <div class="image-modal-info" id="image-modal-info"></div>
    </div>
  `;

    document.body.appendChild(modal);

    // Set up event listeners
    const closeBtn = document.getElementById("image-modal-close");
    const modalImg = document.getElementById("image-modal-image");

    // Close modal on close button click
    closeBtn.addEventListener("click", closeImageModal);

    // Close modal on background click
    modal.addEventListener("click", (e) => {
        if (e.target === modal) {
            closeImageModal();
        }
    });

    // Close modal on escape key
    document.addEventListener("keydown", (e) => {
        if (e.key === "Escape" && modal.classList.contains("show")) {
            closeImageModal();
        }
    });

    // Prevent click propagation on modal content
    const modalContent = modal.querySelector(".image-modal-content");
    modalContent.addEventListener("click", (e) => {
        e.stopPropagation();
    });
}

// Open image modal with the given image
function openImageModal(imageSrc, imageAlt = "", imageInfo = "") {
    const modal = document.getElementById("image-modal");
    const modalImg = document.getElementById("image-modal-image");
    const modalInfo = document.getElementById("image-modal-info");

    if (!modal || !modalImg) {
        console.error("Image modal elements not found");
        return;
    }

    modalImg.src = imageSrc;
    modalImg.alt = imageAlt;
    modalInfo.textContent = imageInfo || imageAlt || "";

    modal.classList.add("show");
    document.body.style.overflow = "hidden"; // Prevent background scrolling
}

// Close image modal
function closeImageModal() {
    const modal = document.getElementById("image-modal");
    if (modal) {
        modal.classList.remove("show");
        document.body.style.overflow = ""; // Restore scrolling

        // Clear the image source after animation
        setTimeout(() => {
            const modalImg = document.getElementById("image-modal-image");
            if (modalImg) {
                modalImg.src = "";
            }
        }, 300);
    }
}

// Add click handler to an image element
function addImageClickHandler(imgElement) {
    if (!imgElement || imgElement.dataset.clickHandlerAdded) {
        return; // Already has handler or invalid element
    }

    imgElement.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();

        const imageSrc = imgElement.src;
        const imageAlt = imgElement.alt || "Image";

        // Extract additional info if available
        let imageInfo = "";
        if (imgElement.title && imgElement.title !== imageAlt) {
            imageInfo = imgElement.title;
        }

        openImageModal(imageSrc, imageAlt, imageInfo);
    });

    // Mark as having click handler to avoid duplicates
    imgElement.dataset.clickHandlerAdded = "true";

    // Add visual indication that image is clickable (redundant with CSS but ensures it's set)
    imgElement.style.cursor = "pointer";
}

// Add click handlers to all images in a container
function addImageClickHandlersToContainer(container) {
    if (!container) return;

    const images = container.querySelectorAll("img.note-image");
    images.forEach((img) => {
        addImageClickHandler(img);
    });
}

// External link handling functions

// Add click handler to an external link element
function addExternalLinkHandler(linkElement) {
    if (!linkElement || linkElement.dataset.externalHandlerAdded) {
        return; // Already has handler or invalid element
    }

    linkElement.addEventListener("click", (e) => {
        e.preventDefault();
        e.stopPropagation();

        const href = linkElement.href;

        // Check if it's an external link (starts with http:// or https://)
        if (href && (href.startsWith("http://") || href.startsWith("https://"))) {
            try {
                BrowserOpenURL(href);
            } catch (error) {
                console.error("Error opening external link:", error);
                // Fallback - try window.open (though this might not work in Wails)
                window.open(href, "_blank");
            }
        }
    });

    // Mark as having click handler to avoid duplicates
    linkElement.dataset.externalHandlerAdded = "true";

    // Add visual indication that link opens externally
    linkElement.title = linkElement.title || "Opens in external browser";
}

// Add click handlers to all external links in a container
function addExternalLinkHandlersToContainer(container) {
    if (!container) return;

    const links = container.querySelectorAll("a[href]");
    links.forEach((link) => {
        const href = link.href;
        // Only handle external links (http/https)
        if (href && (href.startsWith("http://") || href.startsWith("https://"))) {
            addExternalLinkHandler(link);
        }
    });
}
