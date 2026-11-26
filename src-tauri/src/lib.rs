pub mod crypto;
pub mod error;
pub mod ids;

pub fn run() {
    tauri::Builder::default()
        .run(tauri::generate_context!())
        .expect("error while running gote");
}
