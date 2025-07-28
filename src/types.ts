export interface Session {
  key: Uint8Array;
  expiresAt: Date;
}

export interface Config {
  notesPath: string;
  passwordHashPath: string;
  configFilePath?: string;
}

export interface EncryptedNote {
  id: string;
  encryptedData: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface Note {
  id: string;
  content: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface CreateNoteRequest {
  content: string;
}

export interface UpdateNoteRequest {
  content: string;
}

export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
}
