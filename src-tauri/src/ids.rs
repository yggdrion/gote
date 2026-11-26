use crate::error::{AppError, Result};

const ID_LEN: usize = 8;

/// 8 lowercase hex chars from the first 8 chars of a v4 UUID.
pub fn generate_short_id() -> String {
    uuid::Uuid::new_v4().to_string()[..ID_LEN].to_string()
}

/// `^[0-9a-fA-F]{8}$`
pub fn is_valid_id(id: &str) -> bool {
    id.len() == ID_LEN && id.bytes().all(|b| b.is_ascii_hexdigit())
}

/// `<valid id>.json`
pub fn is_note_filename(name: &str) -> bool {
    name.strip_suffix(".json").is_some_and(is_valid_id)
}

pub fn ensure_valid_id(id: &str) -> Result<()> {
    if is_valid_id(id) {
        Ok(())
    } else {
        Err(AppError::msg("invalid id"))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn note_filenames() {
        assert!(is_note_filename("3caaec0b.json"));
        assert!(!is_note_filename(".gote_config.json"));
        assert!(!is_note_filename("zzzzzzzz.json"));
        assert!(!is_note_filename("3caaec0b.txt"));
    }

    #[test]
    fn id_validation() {
        assert!(!is_valid_id("../../etc"));
        assert!(!is_valid_id("3caaec0"));
        assert!(!is_valid_id("3caaec0b\n"));
        assert!(is_valid_id("3CAAEC0b"));
        assert!(ensure_valid_id("../x").is_err());
        assert_eq!(ensure_valid_id("../x").unwrap_err().message(), "invalid id");
    }

    #[test]
    fn generated_ids_are_short_lowercase_hex() {
        for _ in 0..50 {
            let id = generate_short_id();
            assert_eq!(id.len(), 8);
            assert!(id.bytes().all(|b| matches!(b, b'0'..=b'9' | b'a'..=b'f')));
        }
    }
}
