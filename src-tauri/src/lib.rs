pub mod app;
pub mod auth;
pub mod backup;
pub mod commands;
pub mod config;
pub mod crypto;
pub mod error;
pub mod ids;
pub mod image_store;
pub mod models;
pub mod note_store;

use std::sync::Mutex;
use std::thread;
use std::time::{Duration, Instant};

use tauri::{Manager, RunEvent};

use crate::app::AppCore;

const SESSION_CLEANUP_INTERVAL: Duration = Duration::from_secs(5 * 60);
const BACKUP_CHECK_INTERVAL: Duration = Duration::from_secs(60 * 60);
/// How often the backup thread looks for "configured but not yet checked".
const BACKUP_POLL: Duration = Duration::from_secs(30);

fn lock_core(state: &Mutex<AppCore>) -> std::sync::MutexGuard<'_, AppCore> {
    state.lock().unwrap_or_else(|e| e.into_inner())
}

fn spawn_background_tasks(handle: tauri::AppHandle) {
    let sessions = handle.clone();
    thread::spawn(move || {
        loop {
            thread::sleep(SESSION_CLEANUP_INTERVAL);
            lock_core(&sessions.state::<Mutex<AppCore>>()).cleanup_sessions();
        }
    });

    // Check at startup (as soon as the app is configured), then hourly.
    thread::spawn(move || {
        let mut last_check: Option<Instant> = None;
        loop {
            if last_check.is_none_or(|t| t.elapsed() >= BACKUP_CHECK_INTERVAL) {
                let job = {
                    let state = handle.state::<Mutex<AppCore>>();
                    let core = lock_core(&state);
                    core.configured.then(|| core.backup_job())
                };
                if let Some(job) = job {
                    job.run_if_due();
                    last_check = Some(Instant::now());
                }
            }
            thread::sleep(BACKUP_POLL);
        }
    });
}

pub fn run() {
    env_logger::Builder::from_env(env_logger::Env::default().default_filter_or("info")).init();

    let app = tauri::Builder::default()
        .plugin(tauri_plugin_clipboard_manager::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_process::init())
        .manage(Mutex::new(AppCore::startup(config::config_file_path())))
        .setup(|app| {
            spawn_background_tasks(app.handle().clone());
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            commands::is_configured,
            commands::is_password_set,
            commands::complete_initial_setup,
            commands::set_password,
            commands::verify_password,
            commands::is_authenticated,
            commands::logout,
            commands::reset_application,
            commands::get_all_notes,
            commands::get_notes_by_category,
            commands::search_notes,
            commands::get_note,
            commands::create_note,
            commands::create_note_with_category,
            commands::update_note,
            commands::update_note_category,
            commands::move_to_trash,
            commands::restore_from_trash,
            commands::delete_note,
            commands::permanently_delete_note,
            commands::get_settings,
            commands::update_settings,
            commands::create_backup,
            commands::save_image_from_clipboard,
            commands::get_image,
            commands::get_image_as_data_url,
            commands::delete_image,
            commands::list_images,
            commands::cleanup_orphaned_images,
            commands::get_image_stats,
        ])
        .build(tauri::generate_context!())
        .expect("error while building gote");

    app.run(|handle, event| {
        if let RunEvent::Exit = event {
            // Drops the file watcher.
            lock_core(&handle.state::<Mutex<AppCore>>()).shutdown();
        }
    });
}
