pub mod auth;
pub mod backup;
pub mod config;
pub mod crypto;
pub mod error;
pub mod ids;
pub mod image_store;
pub mod models;
pub mod note_store;

pub fn run() {
    tauri::Builder::default()
        .run(tauri::generate_context!())
        .expect("error while running gote");
}
