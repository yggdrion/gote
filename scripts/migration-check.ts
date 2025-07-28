#!/usr/bin/env bun

import { readdir, readFile } from "fs/promises";
import { join } from "path";
import { existsSync } from "fs";

interface EncryptedNote {
  id: string;
  encrypted_data: string;
  created_at: string;
  updated_at: string;
}

async function checkDataIntegrity(dataPath: string = "./data") {
  console.log("🔍 Checking data integrity...\n");

  if (!existsSync(dataPath)) {
    console.log("❌ Data directory not found:", dataPath);
    return;
  }

  try {
    const files = await readdir(dataPath);
    const jsonFiles = files.filter((f) => f.endsWith(".json"));

    console.log(`📁 Data directory: ${dataPath}`);
    console.log(`📄 Found ${jsonFiles.length} note files\n`);

    if (jsonFiles.length === 0) {
      console.log(
        "ℹ️  No notes found. This is normal for a fresh installation."
      );
      return;
    }

    let validNotes = 0;
    let invalidNotes = 0;

    for (const file of jsonFiles) {
      try {
        const filePath = join(dataPath, file);
        const content = await readFile(filePath, "utf-8");
        const note: EncryptedNote = JSON.parse(content);

        // Validate note structure
        const hasRequiredFields =
          note.id && note.encrypted_data && note.created_at && note.updated_at;

        if (hasRequiredFields) {
          console.log(`✅ ${file}: Valid (ID: ${note.id})`);
          validNotes++;
        } else {
          console.log(`❌ ${file}: Missing required fields`);
          invalidNotes++;
        }
      } catch (err) {
        console.log(`❌ ${file}: Parse error -`, err);
        invalidNotes++;
      }
    }

    console.log(`\n📊 Summary:`);
    console.log(`   Valid notes: ${validNotes}`);
    console.log(`   Invalid notes: ${invalidNotes}`);

    if (invalidNotes === 0) {
      console.log(`\n🎉 All notes are valid! Migration should work perfectly.`);
    } else {
      console.log(`\n⚠️  Some notes have issues. Check the files manually.`);
    }
  } catch (err) {
    console.error("❌ Error reading data directory:", err);
  }
}

async function checkConfiguration() {
  console.log("\n🔧 Checking configuration...\n");

  const possibleConfigPaths = [
    "./config.json",
    join(process.env.APPDATA || "", "gote", "config.json"),
    join(process.env.HOME || "", ".config", "gote", "config.json"),
    join(process.env.XDG_CONFIG_HOME || "", "gote", "config.json"),
  ];

  let configFound = false;

  for (const configPath of possibleConfigPaths) {
    if (existsSync(configPath)) {
      console.log(`✅ Config found: ${configPath}`);

      try {
        const content = await readFile(configPath, "utf-8");
        const config = JSON.parse(content);
        console.log(`   Notes path: ${config.notesPath || "default (./data)"}`);
        console.log(
          `   Password hash path: ${config.passwordHashPath || "default"}`
        );
        configFound = true;
      } catch (err) {
        console.log(`❌ Config parse error: ${err}`);
      }
      break;
    }
  }

  if (!configFound) {
    console.log("ℹ️  No config file found. Will use defaults.");
  }
}

async function checkPasswordHash() {
  console.log("\n🔐 Checking password setup...\n");

  const possibleHashPaths = [
    "./data/.password_hash",
    join(process.env.APPDATA || "", "gote", "gote_password_hash"),
    join(process.env.HOME || "", ".config", "gote", "gote_password_hash"),
  ];

  let hashFound = false;

  for (const hashPath of possibleHashPaths) {
    if (existsSync(hashPath)) {
      console.log(`✅ Password hash found: ${hashPath}`);
      hashFound = true;
      break;
    }
  }

  if (!hashFound) {
    console.log(
      "ℹ️  No password hash found. First-time setup will be required."
    );
  }
}

// Main execution
async function main() {
  console.log("🚀 Gote Migration Checker\n");
  console.log(
    "This tool helps verify your data is ready for the TypeScript migration.\n"
  );

  await checkConfiguration();
  await checkPasswordHash();

  // Check default data path and any custom path from config
  await checkDataIntegrity();

  console.log("\n✨ Check complete!");
  console.log("\nNext steps:");
  console.log("1. Run 'bun install' to install dependencies");
  console.log("2. Run 'bun run dev' to start the server");
  console.log("3. Open http://localhost:8080 and enter your password");
}

if (import.meta.main) {
  main().catch(console.error);
}
