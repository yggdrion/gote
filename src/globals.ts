import { type Config } from "./types";
import { type NoteStore } from "./noteStore";

// Global application state
export let store: NoteStore;
export let currentConfig: Config;

export function setStore(newStore: NoteStore) {
  store = newStore;
}

export function setConfig(newConfig: Config) {
  currentConfig = newConfig;
}
