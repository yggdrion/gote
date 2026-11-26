use std::collections::HashMap;
use std::fs;
use std::io::Write;
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};

use base64::Engine;
use base64::engine::general_purpose::{STANDARD, URL_SAFE};
use chrono::{Local, SecondsFormat};
use rand::RngCore;
use serde::{Deserialize, Serialize};

use crate::crypto;
use crate::error::{AppError, Result};
use crate::ids::is_note_filename;

pub const SESSION_TIMEOUT: Duration = Duration::from_secs(30 * 60);
const CROSS_PLATFORM_FILE: &str = ".gote_config.json";

/// Local password file: `{"hash":"<b64>","salt":"<b64>"}`.
#[derive(Debug, Serialize, Deserialize)]
struct PasswordData {
    hash: String,
    salt: String,
}

/// `<notesDir>/.gote_config.json`, shared through the synced notes folder.
#[derive(Debug, Serialize, Deserialize)]
struct CrossPlatformConfig {
    salt: String,
    #[serde(rename = "createdAt", default)]
    created_at: String,
    #[serde(default)]
    version: String,
}

/// Only the field of a note file that matters for password verification.
#[derive(Deserialize)]
struct EncryptedField {
    #[serde(default)]
    encrypted_data: String,
}

pub struct AuthManager {
    password_hash_path: PathBuf,
    notes_dir: PathBuf,
    current_salt: Option<Vec<u8>>,
    sessions: HashMap<String, Instant>,
    pub session_timeout: Duration,
}

fn write_private(path: &Path, data: &[u8]) -> std::io::Result<()> {
    let mut opts = fs::OpenOptions::new();
    opts.write(true).create(true).truncate(true);
    #[cfg(unix)]
    {
        use std::os::unix::fs::OpenOptionsExt;
        opts.mode(0o600);
    }
    opts.open(path)?.write_all(data)
}

fn constant_time_eq(a: &[u8], b: &[u8]) -> bool {
    a.len() == b.len() && a.iter().zip(b).fold(0u8, |acc, (x, y)| acc | (x ^ y)) == 0
}

impl AuthManager {
    pub fn new(password_hash_path: PathBuf, notes_dir: PathBuf) -> Self {
        AuthManager {
            password_hash_path,
            notes_dir,
            current_salt: None,
            sessions: HashMap::new(),
            session_timeout: SESSION_TIMEOUT,
        }
    }

    fn cross_platform_path(&self) -> PathBuf {
        self.notes_dir.join(CROSS_PLATFORM_FILE)
    }

    /// True only if neither the local hash file nor the cross-platform config exists.
    pub fn is_first_time_setup(&self) -> bool {
        !self.password_hash_path.exists() && !self.cross_platform_path().exists()
    }

    fn load_cross_platform_salt(&self) -> Result<Vec<u8>> {
        let data = fs::read_to_string(self.cross_platform_path())?;
        let cfg: CrossPlatformConfig = serde_json::from_str(&data)?;
        STANDARD
            .decode(cfg.salt)
            .map_err(|e| AppError::msg(format!("failed to decode salt: {e}")))
    }

    fn save_cross_platform_salt(&self, salt: &[u8]) -> Result<()> {
        let cfg = CrossPlatformConfig {
            salt: STANDARD.encode(salt),
            created_at: Local::now().to_rfc3339_opts(SecondsFormat::Secs, false),
            version: "1.0".to_string(),
        };
        let data = serde_json::to_string_pretty(&cfg)?;
        write_private(&self.cross_platform_path(), data.as_bytes())?;
        Ok(())
    }

    fn load_local_salt(&self) -> Result<Vec<u8>> {
        let data = fs::read_to_string(&self.password_hash_path)
            .map_err(|e| AppError::msg(format!("failed to read password file: {e}")))?;
        let pd: PasswordData = serde_json::from_str(&data)
            .map_err(|e| AppError::msg(format!("failed to parse password data: {e}")))?;
        STANDARD
            .decode(pd.salt)
            .map_err(|e| AppError::msg(format!("failed to decode salt: {e}")))
    }

