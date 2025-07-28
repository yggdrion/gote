import { createHash, randomBytes } from "crypto";
import { existsSync } from "fs";
import { dirname } from "path";
import { mkdirSync } from "fs";
import { type Session } from "./types";
import { currentConfig } from "./index";

// Session management
const sessions = new Map<string, Session>();
const sessionTimeout = 30 * 60 * 1000; // 30 minutes in milliseconds

export function getPasswordHashFile(): string {
  return currentConfig?.passwordHashPath || "./data/.password_hash";
}

export function isFirstTimeSetup(): boolean {
  return !existsSync(getPasswordHashFile());
}

export async function storePasswordHash(password: string): Promise<void> {
  const verificationHash = createHash("sha256")
    .update(password + "verification")
    .digest();

  const hashFile = getPasswordHashFile();
  const hashDir = dirname(hashFile);

  if (!existsSync(hashDir)) {
    mkdirSync(hashDir, { recursive: true });
  }

  await Bun.write(hashFile, verificationHash);
}

export async function verifyPassword(password: string): Promise<boolean> {
  if (isFirstTimeSetup()) {
    return false;
  }

  try {
    const hashFile = getPasswordHashFile();
    const file = Bun.file(hashFile);
    const storedHash = new Uint8Array(await file.arrayBuffer());

    const verificationHash = createHash("sha256")
      .update(password + "verification")
      .digest();

    return Buffer.compare(Buffer.from(storedHash), verificationHash) === 0;
  } catch (err) {
    return false;
  }
}

export function generateSessionID(): string {
  return randomBytes(32).toString("base64url");
}

export function generateShortUUID(): string {
  // Generate a UUID-like string with 8 characters
  return randomBytes(4).toString("hex");
}

export function isAuthenticated(request: Request): Session | null {
  const cookieHeader = request.headers.get("cookie");
  if (!cookieHeader) {
    return null;
  }

  const cookies = parseCookies(cookieHeader);
  const sessionId = cookies.session;

  if (!sessionId) {
    return null;
  }

  const session = sessions.get(sessionId);
  if (!session || new Date() > session.expiresAt) {
    if (session) {
      sessions.delete(sessionId);
    }
    return null;
  }

  // Extend session
  session.expiresAt = new Date(Date.now() + sessionTimeout);
  return session;
}

export function createSession(key: Uint8Array): string {
  const sessionId = generateSessionID();
  const session: Session = {
    key,
    expiresAt: new Date(Date.now() + sessionTimeout),
  };

  sessions.set(sessionId, session);
  return sessionId;
}

export function destroySession(sessionId: string): void {
  sessions.delete(sessionId);
}

function parseCookies(cookieHeader: string): Record<string, string> {
  const cookies: Record<string, string> = {};

  cookieHeader.split(";").forEach((cookie) => {
    const [name, value] = cookie.trim().split("=");
    if (name && value) {
      cookies[name] = decodeURIComponent(value);
    }
  });

  return cookies;
}
