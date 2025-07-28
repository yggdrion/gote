import { serve } from "bun";
import { router } from "./router";
import { loadConfig } from "./config";
import { NoteStore } from "./noteStore";
import { type Config } from "./types";

// Global state
export let store: NoteStore;
export let currentConfig: Config;

// Initialize application
async function init() {
  currentConfig = await loadConfig();

  console.log("Configuration loaded:");
  console.log(`  Notes directory: ${currentConfig.notesPath}`);
  console.log(`  Password hash file: ${currentConfig.passwordHashPath}`);
  console.log(`  Config file: ${currentConfig.configFilePath}`);

  store = new NoteStore(currentConfig.notesPath);
}

// Start server
async function startServer() {
  await init();

  const server = serve({
    port: 8080,
    fetch: router,
  });

  console.log(`Server starting on http://localhost:${server.port}`);
}

startServer().catch(console.error);