    fn write_local_hash(&self, password: &str, salt: &[u8]) -> Result<()> {
        let verification = crypto::derive_key(&format!("{password}verification"), salt);
        let pd = PasswordData {
            hash: STANDARD.encode(verification),
            salt: STANDARD.encode(salt),
        };
        if let Some(parent) = self.password_hash_path.parent() {
            fs::create_dir_all(parent)?;
        }
        write_private(&self.password_hash_path, &serde_json::to_vec(&pd)?)?;
        Ok(())
    }

    /// Stores the verification hash locally. Reuses the synced salt when the
    /// cross-platform config exists, otherwise generates a fresh salt and writes both.
    pub fn store_password_hash(&mut self, password: &str) -> Result<()> {
        if let Ok(salt) = self.load_cross_platform_salt() {
            self.current_salt = Some(salt.clone());
            return self.write_local_hash(password, &salt);
        }
        let salt = crypto::generate_salt().to_vec();
        self.current_salt = Some(salt.clone());
        self.write_local_hash(password, &salt)?;
        if let Err(e) = self.save_cross_platform_salt(&salt) {
            log::warn!("could not save cross-platform config: {e}");
        }
        Ok(())
    }

    pub fn verify_password(&mut self, password: &str) -> bool {
        if self.is_first_time_setup() {
            return false;
        }

        // A usable local hash that rejects the password is definitive when there
        // are no notes to check against (otherwise any password would pass the
        // "empty synced folder" rule below).
        let mut local_hash_usable = false;
        if let Ok(data) = fs::read_to_string(&self.password_hash_path)
            && let Ok(pd) = serde_json::from_str::<PasswordData>(&data)
            && let Ok(salt) = STANDARD.decode(&pd.salt)
        {
            let computed = STANDARD.encode(crypto::derive_key(
                &format!("{password}verification"),
                &salt,
            ));
            self.current_salt = Some(salt);
            if constant_time_eq(computed.as_bytes(), pd.hash.as_bytes()) {
                return true;
            }
            local_hash_usable = true;
        }

        // Second-machine bootstrap: verify against the synced notes.
        if let Ok(salt) = self.load_cross_platform_salt() {
            self.current_salt = Some(salt.clone());
            if self.verify_with_cross_platform_data(password, &salt, !local_hash_usable) {
                if let Err(e) = self.write_local_hash(password, &salt) {
                    log::warn!("could not create local password hash: {e}");
                }
                return true;
            }
        }
        false
    }

    fn verify_with_cross_platform_data(
        &self,
        password: &str,
        salt: &[u8],
        accept_if_no_notes: bool,
    ) -> bool {
        let key = crypto::derive_key(password, salt);
        let Ok(entries) = fs::read_dir(&self.notes_dir) else {
            return false;
        };
        let mut json_files = 0;
        for entry in entries.flatten() {
            let name = entry.file_name().to_string_lossy().into_owned();
            let is_file = entry.file_type().is_ok_and(|t| t.is_file());
            if !is_file || !name.ends_with(".json") || name == CROSS_PLATFORM_FILE {
                continue;
            }
            json_files += 1;
            let Ok(text) = fs::read_to_string(entry.path()) else {
                continue;
            };
            // Only real note files can prove the password.
            if !is_note_filename(&name) {
                continue;
            }
            let Ok(field) = serde_json::from_str::<EncryptedField>(&text) else {
                continue;
            };
            if crypto::decrypt(&field.encrypted_data, &key).is_ok() {
                return true;
            }
        }
        // JSON files exist but none decrypted: wrong password. With no JSON files
        // but a config present this is a fresh synced setup, so accept.
        accept_if_no_notes && json_files == 0 && self.cross_platform_path().exists()
    }

    pub fn derive_encryption_key(&mut self, password: &str) -> Result<[u8; 32]> {
        if self.current_salt.is_none() {
            if let Ok(salt) = self.load_cross_platform_salt() {
                self.current_salt = Some(salt);
            } else {
                if self.is_first_time_setup() {
                    return Err(AppError::msg("no password set up"));
                }
                let salt = self.load_local_salt()?;
                if !self.cross_platform_path().exists()
                    && let Err(e) = self.save_cross_platform_salt(&salt)
                {
                    log::warn!("could not create cross-platform config: {e}");
                }
                self.current_salt = Some(salt);
            }
        }
        let salt = self.current_salt.as_deref().expect("salt set above");
        Ok(crypto::derive_key(password, salt))
    }

