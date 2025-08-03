// Frontend Performance Utilities

// Debouncer for search input and other frequent operations
class Debouncer {
  constructor(delay = 300) {
    this.delay = delay;
    this.timeouts = new Map();
  }

  debounce(key, callback) {
    // Clear existing timeout for this key
    if (this.timeouts.has(key)) {
      clearTimeout(this.timeouts.get(key));
    }

    // Set new timeout
    const timeoutId = setTimeout(() => {
      this.timeouts.delete(key);
      callback();
    }, this.delay);

    this.timeouts.set(key, timeoutId);
  }

  cancel(key) {
    if (this.timeouts.has(key)) {
      clearTimeout(this.timeouts.get(key));
      this.timeouts.delete(key);
    }
  }

  clear() {
    this.timeouts.forEach((timeoutId) => clearTimeout(timeoutId));
    this.timeouts.clear();
  }
}

// Throttler for limiting execution frequency
class Throttler {
  constructor(delay = 1000) {
    this.delay = delay;
    this.lastExecution = new Map();
    this.pendingCallbacks = new Map();
  }

  throttle(key, callback) {
    const now = Date.now();
    const lastExec = this.lastExecution.get(key) || 0;
    const timeSinceLastExec = now - lastExec;

    if (timeSinceLastExec >= this.delay) {
      // Can execute immediately
      this.lastExecution.set(key, now);
      callback();
      return;
    }

    // Need to throttle - schedule for later
    if (this.pendingCallbacks.has(key)) {
      clearTimeout(this.pendingCallbacks.get(key));
    }

    const waitTime = this.delay - timeSinceLastExec;
    const timeoutId = setTimeout(() => {
      this.lastExecution.set(key, Date.now());
      this.pendingCallbacks.delete(key);
      callback();
    }, waitTime);

    this.pendingCallbacks.set(key, timeoutId);
  }
}

// Virtual scrolling for large note lists
class VirtualScroller {
  constructor(container, itemHeight = 80, bufferSize = 10) {
    this.container = container;
    this.itemHeight = itemHeight;
    this.bufferSize = bufferSize;
    this.scrollTop = 0;
    this.visibleStart = 0;
    this.visibleEnd = 0;
    this.items = [];
    this.renderCallback = null;

    this.setupScrolling();
  }

  setupScrolling() {
    this.container.addEventListener("scroll", (e) => {
      this.scrollTop = e.target.scrollTop;
      this.updateVisibleRange();
      this.render();
    });
  }

  setItems(items) {
    this.items = items;
    this.updateVisibleRange();
    this.render();
  }

  setRenderCallback(callback) {
    this.renderCallback = callback;
  }

  updateVisibleRange() {
    const containerHeight = this.container.clientHeight;
    const totalItems = this.items.length;

    this.visibleStart = Math.max(
      0,
      Math.floor(this.scrollTop / this.itemHeight) - this.bufferSize
    );
    this.visibleEnd = Math.min(
      totalItems,
      Math.ceil((this.scrollTop + containerHeight) / this.itemHeight) +
        this.bufferSize
    );
  }

  render() {
    if (!this.renderCallback) return;

    const visibleItems = this.items.slice(this.visibleStart, this.visibleEnd);
    const topOffset = this.visibleStart * this.itemHeight;
    const totalHeight = this.items.length * this.itemHeight;

    this.renderCallback({
      items: visibleItems,
      startIndex: this.visibleStart,
      topOffset,
      totalHeight,
    });
  }
}

// Optimized search with caching and indexing
class SearchOptimizer {
  constructor() {
    this.searchCache = new Map();
    this.searchIndex = new Map();
    this.maxCacheSize = 100;
  }

  // Build search index for faster searching
  buildIndex(notes) {
    this.searchIndex.clear();

    notes.forEach((note) => {
      const words = this.extractWords(note.content);
      words.forEach((word) => {
        if (!this.searchIndex.has(word)) {
          this.searchIndex.set(word, new Set());
        }
        this.searchIndex.get(word).add(note.id);
      });
    });
  }

  extractWords(content) {
    return content
      .toLowerCase()
      .replace(/[^\w\s]/g, " ")
      .split(/\s+/)
      .filter((word) => word.length > 2); // Only index words longer than 2 characters
  }

  search(query, notes) {
    const cacheKey = query.toLowerCase().trim();

    // Check cache first
    if (this.searchCache.has(cacheKey)) {
      return this.searchCache.get(cacheKey);
    }

    let results;

    if (query.trim() === "") {
      results = notes;
    } else {
      results = this.performSearch(query, notes);
    }

    // Cache results (with size limit)
    if (this.searchCache.size >= this.maxCacheSize) {
      // Remove oldest entry
      const firstKey = this.searchCache.keys().next().value;
      this.searchCache.delete(firstKey);
    }
    this.searchCache.set(cacheKey, results);

    return results;
  }

