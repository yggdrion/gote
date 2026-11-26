use std::fs::{self, File};
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use std::time::{Duration, SystemTime};

use chrono::Local;
use zip::write::SimpleFileOptions;

use crate::error::{AppError, Result};

const BACKUP_INTERVAL: Duration = Duration::from_secs(24 * 60 * 60);

fn zip_err(e: zip::result::ZipError) -> AppError {
    AppError::msg(format!("backup failed: {e}"))
}

fn add_file<W: Write + io::Seek>(
    zip: &mut zip::ZipWriter<W>,
    folder: &str,
    rel: &str,
    path: &Path,
) -> Result<()> {
    let mut src = File::open(path)?;
    zip.start_file(
        format!("{folder}{rel}"),
        SimpleFileOptions::default().compression_method(zip::CompressionMethod::Deflated),
    )
    .map_err(zip_err)?;
    io::copy(&mut src, zip)?;
    Ok(())
}

fn add_dir_recursive<W: Write + io::Seek>(
    zip: &mut zip::ZipWriter<W>,
    folder: &str,
    rel: &str,
    dir: &Path,
) {
    let Ok(entries) = fs::read_dir(dir) else {
        return;
    };
    for entry in entries.flatten() {
        let name = entry.file_name().to_string_lossy().into_owned();
        let rel_path = format!("{rel}/{name}");
        let Ok(kind) = entry.file_type() else {
            continue;
        };
        if kind.is_dir() {
            add_dir_recursive(zip, folder, &rel_path, &entry.path());
        } else if kind.is_file()
            && let Err(e) = add_file(zip, folder, &rel_path, &entry.path())
        {
            log::warn!("backup: skipping {rel_path}: {e}");
        }
    }
}

/// Creates `<notesDir>/backups/backup-<YYYYMMDD-HHMM>.zip` and returns its path.
pub fn backup_notes(notes_dir: &Path) -> Result<PathBuf> {
    let backups_dir = notes_dir.join("backups");
    fs::create_dir_all(&backups_dir)?;

    let stamp = Local::now().format("%Y%m%d-%H%M").to_string();
    let zip_path = backups_dir.join(format!("backup-{stamp}.zip"));
    let tmp_path = backups_dir.join(format!("backup-{stamp}.zip.tmp"));
    let folder = format!("backup-{stamp}/");

    let result = (|| -> Result<()> {
        let mut zip = zip::ZipWriter::new(File::create(&tmp_path)?);

        // Root-level *.json files (this includes .gote_config.json), each once.
        let mut names: Vec<String> = fs::read_dir(notes_dir)?
            .flatten()
            .filter(|e| e.file_type().is_ok_and(|t| t.is_file()))
            .map(|e| e.file_name().to_string_lossy().into_owned())
            .filter(|n| n.ends_with(".json"))
            .collect();
        names.sort();
        for name in names {
            if let Err(e) = add_file(&mut zip, &folder, &name, &notes_dir.join(&name)) {
                log::warn!("backup: skipping {name}: {e}");
            }
        }

        let images = notes_dir.join("images");
        if images.is_dir() {
            add_dir_recursive(&mut zip, &folder, "images", &images);
        }

        zip.finish().map_err(zip_err)?.sync_all()?;
        Ok(())
    })();

    match result {
        Ok(()) => {
            // Replaces an existing backup of the same minute.
            fs::rename(&tmp_path, &zip_path)?;
            Ok(zip_path)
        }
        Err(e) => {
            let _ = fs::remove_file(&tmp_path);
            Err(e)
        }
    }
}

/// Modification time of the newest `backup-*.zip` in `<notesDir>/backups`.
pub fn latest_backup_mtime(notes_dir: &Path) -> Option<SystemTime> {
    fs::read_dir(notes_dir.join("backups"))
        .ok()?
        .flatten()
        .filter(|e| {
            let n = e.file_name().to_string_lossy().into_owned();
            n.starts_with("backup-") && n.ends_with(".zip")
        })
        .filter_map(|e| e.metadata().ok()?.modified().ok())
        .max()
}

/// True when no backup exists or the newest one is at least 24 hours old.
pub fn backup_is_due(notes_dir: &Path, now: SystemTime) -> bool {
    match latest_backup_mtime(notes_dir) {
        None => true,
        Some(mtime) => now
            .duration_since(mtime)
            .is_ok_and(|age| age >= BACKUP_INTERVAL),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::BTreeSet;

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

    #[test]
    fn backup_contents() {
        let tmp = tempfile::tempdir().unwrap();
        let notes = tmp.path().join("notes");
        copy_dir(&Path::new(FIXTURES).join("notes"), &notes);
        fs::create_dir_all(notes.join("corrupted")).unwrap();
        fs::write(notes.join("corrupted/x.json"), "{}").unwrap();
        fs::create_dir_all(notes.join("backups")).unwrap();
        fs::write(notes.join("backups/old.zip"), "zip").unwrap();
        fs::write(notes.join("readme.txt"), "not a json file").unwrap();

        let zip_path = backup_notes(&notes).unwrap();
        let file_name = zip_path.file_name().unwrap().to_string_lossy().into_owned();
        assert!(file_name.starts_with("backup-") && file_name.ends_with(".zip"));
        let stamp = file_name
            .strip_prefix("backup-")
            .unwrap()
            .strip_suffix(".zip")
            .unwrap();
        assert_eq!(stamp.len(), 13);

        let mut archive = zip::ZipArchive::new(File::open(&zip_path).unwrap()).unwrap();
        let names: Vec<String> = (0..archive.len())
            .map(|i| archive.by_index(i).unwrap().name().to_string())
            .collect();
        let unique: BTreeSet<_> = names.iter().cloned().collect();
        assert_eq!(names.len(), unique.len(), "each entry once: {names:?}");
        let expected: BTreeSet<String> = [
            "1a90778d.json",
            "3caaec0b.json",
            "abcdef01.json",
            ".gote_config.json",
            "images/0c93c8fc.json",
        ]
        .iter()
        .map(|n| format!("backup-{stamp}/{n}"))
        .collect();
        assert_eq!(unique, expected);

        // Content is preserved.
        let mut entry = archive
            .by_name(&format!("backup-{stamp}/abcdef01.json"))
            .unwrap();
        let mut text = String::new();
        io::Read::read_to_string(&mut entry, &mut text).unwrap();
        assert_eq!(
            text,
            fs::read_to_string(Path::new(FIXTURES).join("notes/abcdef01.json")).unwrap()
        );
        assert!(
            !notes
                .join("backups")
                .join(format!("{file_name}.tmp"))
                .exists()
        );

        // Re-running in the same minute replaces the zip.
        backup_notes(&notes).unwrap();
    }

    #[test]
    fn backup_is_due_logic() {
        let tmp = tempfile::tempdir().unwrap();
        let notes = tmp.path();
        let now = SystemTime::now();
        assert!(backup_is_due(notes, now));
        assert!(latest_backup_mtime(notes).is_none());

        fs::create_dir_all(notes.join("backups")).unwrap();
        // Non-matching files do not count.
        fs::write(notes.join("backups/other.zip"), "x").unwrap();
        assert!(backup_is_due(notes, now));

        let zip = notes.join("backups/backup-20250101-0000.zip");
        fs::write(&zip, "x").unwrap();
        assert!(!backup_is_due(notes, now));
        assert!(!backup_is_due(notes, now + Duration::from_secs(23 * 3600)));
        assert!(backup_is_due(notes, now + Duration::from_secs(25 * 3600)));
        assert!(latest_backup_mtime(notes).is_some());
    }
}
