//! Plain-Rust facade mirroring the Go `App` semantics. No Tauri dependency, so it
//! is fully testable with `cargo test`.

use std::collections::HashSet;
use std::fs;
use std::path::PathBuf;
use std::sync::{Arc, Mutex};
use std::time::SystemTime;

use base64::Engine;
use base64::engine::general_purpose::STANDARD;
use chrono::{DateTime, Utc};
use serde::Serialize;

use crate::auth::AuthManager;
use crate::backup;
use crate::config::{self, Config};
use crate::error::{AppError, Result, SESSION_EXPIRED};
use crate::ids::is_valid_id;
use crate::image_store::{ImageStore, extract_image_ids};
use crate::models::{Category, Image, NoteDto};
use crate::note_store::NoteStore;

#[derive(Debug, Serialize)]
pub struct Settings {
    #[serde(rename = "notesPath")]
    pub notes_path: String,
    #[serde(rename = "passwordHashPath")]
    pub password_hash_path: String,
}

#[derive(Debug, Serialize)]
pub struct ImageData {
    pub id: String,
    pub filename: String,
    pub content_type: String,
    pub size: i64,
    pub created_at: DateTime<Utc>,
    pub data: String,
}

#[derive(Debug, Serialize)]
pub struct ImageStats {
    pub total_images: usize,
    pub referenced_images: usize,
    pub orphaned_images: usize,
    pub total_references: usize,
    pub total_size_bytes: i64,
    pub total_size_mb: f64,
}

/// Everything needed to run a backup without holding the `AppCore` mutex.
#[derive(Clone)]
pub struct BackupJob {
    notes_path: PathBuf,
    lock: Arc<Mutex<()>>,
}

impl BackupJob {
    /// Creates a backup now. Manual and scheduled backups share one mutex.
    pub fn run(&self) -> Result<String> {
        let _guard = self.lock.lock().unwrap_or_else(|e| e.into_inner());
        let path = backup::backup_notes(&self.notes_path)
            .map_err(|e| AppError::msg(format!("failed to create backup: {e}")))?;
        Ok(path.to_string_lossy().into_owned())
    }

    /// Creates a backup if none exists or the newest is at least 24h old.
    pub fn run_if_due(&self) {
        let _guard = self.lock.lock().unwrap_or_else(|e| e.into_inner());
        if !backup::backup_is_due(&self.notes_path, SystemTime::now()) {
            return;
        }
        match backup::backup_notes(&self.notes_path) {
            Ok(p) => log::info!("auto-backup: created {}", p.display()),
            Err(e) => log::warn!("auto-backup failed: {e}"),
        }
    }
}

pub struct AppCore {
    pub config_path: PathBuf,
    pub config: Config,
    pub configured: bool,
    pub auth: Option<AuthManager>,
    pub store: Option<NoteStore>,
    pub images: Option<ImageStore>,
    key: Option<[u8; 32]>,
    session: Option<String>,
    pub backup_lock: Arc<Mutex<()>>,
}

fn not_initialized() -> AppError {
    AppError::msg("application is not set up")
}

fn require_id(id: &str) -> Result<()> {
    if id.trim().is_empty() {
        Err(AppError::msg("note ID cannot be empty"))
    } else {
        Ok(())
    }
}

impl AppCore {
    pub fn startup(config_path: PathBuf) -> AppCore {
        let configured = config_path.exists();
        let config = if configured {
            Config::load_from(&config_path).unwrap_or_else(|e| {
                log::warn!("failed to load configuration, using defaults: {e}");
                Config::defaults()
            })
        } else {
            log::info!("first-time setup required - no configuration file found");
            Config::defaults()
        };
        let mut core = AppCore {
            config_path,
            config,
            configured,
            auth: None,
            store: None,
            images: None,
            key: None,
            session: None,
            backup_lock: Arc::new(Mutex::new(())),
        };
        if configured {
            core.build_components();
            log::info!("notes directory: {}", core.config.notes_path.display());
        }
        core
    }

