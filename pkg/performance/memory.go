package performance

import (
	"sync"
)

// ByteBufferPool provides a pool of reusable byte buffers to reduce memory allocations
type ByteBufferPool struct {
	pool sync.Pool
}

// NewByteBufferPool creates a new byte buffer pool
func NewByteBufferPool() *ByteBufferPool {
	return &ByteBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				// Create buffers with reasonable initial capacity
				return make([]byte, 0, 4096) // 4KB initial capacity
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (bp *ByteBufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

// Put returns a buffer to the pool (buffer will be reset)
func (bp *ByteBufferPool) Put(buf []byte) {
	// Reset the buffer but keep the underlying capacity
	buf = buf[:0]
	bp.pool.Put(buf)
}

// StringBufferPool provides a pool of reusable string builders
type StringBufferPool struct {
	pool sync.Pool
}

// NewStringBufferPool creates a new string buffer pool
func NewStringBufferPool() *StringBufferPool {
	return &StringBufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]string, 0, 32) // Initial capacity for 32 strings
			},
		},
	}
}

// Get retrieves a string slice from the pool
func (sp *StringBufferPool) Get() []string {
	return sp.pool.Get().([]string)
}

// Put returns a string slice to the pool (slice will be reset)
func (sp *StringBufferPool) Put(buf []string) {
	// Reset the slice but keep the underlying capacity
	for i := range buf {
		buf[i] = "" // Clear strings to help GC
	}
	buf = buf[:0]
	sp.pool.Put(buf)
}

// NoteCache provides an LRU cache for frequently accessed notes
type NoteCache struct {
	mutex    sync.RWMutex
	capacity int
	cache    map[string]*cacheEntry
	head     *cacheEntry
	tail     *cacheEntry
}

type cacheEntry struct {
	key   string
	value interface{}
	prev  *cacheEntry
	next  *cacheEntry
}

// NewNoteCache creates a new LRU cache with the specified capacity
func NewNoteCache(capacity int) *NoteCache {
	if capacity <= 0 {
		capacity = 100 // Default capacity
	}

	cache := &NoteCache{
		capacity: capacity,
		cache:    make(map[string]*cacheEntry, capacity),
	}

	// Initialize sentinel nodes
	cache.head = &cacheEntry{}
	cache.tail = &cacheEntry{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	return cache
}

// Get retrieves a value from the cache
func (nc *NoteCache) Get(key string) (interface{}, bool) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	if entry, exists := nc.cache[key]; exists {
		nc.moveToHead(entry)
		return entry.value, true
	}
	return nil, false
}

// Put adds or updates a value in the cache
func (nc *NoteCache) Put(key string, value interface{}) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	if entry, exists := nc.cache[key]; exists {
		// Update existing entry
		entry.value = value
		nc.moveToHead(entry)
		return
	}

	// Create new entry
	newEntry := &cacheEntry{
		key:   key,
		value: value,
	}

	nc.cache[key] = newEntry
	nc.addToHead(newEntry)

	// Check capacity and evict if necessary
	if len(nc.cache) > nc.capacity {
		tail := nc.removeTail()
		delete(nc.cache, tail.key)
	}
}

// Remove removes a key from the cache
func (nc *NoteCache) Remove(key string) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	if entry, exists := nc.cache[key]; exists {
		nc.removeEntry(entry)
		delete(nc.cache, key)
	}
}

// Clear removes all entries from the cache
func (nc *NoteCache) Clear() {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	nc.cache = make(map[string]*cacheEntry, nc.capacity)
	nc.head.next = nc.tail
	nc.tail.prev = nc.head
}

// Size returns the current number of entries in the cache
func (nc *NoteCache) Size() int {
	nc.mutex.RLock()
	defer nc.mutex.RUnlock()
	return len(nc.cache)
}

// moveToHead moves an entry to the head of the doubly linked list
func (nc *NoteCache) moveToHead(entry *cacheEntry) {
	nc.removeEntry(entry)
	nc.addToHead(entry)
}

// addToHead adds an entry to the head of the doubly linked list
func (nc *NoteCache) addToHead(entry *cacheEntry) {
	entry.prev = nc.head
	entry.next = nc.head.next
	nc.head.next.prev = entry
	nc.head.next = entry
}

// removeEntry removes an entry from the doubly linked list
func (nc *NoteCache) removeEntry(entry *cacheEntry) {
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
}

// removeTail removes and returns the tail entry
func (nc *NoteCache) removeTail() *cacheEntry {
	lastEntry := nc.tail.prev
	nc.removeEntry(lastEntry)
	return lastEntry
}

// MemoryMonitor provides memory usage monitoring and optimization
type MemoryMonitor struct {
	maxMemoryMB     int64
	cleanupCallback func()
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(maxMemoryMB int64, cleanupCallback func()) *MemoryMonitor {
	return &MemoryMonitor{
		maxMemoryMB:     maxMemoryMB,
		cleanupCallback: cleanupCallback,
	}
}

// CheckMemoryUsage checks current memory usage and triggers cleanup if needed
func (mm *MemoryMonitor) CheckMemoryUsage() {
	// This is a simplified implementation
	// In a production environment, you would use runtime.MemStats
	// for more accurate memory monitoring

	if mm.cleanupCallback != nil {
		// For now, we'll trigger cleanup based on cache size or other heuristics
		// This can be enhanced with actual memory statistics
		mm.cleanupCallback()
	}
}
