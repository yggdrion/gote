import { join, dirname } from "path";
import { existsSync, mkdirSync } from "fs";
import {
  isAuthenticated,
  isFirstTimeSetup,
  verifyPassword,
  storePasswordHash,
  createSession,
  destroySession,
} from "./auth";
import { deriveKey } from "./crypto";
import { store, currentConfig } from "./globals";
import { renderTemplate } from "./templates";
import { saveConfig } from "./config";
import {
  type CreateNoteRequest,
  type UpdateNoteRequest,
  type Config,
} from "./types";

// Helper function to serve static files
async function serveStatic(pathname: string): Promise<Response | null> {
  if (pathname.startsWith("/static/")) {
    const filePath = join(".", pathname);
    const file = Bun.file(filePath);

    if (await file.exists()) {
      const ext = pathname.split(".").pop()?.toLowerCase();
      const contentType = getContentType(ext);

      return new Response(file, {
        headers: { "Content-Type": contentType },
      });
    }
  }
  return null;
}

function getContentType(ext: string | undefined): string {
  const types: Record<string, string> = {
    html: "text/html; charset=utf-8",
    css: "text/css",
    js: "application/javascript",
    json: "application/json",
    png: "image/png",
    jpg: "image/jpeg",
    jpeg: "image/jpeg",
    gif: "image/gif",
    svg: "image/svg+xml",
  };
  return types[ext || ""] || "application/octet-stream";
}

function setCookie(
  response: Response,
  name: string,
  value: string,
  options: any = {}
): void {
  const cookieValue = `${name}=${encodeURIComponent(value)}; Path=/; HttpOnly${
    options.maxAge ? `; Max-Age=${options.maxAge}` : ""
  }${options.secure ? "; Secure" : ""}`;
  response.headers.set("Set-Cookie", cookieValue);
}

function redirect(url: string, status = 302): Response {
  return new Response(null, {
    status,
    headers: { Location: url },
  });
}

export async function router(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const pathname = url.pathname;
  const method = request.method;

  console.log(`📝 ${method} ${pathname}`);

  // Serve static files
  const staticResponse = await serveStatic(pathname);
  if (staticResponse) {
    console.log(`📁 Serving static file: ${pathname}`);
    return staticResponse;
  }

  // Authentication routes (no auth required)
  if (pathname === "/login" && method === "GET") {
    return loginHandler(request);
  }

  if (pathname === "/auth" && method === "POST") {
    return authHandler(request);
  }

  if (pathname === "/logout" && method === "POST") {
    return logoutHandler(request);
  }

  // Protected routes
  const session = isAuthenticated(request);
  if (!session) {
    if (pathname.startsWith("/api/")) {
      return new Response("Unauthorized", { status: 401 });
    }
    return redirect("/login");
  }

  // Main app route
  if (pathname === "/" && method === "GET") {
    return indexHandler(request);
  }

  // API routes
  if (pathname === "/api/notes" && method === "GET") {
    return apiGetNotesHandler();
  }

  if (pathname === "/api/notes" && method === "POST") {
    return apiCreateNoteHandler(request, session.key);
  }

  if (pathname.startsWith("/api/notes/") && method === "GET") {
    const id = pathname.split("/").pop();
    return apiGetNoteHandler(id || "");
  }

  if (pathname.startsWith("/api/notes/") && method === "PUT") {
    const id = pathname.split("/").pop();
    return apiUpdateNoteHandler(request, id || "", session.key);
  }

  if (pathname.startsWith("/api/notes/") && method === "DELETE") {
    const id = pathname.split("/").pop();
    return apiDeleteNoteHandler(id || "");
  }

  if (pathname === "/api/search" && method === "GET") {
    return apiSearchHandler(request);
  }

  if (pathname === "/api/settings" && method === "GET") {
    return apiGetSettingsHandler();
  }

  if (pathname === "/api/settings" && method === "POST") {
    return apiSettingsHandler(request);
  }

  return new Response("Not Found", { status: 404 });
}

async function loginHandler(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const error = url.searchParams.get("error") || "";
  const isFirstTime = isFirstTimeSetup();

  const data = {
    Error: error,
    IsFirstTime: isFirstTime,
  };

  try {
    const html = renderTemplate("./static/login.html", data);
    return new Response(html, {
      headers: { "Content-Type": "text/html; charset=utf-8" },
    });
  } catch (err) {
    console.error("Template error:", err);
    return new Response("Template error", { status: 500 });
  }
}

async function authHandler(request: Request): Promise<Response> {
  const formData = await request.formData();
  const password = formData.get("password")?.toString() || "";

  if (!password) {
    return redirect("/login?error=Password required");
  }

  // Handle first-time setup
  if (isFirstTimeSetup()) {
    const confirmPassword = formData.get("confirm_password")?.toString() || "";

    if (!confirmPassword) {
      return redirect("/login?error=Please confirm your password");
    }

    if (password !== confirmPassword) {
      return redirect("/login?error=Passwords do not match");
    }

    if (password.length < 6) {
      return redirect("/login?error=Password must be at least 6 characters");
    }

    try {
      await storePasswordHash(password);
    } catch (err) {
      return redirect("/login?error=Failed to create password");
    }
  } else {
    // Verify existing password
    if (!(await verifyPassword(password))) {
      return redirect("/login?error=Invalid password");
    }
  }

  const key = deriveKey(password);

  // Try to load notes with this password
  try {
    await store.loadNotes(key);
  } catch (err) {
    if (!isFirstTimeSetup()) {
      return redirect("/login?error=Failed to decrypt notes");
    }
  }

  // Create session
  const sessionId = createSession(key);

  const response = redirect("/");
  setCookie(response, "session", sessionId);
  return response;
}