    /// (Re)creates the auth manager, note store and image store for the current config.
    fn build_components(&mut self) {
        if let Some(old) = self.store.take() {
            old.clear();
        }
        self.auth = Some(AuthManager::new(
            self.config.password_hash_path.clone(),
            self.config.notes_path.clone(),
        ));
        self.store = match NoteStore::new(self.config.notes_path.clone()) {
            Ok(s) => Some(s),
            Err(e) => {
                log::error!("failed to open notes directory: {e}");
                None
            }
        };
        self.images = Some(ImageStore::new(&self.config.notes_path));
    }

    /// Drops the key, the session and all decrypted data.
    fn clear_auth_state(&mut self) {
        if let (Some(auth), Some(id)) = (self.auth.as_mut(), self.session.take()) {
            auth.delete_session(&id);
        }
        self.session = None;
        self.key = None;
        if let Some(store) = &self.store {
            store.clear();
        }
        if let Some(images) = &self.images {
            images.set_key(None);
        }
    }

    fn start_session(&mut self, key: [u8; 32]) -> Result<()> {
        let auth = self.auth.as_mut().ok_or_else(not_initialized)?;
        if let Some(old) = self.session.take() {
            auth.delete_session(&old);
        }
        self.session = Some(auth.create_session());
        self.key = Some(key);
        if let Some(images) = &self.images {
            images.set_key(Some(key));
        }
        if let Some(store) = &self.store
            && let Err(e) = store.load_notes(key)
        {
            log::warn!("failed to load notes: {e}");
        }
        Ok(())
    }

    /// Valid session check that slides the expiry; clears all state when expired.
    fn require_auth(&mut self) -> Result<()> {
        if self.check_session() {
            Ok(())
        } else {
            Err(AppError::msg(SESSION_EXPIRED))
        }
    }

    fn check_session(&mut self) -> bool {
        let valid = match (self.auth.as_mut(), self.session.as_deref()) {
            (Some(auth), Some(id)) => auth.validate_session(id) && self.key.is_some(),
            _ => false,
        };
        if !valid && (self.session.is_some() || self.key.is_some()) {
            self.clear_auth_state();
        }
        valid
    }

    fn store(&self) -> Result<&NoteStore> {
        self.store.as_ref().ok_or_else(not_initialized)
    }

    fn images(&self) -> Result<&ImageStore> {
        self.images.as_ref().ok_or_else(not_initialized)
    }

    // ---- setup & authentication -------------------------------------------------

    pub fn is_configured(&self) -> bool {
        self.config_path.exists()
    }

    pub fn is_password_set(&self) -> bool {
        self.auth.as_ref().is_some_and(|a| !a.is_first_time_setup())
    }

    pub fn complete_initial_setup(
        &mut self,
        notes_path: &str,
        password_hash_path: &str,
        password: &str,
        confirm_password: &str,
    ) -> Result<()> {
        if password.chars().count() < 6 {
            return Err(AppError::msg("password must be at least 6 characters long"));
        }
        if password != confirm_password {
            return Err(AppError::msg("passwords do not match"));
        }
        let notes_path = if notes_path.is_empty() {
            config::default_notes_path()
        } else {
            PathBuf::from(notes_path)
        };
        let hash_path = if password_hash_path.is_empty() {
            config::default_password_hash_path()
        } else {
            PathBuf::from(password_hash_path)
        };
        fs::create_dir_all(&notes_path)
            .map_err(|e| AppError::msg(format!("failed to create notes directory: {e}")))?;
        if let Some(parent) = hash_path.parent() {
            fs::create_dir_all(parent)
                .map_err(|e| AppError::msg(format!("failed to create password directory: {e}")))?;
        }

        self.config = Config {
            notes_path,
            password_hash_path: hash_path,
        };
        self.config
            .save_to(&self.config_path)
            .map_err(|e| AppError::msg(format!("failed to save configuration: {e}")))?;
        self.configured = true;
        self.clear_auth_state();
        self.build_components();

        let auth = self.auth.as_mut().ok_or_else(not_initialized)?;
        auth.store_password_hash(password)
            .map_err(|e| AppError::msg(format!("failed to store password: {e}")))?;
        let key = auth
            .derive_encryption_key(password)
            .map_err(|e| AppError::msg(format!("failed to derive encryption key: {e}")))?;
        // Unlike the Go version this also creates a session, so the first save
        // right after setup passes `require_auth`.
        self.start_session(key)
    }

