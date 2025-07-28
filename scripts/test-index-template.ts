#!/usr/bin/env bun

import { renderTemplate } from "../src/templates";

// Test with sample notes data
const data = {
  Notes: [
    {
      id: "test1",
      content: "This is a **test note** with `code`",
      createdAt: new Date(),
      updatedAt: new Date(),
    },
    {
      id: "test2",
      content: "Another note\n\nWith multiple lines",
      createdAt: new Date(),
      updatedAt: new Date(),
    },
  ],
  Query: "",
};

console.log("Testing index template...");
console.log("Notes data:", JSON.stringify(data.Notes, null, 2));
console.log("=".repeat(50));

try {
  const html = renderTemplate("./static/index.html", data);
  console.log("Template rendered successfully!");
  console.log("Length:", html.length);

  // Check if notes content appears
  if (html.includes("test note")) {
    console.log("✅ Notes content found in rendered HTML");
  } else {
    console.log("❌ Notes content NOT found in rendered HTML");
  }

  // Save to file for inspection
  await Bun.write("test-index.html", html);
  console.log("Saved to test-index.html");
} catch (err) {
  console.error("Error:", err);
}