async function logoutHandler(request: Request): Promise<Response> {
  const cookieHeader = request.headers.get("cookie");
  if (cookieHeader) {
    const sessionMatch = cookieHeader.match(/session=([^;]+)/);
    if (sessionMatch && sessionMatch[1]) {
      destroySession(decodeURIComponent(sessionMatch[1]));
    }
  }

  const response = redirect("/login");
  setCookie(response, "session", "", { maxAge: -1 });
  return response;
}

async function indexHandler(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const query = url.searchParams.get("q") || "";

  const notes = query ? store.searchNotes(query) : store.getAllNotes();

  const data = {
    Notes: notes,
    Query: query,
  };

  try {
    const html = renderTemplate("./static/index.html", data);
    return new Response(html, {
      headers: { "Content-Type": "text/html; charset=utf-8" },
    });
  } catch (err) {
    console.error("Template error:", err);
    return new Response("Template error: " + String(err), { status: 500 });
  }
}

// API Handlers
async function apiGetNotesHandler(): Promise<Response> {
  const notes = store.getAllNotes();
  return new Response(JSON.stringify(notes), {
    headers: { "Content-Type": "application/json" },
  });
}

async function apiCreateNoteHandler(
  request: Request,
  key: Uint8Array
): Promise<Response> {
  try {
    const { content }: CreateNoteRequest = await request.json();
    const note = await store.createNote(content, key);

    return new Response(JSON.stringify(note), {
      status: 201,
      headers: { "Content-Type": "application/json" },
    });
  } catch (err) {
    return new Response(JSON.stringify({ error: "Invalid JSON" }), {
      status: 400,
      headers: { "Content-Type": "application/json" },
    });
  }
}

async function apiGetNoteHandler(id: string): Promise<Response> {
  if (!id) {
    return new Response("Invalid note ID", { status: 400 });
  }

  const note = store.getNote(id);
  if (!note) {
    return new Response("Note not found", { status: 404 });
  }

  return new Response(JSON.stringify(note), {
    headers: { "Content-Type": "application/json" },
  });
}

async function apiUpdateNoteHandler(
  request: Request,
  id: string,
  key: Uint8Array
): Promise<Response> {
  if (!id) {
    return new Response("Invalid note ID", { status: 400 });
  }

  try {
    const { content }: UpdateNoteRequest = await request.json();
    const note = await store.updateNote(id, content, key);

    return new Response(JSON.stringify(note), {
      headers: { "Content-Type": "application/json" },
    });
  } catch (err) {
    if (err instanceof Error && err.message === "Note not found") {
      return new Response(err.message, { status: 404 });
    }
    return new Response("Invalid JSON", { status: 400 });
  }
}

async function apiDeleteNoteHandler(id: string): Promise<Response> {
  if (!id) {
    return new Response("Invalid note ID", { status: 400 });
  }

  try {
    await store.deleteNoteById(id);
    return new Response(null, { status: 204 });
  } catch (err) {
    return new Response("Note not found", { status: 404 });
  }
}

async function apiSearchHandler(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const query = url.searchParams.get("q");

  if (!query) {
    return new Response("Missing query parameter", { status: 400 });
  }

  const notes = store.searchNotes(query);
  return new Response(JSON.stringify(notes), {
    headers: { "Content-Type": "application/json" },
  });
}

async function apiGetSettingsHandler(): Promise<Response> {
  return new Response(JSON.stringify(currentConfig), {
    headers: { "Content-Type": "application/json" },
  });
}

async function apiSettingsHandler(request: Request): Promise<Response> {
  try {
    const req: Config = await request.json();

    // Validate and set default paths if empty
    if (!req.notesPath) {
      req.notesPath = "./data";
    }
    if (!req.passwordHashPath) {
      req.passwordHashPath = currentConfig.passwordHashPath;
    }

    // Ensure directories exist before saving config
    if (!existsSync(req.notesPath)) {
      try {
        mkdirSync(req.notesPath, { recursive: true });
      } catch (err) {
        return new Response(`Failed to create notes directory: ${err}`, {
          status: 400,
        });
      }
    }

    const passwordDir = dirname(req.passwordHashPath);
    if (!existsSync(passwordDir)) {
      try {
        mkdirSync(passwordDir, { recursive: true });
      } catch (err) {
        return new Response(
          `Failed to create password hash directory: ${err}`,
          { status: 400 }
        );
      }
    }

    // Update global config
    currentConfig.notesPath = req.notesPath;
    currentConfig.passwordHashPath = req.passwordHashPath;

    // Save config to file
    try {
      await saveConfig(currentConfig);
    } catch (err) {
      return new Response("Failed to save configuration", { status: 500 });
    }

    return new Response(
      JSON.stringify({
        success: true,
        message: "Settings saved successfully",
      }),
      {
        headers: { "Content-Type": "application/json" },
      }
    );
  } catch (err) {
    return new Response("Invalid JSON", { status: 400 });
  }
}
