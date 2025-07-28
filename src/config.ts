import { join, dirname } from "path";
import { existsSync, mkdirSync } from "fs";
import { type Config } from "./types";

function getDefaultDataPath(): string {
  return "./data";
}

function getDefaultPasswordHashPath(): string {
  const homeDir = process.env.HOME || process.env.USERPROFILE;
  if (!homeDir) {
    return join("./data", ".password_hash");
  }

  let configDir: string;
  if (process.platform === "win32") {
    configDir = process.env.APPDATA || homeDir;
  } else {
    configDir = process.env.XDG_CONFIG_HOME || join(homeDir, ".config");
  }

  const configPath = join(configDir, "gote");
  if (!existsSync(configPath)) {
    try {
      mkdirSync(configPath, { recursive: true });
    } catch (err) {
      return join("./data", ".password_hash");
    }
  }

  return join(configPath, "gote_password_hash");
}

function getConfigFilePath(): string {
  const homeDir = process.env.HOME || process.env.USERPROFILE;
  if (!homeDir) {
    return "./config.json";
  }

  let configDir: string;
  if (process.platform === "win32") {
    configDir = process.env.APPDATA || homeDir;
  } else {
    configDir = process.env.XDG_CONFIG_HOME || join(homeDir, ".config");
  }

  const configPath = join(configDir, "gote");
  if (!existsSync(configPath)) {
    try {
      mkdirSync(configPath, { recursive: true });
    } catch (err) {
      return "./config.json";
    }
  }

  return join(configPath, "config.json");
}

export async function loadConfig(): Promise<Config> {
  const config: Config = {
    notesPath: getDefaultDataPath(),
    passwordHashPath: getDefaultPasswordHashPath(),
    configFilePath: getConfigFilePath(),
  };

  const configFile = getConfigFilePath();
  try {
    const file = Bun.file(configFile);
    if (await file.exists()) {
      const data = await file.json();
      Object.assign(config, data);
    }
  } catch (err) {
    console.warn("Could not load config file:", err);
  }

  // Ensure directories exist
  if (!existsSync(config.notesPath)) {
    try {
      mkdirSync(config.notesPath, { recursive: true });
    } catch (err) {
      console.warn(`Could not create data directory ${config.notesPath}:`, err);
    }
  }

  const passwordDir = dirname(config.passwordHashPath);
  if (!existsSync(passwordDir)) {
    try {
      mkdirSync(passwordDir, { recursive: true });
    } catch (err) {
      console.warn(
        `Could not create password hash directory ${passwordDir}:`,
        err
      );
    }
  }

  return config;
}

export async function saveConfig(config: Config): Promise<void> {
  const configFile = getConfigFilePath();
  const configDir = dirname(configFile);

  if (!existsSync(configDir)) {
    mkdirSync(configDir, { recursive: true });
  }

  await Bun.write(configFile, JSON.stringify(config, null, 2));
}
