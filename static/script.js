let currentNoteId = null;
let isEditing = false;

// Configure marked for GitHub-flavored markdown
marked.setOptions({
    breaks: true,
    gfm: true,
    headerIds: false,
    mangle: false
});

// Configure syntax highlighting
marked.setOptions({
    highlight: function(code, lang) {
        if (lang && hljs.getLanguage(lang)) {
            try {
                return hljs.highlight(code, { language: lang }).value;
            } catch (__) {
                // Fall back to plain text if highlighting fails
            }
        }
        return hljs.highlightAuto(code).value;
    }
});

// Function to render markdown content
function renderMarkdown(content) {
    if (!content || content.trim() === '') {
        return '<em style="color: #999;">Empty note...</em>';
    }
    return marked.parse(content);
}

// Function to render all markdown content on the page
function renderAllMarkdownContent() {
    document.querySelectorAll('.markdown-content').forEach(element => {
        const rawContent = element.getAttribute('data-raw-content');
        if (rawContent) {
            element.innerHTML = renderMarkdown(rawContent);
        }
    });
}

// Create a new note
function createNewNote() {
    currentNoteId = null;
    openEditor('');
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
            alert('Error loading note');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Error loading note');
    }
}

// Open the editor panel
function openEditor(content) {
    const editor = document.getElementById('note-editor');
    const textarea = document.getElementById('note-content');

    textarea.value = content || '';
    editor.classList.remove('hidden');
    textarea.focus();
    isEditing = true;

    // Add backdrop
    const backdrop = document.createElement('div');
    backdrop.className = 'editor-backdrop';
    const isMobile = window.innerWidth <= 768;
    backdrop.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: ${isMobile ? '100%' : '50%'};
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
    const editor = document.getElementById('note-editor');
    const backdrop = document.querySelector('.editor-backdrop');

    editor.classList.add('hidden');
    if (backdrop) {
        backdrop.remove();
    }
    currentNoteId = null;
    isEditing = false;
}

// Save the current note
async function saveNote() {
    const content = document.getElementById('note-content').value.trim();

    if (!content) {
        alert('Please enter some content for your note');
        return;
    }

    try {
        let response;
        if (currentNoteId) {
            // Update existing note
            response = await fetch(`/api/notes/${currentNoteId}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ content })
            });
        } else {
            // Create new note
            response = await fetch('/api/notes', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ content })
            });
        }

        if (response.ok) {
            closeEditor();
            window.location.reload(); // Refresh to show updated notes and render markdown
        } else {
            alert('Error saving note');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Error saving note');
    }
}

// Delete a note
async function deleteNote(noteId) {
    if (!confirm('Are you sure you want to delete this note?')) {
        return;
    }

    try {
        const response = await fetch(`/api/notes/${noteId}`, {
            method: 'DELETE'
        });

        if (response.ok) {
            // If we're currently editing the deleted note, close the editor
            if (currentNoteId === noteId) {
                closeEditor();
            }
            window.location.reload(); // Refresh to remove deleted note
        } else {
            alert('Error deleting note');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Error deleting note');
    }
}

// Clear search and go back to all notes
function clearSearch() {
    window.location.href = '/';
}

// Auto-save functionality
let autoSaveTimeout;

function setupAutoSave() {
    const textarea = document.getElementById('note-content');
    if (!textarea) return;

    function scheduleAutoSave() {
        if (!isEditing || !currentNoteId) return; // Only auto-save existing notes

        clearTimeout(autoSaveTimeout);
        autoSaveTimeout = setTimeout(async () => {
            const content = textarea.value.trim();
            if (content && currentNoteId) {
                try {
                    await fetch(`/api/notes/${currentNoteId}`, {
                        method: 'PUT',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({ content })
                    });
                    console.log('Auto-saved');
                } catch (error) {
                    console.error('Auto-save failed:', error);
                }
            }
        }, 3000); // Auto-save after 3 seconds of inactivity
    }

    textarea.addEventListener('input', scheduleAutoSave);
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', function () {
    // Render all markdown content
    renderAllMarkdownContent();
    
    setupAutoSave();

    // Add keyboard shortcuts
    document.addEventListener('keydown', function (event) {
        // Ctrl/Cmd + S to save
        if ((event.ctrlKey || event.metaKey) && event.key === 's') {
            event.preventDefault();
            if (isEditing) {
                saveNote();
            }
        }

        // Ctrl/Cmd + N for new note
        if ((event.ctrlKey || event.metaKey) && event.key === 'n') {
            event.preventDefault();
            createNewNote();
        }

        // Escape to close editor
        if (event.key === 'Escape' && isEditing) {
            closeEditor();
        }
    });

    // Focus management for better UX
    const searchInput = document.querySelector('.search-form input');
    if (searchInput && !isEditing) {
        // Focus search if no notes are being edited and it's empty
        if (searchInput.value === '') {
            setTimeout(() => searchInput.focus(), 100);
        }
    }
});