    /// Removes the local password hash file; a missing file is fine.
    pub fn remove_password_hash(&self) -> Result<()> {
        match fs::remove_file(&self.password_hash_path) {
            Ok(()) => Ok(()),
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
            Err(e) => Err(e.into()),
        }
    }

    pub fn create_session(&mut self) -> String {
        let mut bytes = [0u8; 32];
        rand::rngs::OsRng.fill_bytes(&mut bytes);
        let id = URL_SAFE.encode(bytes);
        self.sessions
            .insert(id.clone(), Instant::now() + self.session_timeout);
        id
    }

    /// Validates a session and slides its expiry; expired sessions are removed.
    pub fn validate_session(&mut self, id: &str) -> bool {
        match self.sessions.get(id).copied() {
            None => false,
            Some(expires) if Instant::now() > expires => {
                self.sessions.remove(id);
                false
            }
            Some(_) => {
                self.sessions
                    .insert(id.to_string(), Instant::now() + self.session_timeout);
                true
            }
        }
    }

    pub fn delete_session(&mut self, id: &str) {
        self.sessions.remove(id);
    }

    pub fn cleanup_expired(&mut self) {
        let now = Instant::now();
        self.sessions.retain(|_, expires| now <= *expires);
    }

    #[cfg(test)]
    fn has_session(&self, id: &str) -> bool {
        self.sessions.contains_key(id)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::thread::sleep;

    const FIXTURES: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/../testdata/go-fixtures");

    fn copy_dir(from: &Path, to: &Path) {
        fs::create_dir_all(to).unwrap();
        for e in fs::read_dir(from).unwrap().flatten() {
            let dest = to.join(e.file_name());
            if e.file_type().unwrap().is_dir() {
                copy_dir(&e.path(), &dest);
            } else {
                fs::copy(e.path(), dest).unwrap();
            }
        }
    }

    fn fixture_copy(tmp: &Path) -> PathBuf {
        let notes = tmp.join("notes");
        copy_dir(&Path::new(FIXTURES).join("notes"), &notes);
        notes
    }

    fn read_json(p: &Path) -> serde_json::Value {
        serde_json::from_str(&fs::read_to_string(p).unwrap()).unwrap()
    }

    #[test]
    fn first_time_setup_then_verify() {
        let tmp = tempfile::tempdir().unwrap();
        let hash = tmp.path().join("cfg").join("password_hash");
        let notes = tmp.path().join("notes");
        fs::create_dir_all(&notes).unwrap();
        let mut auth = AuthManager::new(hash.clone(), notes.clone());
        assert!(auth.is_first_time_setup());
        assert!(!auth.verify_password("secret1"));

        auth.store_password_hash("secret1").unwrap();
        assert!(!auth.is_first_time_setup());
        assert!(hash.exists());
        let cfg = read_json(&notes.join(".gote_config.json"));
        assert_eq!(cfg["version"], "1.0");
        assert_eq!(cfg["salt"], read_json(&hash)["salt"]);
        assert!(cfg["createdAt"].as_str().unwrap().len() >= 20);

        assert!(auth.verify_password("secret1"));
        assert!(!auth.verify_password("nope"));

        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            for p in [&hash, &notes.join(".gote_config.json")] {
                assert_eq!(fs::metadata(p).unwrap().permissions().mode() & 0o777, 0o600);
            }
        }
    }

    #[test]
    fn verifies_go_written_hash() {
        let tmp = tempfile::tempdir().unwrap();
        let notes = fixture_copy(tmp.path());
        let hash = tmp.path().join("password_hash");
        fs::copy(Path::new(FIXTURES).join("local-password-hash.json"), &hash).unwrap();
        let mut auth = AuthManager::new(hash, notes.clone());
        assert!(auth.verify_password("testpass123"));
        assert!(!auth.verify_password("wrong"));
        assert!(auth.verify_password("testpass123"));

        let key = auth.derive_encryption_key("testpass123").unwrap();
        let note = read_json(&notes.join("abcdef01.json"));
        let plain = crypto::decrypt(note["encrypted_data"].as_str().unwrap(), &key).unwrap();
        assert_eq!(plain, b"legacy plain content");
    }

