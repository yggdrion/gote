use aes_gcm::aead::{Aead, AeadCore, KeyInit, OsRng};
use aes_gcm::{Aes256Gcm, Key, Nonce};
use base64::Engine;
use base64::engine::general_purpose::STANDARD;
use rand::RngCore;

use crate::error::{AppError, Result};

pub const KEY_LEN: usize = 32;
pub const SALT_LEN: usize = 32;
pub const PBKDF2_ITERATIONS: u32 = 100_000;
const NONCE_LEN: usize = 12;

/// PBKDF2-HMAC-SHA256(password, salt, 100_000, 32 bytes).
pub fn derive_key(password: &str, salt: &[u8]) -> [u8; KEY_LEN] {
    let mut key = [0u8; KEY_LEN];
    pbkdf2::pbkdf2_hmac::<sha2::Sha256>(password.as_bytes(), salt, PBKDF2_ITERATIONS, &mut key);
    key
}

pub fn generate_salt() -> [u8; SALT_LEN] {
    let mut salt = [0u8; SALT_LEN];
    OsRng.fill_bytes(&mut salt);
    salt
}

fn cipher(key: &[u8]) -> Result<Aes256Gcm> {
    if key.len() != KEY_LEN {
        return Err(AppError::msg("invalid key length"));
    }
    Ok(Aes256Gcm::new(Key::<Aes256Gcm>::from_slice(key)))
}

/// Returns `base64(STD)(nonce || ciphertext || tag)`.
pub fn encrypt(plain: &[u8], key: &[u8]) -> Result<String> {
    let cipher = cipher(key)?;
    let nonce = Aes256Gcm::generate_nonce(&mut OsRng);
    let sealed = cipher
        .encrypt(&nonce, plain)
        .map_err(|_| AppError::msg("encryption failed"))?;
    let mut out = Vec::with_capacity(NONCE_LEN + sealed.len());
    out.extend_from_slice(&nonce);
    out.extend_from_slice(&sealed);
    Ok(STANDARD.encode(out))
}

pub fn decrypt(b64: &str, key: &[u8]) -> Result<Vec<u8>> {
    let data = STANDARD
        .decode(b64)
        .map_err(|e| AppError::msg(format!("invalid base64: {e}")))?;
    let cipher = cipher(key)?;
    if data.len() < NONCE_LEN {
        return Err(AppError::msg("ciphertext too short"));
    }
    let (nonce, body) = data.split_at(NONCE_LEN);
    cipher
        .decrypt(Nonce::from_slice(nonce), body)
        .map_err(|_| AppError::msg("decryption failed"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::Value;

    const FIXTURES: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/../testdata/go-fixtures");

    fn read_json(rel: &str) -> Value {
        let text = std::fs::read_to_string(format!("{FIXTURES}/{rel}")).unwrap();
        serde_json::from_str(&text).unwrap()
    }

    fn fixture_salt() -> Vec<u8> {
        let cfg = read_json("notes/.gote_config.json");
        STANDARD.decode(cfg["salt"].as_str().unwrap()).unwrap()
    }

    #[test]
    fn decrypts_go_legacy_note() {
        let key = derive_key("testpass123", &fixture_salt());
        let note = read_json("notes/abcdef01.json");
        let plain = decrypt(note["encrypted_data"].as_str().unwrap(), &key).unwrap();
        assert_eq!(plain, b"legacy plain content");
    }

    #[test]
    fn verification_hash_matches_go() {
        let hash = derive_key("testpass123verification", &fixture_salt());
        let local = read_json("local-password-hash.json");
        assert_eq!(STANDARD.encode(hash), local["hash"].as_str().unwrap());
    }

    #[test]
    fn roundtrip() {
        let key = derive_key("pw", &generate_salt());
        let a = encrypt(b"x\0y", &key).unwrap();
        let b = encrypt(b"x\0y", &key).unwrap();
        assert_ne!(a, b);
        assert_eq!(decrypt(&a, &key).unwrap(), b"x\0y");
    }

    #[test]
    fn wrong_key_fails() {
        let salt = generate_salt();
        let sealed = encrypt(b"secret", &derive_key("a", &salt)).unwrap();
        assert!(decrypt(&sealed, &derive_key("b", &salt)).is_err());
    }

    #[test]
    fn short_ciphertext_fails() {
        let key = derive_key("a", &generate_salt());
        assert!(decrypt("AAAA", &key).is_err());
    }
}
