#!/usr/bin/env bun

import { storePasswordHash } from "../src/auth";

// Set up test password
async function setupPassword() {
  await storePasswordHash("testpass123");
  console.log("Password set to: testpass123");
}

setupPassword().catch(console.error);
