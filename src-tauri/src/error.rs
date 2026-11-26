use std::fmt;

/// Error string the frontend matches on (substring `session expired`) to auto-logout.
pub const SESSION_EXPIRED: &str = "session expired - re-authentication required";

/// Application error. Serializes as its display string so that Tauri rejects
/// `invoke` calls with a plain message.
#[derive(Debug)]
pub struct AppError(String);

impl AppError {
    pub fn msg(message: impl Into<String>) -> Self {
        AppError(message.into())
    }

    pub fn message(&self) -> &str {
        &self.0
    }
}

impl fmt::Display for AppError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(&self.0)
    }
}

impl std::error::Error for AppError {}

impl serde::Serialize for AppError {
    fn serialize<S: serde::Serializer>(
        &self,
        serializer: S,
    ) -> std::result::Result<S::Ok, S::Error> {
        serializer.serialize_str(&self.0)
    }
}

impl From<std::io::Error> for AppError {
    fn from(e: std::io::Error) -> Self {
        AppError(e.to_string())
    }
}

impl From<serde_json::Error> for AppError {
    fn from(e: serde_json::Error) -> Self {
        AppError(e.to_string())
    }
}

pub type Result<T> = std::result::Result<T, AppError>;
