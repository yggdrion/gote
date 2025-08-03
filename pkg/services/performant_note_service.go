package services

import (
	"gote/pkg/models"
	"gote/pkg/storage"
)

// PerformantNoteService wraps NoteService with performance optimizations
type PerformantNoteService struct {
	*NoteService
	performantStore *storage.PerformantNoteStore
}

// NewPerformantNoteService creates a new performant note service
func NewPerformantNoteService(dataDir string) *PerformantNoteService {
	performantStore := storage.NewPerformantNoteStore(dataDir)
	baseService := &NoteService{
		store: performantStore.NoteStore, // Use the embedded NoteStore
	}

	return &PerformantNoteService{
		NoteService:     baseService,
		performantStore: performantStore,
	}
}

// LoadNotes initializes the note store with an encryption key
func (pns *PerformantNoteService) LoadNotes(key []byte) error {
	return pns.performantStore.LoadNotes(key)
}

// GetNote returns a specific note by ID using optimized caching
func (pns *PerformantNoteService) GetNote(id string) (*models.Note, error) {
	return pns.performantStore.GetNoteOptimized(id)
}

// CreateNote creates a new note with performance optimizations
func (pns *PerformantNoteService) CreateNote(content string, key []byte) (*models.Note, error) {
	return pns.performantStore.CreateNoteOptimized(content, key)
}

// UpdateNote updates an existing note with performance optimizations
func (pns *PerformantNoteService) UpdateNote(id, content string, key []byte) (*models.Note, error) {
	return pns.performantStore.UpdateNoteOptimized(id, content, key)
}

// DeleteNote deletes a note with cache cleanup
func (pns *PerformantNoteService) DeleteNote(id string) error {
	return pns.performantStore.DeleteNoteOptimized(id)
}

// SearchNotes performs optimized search
func (pns *PerformantNoteService) SearchNotes(query string) []*models.Note {
	return pns.performantStore.SearchNotesOptimized(query)
}

// SyncFromDisk syncs from disk with throttling
func (pns *PerformantNoteService) SyncFromDisk() error {
	return pns.performantStore.SyncFromDiskOptimized()
}

// GetPerformanceStats returns performance statistics
func (pns *PerformantNoteService) GetPerformanceStats() map[string]interface{} {
	return pns.performantStore.GetPerformanceStats()
}

// Cleanup releases resources
func (pns *PerformantNoteService) Cleanup() {
	pns.performantStore.Cleanup()
}
