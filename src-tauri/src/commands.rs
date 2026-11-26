//! Thin `#[tauri::command]` wrappers over `AppCore`. JS argument names are the
//! camelCase forms of the parameter names (Tauri converts them).

use std::sync::{Mutex, MutexGuard};

use tauri::State;

use crate::app::{AppCore, ImageData, ImageStats, Settings};
use crate::error::Result;
use crate::models::{Image, NoteDto};

type Core<'a> = State<'a, Mutex<AppCore>>;

fn lock<'a>(state: &'a Core<'_>) -> MutexGuard<'a, AppCore> {
    state.lock().unwrap_or_else(|e| e.into_inner())
}

#[tauri::command(async)]
pub fn is_configured(state: Core<'_>) -> bool {
    lock(&state).is_configured()
}

#[tauri::command(async)]
pub fn is_password_set(state: Core<'_>) -> bool {
    lock(&state).is_password_set()
}

#[tauri::command(async)]
pub fn complete_initial_setup(
    state: Core<'_>,
    notes_path: String,
    password_hash_path: String,
    password: String,
    confirm_password: String,
) -> Result<()> {
    lock(&state).complete_initial_setup(
        &notes_path,
        &password_hash_path,
        &password,
        &confirm_password,
    )
}

#[tauri::command(async)]
pub fn set_password(state: Core<'_>, password: String) -> Result<()> {
    lock(&state).set_password(&password)
}

#[tauri::command(async)]
pub fn verify_password(state: Core<'_>, password: String) -> bool {
    lock(&state).verify_password(&password)
}

#[tauri::command(async)]
pub fn is_authenticated(state: Core<'_>) -> bool {
    lock(&state).is_authenticated()
}

#[tauri::command(async)]
pub fn logout(state: Core<'_>) {
    lock(&state).logout()
}

#[tauri::command(async)]
pub fn reset_application(state: Core<'_>) -> Result<()> {
    lock(&state).reset_application()
}

#[tauri::command(async)]
pub fn get_all_notes(state: Core<'_>) -> Vec<NoteDto> {
    lock(&state).get_all_notes()
}

#[tauri::command(async)]
pub fn get_notes_by_category(state: Core<'_>, category: String) -> Vec<NoteDto> {
    lock(&state).get_notes_by_category(&category)
}

#[tauri::command(async)]
pub fn search_notes(state: Core<'_>, query: String) -> Vec<NoteDto> {
    lock(&state).search_notes(&query)
}

#[tauri::command(async)]
pub fn get_note(state: Core<'_>, id: String) -> Result<NoteDto> {
    lock(&state).get_note(&id)
}

#[tauri::command(async)]
pub fn create_note(state: Core<'_>, content: String) -> Result<NoteDto> {
    lock(&state).create_note(&content)
}

#[tauri::command(async)]
pub fn create_note_with_category(
    state: Core<'_>,
    content: String,
    category: String,
) -> Result<NoteDto> {
    lock(&state).create_note_with_category(&content, &category)
}

#[tauri::command(async)]
pub fn update_note(state: Core<'_>, id: String, content: String) -> Result<NoteDto> {
    lock(&state).update_note(&id, &content)
}

#[tauri::command(async)]
pub fn update_note_category(state: Core<'_>, id: String, category: String) -> Result<NoteDto> {
    lock(&state).update_note_category(&id, &category)
}

#[tauri::command(async)]
pub fn move_to_trash(state: Core<'_>, id: String) -> Result<NoteDto> {
    lock(&state).move_to_trash(&id)
}

#[tauri::command(async)]
pub fn restore_from_trash(state: Core<'_>, id: String) -> Result<NoteDto> {
    lock(&state).restore_from_trash(&id)
}

#[tauri::command(async)]
pub fn delete_note(state: Core<'_>, id: String) -> Result<()> {
    lock(&state).delete_note(&id)
}

#[tauri::command(async)]
pub fn permanently_delete_note(state: Core<'_>, id: String) -> Result<()> {
    lock(&state).permanently_delete_note(&id)
}

#[tauri::command(async)]
pub fn get_settings(state: Core<'_>) -> Settings {
    lock(&state).get_settings()
}

#[tauri::command(async)]
pub fn update_settings(
    state: Core<'_>,
    notes_path: String,
    password_hash_path: String,
) -> Result<()> {
    lock(&state).update_settings(&notes_path, &password_hash_path)
}

#[tauri::command(async)]
pub fn create_backup(state: Core<'_>) -> Result<String> {
    // Release the AppCore mutex before the (slow) backup runs.
    let job = lock(&state).begin_backup()?;
    job.run()
}

#[tauri::command(async)]
pub fn save_image_from_clipboard(
    state: Core<'_>,
    image_data: String,
    content_type: String,
) -> Result<Image> {
    lock(&state).save_image_from_clipboard(&image_data, &content_type)
}

#[tauri::command(async)]
pub fn get_image(state: Core<'_>, image_id: String) -> Result<ImageData> {
    lock(&state).get_image(&image_id)
}

#[tauri::command(async)]
pub fn get_image_as_data_url(state: Core<'_>, image_id: String) -> Result<String> {
    lock(&state).get_image_as_data_url(&image_id)
}

#[tauri::command(async)]
pub fn delete_image(state: Core<'_>, image_id: String) -> Result<()> {
    lock(&state).delete_image(&image_id)
}

#[tauri::command(async)]
pub fn list_images(state: Core<'_>) -> Result<Vec<Image>> {
    lock(&state).list_images()
}

#[tauri::command(async)]
pub fn cleanup_orphaned_images(state: Core<'_>) -> Result<usize> {
    lock(&state).cleanup_orphaned_images()
}

#[tauri::command(async)]
pub fn get_image_stats(state: Core<'_>) -> Result<ImageStats> {
    lock(&state).get_image_stats()
}
