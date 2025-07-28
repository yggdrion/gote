import {
  randomBytes,
  createCipheriv,
  createDecipheriv,
  createHash,
} from "crypto";

export function encrypt(plaintext: string, key: Uint8Array): string {
  const algorithm = "aes-256-gcm";
  const iv = randomBytes(12); // GCM standard nonce size
  const cipher = createCipheriv(algorithm, key, iv);

  let encrypted = cipher.update(plaintext, "utf8");
  const final = cipher.final();
  const authTag = cipher.getAuthTag();

  // Combine iv, encrypted data, and auth tag (Go GCM.Seal format)
  const combined = Buffer.concat([iv, encrypted, final, authTag]);
  return combined.toString("base64");
}

export function decrypt(ciphertext: string, key: Uint8Array): string {
  const algorithm = "aes-256-gcm";
  const data = Buffer.from(ciphertext, "base64");

  if (data.length < 28) {
    // 12 bytes nonce + 16 bytes auth tag minimum
    throw new Error("ciphertext too short");
  }

  // Go GCM format: nonce + encrypted_data + auth_tag
  const nonceSize = 12;
  const authTagSize = 16;

  const nonce = data.subarray(0, nonceSize);
  const authTag = data.subarray(data.length - authTagSize);
  const encrypted = data.subarray(nonceSize, data.length - authTagSize);

  const decipher = createDecipheriv(algorithm, key, nonce);
  decipher.setAuthTag(authTag);

  let decrypted = decipher.update(encrypted, undefined, "utf8");
  decrypted += decipher.final("utf8");

  return decrypted;
}

export function deriveKey(password: string): Uint8Array {
  const hash = createHash("sha256");
  hash.update(password);
  return new Uint8Array(hash.digest());
}
