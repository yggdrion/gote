use std::collections::HashMap;
use std::fs;
use std::panic::{AssertUnwindSafe, catch_unwind};
use std::path::{Path, PathBuf};
use std::sync::{Arc, RwLock, RwLockReadGuard, RwLockWriteGuard, Weak};
use std::time::SystemTime;

use chrono::Utc;
use notify::{Event, EventKind, RecommendedWatcher, RecursiveMode, Watcher};

use crate::crypto;
use crate::error::{AppError, Result};
use crate::ids::{ensure_valid_id, generate_short_id, is_note_filename};
use crate::models::{Category, EncryptedNote, Note, NotePayload};

struct Inner {
    dir: PathBuf,
    notes: HashMap<String, Note>,
    key: Option<[u8; 32]>,
    mod_times: HashMap<PathBuf, SystemTime>,
    watcher: Option<RecommendedWatcher>,
}

/// Encrypted note storage with an in-memory cache and a directory watcher.
/// Cloning is cheap: all clones share the same state.
#[derive(Clone)]
pub struct NoteStore {
    inner: Arc<RwLock<Inner>>,
}

fn sorted(mut notes: Vec<Note>) -> Vec<Note> {
    notes.sort_by(|a, b| {
        b.updated_at
            .cmp(&a.updated_at)
            .then_with(|| a.id.cmp(&b.id))
    });
    notes
}

impl Inner {
    fn note_path(&self, id: &str) -> PathBuf {
        self.dir.join(format!("{id}.json"))
    }

    fn require_key(&self) -> Result<[u8; 32]> {
        self.key.ok_or_else(|| AppError::msg("not authenticated"))
    }

    /// Reads, decrypts and decodes a note file. The id comes from the (validated)
    /// file name, never from the file content.
    fn read_note_file(&self, path: &Path, id: &str) -> Result<Note> {
        let key = self.require_key()?;
        let enc: EncryptedNote = serde_json::from_str(&fs::read_to_string(path)?)?;
        let plain = crypto::decrypt(&enc.encrypted_data, &key)?;
        let plain = String::from_utf8(plain)
            .map_err(|_| AppError::msg("decrypted note is not valid UTF-8"))?;
        let (content, category, original_category, images) = NotePayload::decode(&plain);
        Ok(Note {
            id: id.to_string(),
            content,
            category,
            original_category,
            images,
            created_at: enc.created_at,
            updated_at: enc.updated_at,
        })
    }

    /// Inserts the note if it is new or newer than the cached one.
    fn merge(&mut self, note: Note) {
        let newer = self
            .notes
            .get(&note.id)
            .is_none_or(|existing| note.updated_at > existing.updated_at);
        if newer {
            self.notes.insert(note.id.clone(), note);
        }
    }

    fn save_note(&mut self, note: &Note) -> Result<()> {
        let key = self.require_key()?;
        let enc = EncryptedNote {
            id: note.id.clone(),
            encrypted_data: crypto::encrypt(NotePayload::encode(note).as_bytes(), &key)?,
            created_at: note.created_at,
            updated_at: note.updated_at,
        };
        let json = serde_json::to_string_pretty(&enc)?;
        let path = self.note_path(&note.id);
        // Write-then-rename so sync clients never see a half-written note. The temp
        // name is not a note file name, so the watcher ignores it.
        let tmp = self.dir.join(format!(".{}.json.tmp", note.id));
        fs::write(&tmp, json)?;
        if let Err(e) = fs::rename(&tmp, &path) {
            let _ = fs::remove_file(&tmp);
            return Err(e.into());
        }
        if let Ok(mtime) = fs::metadata(&path).and_then(|m| m.modified()) {
            self.mod_times.insert(path, mtime);
        }
        Ok(())
    }