    pub fn set_password(&mut self, password: &str) -> Result<()> {
        let auth = self.auth.as_mut().ok_or_else(not_initialized)?;
        auth.store_password_hash(password)
            .map_err(|e| AppError::msg(format!("failed to store password: {e}")))?;
        let key = auth
            .derive_encryption_key(password)
            .map_err(|e| AppError::msg(format!("failed to derive encryption key: {e}")))?;
        self.start_session(key)
    }

    pub fn verify_password(&mut self, password: &str) -> bool {
        let Some(auth) = self.auth.as_mut() else {
            return false;
        };
        if !auth.verify_password(password) {
            return false;
        }
        let key = match auth.derive_encryption_key(password) {
            Ok(k) => k,
            Err(e) => {
                log::warn!("failed to derive encryption key: {e}");
                return false;
            }
        };
        self.start_session(key).is_ok()
    }

    pub fn is_authenticated(&mut self) -> bool {
        self.check_session()
    }

    pub fn logout(&mut self) {
        self.clear_auth_state();
    }

    /// Logs out and removes the local password hash (notes are kept).
    pub fn reset_application(&mut self) -> Result<()> {
        self.clear_auth_state();
        if let Some(auth) = &self.auth {
            auth.remove_password_hash()
                .map_err(|e| AppError::msg(format!("failed to remove password hash: {e}")))?;
        }
        Ok(())
    }

    pub fn cleanup_sessions(&mut self) {
        if let Some(auth) = self.auth.as_mut() {
            auth.cleanup_expired();
        }
        // If our own session just expired, drop the key and decrypted notes too.
        let alive = match (self.auth.as_ref(), self.session.as_deref()) {
            (Some(auth), Some(id)) => auth.has_session(id),
            _ => false,
        };
        if !alive && (self.session.is_some() || self.key.is_some()) {
            self.clear_auth_state();
        }
    }

    // ---- notes -------------------------------------------------------------------

    pub fn get_all_notes(&mut self) -> Vec<NoteDto> {
        if !self.check_session() {
            return Vec::new();
        }
        self.store
            .as_ref()
            .map(|s| s.all_notes().iter().map(NoteDto::from).collect())
            .unwrap_or_default()
    }

    pub fn get_notes_by_category(&mut self, category: &str) -> Vec<NoteDto> {
        let Some(category) = Category::parse(category) else {
            return Vec::new();
        };
        if !self.check_session() {
            return Vec::new();
        }
        self.store
            .as_ref()
            .map(|s| {
                s.notes_by_category(category)
                    .iter()
                    .map(NoteDto::from)
                    .collect()
            })
            .unwrap_or_default()
    }

    pub fn search_notes(&mut self, query: &str) -> Vec<NoteDto> {
        if !self.check_session() {
            return Vec::new();
        }
        self.store
            .as_ref()
            .map(|s| s.search(query).iter().map(NoteDto::from).collect())
            .unwrap_or_default()
    }

    pub fn get_note(&mut self, id: &str) -> Result<NoteDto> {
        self.require_auth()?;
        require_id(id)?;
        Ok(NoteDto::from(&self.store()?.get_note(id)?))
    }

    pub fn create_note(&mut self, content: &str) -> Result<NoteDto> {
        self.create_note_with_category(content, "private")
    }

    pub fn create_note_with_category(&mut self, content: &str, category: &str) -> Result<NoteDto> {
        self.require_auth()?;
        let category = Category::parse(category).unwrap_or(Category::Private);
        Ok(NoteDto::from(
            &self.store()?.create_note(content, category)?,
        ))
    }

    pub fn update_note(&mut self, id: &str, content: &str) -> Result<NoteDto> {
        self.require_auth()?;
        require_id(id)?;
        if content.trim().is_empty() {
            return Err(AppError::msg("note content cannot be empty"));
        }
        Ok(NoteDto::from(&self.store()?.update_note(id, content)?))
    }

