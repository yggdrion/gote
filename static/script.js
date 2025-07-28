let currentNoteId = null;
let isEditing = false;
let markedInstance = null;

// Initialize highlight.js when available
function initializeHighlightJS() {
  if (typeof hljs !== "undefined") {
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
      ],
    });
  }
}

// Configure marked with marked-highlight extension
function initializeMarked() {
  if (typeof marked !== "undefined" && typeof markedHighlight !== "undefined") {
    // Create marked instance with highlight extension
    markedInstance = new marked.Marked(
      markedHighlight.markedHighlight({
        langPrefix: "hljs language-",
        highlight(code, lang) {
          if (typeof hljs !== "undefined") {
            const language = hljs.getLanguage(lang) ? lang : "plaintext";
            return hljs.highlight(code, { language }).value;
          }
          return code; // Fallback if hljs is not available
        },
      })
    );

    // Configure additional options
    markedInstance.setOptions({
      breaks: true,
      gfm: true,
      headerIds: false,
      mangle: false,
    });
  } else if (typeof marked !== "undefined") {
    // Fallback to legacy marked configuration if marked-highlight is not available
    marked.setOptions({
      breaks: true,
      gfm: true,
      headerIds: false,
      mangle: false,
      highlight: function (code, lang) {
        if (typeof hljs !== "undefined") {
          if (lang && hljs.getLanguage(lang)) {
            try {
              return hljs.highlight(code, { language: lang }).value;
            } catch (__) {
              return hljs.highlightAuto(code).value;
            }
          }
          return hljs.highlightAuto(code).value;
        }
        return code;
      },
    });
  }
}

// Function to render markdown content
function renderMarkdown(content) {
  if (!content || content.trim() === "") {
    return '<em style="color: #999;">Empty note...</em>';
  }

  if (markedInstance) {
    const html = markedInstance.parse(content);
    return html;
  } else if (typeof marked !== "undefined") {
    const html = marked.parse(content);
    return html;
  }

  // Fallback if marked is not available - basic text with line breaks
  return content.replace(/\n/g, "<br>");
}

// Function to render all markdown content on the page
function renderAllMarkdownContent() {
  document.querySelectorAll(".markdown-content").forEach((element) => {
    const rawContent = element.getAttribute("data-raw-content");
    if (rawContent) {
      element.innerHTML = renderMarkdown(rawContent);

      // Note: With marked-highlight, syntax highlighting is applied during parsing,
      // so we don't need to manually apply it to code blocks
    }
  });
}

// Create a new note
function createNewNote() {
  currentNoteId = null;
  openEditor("");
}

// Select and edit an existing note
async function selectNote(noteId) {
  try {
    const response = await fetch(`/api/notes/${noteId}`);
    if (response.ok) {
      const note = await response.json();
      currentNoteId = noteId;
      openEditor(note.content);
    } else {
      alert("Error loading note");
    }
  } catch (error) {
    console.error("Error:", error);
    alert("Error loading note");
  }
}

// Open the editor panel
function openEditor(content) {
  const editor = document.getElementById("note-editor");
  const textarea = document.getElementById("note-content");

  textarea.value = content || "";
  editor.classList.remove("hidden");
  textarea.focus();
  isEditing = true;

  // Add backdrop
  const backdrop = document.createElement("div");
  backdrop.className = "editor-backdrop";
  const isMobile = window.innerWidth <= 768;
  backdrop.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: ${isMobile ? "100%" : "50%"};
        height: 100vh;
        background: rgba(0, 0, 0, 0.3);
        z-index: 999;
        backdrop-filter: blur(5px);
    `;
  backdrop.onclick = closeEditor;
  document.body.appendChild(backdrop);
}

// Close the editor panel
function closeEditor() {
  const editor = document.getElementById("note-editor");
  const backdrop = document.querySelector(".editor-backdrop");

  editor.classList.add("hidden");
  if (backdrop) {
    backdrop.remove();
  }
  currentNoteId = null;
  isEditing = false;
}

// Save the current note
async function saveNote() {
  const content = document.getElementById("note-content").value.trim();

  if (!content) {
    alert("Please enter some content for your note");
    return;
  }

  try {
    let response;
    if (currentNoteId) {
      // Update existing note
      response = await fetch(`/api/notes/${currentNoteId}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ content }),
      });
    } else {
      // Create new note
      response = await fetch("/api/notes", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ content }),
      });
    }

    if (response.ok) {
      closeEditor();
      window.location.reload(); // Refresh to show updated notes and render markdown
    } else {
      alert("Error saving note");
    }
  } catch (error) {
    console.error("Error:", error);
    alert("Error saving note");
  }
}

