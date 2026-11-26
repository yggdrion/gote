// Thin wrappers around Tauri commands. They keep the PascalCase names of the
// previous backend bindings, so main.js only changes its imports.
import { invoke } from "@tauri-apps/api/core";

// Tauri rejects with a bare string; main.js reads `error.message`.
function call(cmd, args) {
  return invoke(cmd, args).catch((e) => {
    throw e instanceof Error ? e : new Error(String(e));
  });
}

export const IsConfigured = () => call("is_configured");
export const IsPasswordSet = () => call("is_password_set");
export const CompleteInitialSetup = (
  notesPath,
  passwordHashPath,
  password,
  confirmPassword
) =>
  call("complete_initial_setup", {
    notesPath,
    passwordHashPath,
    password,
    confirmPassword,
  });
export const SetPassword = (password) => call("set_password", { password });
export const VerifyPassword = (password) =>
  call("verify_password", { password });
export const IsAuthenticated = () => call("is_authenticated");
export const Logout = () => call("logout");
export const ResetApplication = () => call("reset_application");

export const GetAllNotes = () => call("get_all_notes");
export const GetNotesByCategory = (category) =>
  call("get_notes_by_category", { category });
export const SearchNotes = (query) => call("search_notes", { query });
export const GetNote = (id) => call("get_note", { id });
export const CreateNote = (content) => call("create_note", { content });
export const CreateNoteWithCategory = (content, category) =>
  call("create_note_with_category", { content, category });
export const UpdateNote = (id, content) => call("update_note", { id, content });
export const UpdateNoteCategory = (id, category) =>
  call("update_note_category", { id, category });
export const MoveToTrash = (id) => call("move_to_trash", { id });
export const RestoreFromTrash = (id) => call("restore_from_trash", { id });
export const DeleteNote = (id) => call("delete_note", { id });
export const PermanentlyDeleteNote = (id) =>
  call("permanently_delete_note", { id });

export const GetSettings = () => call("get_settings");
export const UpdateSettings = (notesPath, passwordHashPath) =>
  call("update_settings", { notesPath, passwordHashPath });
export const CreateBackup = () => call("create_backup");

export const SaveImageFromClipboard = (imageData, contentType) =>
  call("save_image_from_clipboard", { imageData, contentType });
export const GetImage = (imageId) => call("get_image", { imageId });
export const GetImageAsDataURL = (imageId) =>
  call("get_image_as_data_url", { imageId });
export const DeleteImage = (imageId) => call("delete_image", { imageId });
export const ListImages = () => call("list_images");
export const CleanupOrphanedImages = () => call("cleanup_orphaned_images");
export const GetImageStats = () => call("get_image_stats");