    #[test]
    fn bootstrap_from_synced_folder_accepts_right_password() {
        let tmp = tempfile::tempdir().unwrap();
        let notes = fixture_copy(tmp.path());
        let hash = tmp.path().join("fresh").join("password_hash");
        let mut auth = AuthManager::new(hash.clone(), notes.clone());
        assert!(!auth.is_first_time_setup());
        assert!(auth.verify_password("testpass123"));
        assert!(hash.exists());
        let cfg = read_json(&notes.join(".gote_config.json"));
        assert_eq!(read_json(&hash)["salt"], cfg["salt"]);
        // The created local hash is a valid verification hash for later logins.
        let mut again = AuthManager::new(hash, notes);
        assert!(again.verify_password("testpass123"));
    }

    #[test]
    fn bootstrap_from_synced_folder_rejects_wrong_password() {
        let tmp = tempfile::tempdir().unwrap();
        let notes = fixture_copy(tmp.path());
        let hash = tmp.path().join("fresh").join("password_hash");
        let mut auth = AuthManager::new(hash.clone(), notes);
        assert!(!auth.verify_password("wrong"));
        assert!(!hash.exists());
    }

    #[test]
    fn bootstrap_with_config_but_no_notes_accepts() {
        let tmp = tempfile::tempdir().unwrap();
        let notes = tmp.path().join("notes");
        fs::create_dir_all(&notes).unwrap();
        let mut other = AuthManager::new(tmp.path().join("other_hash"), notes.clone());
        other.store_password_hash("secret1").unwrap();

        let hash = tmp.path().join("fresh_hash");
        let mut auth = AuthManager::new(hash.clone(), notes);
        assert!(auth.verify_password("anything-goes"));
        assert!(hash.exists());
    }

    #[test]
    fn store_password_hash_reuses_synced_salt() {
        let tmp = tempfile::tempdir().unwrap();
        let notes = fixture_copy(tmp.path());
        let hash = tmp.path().join("h").join("password_hash");
        let mut auth = AuthManager::new(hash.clone(), notes.clone());
        auth.store_password_hash("testpass123").unwrap();
        assert_eq!(
            read_json(&hash)["salt"],
            read_json(&notes.join(".gote_config.json"))["salt"]
        );
    }

    #[test]
    fn derive_key_without_setup_errors() {
        let tmp = tempfile::tempdir().unwrap();
        let mut auth = AuthManager::new(tmp.path().join("h"), tmp.path().join("n"));
        assert_eq!(
            auth.derive_encryption_key("x").unwrap_err().message(),
            "no password set up"
        );
    }

    #[test]
    fn remove_password_hash_tolerates_missing() {
        let tmp = tempfile::tempdir().unwrap();
        let hash = tmp.path().join("h");
        let auth = AuthManager::new(hash.clone(), tmp.path().join("n"));
        auth.remove_password_hash().unwrap();
        fs::write(&hash, "x").unwrap();
        auth.remove_password_hash().unwrap();
        assert!(!hash.exists());
    }

    #[test]
    fn session_expiry() {
        let tmp = tempfile::tempdir().unwrap();
        let mut auth = AuthManager::new(tmp.path().join("h"), tmp.path().join("n"));
        auth.session_timeout = Duration::from_millis(50);
        let id = auth.create_session();
        assert!(auth.validate_session(&id));
        sleep(Duration::from_millis(100));
        assert!(!auth.validate_session(&id));
        assert!(!auth.has_session(&id));
    }

    #[test]
    fn session_slides_and_cleanup() {
        let tmp = tempfile::tempdir().unwrap();
        let mut auth = AuthManager::new(tmp.path().join("h"), tmp.path().join("n"));
        auth.session_timeout = Duration::from_millis(150);
        let keep = auth.create_session();
        let drop_me = auth.create_session();
        sleep(Duration::from_millis(100));
        assert!(auth.validate_session(&keep));
        sleep(Duration::from_millis(100));
        auth.cleanup_expired();
        assert!(auth.has_session(&keep));
        assert!(!auth.has_session(&drop_me));
        auth.delete_session(&keep);
        assert!(!auth.validate_session(&keep));
    }
}