    pub fn update_note_category(&mut self, id: &str, category: &str) -> Result<NoteDto> {
        self.require_auth()?;
        let parsed = Category::parse(category)
            .ok_or_else(|| AppError::msg(format!("invalid category: {category}")))?;
        require_id(id)?;
        Ok(NoteDto::from(&self.store()?.update_category(id, parsed)?))
    }

    pub fn move_to_trash(&mut self, id: &str) -> Result<NoteDto> {
        self.require_auth()?;
        require_id(id)?;
        Ok(NoteDto::from(&self.store()?.move_to_trash(id)?))
    }

    pub fn restore_from_trash(&mut self, id: &str) -> Result<NoteDto> {
        self.require_auth()?;
        require_id(id)?;
        Ok(NoteDto::from(&self.store()?.restore_from_trash(id)?))
    }

    /// Trash notes are deleted permanently (with orphan-image cleanup); others go to trash.
    pub fn delete_note(&mut self, id: &str) -> Result<()> {
        self.require_auth()?;
        require_id(id)?;
        let note = self.store()?.get_note(id)?;
        if note.category == Category::Trash {
            self.permanently_delete_inner(id)
        } else {
            self.store()?.move_to_trash(id).map(|_| ())
        }
    }

    pub fn permanently_delete_note(&mut self, id: &str) -> Result<()> {
        self.require_auth()?;
        require_id(id)?;
        self.permanently_delete_inner(id)
    }

    fn permanently_delete_inner(&self, id: &str) -> Result<()> {
        let store = self.store()?;
        let content = store.get_note(id).map(|n| n.content).unwrap_or_default();
        store.permanently_delete(id)?;
        let others: HashSet<String> = store
            .all_notes()
            .iter()
            .flat_map(|n| extract_image_ids(&n.content))
            .collect();
        for image_id in extract_image_ids(&content) {
            if others.contains(&image_id) || !is_valid_id(&image_id) {
                continue;
            }
            match self.images()?.delete_image(&image_id) {
                Ok(()) => log::info!("cleaned up orphaned image {image_id}"),
                Err(e) => log::warn!("failed to delete orphaned image {image_id}: {e}"),
            }
        }
        Ok(())
    }

    // ---- settings ----------------------------------------------------------------

    pub fn get_settings(&self) -> Settings {
        Settings {
            notes_path: self.config.notes_path.to_string_lossy().into_owned(),
            password_hash_path: self
                .config
                .password_hash_path
                .to_string_lossy()
                .into_owned(),
        }
    }

    /// Saves new paths, logs out and rebuilds the components for them.
    pub fn update_settings(&mut self, notes_path: &str, password_hash_path: &str) -> Result<()> {
        let notes_path = if notes_path.is_empty() {
            config::default_notes_path()
        } else {
            PathBuf::from(notes_path)
        };
        let hash_path = if password_hash_path.is_empty() {
            config::default_password_hash_path()
        } else {
            PathBuf::from(password_hash_path)
        };
        fs::create_dir_all(&notes_path)
            .map_err(|e| AppError::msg(format!("failed to create notes directory: {e}")))?;
        if let Some(parent) = hash_path.parent() {
            fs::create_dir_all(parent).map_err(|e| {
                AppError::msg(format!("failed to create password hash directory: {e}"))
            })?;
        }
        let new_config = Config {
            notes_path,
            password_hash_path: hash_path,
        };
        new_config
            .save_to(&self.config_path)
            .map_err(|e| AppError::msg(format!("failed to save configuration: {e}")))?;

        self.clear_auth_state();
        self.config = new_config;
        self.configured = true;
        self.build_components();
        Ok(())
    }

    // ---- backups -----------------------------------------------------------------

    /// Authenticated manual backup. Run the returned job after releasing any
    /// shared lock on the `AppCore`.
    pub fn begin_backup(&mut self) -> Result<BackupJob> {
        self.require_auth()?;
        Ok(self.backup_job())
    }

    pub fn backup_job(&self) -> BackupJob {
        BackupJob {
            notes_path: self.config.notes_path.clone(),
            lock: Arc::clone(&self.backup_lock),
        }
    }

