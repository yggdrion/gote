#!/usr/bin/env bun

import { deriveKey } from "../src/crypto";
import { NoteStore } from "../src/noteStore";

// Test the app with a simple note
async function testApp() {
  try {
    const password = "testpass123";
    const key = deriveKey(password);

    const store = new NoteStore("./data");

    // Create a test note
    const noteId = await store.createNote(
      "# Test Note\n\nThis is a test note with **bold** text and *italic* text.\n\n```javascript\nconsole.log('Hello World');\n```",
      key
    );
    console.log("Created note with ID:", noteId.id);

    // Load notes and display
    await store.loadNotes(key);
    const notes = store.getAllNotes();
    console.log("Loaded notes:", notes.length);

    for (const note of notes) {
      console.log(`Note ${note.id}: ${note.content.substring(0, 50)}...`);
    }

    console.log("✅ Crypto compatibility test passed!");
    process.exit(0);
  } catch (error) {
    console.error("❌ Crypto compatibility test failed:", error);
    process.exit(1);
  }
}

testApp();