    fn sync_from_disk(&mut self) -> Result<()> {
        self.require_key()?;
        let entries = fs::read_dir(&self.dir)
            .map_err(|e| AppError::msg(format!("error reading data directory: {e}")))?;
        let mut on_disk = Vec::new();
        for entry in entries.flatten() {
            let name = entry.file_name().to_string_lossy().into_owned();
            if !is_note_filename(&name) || !entry.file_type().is_ok_and(|t| t.is_file()) {
                continue;
            }
            let id = name.trim_end_matches(".json").to_string();
            let path = self.dir.join(&name);
            on_disk.push(id.clone());
            match self.read_note_file(&path, &id) {
                Ok(note) => {
                    if let Ok(mtime) = fs::metadata(&path).and_then(|m| m.modified()) {
                        self.mod_times.insert(path, mtime);
                    }
                    self.merge(note);
                }
                Err(e) => log::warn!("skipping note file {name}: {e}"),
            }
        }
        self.notes.retain(|id, _| on_disk.contains(id));
        self.mod_times.retain(|path, _| path.exists());
        Ok(())
    }

    /// Applies one watcher event path to the cache.
    fn handle_path(&mut self, path: &Path) {
        let Some(name) = path.file_name().map(|n| n.to_string_lossy().into_owned()) else {
            return;
        };
        if !is_note_filename(&name) {
            return;
        }
        let id = name.trim_end_matches(".json").to_string();
        let path = self.dir.join(&name);

        if !path.exists() {
            self.notes.remove(&id);
            self.mod_times.remove(&path);
            return;
        }
        if self.key.is_none() {
            return;
        }
        let mtime = fs::metadata(&path).and_then(|m| m.modified()).ok();
        if let (Some(last), Some(current)) = (self.mod_times.get(&path), mtime)
            && current <= *last
        {
            return; // our own write, or already processed
        }
        match self.read_note_file(&path, &id) {
            Ok(note) => {
                if let Some(m) = mtime {
                    self.mod_times.insert(path, m);
                }
                self.merge(note);
            }
            Err(e) => log::warn!("could not process changed note file {name}: {e}"),
        }
    }
}

impl NoteStore {
    pub fn new(data_dir: PathBuf) -> Result<Self> {
        fs::create_dir_all(&data_dir)?;
        Ok(NoteStore {
            inner: Arc::new(RwLock::new(Inner {
                dir: data_dir,
                notes: HashMap::new(),
                key: None,
                mod_times: HashMap::new(),
                watcher: None,
            })),
        })
    }