    pub fn create_backup(&mut self) -> Result<String> {
        self.begin_backup()?.run()
    }

    /// For the scheduler: no auth needed (the archive contains ciphertext only).
    pub fn run_backup_if_due(&self) {
        if self.configured {
            self.backup_job().run_if_due();
        }
    }

    // ---- images ------------------------------------------------------------------

    pub fn save_image_from_clipboard(
        &mut self,
        image_data: &str,
        content_type: &str,
    ) -> Result<Image> {
        self.require_auth()?;
        let data = STANDARD
            .decode(image_data)
            .map_err(|e| AppError::msg(format!("invalid image data: {e}")))?;
        let ext = match content_type {
            "image/png" => ".png",
            "image/jpeg" => ".jpg",
            _ => ".img",
        };
        let filename = format!("clipboard_{}{ext}", Utc::now().timestamp());
        self.images()?.store_image(&data, content_type, &filename)
    }

    pub fn get_image(&mut self, image_id: &str) -> Result<ImageData> {
        self.require_auth()?;
        let (data, meta) = self.images()?.get_image(image_id)?;
        Ok(ImageData {
            id: meta.id,
            filename: meta.filename,
            content_type: meta.content_type,
            size: meta.size,
            created_at: meta.created_at,
            data: STANDARD.encode(data),
        })
    }

    pub fn get_image_as_data_url(&mut self, image_id: &str) -> Result<String> {
        self.require_auth()?;
        let (data, meta) = self.images()?.get_image(image_id)?;
        Ok(format!(
            "data:{};base64,{}",
            meta.content_type,
            STANDARD.encode(data)
        ))
    }

    pub fn delete_image(&mut self, image_id: &str) -> Result<()> {
        self.require_auth()?;
        self.images()?.delete_image(image_id)
    }

    pub fn list_images(&mut self) -> Result<Vec<Image>> {
        self.require_auth()?;
        self.images()?.list_images()
    }

    /// Image ids referenced by every note, plus the total number of references.
    fn referenced_images(&self) -> Result<(HashSet<String>, usize)> {
        let mut set = HashSet::new();
        let mut total = 0;
        for note in self.store()?.all_notes() {
            let ids = extract_image_ids(&note.content);
            total += ids.len();
            set.extend(ids);
        }
        Ok((set, total))
    }

    pub fn cleanup_orphaned_images(&mut self) -> Result<usize> {
        self.require_auth()?;
        let all = self
            .images()?
            .list_images()
            .map_err(|e| AppError::msg(format!("failed to list images: {e}")))?;
        let (referenced, _) = self.referenced_images()?;
        let mut cleaned = 0;
        for image in all.iter().filter(|i| !referenced.contains(&i.id)) {
            match self.images()?.delete_image(&image.id) {
                Ok(()) => cleaned += 1,
                Err(e) => log::warn!("failed to delete orphaned image {}: {e}", image.id),
            }
        }
        Ok(cleaned)
    }

    pub fn get_image_stats(&mut self) -> Result<ImageStats> {
        self.require_auth()?;
        let all = self
            .images()?
            .list_images()
            .map_err(|e| AppError::msg(format!("failed to list images: {e}")))?;
        let (referenced, total_references) = self.referenced_images()?;
        let total_size: i64 = all.iter().map(|i| i.size).sum();
        let orphaned = all.iter().filter(|i| !referenced.contains(&i.id)).count();
        Ok(ImageStats {
            total_images: all.len(),
            referenced_images: referenced.len(),
            orphaned_images: orphaned,
            total_references,
            total_size_bytes: total_size,
            total_size_mb: total_size as f64 / (1024.0 * 1024.0),
        })
    }