// Delete a note with custom confirmation modal
async function deleteNote(noteId) {
  // Find the note content to show in preview
  const noteCard = document
    .querySelector(`[data-note-id="${noteId}"]`)
    .closest(".note-card");
  const noteContent = noteCard.querySelector(".note-content");
  const rawContent =
    noteContent.getAttribute("data-raw-content") || noteContent.textContent;

  // Show custom delete modal
  showDeleteModal(noteId, rawContent);
}

// Show the delete confirmation modal
function showDeleteModal(noteId, noteContent) {
  const modal = document.getElementById("delete-modal");
  const preview = document.getElementById("delete-note-preview");

  // Set up note preview
  if (noteContent && noteContent.trim()) {
    // Truncate content if too long
    let previewContent =
      noteContent.length > 150
        ? noteContent.substring(0, 150) + "..."
        : noteContent;

    // Enhanced markdown rendering for preview
    previewContent = previewContent
      // Remove code blocks first to avoid conflicts
      .replace(/```[\s\S]*?```/g, "[code block]")
      // Headers
      .replace(/^### (.*$)/gm, "<strong>$1</strong>")
      .replace(/^## (.*$)/gm, "<strong>$1</strong>")
      .replace(/^# (.*$)/gm, "<strong>$1</strong>")
      // Bold and italic
      .replace(/\*\*(.*?)\*\*/g, "<strong>$1</strong>")
      .replace(/\*(.*?)\*/g, "<em>$1</em>")
      // Inline code
      .replace(/`(.*?)`/g, "<code>$1</code>")
      // Remove markdown list markers
      .replace(/^[\s]*[-\*\+]\s/gm, "• ")
      .replace(/^\d+\.\s/gm, "• ")
      // Line breaks
      .replace(/\n/g, "<br>");

    preview.innerHTML = `<div class="note-content">${previewContent}</div>`;
  } else {
    preview.innerHTML =
      '<div class="note-content"><em style="color: #9ca3af;">Empty note</em></div>';
  }

  // Store the noteId for later use
  modal.setAttribute("data-note-id", noteId);

  // Show modal with animation
  modal.classList.remove("hidden");

  // Focus on cancel button for better UX (keyboard navigation)
  setTimeout(() => {
    const cancelBtn = modal.querySelector(".cancel-delete-btn");
    if (cancelBtn) cancelBtn.focus();
  }, 100);
}

// Hide the delete confirmation modal
function hideDeleteModal() {
  const modal = document.getElementById("delete-modal");
  modal.classList.add("hidden");
  modal.removeAttribute("data-note-id");
}

// Confirm and execute note deletion
async function confirmDeleteNote() {
  const modal = document.getElementById("delete-modal");
  const noteId = modal.getAttribute("data-note-id");
  const confirmBtn = modal.querySelector(".confirm-delete-btn");

  if (!noteId) return;

  // Show loading state
  confirmBtn.classList.add("loading");
  confirmBtn.textContent = "Deleting...";
  confirmBtn.disabled = true;

  try {
    const response = await fetch(`/api/notes/${noteId}`, {
      method: "DELETE",
    });

    if (response.ok) {
      // If we're currently editing the deleted note, close the editor
      if (currentNoteId === noteId) {
        closeEditor();
      }

      hideDeleteModal();
      window.location.reload(); // Refresh to remove deleted note
    } else {
      throw new Error("Failed to delete note");
    }
  } catch (error) {
    console.error("Error:", error);

    // Reset button state
    confirmBtn.classList.remove("loading");
    confirmBtn.textContent = "Delete Note";
    confirmBtn.disabled = false;

    // Show error message
    alert("Error deleting note. Please try again.");
  }
}

// Clear search and go back to all notes
function clearSearch() {
  window.location.href = "/";
}

// Auto-save functionality
let autoSaveTimeout;

function setupAutoSave() {
  const textarea = document.getElementById("note-content");
  if (!textarea) return;

  function scheduleAutoSave() {
    if (!isEditing || !currentNoteId) return; // Only auto-save existing notes

    clearTimeout(autoSaveTimeout);
    autoSaveTimeout = setTimeout(async () => {
      const content = textarea.value.trim();
      if (content && currentNoteId) {
        try {
          await fetch(`/api/notes/${currentNoteId}`, {
            method: "PUT",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify({ content }),
          });
          console.log("Auto-saved");
        } catch (error) {
          console.error("Auto-save failed:", error);
        }
      }
    }, 3000); // Auto-save after 3 seconds of inactivity
  }

  textarea.addEventListener("input", scheduleAutoSave);
}

// Initialize when DOM is loaded
document.addEventListener("DOMContentLoaded", function () {
  // Initialize libraries
  initializeHighlightJS();
  initializeMarked();

  // Render all markdown content
  renderAllMarkdownContent();

  // Set up event listeners for edit and delete buttons
  setupNoteActionListeners();

  setupAutoSave();

  // Add keyboard shortcuts
  document.addEventListener("keydown", function (event) {
    // Ctrl/Cmd + S to save
    if ((event.ctrlKey || event.metaKey) && event.key === "s") {
      event.preventDefault();
      if (isEditing) {
        saveNote();
      }
    }

    // Ctrl/Cmd + N for new note
    if ((event.ctrlKey || event.metaKey) && event.key === "n") {
      event.preventDefault();
      createNewNote();
    }

    // Escape to close editor or modal
    if (event.key === "Escape") {
      const deleteModal = document.getElementById("delete-modal");
      if (deleteModal && !deleteModal.classList.contains("hidden")) {
        hideDeleteModal();
      } else if (isEditing) {
        closeEditor();
      }
    }

    // Enter to confirm delete when modal is open
    if (event.key === "Enter") {
      const deleteModal = document.getElementById("delete-modal");
      if (deleteModal && !deleteModal.classList.contains("hidden")) {
        event.preventDefault();
        confirmDeleteNote();
      }
    }
  });

  // Focus management for better UX
  const searchInput = document.querySelector(".search-form input");
  if (searchInput && !isEditing) {
    // Focus search if no notes are being edited and it's empty
    if (searchInput.value === "") {
      setTimeout(() => searchInput.focus(), 100);
    }
  }
});

// Set up event listeners for note action buttons
function setupNoteActionListeners() {
  // Edit buttons
  document.querySelectorAll(".edit-btn").forEach((button) => {
    button.addEventListener("click", function () {
      const noteId = this.getAttribute("data-note-id");
      selectNote(noteId);
    });
  });

  // Delete buttons
  document.querySelectorAll(".delete-btn").forEach((button) => {
    button.addEventListener("click", function () {
      const noteId = this.getAttribute("data-note-id");
      deleteNote(noteId);
    });
  });

  // Clear search buttons
  document
    .querySelectorAll(".clear-search-btn, .clear-search-link")
    .forEach((button) => {
      button.addEventListener("click", function () {
        clearSearch();
      });
    });

  // New note buttons
  document
    .querySelectorAll(".new-note-link, .new-note-btn")
    .forEach((button) => {
      button.addEventListener("click", function () {
        createNewNote();
      });
    });

  // Save note buttons
  document.querySelectorAll(".save-note-btn").forEach((button) => {
    button.addEventListener("click", function () {
      saveNote();
    });
  });

  // Close editor buttons
  document.querySelectorAll(".close-editor-btn").forEach((button) => {
    button.addEventListener("click", function () {
      closeEditor();
    });
  });

  // Delete modal buttons
  document.querySelectorAll(".cancel-delete-btn").forEach((button) => {
    button.addEventListener("click", function () {
      hideDeleteModal();
    });
  });

  document.querySelectorAll(".confirm-delete-btn").forEach((button) => {
    button.addEventListener("click", function () {
      confirmDeleteNote();
    });
  });

  // Close modal when clicking outside
  const deleteModal = document.getElementById("delete-modal");
  if (deleteModal) {
    deleteModal.addEventListener("click", function (e) {
      if (e.target === deleteModal) {
        hideDeleteModal();
      }
    });
  }

  // Settings modal functionality
  document.querySelectorAll(".settings-btn").forEach((button) => {
    button.addEventListener("click", function () {
      showSettingsModal();
    });
  });

  document.querySelectorAll(".close-settings-btn").forEach((button) => {
    button.addEventListener("click", function () {
      hideSettingsModal();
    });
  });

  document.querySelectorAll(".save-settings-btn").forEach((button) => {
    button.addEventListener("click", function () {
      saveSettings();
    });
  });

  // Close settings modal when clicking outside
  const settingsModal = document.getElementById("settings-modal");
  if (settingsModal) {
    settingsModal.addEventListener("click", function (e) {
      if (e.target === settingsModal) {
        hideSettingsModal();
      }
    });
  }
}

// Settings modal functions
function showSettingsModal() {
  const modal = document.getElementById("settings-modal");
  if (modal) {
    // Load current settings
    loadCurrentSettings();
    modal.classList.remove("hidden");
  }
}

function hideSettingsModal() {
  const modal = document.getElementById("settings-modal");
  if (modal) {
    modal.classList.add("hidden");
  }
}

function loadCurrentSettings() {
  // Load settings from server
  fetch("/api/settings", {
    method: "GET",
  })
    .then((response) => response.json())
    .then((data) => {
      const notesPathInput = document.getElementById("notes-path");
      const passwordHashInput = document.getElementById("password-hash");

      if (notesPathInput) notesPathInput.value = data.notesPath || "./data";
      if (passwordHashInput)
        passwordHashInput.value =
          data.passwordHashPath || "./data/.password_hash";
    })
    .catch((error) => {
      console.error("Error loading settings:", error);
      // Fallback to defaults
      const notesPathInput = document.getElementById("notes-path");
      const passwordHashInput = document.getElementById("password-hash");

      if (notesPathInput) notesPathInput.value = "./data";
      if (passwordHashInput) passwordHashInput.value = "./data/.password_hash";
    });
}

function saveSettings() {
  const notesPathInput = document.getElementById("notes-path");
  const passwordHashInput = document.getElementById("password-hash");

  if (notesPathInput && passwordHashInput) {
    const notesPath = notesPathInput.value.trim() || "./data";
    const passwordHashPath =
      passwordHashInput.value.trim() || "./data/.password_hash";

    // Send to server
    fetch("/api/settings", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        notesPath: notesPath,
        passwordHashPath: passwordHashPath,
      }),
    })
      .then((response) => response.json())
      .then((data) => {
        if (data.success) {
          hideSettingsModal();
          // Show success message
          showNotification("Settings saved successfully!", "success");
        } else {
          showNotification(
            "Failed to save settings: " + (data.error || "Unknown error"),
            "error"
          );
        }
      })
      .catch((error) => {
        console.error("Error saving settings:", error);
        showNotification("Failed to save settings", "error");
      });
  }
}

function showNotification(message, type = "info") {
  // Create a simple notification
  const notification = document.createElement("div");
  notification.className = `notification notification-${type}`;
  notification.textContent = message;

  document.body.appendChild(notification);

  // Auto-hide after 3 seconds
  setTimeout(() => {
    if (notification.parentNode) {
      notification.parentNode.removeChild(notification);
    }
  }, 3000);
}