  performSearch(query, notes) {
    const queryWords = this.extractWords(query);

    if (queryWords.length === 0) {
      return notes;
    }

    // Use index for better performance if available
    if (this.searchIndex.size > 0) {
      return this.indexBasedSearch(queryWords, notes);
    }

    // Fallback to linear search
    return this.linearSearch(query, notes);
  }

  indexBasedSearch(queryWords, notes) {
    let candidateIds = null;

    queryWords.forEach((word) => {
      const wordResults = this.searchIndex.get(word) || new Set();

      if (candidateIds === null) {
        candidateIds = new Set(wordResults);
      } else {
        // Intersection - notes must contain all query words
        candidateIds = new Set(
          [...candidateIds].filter((id) => wordResults.has(id))
        );
      }
    });

    if (!candidateIds) return [];

    return notes.filter((note) => candidateIds.has(note.id));
  }

  linearSearch(query, notes) {
    const lowerQuery = query.toLowerCase();
    return notes.filter((note) =>
      note.content.toLowerCase().includes(lowerQuery)
    );
  }

  clearCache() {
    this.searchCache.clear();
  }
}

// Memory-efficient DOM manipulation
class DOMOptimizer {
  constructor() {
    this.elementPool = new Map();
    this.recycledElements = new Map();
  }

  // Create or reuse DOM elements
  getElement(tagName, className = "") {
    const key = `${tagName}-${className}`;

    if (
      this.recycledElements.has(key) &&
      this.recycledElements.get(key).length > 0
    ) {
      const element = this.recycledElements.get(key).pop();
      this.resetElement(element);
      return element;
    }

    const element = document.createElement(tagName);
    if (className) {
      element.className = className;
    }
    return element;
  }

  // Recycle DOM elements instead of destroying them
  recycleElement(element) {
    const key = `${element.tagName.toLowerCase()}-${element.className}`;

    if (!this.recycledElements.has(key)) {
      this.recycledElements.set(key, []);
    }

    this.resetElement(element);
    this.recycledElements.get(key).push(element);
  }

  resetElement(element) {
    element.innerHTML = "";
    element.removeAttribute("style");
    element.className = element.className.split(" ")[0] || ""; // Keep only base class

    // Remove event listeners by cloning the element
    const newElement = element.cloneNode(false);
    element.parentNode?.replaceChild(newElement, element);
    return newElement;
  }

  // Batch DOM operations for better performance
  batchUpdate(callback) {
    requestAnimationFrame(() => {
      callback();
    });
  }

  // Efficiently update text content
  updateTextContent(element, text) {
    if (element.textContent !== text) {
      element.textContent = text;
    }
  }

  // Efficiently toggle classes
  toggleClass(element, className, condition) {
    if (condition) {
      if (!element.classList.contains(className)) {
        element.classList.add(className);
      }
    } else {
      if (element.classList.contains(className)) {
        element.classList.remove(className);
      }
    }
  }
}

// Performance monitoring
class PerformanceMonitor {
  constructor() {
    this.metrics = new Map();
    this.startTimes = new Map();
  }

  startTiming(key) {
    this.startTimes.set(key, performance.now());
  }

  endTiming(key) {
    const startTime = this.startTimes.get(key);
    if (startTime) {
      const duration = performance.now() - startTime;
      this.addMetric(key, duration);
      this.startTimes.delete(key);
      return duration;
    }
    return 0;
  }

  addMetric(key, value) {
    if (!this.metrics.has(key)) {
      this.metrics.set(key, {
        count: 0,
        total: 0,
        min: Infinity,
        max: -Infinity,
        avg: 0,
      });
    }

    const metric = this.metrics.get(key);
    metric.count++;
    metric.total += value;
    metric.min = Math.min(metric.min, value);
    metric.max = Math.max(metric.max, value);
    metric.avg = metric.total / metric.count;
  }

  getMetrics() {
    const result = {};
    this.metrics.forEach((value, key) => {
      result[key] = { ...value };
    });
    return result;
  }

  logMetrics() {
    console.table(this.getMetrics());
  }

  reset() {
    this.metrics.clear();
    this.startTimes.clear();
  }
}

// Export utilities
export {
  Debouncer,
  Throttler,
  VirtualScroller,
  SearchOptimizer,
  DOMOptimizer,
  PerformanceMonitor,
};
