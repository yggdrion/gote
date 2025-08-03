package performance

import (
	"sync"
	"time"
)

// Debouncer provides debouncing functionality for frequent operations
type Debouncer struct {
	mutex    sync.Mutex
	timers   map[string]*time.Timer
	duration time.Duration
}

// NewDebouncer creates a new debouncer with the specified duration
func NewDebouncer(duration time.Duration) *Debouncer {
	return &Debouncer{
		timers:   make(map[string]*time.Timer),
		duration: duration,
	}
}

// Debounce executes the function after the debounce duration has passed
// If called again with the same key before the duration expires, the previous call is cancelled
func (d *Debouncer) Debounce(key string, fn func()) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Cancel existing timer if it exists
	if timer, exists := d.timers[key]; exists {
		timer.Stop()
	}

	// Create new timer
	d.timers[key] = time.AfterFunc(d.duration, func() {
		d.mutex.Lock()
		delete(d.timers, key)
		d.mutex.Unlock()
		fn()
	})
}

// Cancel cancels a pending debounced function call
func (d *Debouncer) Cancel(key string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if timer, exists := d.timers[key]; exists {
		timer.Stop()
		delete(d.timers, key)
	}
}

// Clear cancels all pending debounced function calls
func (d *Debouncer) Clear() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for key, timer := range d.timers {
		timer.Stop()
		delete(d.timers, key)
	}
}

// ThrottledExecutor provides throttling functionality to limit execution frequency
type ThrottledExecutor struct {
	mutex       sync.Mutex
	lastExec    time.Time
	minInterval time.Duration
	pending     bool
	pendingFn   func()
	timer       *time.Timer
}

// NewThrottledExecutor creates a new throttled executor
func NewThrottledExecutor(minInterval time.Duration) *ThrottledExecutor {
	return &ThrottledExecutor{
		minInterval: minInterval,
	}
}

// Execute runs the function with throttling
func (te *ThrottledExecutor) Execute(fn func()) {
	te.mutex.Lock()
	defer te.mutex.Unlock()

	now := time.Now()
	timeSinceLastExec := now.Sub(te.lastExec)

	if timeSinceLastExec >= te.minInterval {
		// Can execute immediately
		te.lastExec = now
		go fn()
		return
	}

	// Need to throttle - schedule for later
	te.pendingFn = fn
	if !te.pending {
		te.pending = true
		waitTime := te.minInterval - timeSinceLastExec

		if te.timer != nil {
			te.timer.Stop()
		}

		te.timer = time.AfterFunc(waitTime, func() {
			te.mutex.Lock()
			if te.pending && te.pendingFn != nil {
				te.pending = false
				fn := te.pendingFn
				te.pendingFn = nil
				te.lastExec = time.Now()
				te.mutex.Unlock()
				go fn()
			} else {
				te.mutex.Unlock()
			}
		})
	}
}

// BatchProcessor processes items in batches to improve performance
type BatchProcessor struct {
	mutex        sync.Mutex
	items        []interface{}
	maxBatchSize int
	maxWaitTime  time.Duration
	processor    func([]interface{})
	timer        *time.Timer
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(maxBatchSize int, maxWaitTime time.Duration, processor func([]interface{})) *BatchProcessor {
	return &BatchProcessor{
		maxBatchSize: maxBatchSize,
		maxWaitTime:  maxWaitTime,
		processor:    processor,
	}
}

// Add adds an item to the batch
func (bp *BatchProcessor) Add(item interface{}) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	bp.items = append(bp.items, item)

	// Process immediately if batch is full
	if len(bp.items) >= bp.maxBatchSize {
		bp.processBatch()
		return
	}

	// Set timer for batch processing if not already set
	if bp.timer == nil {
		bp.timer = time.AfterFunc(bp.maxWaitTime, func() {
			bp.mutex.Lock()
			defer bp.mutex.Unlock()
			if len(bp.items) > 0 {
				bp.processBatch()
			}
		})
	}
}

// processBatch processes the current batch (must be called with mutex held)
func (bp *BatchProcessor) processBatch() {
	if len(bp.items) == 0 {
		return
	}

	items := make([]interface{}, len(bp.items))
	copy(items, bp.items)
	bp.items = bp.items[:0] // Clear the slice

	if bp.timer != nil {
		bp.timer.Stop()
		bp.timer = nil
	}

	// Process in goroutine to avoid blocking
	go bp.processor(items)
}

// Flush processes any pending items immediately
func (bp *BatchProcessor) Flush() {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	bp.processBatch()
}