    /// Releases the file watcher (called on exit).
    pub fn shutdown(&mut self) {
        if let Some(store) = &self.store {
            store.clear();
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::Path;
    use std::thread::sleep;
    use std::time::Duration;

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

    struct Env {
        tmp: tempfile::TempDir,
    }

    impl Env {
        fn new() -> Env {
            Env {
                tmp: tempfile::tempdir().unwrap(),
            }
        }
        fn config_path(&self) -> PathBuf {
            self.tmp.path().join("cfg").join("config")
        }
        fn notes(&self) -> PathBuf {
            self.tmp.path().join("notes")
        }
        fn hash(&self) -> PathBuf {
            self.tmp.path().join("hash").join("password_hash")
        }
        /// An unconfigured core that has completed setup with password `secret1`.
        fn set_up(&self) -> AppCore {
            let mut core = AppCore::startup(self.config_path());
            core.complete_initial_setup(
                self.notes().to_str().unwrap(),
                self.hash().to_str().unwrap(),
                "secret1",
                "secret1",
            )
            .unwrap();
            core
        }
        /// A configured core over a copy of the Go fixtures (not yet logged in).
        fn fixture_core(&self) -> AppCore {
            copy_dir(&Path::new(FIXTURES).join("notes"), &self.notes());
            fs::create_dir_all(self.hash().parent().unwrap()).unwrap();
            fs::copy(
                Path::new(FIXTURES).join("local-password-hash.json"),
                self.hash(),
            )
            .unwrap();
            Config {
                notes_path: self.notes(),
                password_hash_path: self.hash(),
            }
            .save_to(&self.config_path())
            .unwrap();
            AppCore::startup(self.config_path())
        }
    }

    #[test]
    fn setup_then_create_note() {
        let env = Env::new();
        let mut core = AppCore::startup(env.config_path());
        assert!(!core.is_configured());
        assert!(!core.is_password_set());
        core.complete_initial_setup(
            env.notes().to_str().unwrap(),
            env.hash().to_str().unwrap(),
            "secret1",
            "secret1",
        )
        .unwrap();
        assert!(core.is_configured());
        assert!(core.is_password_set());
        assert!(core.is_authenticated());
        let note = core.create_note("hi").unwrap();
        assert_eq!(note.content, "hi");
        assert_eq!(core.get_all_notes().len(), 1);
        assert_eq!(core.get_notes_by_category("private").len(), 1);
        assert!(core.get_notes_by_category("work").is_empty());
        assert!(core.get_notes_by_category("bogus").is_empty());
        assert_eq!(core.search_notes("HI").len(), 1);

        // A restart with the persisted config sees the configured app and logs in.
        let mut again = AppCore::startup(env.config_path());
        assert!(again.is_configured() && again.is_password_set());
        assert!(!again.is_authenticated());
        assert!(!again.verify_password("wrong"));
        assert!(again.verify_password("secret1"));
        assert_eq!(again.get_all_notes().len(), 1);
    }

    #[test]
    fn setup_validation_errors() {
        let env = Env::new();
        let mut core = AppCore::startup(env.config_path());
        let e = core
            .complete_initial_setup("", "", "abc", "abc")
            .unwrap_err()
            .to_string();
        assert!(e.contains("at least 6 characters"), "{e}");
        let e = core
            .complete_initial_setup("", "", "secret1", "secret2")
            .unwrap_err()
            .to_string();
        assert!(e.contains("passwords do not match"), "{e}");
        assert!(!core.is_configured());
    }

    #[test]
    fn logout_clears_notes() {
        let env = Env::new();
        let mut core = env.set_up();
        core.create_note("secret").unwrap();
        assert_eq!(core.get_all_notes().len(), 1);
        core.logout();
        assert!(core.get_all_notes().is_empty());
        assert!(core.search_notes("secret").is_empty());
        assert!(core.get_notes_by_category("private").is_empty());
        let e = core.create_note("x").unwrap_err().to_string();
        assert!(e.contains("session expired"), "{e}");
        assert!(core.get_note("deadbeef").is_err());
        assert!(!core.is_authenticated());
        // Logging back in restores access.
        assert!(core.verify_password("secret1"));
        assert_eq!(core.get_all_notes().len(), 1);
    }

    #[test]
    fn expired_session_rejects_mutation() {
        let env = Env::new();
        let mut core = env.set_up();
        let note = core.create_note("hello").unwrap();
        core.auth.as_mut().unwrap().session_timeout = Duration::from_millis(50);
        // A fresh login creates a session using the shortened timeout.
        core.logout();
        assert!(core.verify_password("secret1"));
        sleep(Duration::from_millis(100));
        let e = core
            .update_note(&note.id, "changed")
            .unwrap_err()
            .to_string();
        assert!(e.contains("session expired"), "{e}");
        assert!(core.get_all_notes().is_empty());
    }

    #[test]
    fn cleanup_sessions_drops_expired_key() {
        let env = Env::new();
        let mut core = env.set_up();
        core.auth.as_mut().unwrap().session_timeout = Duration::from_millis(50);
        core.logout();
        assert!(core.verify_password("secret1"));
        sleep(Duration::from_millis(100));
        core.cleanup_sessions();
        assert!(core.key.is_none() && core.session.is_none());
        assert!(core.store.as_ref().unwrap().all_notes().is_empty());
    }

    #[test]
    fn opens_go_fixture_folder() {
        let env = Env::new();
        let mut core = env.fixture_core();
        assert!(core.is_configured() && core.is_password_set());
        assert!(!core.verify_password("wrong"));
        assert!(core.verify_password("testpass123"));
        assert_eq!(core.get_all_notes().len(), 3);
        let url = core.get_image_as_data_url("0c93c8fc").unwrap();
        assert!(
            url.starts_with("data:image/png;base64,iVBORw0KGgo"),
            "{url}"
        );

        let img = core.get_image("0c93c8fc").unwrap();
        assert_eq!(img.size, 12);
        assert_eq!(img.filename, "clipboard_1700000000.png");
        let stats = core.get_image_stats().unwrap();
        assert_eq!(
            (
                stats.total_images,
                stats.referenced_images,
                stats.orphaned_images
            ),
            (1, 1, 0)
        );
        assert_eq!(stats.total_references, 1);
        assert_eq!(stats.total_size_bytes, 12);

        let work = core.get_notes_by_category("work");
        assert_eq!(work.len(), 1);
        assert_eq!(work[0].id, "3caaec0b");
        let trashed = core.get_note("1a90778d").unwrap();
        assert_eq!(trashed.category, "trash");
        assert_eq!(trashed.original_category.as_deref(), Some("private"));
    }

    #[test]
    fn second_machine_bootstrap_through_app() {
        let env = Env::new();
        let mut core = env.fixture_core();
        fs::remove_file(env.hash()).unwrap();
        assert!(core.is_password_set(), "synced config counts as set up");
        assert!(!core.verify_password("wrong"));
        assert!(!env.hash().exists());
        assert!(core.verify_password("testpass123"));
        assert!(env.hash().exists());
        assert_eq!(core.get_all_notes().len(), 3);
    }

    fn work_note_with_image() -> &'static str {
        "![shot](image:0c93c8fc)"
    }

    #[test]
    fn permanent_delete_cleans_orphan_images() {
        let env = Env::new();
        let mut core = env.fixture_core();
        assert!(core.verify_password("testpass123"));
        let image_file = env.notes().join("images/0c93c8fc.json");

        // Not allowed straight away: the note is not in trash.
        assert!(core.permanently_delete_note("3caaec0b").is_err());
        assert!(image_file.exists());

        core.move_to_trash("3caaec0b").unwrap();
        core.permanently_delete_note("3caaec0b").unwrap();
        assert!(!image_file.exists());
        assert!(core.get_note("3caaec0b").is_err());
    }

    #[test]
    fn permanent_delete_keeps_shared_images() {
        let env = Env::new();
        let mut core = env.fixture_core();
        assert!(core.verify_password("testpass123"));
        let image_file = env.notes().join("images/0c93c8fc.json");
        core.create_note(work_note_with_image()).unwrap();

        // delete_note on a normal note trashes; on a trashed note it deletes.
        core.delete_note("3caaec0b").unwrap();
        assert_eq!(core.get_note("3caaec0b").unwrap().category, "trash");
        core.delete_note("3caaec0b").unwrap();
        assert!(core.get_note("3caaec0b").is_err());
        assert!(image_file.exists(), "still referenced by the second note");
    }

    #[test]
    fn update_settings_logs_out() {
        let env = Env::new();
        let mut core = env.set_up();
        core.create_note("x").unwrap();
        let new_notes = env.tmp.path().join("other-notes");
        let new_hash = env.tmp.path().join("other-hash").join("password_hash");
        core.update_settings(new_notes.to_str().unwrap(), new_hash.to_str().unwrap())
            .unwrap();
        assert!(!core.is_authenticated());
        assert!(core.get_all_notes().is_empty());
        assert!(
            core.create_note("y")
                .unwrap_err()
                .to_string()
                .contains("session expired")
        );
        let s = core.get_settings();
        assert_eq!(s.notes_path, new_notes.to_str().unwrap());
        assert!(new_notes.is_dir());
        // The new location is persisted and first-time again.
        let cfg = Config::load_from(&env.config_path()).unwrap();
        assert_eq!(cfg.notes_path, new_notes);
        assert!(!core.is_password_set());
        let v = serde_json::to_value(core.get_settings()).unwrap();
        assert!(v.get("notesPath").is_some() && v.get("passwordHashPath").is_some());
    }

    #[test]
    fn validation_and_category_errors() {
        let env = Env::new();
        let mut core = env.set_up();
        let note = core.create_note("a").unwrap();
        assert_eq!(
            core.update_note(&note.id, "  \n").unwrap_err().to_string(),
            "note content cannot be empty"
        );
        assert_eq!(
            core.update_note("  ", "x").unwrap_err().to_string(),
            "note ID cannot be empty"
        );
        assert_eq!(
            core.update_note_category(&note.id, "nope")
                .unwrap_err()
                .to_string(),
            "invalid category: nope"
        );
        assert_eq!(
            core.update_note_category(&note.id, "work")
                .unwrap()
                .category,
            "work"
        );
        assert_eq!(
            core.create_note_with_category("b", "weird")
                .unwrap()
                .category,
            "private"
        );
        assert_eq!(
            core.update_note("../x", "c").unwrap_err().to_string(),
            "invalid id"
        );
    }

    #[test]
    fn image_commands_roundtrip() {
        let env = Env::new();
        let mut core = env.set_up();
        let b64 = STANDARD.encode(b"\x89PNG\r\n\x1a\nxyz");
        let img = core.save_image_from_clipboard(&b64, "image/png").unwrap();
        assert!(img.filename.starts_with("clipboard_") && img.filename.ends_with(".png"));
        let jpg = core.save_image_from_clipboard(&b64, "image/jpeg").unwrap();
        assert!(jpg.filename.ends_with(".jpg"));
        let other = core.save_image_from_clipboard(&b64, "image/gif").unwrap();
        assert!(other.filename.ends_with(".img"));
        assert!(
            core.save_image_from_clipboard("!!notb64", "image/png")
                .unwrap_err()
                .to_string()
                .starts_with("invalid image data")
        );
        assert_eq!(core.list_images().unwrap().len(), 3);
        // All three are orphans: no note references them.
        assert_eq!(core.cleanup_orphaned_images().unwrap(), 3);
        assert!(core.list_images().unwrap().is_empty());
        core.logout();
        assert!(
            core.list_images()
                .unwrap_err()
                .to_string()
                .contains("session expired")
        );
        assert!(core.get_image(&img.id).is_err());
    }

    #[test]
    fn create_backup_requires_auth_and_writes_zip() {
        let env = Env::new();
        let mut core = env.set_up();
        core.create_note("x").unwrap();
        let path = core.create_backup().unwrap();
        assert!(Path::new(&path).is_file());
        core.run_backup_if_due(); // a fresh backup exists: no-op, must not panic
        core.logout();
        assert!(
            core.create_backup()
                .unwrap_err()
                .to_string()
                .contains("session expired")
        );
    }

    #[test]
    fn reset_application_removes_hash_keeps_notes() {
        let env = Env::new();
        let mut core = env.set_up();
        let note = core.create_note("keep").unwrap();
        core.reset_application().unwrap();
        assert!(!env.hash().exists());
        assert!(env.notes().join(format!("{}.json", note.id)).exists());
        assert!(!core.is_authenticated());
    }
}
