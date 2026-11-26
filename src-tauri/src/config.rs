use std::fs;
use std::path::{Path, PathBuf};

use serde::{Deserialize, Serialize};

use crate::error::Result;

/// App configuration stored at `~/.config/gote/config`.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Config {
    #[serde(rename = "notesPath")]
    pub notes_path: PathBuf,
    #[serde(rename = "passwordHashPath")]
    pub password_hash_path: PathBuf,
}

fn home() -> PathBuf {
    dirs::home_dir().unwrap_or_else(|| PathBuf::from("."))
}

fn config_dir_for(home: &Path) -> PathBuf {
    home.join(".config").join("gote")
}

/// `~/.config/gote` on every platform.
pub fn config_dir() -> PathBuf {
    config_dir_for(&home())
}

/// `~/.config/gote/config` (no extension).
pub fn config_file_path() -> PathBuf {
    config_dir().join("config")
}

/// `<home>/Documents/Gote/Notes` on every platform.
pub fn default_notes_path() -> PathBuf {
    home().join("Documents").join("Gote").join("Notes")
}

/// `~/.config/gote/password_hash`.
pub fn default_password_hash_path() -> PathBuf {
    config_dir().join("password_hash")
}

impl Config {
    pub fn defaults() -> Config {
        Config {
            notes_path: default_notes_path(),
            password_hash_path: default_password_hash_path(),
        }
    }

    /// Defaults relative to an explicit home directory.
    pub fn defaults_for_home(home: &Path) -> Config {
        Config {
            notes_path: home.join("Documents").join("Gote").join("Notes"),
            password_hash_path: config_dir_for(home).join("password_hash"),
        }
    }

    /// Loads the config at `path`; a missing file yields the defaults. Ensures the
    /// notes directory and the password hash file's parent directory exist.
    pub fn load_from(path: &Path) -> Result<Config> {
        let config = match fs::read_to_string(path) {
            Ok(text) => serde_json::from_str(&text)?,
            Err(_) => Config::defaults(),
        };
        fs::create_dir_all(&config.notes_path)?;
        if let Some(parent) = config.password_hash_path.parent() {
            fs::create_dir_all(parent)?;
        }
        Ok(config)
    }

    pub fn load() -> Result<Config> {
        Config::load_from(&config_file_path())
    }

    pub fn save_to(&self, path: &Path) -> Result<()> {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent)?;
        }
        let data = serde_json::to_string_pretty(self)?;
        fs::write(path, data)?;
        Ok(())
    }

    pub fn save(&self) -> Result<()> {
        self.save_to(&config_file_path())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn config_roundtrip() {
        let dir = tempfile::tempdir().unwrap();
        let cfg_path = dir.path().join("cfg").join("config");
        let cfg = Config {
            notes_path: dir.path().join("notes"),
            password_hash_path: dir.path().join("hash").join("password_hash"),
        };
        cfg.save_to(&cfg_path).unwrap();

        let v: serde_json::Value =
            serde_json::from_str(&fs::read_to_string(&cfg_path).unwrap()).unwrap();
        let obj = v.as_object().unwrap();
        assert!(obj.contains_key("notesPath") && obj.contains_key("passwordHashPath"));
        assert_eq!(obj.len(), 2);

        let loaded = Config::load_from(&cfg_path).unwrap();
        assert_eq!(loaded, cfg);
        assert!(cfg.notes_path.is_dir());
        assert!(cfg.password_hash_path.parent().unwrap().is_dir());
    }

    #[test]
    fn defaults_layout() {
        let c = Config::defaults_for_home(Path::new("/h"));
        assert_eq!(c.notes_path, Path::new("/h/Documents/Gote/Notes"));
        assert_eq!(
            c.password_hash_path,
            Path::new("/h/.config/gote/password_hash")
        );
    }

    #[test]
    fn parses_go_written_config() {
        let dir = tempfile::tempdir().unwrap();
        let p = dir.path().join("config");
        let notes = dir.path().join("n");
        fs::write(
            &p,
            format!(
                "{{\n  \"notesPath\": {:?},\n  \"passwordHashPath\": {:?}\n}}",
                notes.to_str().unwrap(),
                dir.path().join("ph").to_str().unwrap()
            ),
        )
        .unwrap();
        assert_eq!(Config::load_from(&p).unwrap().notes_path, notes);
    }
}
