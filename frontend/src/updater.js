// In-app updates via the Tauri updater plugin. Releases publish a signed
// latest.json; the app only installs updates signed with the key in
// tauri.conf.json (plugins.updater.pubkey).
import { getVersion } from "@tauri-apps/api/app";
import { check } from "@tauri-apps/plugin-updater";
import { relaunch } from "@tauri-apps/plugin-process";

export function getAppVersion() {
  return getVersion();
}

// With `silent`, only an available update produces a dialog; "up to date" and
// network errors are logged instead (used for the startup check).
export async function checkForUpdates({ silent = false } = {}) {
  let update;
  try {
    update = await check();
  } catch (error) {
    console.warn("Update check failed:", error);
    if (!silent) alert("Could not check for updates: " + (error?.message || error));
    return;
  }

  if (!update) {
    if (!silent) alert("gote is up to date.");
    return;
  }

  const ok = confirm(
    `gote ${update.version} is available (you have ${update.currentVersion}).\n\n` +
      "Install it now? The app will restart."
  );
  if (!ok) return;

  try {
    await update.downloadAndInstall();
    await relaunch();
  } catch (error) {
    console.error("Update failed:", error);
    alert("Update failed: " + (error?.message || error));
  }
}
