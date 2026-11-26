use chrono::{DateTime, Local, SecondsFormat, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Category {
    Private,
    Work,
    Trash,
}

impl Category {
    pub fn parse(s: &str) -> Option<Category> {
        match s {
            "private" => Some(Category::Private),
            "work" => Some(Category::Work),
            "trash" => Some(Category::Trash),
            _ => None,
        }
    }

    pub fn as_str(&self) -> &'static str {
        match self {
            Category::Private => "private",
            Category::Work => "work",
            Category::Trash => "trash",
        }
    }
}

/// Image metadata embedded in notes and stored alongside the encrypted bytes.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Image {
    pub id: String,
    pub filename: String,
    pub content_type: String,
    pub size: i64,
    pub created_at: DateTime<Utc>,
}

/// On-disk wrapper of a note file (`<id>.json`).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncryptedNote {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub encrypted_data: String,
    #[serde(default)]
    pub created_at: DateTime<Utc>,
    #[serde(default)]
    pub updated_at: DateTime<Utc>,
}

/// Decrypted note held in memory.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Note {
    pub id: String,
    pub content: String,
    pub category: Category,
    pub original_category: Option<Category>,
    pub images: Vec<Image>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// The JSON object that gets encrypted into `encrypted_data`.
#[derive(Debug, Default, Serialize, Deserialize)]
pub struct NotePayload {
    pub content: String,
    #[serde(default)]
    pub category: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub original_category: String,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub images: Vec<Image>,
}

/// Decoded form of a note payload: (content, category, original category, images).
pub type DecodedPayload = (String, Category, Option<Category>, Vec<Image>);

impl NotePayload {
    /// Decodes decrypted text. A JSON object is the current format; anything else
    /// is a legacy note where the whole text is the content (category `private`).
    pub fn decode(plaintext: &str) -> DecodedPayload {
        let legacy = || (plaintext.to_string(), Category::Private, None, Vec::new());
        let Ok(value @ serde_json::Value::Object(_)) =
            serde_json::from_str::<serde_json::Value>(plaintext)
        else {
            return legacy();
        };
        match serde_json::from_value::<NotePayload>(value) {
            Ok(p) => (
                p.content,
                Category::parse(&p.category).unwrap_or(Category::Private),
                Category::parse(&p.original_category),
                p.images,
            ),
            Err(_) => legacy(),
        }
    }

    /// Compact JSON of the note, omitting empty `original_category` and `images`.
    pub fn encode(note: &Note) -> String {
        let payload = NotePayload {
            content: note.content.clone(),
            category: note.category.as_str().to_string(),
            original_category: note
                .original_category
                .map(|c| c.as_str().to_string())
                .unwrap_or_default(),
            images: note.images.clone(),
        };
        serde_json::to_string(&payload).expect("note payload serializes")
    }
}

/// Note as sent to the frontend.
#[derive(Debug, Clone, Serialize)]
pub struct NoteDto {
    pub id: String,
    pub content: String,
    pub category: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub original_category: Option<String>,
    pub created_at: String,
    pub updated_at: String,
}

fn format_local(t: &DateTime<Utc>) -> String {
    t.with_timezone(&Local)
        .to_rfc3339_opts(SecondsFormat::Secs, true)
}

