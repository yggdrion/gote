// Simplified Performance Utilities for Notes App

// Simple debouncer for search input
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

// Export utilities
export { Debouncer };
