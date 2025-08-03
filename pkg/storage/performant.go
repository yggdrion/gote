package storage

import (
	"strings"
	"time"

	"gote/pkg/models"
	"gote/pkg/performance"
)

// PerformantNoteStore extends NoteStore with performance optimizations
type PerformantNoteStore struct {
	*NoteStore

	// Performance components
	debouncer      *performance.Debouncer
	throttledSync  *performance.ThrottledExecutor
	batchProcessor *performance.BatchProcessor
	noteCache      *performance.NoteCache
	bufferPool     *performance.ByteBufferPool
	stringPool     *performance.StringBufferPool
	memoryMonitor  *performance.MemoryMonitor

	// Performance settings
	fileWatchDebounceTime time.Duration
	syncThrottleTime      time.Duration
	maxCacheSize          int
	maxMemoryMB           int64
}

// NewPerformantNoteStore creates a new performant note store
func NewPerformantNoteStore(dataDir string) *PerformantNoteStore {
	baseStore := NewNoteStore(dataDir)

	pns := &PerformantNoteStore{
		NoteStore:             baseStore,
		fileWatchDebounceTime: 300 * time.Millisecond, // 300ms debounce for file changes
		syncThrottleTime:      1 * time.Second,        // Max 1 sync per second
		maxCacheSize:          200,                    // Cache up to 200 notes
		maxMemoryMB:           100,                    // 100MB memory limit
	}

	// Initialize performance components
	pns.debouncer = performance.NewDebouncer(pns.fileWatchDebounceTime)
	pns.throttledSync = performance.NewThrottledExecutor(pns.syncThrottleTime)
	pns.noteCache = performance.NewNoteCache(pns.maxCacheSize)
	pns.bufferPool = performance.NewByteBufferPool()
	pns.stringPool = performance.NewStringBufferPool()

	// Initialize batch processor for file operations
	pns.batchProcessor = performance.NewBatchProcessor(
		10,                   // Process up to 10 files at once
		500*time.Millisecond, // Wait max 500ms before processing
		pns.processBatchedFileChanges,
	)

	// Initialize memory monitor
	pns.memoryMonitor = performance.NewMemoryMonitor(pns.maxMemoryMB, pns.cleanupMemory)

	return pns
}

// Enhanced file watching with debouncing
func (pns *PerformantNoteStore) handleFileWriteOptimized(filePath string) {
	// Use debouncer to avoid processing rapid file changes
	pns.debouncer.Debounce(filePath, func() {
		// Add to batch processor instead of processing immediately
		pns.batchProcessor.Add(filePath)
	})
}

// Process batched file changes for better performance
func (pns *PerformantNoteStore) processBatchedFileChanges(items []interface{}) {
	for _, item := range items {
		if filePath, ok := item.(string); ok {
			pns.NoteStore.handleFileWrite(filePath)
		}
	}

	// Check memory usage after batch processing
	pns.memoryMonitor.CheckMemoryUsage()
}

// Enhanced sync with throttling
func (pns *PerformantNoteStore) SyncFromDiskOptimized() error {
	var syncErr error
	pns.throttledSync.Execute(func() {
		syncErr = pns.NoteStore.syncFromDisk()
	})
	return syncErr
}

// Cached note retrieval
func (pns *PerformantNoteStore) GetNoteOptimized(id string) (*models.Note, error) {
	// Check cache first
	if cachedNote, found := pns.noteCache.Get(id); found {
		if note, ok := cachedNote.(*models.Note); ok {
			return note, nil
		}
	}

	// Get from base store
	note, err := pns.NoteStore.GetNote(id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	pns.noteCache.Put(id, note)
	return note, nil
}

// Enhanced search with performance optimizations
func (pns *PerformantNoteStore) SearchNotesOptimized(query string) []*models.Note {
	if query == "" {
		return pns.GetAllNotes()
	}

	// Use string pool for search terms
	searchTerms := pns.stringPool.Get()
	defer pns.stringPool.Put(searchTerms)

	// Split query into terms
	terms := strings.Fields(strings.ToLower(query))
	searchTerms = append(searchTerms, terms...)

	var results []*models.Note
	pns.mutex.RLock()
	defer pns.mutex.RUnlock()

	for _, note := range pns.notes {
		if pns.matchesSearch(note, searchTerms) {
			results = append(results, note)
		}
	}

	return results
}

// Optimized search matching
func (pns *PerformantNoteStore) matchesSearch(note *models.Note, searchTerms []string) bool {
	content := strings.ToLower(note.Content)

	for _, term := range searchTerms {
		if !strings.Contains(content, term) {
			return false
		}
	}

	return true
}

// Memory cleanup function
func (pns *PerformantNoteStore) cleanupMemory() {
	// Clear old cache entries (keep most recent half)
	currentSize := pns.noteCache.Size()
	if currentSize > pns.maxCacheSize/2 {
		// This is a simplified cleanup - in practice you might want more sophisticated LRU cleanup
		pns.noteCache.Clear()
	}

	// Flush any pending operations
	pns.batchProcessor.Flush()
}

// Enhanced note creation with buffering
func (pns *PerformantNoteStore) CreateNoteOptimized(content string, key []byte) (*models.Note, error) {
	note, err := pns.NoteStore.CreateNote(content, key)
	if err != nil {
		return nil, err
	}

	// Cache the new note
	pns.noteCache.Put(note.ID, note)

	return note, nil
}

// Enhanced note update with buffering
func (pns *PerformantNoteStore) UpdateNoteOptimized(id, content string, key []byte) (*models.Note, error) {
	note, err := pns.NoteStore.UpdateNote(id, content, key)
	if err != nil {
		return nil, err
	}

	// Update cache
	pns.noteCache.Put(note.ID, note)

	return note, nil
}

// Enhanced note deletion with cache cleanup
func (pns *PerformantNoteStore) DeleteNoteOptimized(id string) error {
	err := pns.NoteStore.DeleteNote(id)
	if err != nil {
		return err
	}

	// Remove from cache
	pns.noteCache.Remove(id)

	return nil
}

// GetPerformanceStats returns performance statistics
func (pns *PerformantNoteStore) GetPerformanceStats() map[string]interface{} {
	return map[string]interface{}{
		"cache_size":       pns.noteCache.Size(),
		"max_cache_size":   pns.maxCacheSize,
		"debounce_time_ms": pns.fileWatchDebounceTime.Milliseconds(),
		"throttle_time_ms": pns.syncThrottleTime.Milliseconds(),
		"max_memory_mb":    pns.maxMemoryMB,
	}
}

// Cleanup resources when done
func (pns *PerformantNoteStore) Cleanup() {
	if pns.debouncer != nil {
		pns.debouncer.Clear()
	}
	if pns.batchProcessor != nil {
		pns.batchProcessor.Flush()
	}
	if pns.noteCache != nil {
		pns.noteCache.Clear()
	}
}
