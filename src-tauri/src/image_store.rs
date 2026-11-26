use std::fs;
use std::path::{Path, PathBuf};
use std::sync::{OnceLock, RwLock};

use chrono::{DateTime, Utc};
use regex::Regex;
use serde::{Deserialize, Serialize};

use crate::crypto;
use crate::error::{AppError, Result};
use crate::ids::{ensure_valid_id, generate_short_id};
use crate::models::Image;

/// Image file layout: `<notesDir>/images/<id>.json`.
#[derive(Debug, Serialize, Deserialize)]
struct EncryptedImage {
    #[serde(default)]
    id: String,
    #[serde(default)]
    filename: String,
    #[serde(default)]
    content_type: String,
    #[serde(default)]
    size: i64,
    #[serde(default)]
    encrypted_data: String,
    #[serde(default)]
    created_at: DateTime<Utc>,
}

impl EncryptedImage {
    fn meta(&self) -> Image {
        Image {
            id: self.id.clone(),
            filename: self.filename.clone(),
            content_type: self.content_type.clone(),
            size: self.size,
            created_at: self.created_at,
        }
    }
}

pub struct ImageStore {
    dir: PathBuf,
    key: RwLock<Option<[u8; 32]>>,
}

impl ImageStore {
    pub fn new(notes_dir: &Path) -> Self {
        let dir = notes_dir.join("images");
        if let Err(e) = fs::create_dir_all(&dir) {
            log::warn!("failed to create images directory {}: {e}", dir.display());
        }
        ImageStore {
            dir,
            key: RwLock::new(None),
        }
    }

    pub fn set_key(&self, key: Option<[u8; 32]>) {
        *self.key.write().unwrap_or_else(|e| e.into_inner()) = key;
    }

    fn key(&self) -> Result<[u8; 32]> {
        (*self.key.read().unwrap_or_else(|e| e.into_inner()))
            .ok_or_else(|| AppError::msg("encryption key not set"))
    }

    fn path_for(&self, id: &str) -> Result<PathBuf> {
        ensure_valid_id(id)?;
        Ok(self.dir.join(format!("{id}.json")))
    }

    fn load(path: &Path) -> Result<EncryptedImage> {
        let text = fs::read_to_string(path)?;
        Ok(serde_json::from_str(&text)?)
    }

    pub fn store_image(&self, data: &[u8], content_type: &str, filename: &str) -> Result<Image> {
        let key = self.key()?;
        let image = Image {
            id: generate_short_id(),
            filename: filename.to_string(),
            content_type: content_type.to_string(),
            size: data.len() as i64,
            created_at: Utc::now(),
        };
        let encrypted = EncryptedImage {
            id: image.id.clone(),
            filename: image.filename.clone(),
            content_type: image.content_type.clone(),
            size: image.size,
            encrypted_data: crypto::encrypt(data, &key)
                .map_err(|e| AppError::msg(format!("failed to encrypt image: {e}")))?,
            created_at: image.created_at,
        };
        fs::create_dir_all(&self.dir)
            .map_err(|e| AppError::msg(format!("failed to create images directory: {e}")))?;
        let json = serde_json::to_string_pretty(&encrypted)?;
        fs::write(self.path_for(&image.id)?, json.as_bytes())
            .map_err(|e| AppError::msg(format!("failed to save image: {e}")))?;
        Ok(image)
    }

    pub fn get_image(&self, id: &str) -> Result<(Vec<u8>, Image)> {
        let key = self.key()?;
        let path = self.path_for(id)?;
        let encrypted =
            Self::load(&path).map_err(|e| AppError::msg(format!("failed to load image: {e}")))?;
        let data = crypto::decrypt(&encrypted.encrypted_data, &key)
            .map_err(|e| AppError::msg(format!("failed to decrypt image: {e}")))?;
        Ok((data, encrypted.meta()))
    }

    /// Deletes an image file; a missing file is not an error.
    pub fn delete_image(&self, id: &str) -> Result<()> {
        match fs::remove_file(self.path_for(id)?) {
            Ok(()) => Ok(()),
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => Ok(()),
            Err(e) => Err(e.into()),
        }
    }

    pub fn list_images(&self) -> Result<Vec<Image>> {
        let entries = fs::read_dir(&self.dir)
            .map_err(|e| AppError::msg(format!("failed to read images directory: {e}")))?;
        let mut images = Vec::new();
        for entry in entries.flatten() {
            let path = entry.path();
            if path.extension().and_then(|e| e.to_str()) != Some("json") {
                continue;
            }
            if let Ok(encrypted) = Self::load(&path) {
                images.push(encrypted.meta());
            }
        }
        Ok(images)
    }
}

