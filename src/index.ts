import { serve } from "bun";
import { router } from "./router";
import { loadConfig } from "./config";
import { NoteStore } from "./noteStore";
import { setStore, setConfig } from "./globals";

// Initialize application
async function init() {
  const currentConfig = await loadConfig();
  setConfig(currentConfig);

  console.log("Configuration loaded:");
  console.log(`  Notes directory: ${currentConfig.notesPath}`);
  console.log(`  Password hash file: ${currentConfig.passwordHashPath}`);
  console.log(`  Config file: ${currentConfig.configFilePath}`);

  const store = new NoteStore(currentConfig.notesPath);
  setStore(store);
}

// Start server
async function startServer() {
  console.log("🔧 Starting initialization...");
  await init();

  console.log("🚀 Creating server...");
  const server = serve({
    port: 8080,
    fetch: router,
  });

  console.log(`✅ Server starting on http://localhost:${server.port}`);
  console.log("🌐 Server is ready to accept connections");
}

startServer().catch(console.error);