    fn read(&self) -> RwLockReadGuard<'_, Inner> {
        self.inner.read().unwrap_or_else(|e| e.into_inner())
    }

    fn write(&self) -> RwLockWriteGuard<'_, Inner> {
        self.inner.write().unwrap_or_else(|e| e.into_inner())
    }

    pub fn data_dir(&self) -> PathBuf {
        self.read().dir.clone()
    }

    /// Sets the key, starts the watcher (once) and syncs from disk.
    pub fn load_notes(&self, key: [u8; 32]) -> Result<()> {
        let dir = {
            let mut inner = self.write();
            inner.key = Some(key);
            inner.watcher.is_none().then(|| inner.dir.clone())
        };
        if let Some(dir) = dir {
            // Created without holding the lock: `watch()` waits on the watcher's
            // event thread, which may be blocked in the callback on this lock.
            let started = Self::start_watcher(&dir, Arc::downgrade(&self.inner));
            let spare = {
                let mut inner = self.write();
                match started {
                    Ok(w) if inner.watcher.is_none() => {
                        inner.watcher = Some(w);
                        None
                    }
                    Ok(w) => Some(w),
                    Err(e) => {
                        log::warn!("could not watch notes directory: {e}");
                        None
                    }
                }
            };
            drop(spare);
        }
        self.sync_from_disk()
    }

    fn start_watcher(dir: &Path, weak: Weak<RwLock<Inner>>) -> notify::Result<RecommendedWatcher> {
        let mut watcher = notify::recommended_watcher(move |res: notify::Result<Event>| {
            let Ok(event) = res else {
                return;
            };
            if event.kind.is_access() || matches!(event.kind, EventKind::Other) {
                return;
            }
            let Some(inner) = weak.upgrade() else {
                return;
            };
            let handled = catch_unwind(AssertUnwindSafe(|| {
                let mut inner = inner.write().unwrap_or_else(|e| e.into_inner());
                for path in &event.paths {
                    inner.handle_path(path);
                }
            }));
            if handled.is_err() {
                log::error!("panic while handling a file event");
            }
        })?;
        watcher.watch(dir, RecursiveMode::NonRecursive)?;
        Ok(watcher)
    }

    pub fn sync_from_disk(&self) -> Result<()> {
        self.write().sync_from_disk()
    }

    /// Drops the key, all cached notes and the watcher (logout / shutdown).
    pub fn clear(&self) {
        let watcher = {
            let mut inner = self.write();
            inner.key = None;
            inner.notes.clear();
            inner.mod_times.clear();
            inner.watcher.take()
        };
        // Dropped after the lock is released: stopping a watcher can join its
        // thread, which may be blocked in the callback waiting for this lock.
        drop(watcher);
    }

    pub fn create_note(&self, content: &str, category: Category) -> Result<Note> {
        let mut inner = self.write();
        inner.require_key()?;
        let now = Utc::now();
        let mut id = generate_short_id();
        while inner.notes.contains_key(&id) || inner.note_path(&id).exists() {
            id = generate_short_id();
        }
        let note = Note {
            id,
            content: content.to_string(),
            category,
            original_category: None,
            images: Vec::new(),
            created_at: now,
            updated_at: now,
        };
        inner.save_note(&note)?;
        inner.notes.insert(note.id.clone(), note.clone());
        Ok(note)
    }

    /// Applies `change` to a copy of the note, saves it and then caches it.
    fn mutate(&self, id: &str, change: impl FnOnce(&mut Note) -> Result<()>) -> Result<Note> {
        ensure_valid_id(id)?;
        let mut inner = self.write();
        inner.require_key()?;
        let mut note = inner
            .notes
            .get(id)
            .cloned()
            .ok_or_else(|| AppError::msg("note not found"))?;
        change(&mut note)?;
        note.updated_at = Utc::now();
        inner.save_note(&note)?;
        inner.notes.insert(note.id.clone(), note.clone());
        Ok(note)
    }

    pub fn update_note(&self, id: &str, content: &str) -> Result<Note> {
        self.mutate(id, |n| {
            n.content = content.to_string();
            Ok(())
        })
    }

    pub fn update_category(&self, id: &str, category: Category) -> Result<Note> {
        self.mutate(id, |n| {
            n.category = category;
            Ok(())
        })
    }

    pub fn move_to_trash(&self, id: &str) -> Result<Note> {
        self.mutate(id, |n| {
            if n.category != Category::Trash {
                n.original_category = Some(n.category);
            }
            n.category = Category::Trash;
            Ok(())
        })
    }

    pub fn restore_from_trash(&self, id: &str) -> Result<Note> {
        self.mutate(id, |n| {
            if n.category != Category::Trash {
                return Err(AppError::msg("note is not in trash"));
            }
            n.category = n.original_category.take().unwrap_or(Category::Private);
            Ok(())
        })
    }

    pub fn permanently_delete(&self, id: &str) -> Result<()> {
        ensure_valid_id(id)?;
        let mut inner = self.write();
        let note = inner
            .notes
            .get(id)
            .ok_or_else(|| AppError::msg("note not found"))?;
        if note.category != Category::Trash {
            return Err(AppError::msg(
                "note must be in trash to be permanently deleted",
            ));
        }
        let path = inner.note_path(id);
        match fs::remove_file(&path) {
            Ok(()) => {}
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
            Err(e) => return Err(e.into()),
        }
        inner.notes.remove(id);
        inner.mod_times.remove(&path);
        Ok(())
    }

    pub fn get_note(&self, id: &str) -> Result<Note> {
        ensure_valid_id(id)?;
        self.read()
            .notes
            .get(id)
            .cloned()
            .ok_or_else(|| AppError::msg("note not found"))
    }

    pub fn all_notes(&self) -> Vec<Note> {
        sorted(self.read().notes.values().cloned().collect())
    }

    pub fn notes_by_category(&self, category: Category) -> Vec<Note> {
        sorted(
            self.read()
                .notes
                .values()
                .filter(|n| n.category == category)
                .cloned()
                .collect(),
        )
    }

    pub fn search(&self, query: &str) -> Vec<Note> {
        let q = query.to_lowercase();
        sorted(
            self.read()
                .notes
                .values()
                .filter(|n| n.content.to_lowercase().contains(&q))
                .cloned()
                .collect(),
        )
    }

    /// Moves a note file to `<dir>/corrupted/<id>.json` and drops it from the cache.
    pub fn move_to_corrupted(&self, id: &str) -> Result<()> {
        ensure_valid_id(id)?;
        let mut inner = self.write();
        let corrupted = inner.dir.join("corrupted");
        fs::create_dir_all(&corrupted)?;
        let from = inner.note_path(id);
        fs::rename(&from, corrupted.join(format!("{id}.json")))?;
        inner.notes.remove(id);
        inner.mod_times.remove(&from);
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use base64::Engine;
    use base64::engine::general_purpose::STANDARD;
    use std::thread::sleep;
    use std::time::{Duration, Instant};

    const FIXTURES: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/../testdata/go-fixtures");

    fn fixture_key() -> [u8; 32] {
        let cfg: serde_json::Value = serde_json::from_str(
            &fs::read_to_string(format!("{FIXTURES}/notes/.gote_config.json")).unwrap(),
        )
        .unwrap();
        let salt = STANDARD.decode(cfg["salt"].as_str().unwrap()).unwrap();
        crypto::derive_key("testpass123", &salt)
    }

    fn manifest() -> serde_json::Value {
        serde_json::from_str(&fs::read_to_string(format!("{FIXTURES}/manifest.json")).unwrap())
            .unwrap()
    }

    fn copy_fixture_note(id: &str, dir: &Path) {
        fs::copy(
            format!("{FIXTURES}/notes/{id}.json"),
            dir.join(format!("{id}.json")),
        )
        .unwrap();
    }

    /// A store over a full copy of the Go fixture notes dir.
    fn fixture_store() -> (tempfile::TempDir, NoteStore) {
        let tmp = tempfile::tempdir().unwrap();
        for id in ["1a90778d", "3caaec0b", "abcdef01"] {
            copy_fixture_note(id, tmp.path());
        }
        let store = NoteStore::new(tmp.path().to_path_buf()).unwrap();
        store.load_notes(fixture_key()).unwrap();
        (tmp, store)
    }

    fn empty_store() -> (tempfile::TempDir, NoteStore) {
        let tmp = tempfile::tempdir().unwrap();
        let store = NoteStore::new(tmp.path().to_path_buf()).unwrap();
        store.load_notes(fixture_key()).unwrap();
        (tmp, store)
    }

    fn wait_until(what: &str, cond: impl Fn() -> bool) {
        let deadline = Instant::now() + Duration::from_secs(3);
        while Instant::now() < deadline {
            if cond() {
                return;
            }
            sleep(Duration::from_millis(25));
        }
        panic!("timed out waiting for: {what}");
    }

    #[test]
    fn loads_go_fixture_notes() {
        let (_tmp, store) = fixture_store();
        let notes = store.all_notes();
        assert_eq!(notes.len(), 3);
        let ids: Vec<_> = notes.iter().map(|n| n.id.as_str()).collect();
        assert_eq!(ids, ["1a90778d", "3caaec0b", "abcdef01"]);
        let work = store.get_note("3caaec0b").unwrap();
        assert_eq!(work.category, Category::Work);
        assert_eq!(
            work.content,
            manifest()["work_note_content"].as_str().unwrap()
        );
        let trashed = store.get_note("1a90778d").unwrap();
        assert_eq!(trashed.category, Category::Trash);
        assert_eq!(trashed.original_category, Some(Category::Private));
        let legacy = store.get_note("abcdef01").unwrap();
        assert_eq!(legacy.content, "legacy plain content");
        assert_eq!(legacy.category, Category::Private);
        assert_eq!(store.notes_by_category(Category::Work).len(), 1);
    }

    #[test]
    fn crud_and_trash_flow() {
        let (tmp, store) = empty_store();
        let note = store.create_note("first", Category::Work).unwrap();
        let file = tmp.path().join(format!("{}.json", note.id));
        assert!(file.exists());

        let updated = store.update_note(&note.id, "second").unwrap();
        assert_eq!(updated.content, "second");
        assert!(updated.updated_at >= note.updated_at);
        assert_eq!(updated.created_at, note.created_at);

        assert!(store.permanently_delete(&note.id).is_err());
        assert_eq!(
            store.restore_from_trash(&note.id).unwrap_err().message(),
            "note is not in trash"
        );

        let trashed = store.move_to_trash(&note.id).unwrap();
        assert_eq!(trashed.category, Category::Trash);
        assert_eq!(trashed.original_category, Some(Category::Work));
        // Trashing again keeps the original category.
        let again = store.move_to_trash(&note.id).unwrap();
        assert_eq!(again.original_category, Some(Category::Work));

        let restored = store.restore_from_trash(&note.id).unwrap();
        assert_eq!(restored.category, Category::Work);
        assert_eq!(restored.original_category, None);

        store.move_to_trash(&note.id).unwrap();
        store.permanently_delete(&note.id).unwrap();
        assert!(!file.exists());
        assert_eq!(
            store.get_note(&note.id).unwrap_err().message(),
            "note not found"
        );

        // A fresh store sees what was persisted (round trip through disk).
        let note2 = store.create_note("persist me", Category::Private).unwrap();
        store.update_category(&note2.id, Category::Work).unwrap();
        let reloaded = NoteStore::new(tmp.path().to_path_buf()).unwrap();
        reloaded.load_notes(fixture_key()).unwrap();
        let got = reloaded.get_note(&note2.id).unwrap();
        assert_eq!(
            (got.content.as_str(), got.category),
            ("persist me", Category::Work)
        );
    }

    #[test]
    fn rust_written_note_has_go_shape() {
        let (tmp, store) = empty_store();
        let note = store
            .create_note("shape \u{1f600} <b>", Category::Private)
            .unwrap();
        let raw = fs::read_to_string(tmp.path().join(format!("{}.json", note.id))).unwrap();
        assert!(
            raw.contains("\n  \"id\""),
            "pretty JSON with 2-space indent"
        );
        let v: serde_json::Value = serde_json::from_str(&raw).unwrap();
        let mut keys: Vec<_> = v.as_object().unwrap().keys().cloned().collect();
        keys.sort();
        assert_eq!(keys, ["created_at", "encrypted_data", "id", "updated_at"]);
        assert_eq!(v["id"], note.id.as_str());
        // RFC3339 timestamps parseable by Go's time.Time.
        chrono::DateTime::parse_from_rfc3339(v["created_at"].as_str().unwrap()).unwrap();

        let plain = crypto::decrypt(v["encrypted_data"].as_str().unwrap(), &fixture_key()).unwrap();
        let payload: serde_json::Value = serde_json::from_slice(&plain).unwrap();
        let obj = payload.as_object().unwrap();
        assert_eq!(obj["content"], "shape \u{1f600} <b>");
        assert_eq!(obj["category"], "private");
        assert!(!obj.contains_key("original_category") && !obj.contains_key("images"));
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let mode = fs::metadata(tmp.path().join(format!("{}.json", note.id)))
                .unwrap()
                .permissions()
                .mode();
            assert_eq!(mode & 0o111, 0, "not executable");
        }
        // No stray temp files remain.
        let leftovers: Vec<_> = fs::read_dir(tmp.path())
            .unwrap()
            .flatten()
            .filter(|e| e.file_name().to_string_lossy().ends_with(".tmp"))
            .collect();
        assert!(leftovers.is_empty());
    }

    #[test]
    fn search_is_case_insensitive() {
        let (_tmp, store) = empty_store();
        store.create_note("Hello World", Category::Private).unwrap();
        store.create_note("other", Category::Private).unwrap();
        assert_eq!(store.search("hELLo").len(), 1);
        assert_eq!(store.search("zzz").len(), 0);
    }

    #[test]
    fn rejects_invalid_id() {
        let (_tmp, store) = empty_store();
        assert_eq!(store.get_note("../x").unwrap_err().message(), "invalid id");
        assert_eq!(
            store.update_note("../x", "c").unwrap_err().message(),
            "invalid id"
        );
        assert_eq!(
            store.permanently_delete("../../etc").unwrap_err().message(),
            "invalid id"
        );
        assert_eq!(
            store.move_to_corrupted("../x").unwrap_err().message(),
            "invalid id"
        );
    }

    #[test]
    fn mutations_require_key() {
        let tmp = tempfile::tempdir().unwrap();
        let store = NoteStore::new(tmp.path().to_path_buf()).unwrap();
        assert_eq!(
            store
                .create_note("x", Category::Private)
                .unwrap_err()
                .message(),
            "not authenticated"
        );
        let (_t, store) = fixture_store();
        store.clear();
        assert!(store.all_notes().is_empty());
        assert!(store.update_note("3caaec0b", "x").is_err());
    }

    #[test]
    fn move_to_corrupted_moves_file() {
        let (tmp, store) = fixture_store();
        store.move_to_corrupted("abcdef01").unwrap();
        assert!(!tmp.path().join("abcdef01.json").exists());
        assert!(tmp.path().join("corrupted/abcdef01.json").exists());
        assert_eq!(store.all_notes().len(), 2);
    }

    #[test]
    fn undecryptable_files_are_skipped_not_fatal() {
        let tmp = tempfile::tempdir().unwrap();
        fs::write(tmp.path().join("deadbeef.json"), "garbage").unwrap();
        fs::write(
            tmp.path().join("cafebabe.json"),
            r#"{"id":"cafebabe","encrypted_data":"AAAA"}"#,
        )
        .unwrap();
        copy_fixture_note("abcdef01", tmp.path());
        let store = NoteStore::new(tmp.path().to_path_buf()).unwrap();
        store.load_notes(fixture_key()).unwrap();
        assert_eq!(store.all_notes().len(), 1);
    }

    #[test]
    fn watcher_picks_up_external_note() {
        let (tmp, store) = empty_store();
        assert!(store.all_notes().is_empty());
        copy_fixture_note("3caaec0b", tmp.path());
        wait_until("external note to appear", || store.all_notes().len() == 1);
        assert_eq!(store.all_notes()[0].id, "3caaec0b");
    }

    #[test]
    fn watcher_drops_deleted_note() {
        let (tmp, store) = fixture_store();
        assert_eq!(store.all_notes().len(), 3);
        fs::remove_file(tmp.path().join("abcdef01.json")).unwrap();
        wait_until("deleted note to disappear", || store.all_notes().len() == 2);
        assert!(store.get_note("abcdef01").is_err());
    }

    #[test]
    fn watcher_ignores_non_note_files() {
        let (tmp, store) = fixture_store();
        fs::write(tmp.path().join("notes.txt"), "hi").unwrap();
        fs::write(tmp.path().join("zzzzzzzz.json"), "{}").unwrap();
        fs::write(tmp.path().join(".gote_config.json"), "{}").unwrap();
        sleep(Duration::from_millis(500));
        assert_eq!(store.all_notes().len(), 3);
        // The watcher is still alive afterwards.
        fs::remove_file(tmp.path().join("abcdef01.json")).unwrap();
        wait_until("watcher still alive", || store.all_notes().len() == 2);
    }

    fn write_external_version(dir: &Path, id: &str, content: &str, updated_at: &str) {
        let created = chrono::DateTime::parse_from_rfc3339("2020-01-01T00:00:00Z")
            .unwrap()
            .with_timezone(&Utc);
        let updated = chrono::DateTime::parse_from_rfc3339(updated_at)
            .unwrap()
            .with_timezone(&Utc);
        let payload = serde_json::json!({"content": content, "category": "private"});
        let enc = EncryptedNote {
            id: id.to_string(),
            encrypted_data: crypto::encrypt(payload.to_string().as_bytes(), &fixture_key())
                .unwrap(),
            created_at: created,
            updated_at: updated,
        };
        fs::write(
            dir.join(format!("{id}.json")),
            serde_json::to_string_pretty(&enc).unwrap(),
        )
        .unwrap();
    }

    #[test]
    fn stale_external_write_does_not_override() {
        let (tmp, store) = empty_store();
        let note = store
            .create_note("newest in memory", Category::Private)
            .unwrap();
        write_external_version(
            tmp.path(),
            &note.id,
            "stale external",
            "2021-01-01T00:00:00Z",
        );
        sleep(Duration::from_millis(700));
        assert_eq!(
            store.get_note(&note.id).unwrap().content,
            "newest in memory"
        );
    }

    #[test]
    fn newer_external_write_wins() {
        let (tmp, store) = empty_store();
        let note = store
            .create_note("old in memory", Category::Private)
            .unwrap();
        sleep(Duration::from_millis(50));
        write_external_version(
            tmp.path(),
            &note.id,
            "from another machine",
            "2999-01-01T00:00:00Z",
        );
        wait_until("newer external version", || {
            store.get_note(&note.id).unwrap().content == "from another machine"
        });
    }

    #[test]
    fn clear_while_watcher_is_busy_does_not_hang() {
        let (tmp, store) = empty_store();
        let dir = tmp.path().to_path_buf();
        let stop = Arc::new(std::sync::atomic::AtomicBool::new(false));
        // Build the (expensive-to-key) file content once, then rewrite it in a loop.
        write_external_version(&dir, "3caaec0b", "payload", "2999-01-01T00:00:00Z");
        let content = fs::read_to_string(dir.join("3caaec0b.json")).unwrap();
        let writer = {
            let stop = Arc::clone(&stop);
            std::thread::spawn(move || {
                while !stop.load(std::sync::atomic::Ordering::Relaxed) {
                    fs::write(dir.join("3caaec0b.json"), &content).unwrap();
                    sleep(Duration::from_millis(2));
                }
            })
        };
        let key = fixture_key();
        let (tx, rx) = std::sync::mpsc::channel();
        let s2 = store.clone();
        std::thread::spawn(move || {
            for _ in 0..30 {
                s2.clear();
                s2.load_notes(key).unwrap();
            }
            s2.clear();
            tx.send(()).unwrap();
        });
        let finished = rx.recv_timeout(Duration::from_secs(10)).is_ok();
        stop.store(true, std::sync::atomic::Ordering::Relaxed);
        writer.join().unwrap();
        assert!(finished, "clear()/load_notes() hung");
    }

    #[test]
    fn own_writes_do_not_clobber_cache() {
        let (_tmp, store) = empty_store();
        let note = store.create_note("a", Category::Private).unwrap();
        for i in 0..5 {
            store.update_note(&note.id, &format!("v{i}")).unwrap();
        }
        sleep(Duration::from_millis(400));
        assert_eq!(store.get_note(&note.id).unwrap().content, "v4");
    }
}
