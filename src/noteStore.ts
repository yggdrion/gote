import { join } from "path";
import { existsSync, mkdirSync, readdirSync, unlinkSync } from "fs";
import { type Note, type EncryptedNote } from "./types";
import { encrypt, decrypt } from "./crypto";
import { generateShortUUID } from "./auth";

export class NoteStore {
  private dataDir: string;
  private notes: Map<string, Note> = new Map();

  constructor(dataDir: string) {
    this.dataDir = dataDir;

    if (!existsSync(dataDir)) {
      mkdirSync(dataDir, { recursive: true });
    }
  }

  async loadNotes(key: Uint8Array): Promise<void> {
    this.notes.clear();

    try {
      const files = readdirSync(this.dataDir).filter((file) =>
        file.endsWith(".json")
      );

      for (const file of files) {
        try {
          const filePath = join(this.dataDir, file);
          const fileContent = (await Bun.file(
            filePath
          ).json()) as EncryptedNote;

          const decryptedContent = decrypt(fileContent.encryptedData, key);

          const note: Note = {
            id: fileContent.id,
            content: decryptedContent,
            createdAt: new Date(fileContent.createdAt),
            updatedAt: new Date(fileContent.updatedAt),
          };

          this.notes.set(note.id, note);
        } catch (err) {
          console.warn(`Error loading note from ${file}:`, err);
        }
      }
    } catch (err) {
      console.warn("Error reading data directory:", err);
    }
  }

  private async saveNote(note: Note, key: Uint8Array): Promise<void> {
    const encryptedContent = encrypt(note.content, key);

    const encryptedNote: EncryptedNote = {
      id: note.id,
      encryptedData: encryptedContent,
      createdAt: note.createdAt,
      updatedAt: note.updatedAt,
    };

    const filename = join(this.dataDir, `${note.id}.json`);
    await Bun.write(filename, JSON.stringify(encryptedNote, null, 2));
  }

  private async deleteNote(id: string): Promise<void> {
    const filename = join(this.dataDir, `${id}.json`);
    this.notes.delete(id);

    try {
      unlinkSync(filename);
    } catch (err) {
      console.warn(`Error deleting note file ${filename}:`, err);
    }
  }

  async createNote(content: string, key: Uint8Array): Promise<Note> {
    const note: Note = {
      id: generateShortUUID(),
      content,
      createdAt: new Date(),
      updatedAt: new Date(),
    };

    this.notes.set(note.id, note);

    try {
      await this.saveNote(note, key);
      return note;
    } catch (err) {
      this.notes.delete(note.id);
      throw err;
    }
  }

  async updateNote(
    id: string,
    content: string,
    key: Uint8Array
  ): Promise<Note> {
    const note = this.notes.get(id);
    if (!note) {
      throw new Error("Note not found");
    }

    note.content = content;
    note.updatedAt = new Date();

    await this.saveNote(note, key);
    return note;
  }

  getNote(id: string): Note | null {
    return this.notes.get(id) || null;
  }

  getAllNotes(): Note[] {
    const notes = Array.from(this.notes.values());
    // Sort by updated time, newest first
    return notes.sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime());
  }

  searchNotes(query: string): Note[] {
    const lowercaseQuery = query.toLowerCase();
    const results = Array.from(this.notes.values()).filter((note) =>
      note.content.toLowerCase().includes(lowercaseQuery)
    );

    // Sort by updated time, newest first
    return results.sort(
      (a, b) => b.updatedAt.getTime() - a.updatedAt.getTime()
    );
  }

  async deleteNoteById(id: string): Promise<void> {
    const note = this.notes.get(id);
    if (!note) {
      throw new Error("Note not found");
    }

    await this.deleteNote(id);
  }
}
