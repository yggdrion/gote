#!/usr/bin/env bun

import { deriveKey } from "../src/crypto";
import { NoteStore } from "../src/noteStore";
import { renderTemplate } from "../src/templates";

// Test template rendering with actual data
async function testTemplateRendering() {
  try {
    const password = "testpass123";
    const key = deriveKey(password);

    const store = new NoteStore("./data");

    // Load notes
    try {
      await store.loadNotes(key);
      console.log("✅ Notes loaded successfully");
    } catch (err) {
      console.error("❌ Error loading notes:", err);
      throw err;
    }

    const notes = store.getAllNotes();
    console.log(`📝 Found ${notes.length} notes`);

    const data = {
      Notes: notes,
      Query: "",
    };

    console.log("🔧 Template data:", JSON.stringify(data, null, 2));

    try {
      const html = renderTemplate("./static/index.html", data);
      console.log("✅ Template rendered successfully");

      // Check if template variables are properly replaced
      if (html.includes("{{")) {
        console.log("❌ Found unreplaced template variables:");
        const matches = html.match(/\{\{[^}]+\}\}/g);
        if (matches) {
          matches.forEach((match) => console.log("  -", match));
        }
        throw new Error("Template variables not properly replaced");
      } else {
        console.log("✅ All template variables replaced");
      }

      // Check if notes content is in the output
      if (notes.length > 0) {
        const noteId = notes[0].id;
        const noteContentStart = notes[0].content.substring(0, 20);

        if (html.includes(noteId)) {
          console.log("✅ Note ID found in rendered HTML");
        } else {
          console.log("❌ Note ID NOT found in rendered HTML");
          throw new Error("Note ID not found in rendered HTML");
        }

        if (html.includes("Test Note")) {
          console.log("✅ Note content found in rendered HTML");
        } else {
          console.log("❌ Note content NOT found in rendered HTML");
          throw new Error("Note content not found in rendered HTML");
        }
      } else {
        console.log("⚠️ No notes to check");
      }

      console.log("✅ Template rendering test passed!");
      process.exit(0);
    } catch (err) {
      console.error("❌ Template rendering error:", err);
      throw err;
    }
  } catch (error) {
    console.error("❌ Template rendering test failed:", error);
    process.exit(1);
  }
}

testTemplateRendering();