/// Image ids referenced as `![alt](image:<id>)`, in order of appearance.
pub fn extract_image_ids(content: &str) -> Vec<String> {
    static RE: OnceLock<Regex> = OnceLock::new();
    let re = RE.get_or_init(|| Regex::new(r"!\[[^\]]*\]\(image:([^)]+)\)").expect("valid regex"));
    re.captures_iter(content)
        .map(|c| c[1].to_string())
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use base64::Engine;
    use base64::engine::general_purpose::STANDARD;

    const FIXTURES: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/../testdata/go-fixtures");

    fn fixture_key() -> [u8; 32] {
        let cfg: serde_json::Value = serde_json::from_str(
            &fs::read_to_string(format!("{FIXTURES}/notes/.gote_config.json")).unwrap(),
        )
        .unwrap();
        let salt = STANDARD.decode(cfg["salt"].as_str().unwrap()).unwrap();
        crypto::derive_key("testpass123", &salt)
    }

    fn fixture_store(tmp: &Path) -> ImageStore {
        let images = tmp.join("images");
        fs::create_dir_all(&images).unwrap();
        fs::copy(
            format!("{FIXTURES}/notes/images/0c93c8fc.json"),
            images.join("0c93c8fc.json"),
        )
        .unwrap();
        let store = ImageStore::new(tmp);
        store.set_key(Some(fixture_key()));
        store
    }

    #[test]
    fn reads_go_image() {
        let tmp = tempfile::tempdir().unwrap();
        let store = fixture_store(tmp.path());
        let manifest: serde_json::Value =
            serde_json::from_str(&fs::read_to_string(format!("{FIXTURES}/manifest.json")).unwrap())
                .unwrap();
        let (data, meta) = store.get_image("0c93c8fc").unwrap();
        let expected: Vec<u8> = (0..24)
            .step_by(2)
            .map(|i| {
                u8::from_str_radix(&manifest["image_bytes_hex"].as_str().unwrap()[i..i + 2], 16)
                    .unwrap()
            })
            .collect();
        assert_eq!(data, expected);
        assert_eq!(meta.filename, "clipboard_1700000000.png");
        assert_eq!(meta.content_type, "image/png");
        assert_eq!(meta.size, 12);
    }

    #[test]
    fn store_get_roundtrip_and_list() {
        let tmp = tempfile::tempdir().unwrap();
        let store = ImageStore::new(tmp.path());
        assert_eq!(
            store
                .store_image(b"abc", "image/png", "a.png")
                .unwrap_err()
                .message(),
            "encryption key not set"
        );
        store.set_key(Some(crypto::derive_key("pw", &crypto::generate_salt())));
        let meta = store
            .store_image(b"\x89PNG\0data", "image/png", "a.png")
            .unwrap();
        assert_eq!(meta.size, 9);
        let (data, got) = store.get_image(&meta.id).unwrap();
        assert_eq!(data, b"\x89PNG\0data");
        assert_eq!(got, meta);

        // Go-shaped file: exactly these keys.
        let raw: serde_json::Value = serde_json::from_str(
            &fs::read_to_string(tmp.path().join("images").join(format!("{}.json", meta.id)))
                .unwrap(),
        )
        .unwrap();
        let mut keys: Vec<_> = raw.as_object().unwrap().keys().cloned().collect();
        keys.sort();
        assert_eq!(
            keys,
            [
                "content_type",
                "created_at",
                "encrypted_data",
                "filename",
                "id",
                "size"
            ]
        );

        fs::write(tmp.path().join("images").join("junk.json"), "not json").unwrap();
        let list = store.list_images().unwrap();
        assert_eq!(list, vec![meta.clone()]);

        store.delete_image(&meta.id).unwrap();
        store.delete_image(&meta.id).unwrap();
        assert!(store.get_image(&meta.id).is_err());
    }

    #[test]
    fn get_rejects_traversal_id() {
        let tmp = tempfile::tempdir().unwrap();
        let store = fixture_store(tmp.path());
        for bad in ["../../etc/passwd", "../x", "", "0c93c8f", "0c93c8fc/.."] {
            assert_eq!(store.get_image(bad).unwrap_err().message(), "invalid id");
            assert_eq!(store.delete_image(bad).unwrap_err().message(), "invalid id");
        }
    }

    #[test]
    fn extract_image_ids_in_order() {
        let ids = extract_image_ids("a ![x](image:0c93c8fc) b ![](image:deadbeef)");
        assert_eq!(ids, ["0c93c8fc", "deadbeef"]);
        assert!(extract_image_ids("![x](http://a/b.png) plain").is_empty());
    }
}