impl From<&Note> for NoteDto {
    fn from(n: &Note) -> Self {
        NoteDto {
            id: n.id.clone(),
            content: n.content.clone(),
            category: n.category.as_str().to_string(),
            original_category: n.original_category.map(|c| c.as_str().to_string()),
            created_at: format_local(&n.created_at),
            updated_at: format_local(&n.updated_at),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::crypto;
    use base64::Engine;
    use base64::engine::general_purpose::STANDARD;
    use chrono::TimeZone;
    use serde_json::Value;

    const FIXTURES: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/../testdata/go-fixtures");

    fn read_json(rel: &str) -> Value {
        serde_json::from_str(&std::fs::read_to_string(format!("{FIXTURES}/{rel}")).unwrap())
            .unwrap()
    }

    fn fixture_key() -> [u8; 32] {
        let cfg = read_json("notes/.gote_config.json");
        let salt = STANDARD.decode(cfg["salt"].as_str().unwrap()).unwrap();
        crypto::derive_key("testpass123", &salt)
    }

    fn decode_fixture(id: &str) -> DecodedPayload {
        let raw = std::fs::read_to_string(format!("{FIXTURES}/notes/{id}.json")).unwrap();
        let enc: EncryptedNote = serde_json::from_str(&raw).unwrap();
        let plain = crypto::decrypt(&enc.encrypted_data, &fixture_key()).unwrap();
        NotePayload::decode(std::str::from_utf8(&plain).unwrap())
    }

    #[test]
    fn decodes_go_payloads() {
        let manifest = read_json("manifest.json");

        let (content, cat, orig, images) = decode_fixture("3caaec0b");
        assert_eq!(content, manifest["work_note_content"].as_str().unwrap());
        assert_eq!(cat, Category::Work);
        assert_eq!(orig, None);
        assert!(images.is_empty());

        let (content, cat, orig, _) = decode_fixture("1a90778d");
        assert_eq!(content, "private note");
        assert_eq!(cat, Category::Trash);
        assert_eq!(orig, Some(Category::Private));

        let (content, cat, orig, images) = decode_fixture("abcdef01");
        assert_eq!(content, "legacy plain content");
        assert_eq!(cat, Category::Private);
        assert_eq!(orig, None);
        assert!(images.is_empty());
    }

    #[test]
    fn decode_edge_cases() {
        // Empty / missing / unknown category reads as private.
        let (_, cat, _, _) = NotePayload::decode(r#"{"content":"a","category":""}"#);
        assert_eq!(cat, Category::Private);
        let (c, cat, _, _) = NotePayload::decode(r#"{"content":"a"}"#);
        assert_eq!((c.as_str(), cat), ("a", Category::Private));
        // Non-object JSON is legacy content, verbatim.
        let (c, cat, _, _) = NotePayload::decode("\"quoted\"");
        assert_eq!((c.as_str(), cat), ("\"quoted\"", Category::Private));
        let (c, _, _, _) = NotePayload::decode("42");
        assert_eq!(c, "42");
    }

    #[test]
    fn parses_go_timestamps() {
        let raw = std::fs::read_to_string(format!("{FIXTURES}/notes/3caaec0b.json")).unwrap();
        let enc: EncryptedNote = serde_json::from_str(&raw).unwrap();
        let expected = DateTime::parse_from_rfc3339("2026-11-26T21:27:05.523033578+01:00")
            .unwrap()
            .with_timezone(&Utc);
        assert_eq!(enc.created_at, expected);
        assert_eq!(enc.created_at.timestamp_subsec_nanos(), 523_033_578);
        // Re-serialization keeps the same instant.
        let back: EncryptedNote =
            serde_json::from_str(&serde_json::to_string(&enc).unwrap()).unwrap();
        assert_eq!(back.created_at, enc.created_at);
        assert_eq!(back.updated_at, enc.updated_at);
    }

    fn sample_note() -> Note {
        Note {
            id: "deadbeef".into(),
            content: "hello".into(),
            category: Category::Private,
            original_category: None,
            images: vec![],
            created_at: Utc.with_ymd_and_hms(2025, 1, 2, 3, 4, 5).unwrap(),
            updated_at: Utc.with_ymd_and_hms(2025, 1, 2, 3, 4, 6).unwrap(),
        }
    }

    #[test]
    fn encode_omits_empty() {
        let v: Value = serde_json::from_str(&NotePayload::encode(&sample_note())).unwrap();
        let obj = v.as_object().unwrap();
        assert!(!obj.contains_key("original_category"));
        assert!(!obj.contains_key("images"));
        assert_eq!(obj["content"], "hello");
        assert_eq!(obj["category"], "private");
    }

    #[test]
    fn encode_roundtrip_with_original_and_images() {
        let mut n = sample_note();
        n.category = Category::Trash;
        n.original_category = Some(Category::Work);
        n.images = vec![Image {
            id: "0c93c8fc".into(),
            filename: "a.png".into(),
            content_type: "image/png".into(),
            size: 3,
            created_at: n.created_at,
        }];
        let (c, cat, orig, imgs) = NotePayload::decode(&NotePayload::encode(&n));
        assert_eq!(c, "hello");
        assert_eq!(cat, Category::Trash);
        assert_eq!(orig, Some(Category::Work));
        assert_eq!(imgs, n.images);
    }

    #[test]
    fn dto_time_format() {
        let dto = NoteDto::from(&sample_note());
        for s in [&dto.created_at, &dto.updated_at] {
            assert!(!s.contains('.'), "no fractional seconds: {s}");
            let tail = &s[s.len() - 6..];
            assert!(
                s.ends_with('Z') || tail.starts_with('+') || tail.starts_with('-'),
                "offset form: {s}"
            );
        }
        let parsed = DateTime::parse_from_rfc3339(&dto.created_at).unwrap();
        assert_eq!(parsed.with_timezone(&Utc), sample_note().created_at);
        let v = serde_json::to_value(&dto).unwrap();
        assert!(v.get("original_category").is_none());
    }
}
