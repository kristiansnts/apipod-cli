#!/usr/bin/env node

import inquirer from "inquirer";
import chalk from "chalk";
import fs from "fs";
import path from "path";
import os from "os";

const PROXY_BASE_URL = "http://localhost:8081";

const CLAUDE_SETTINGS_PATH = path.join(os.homedir(), ".claude", "settings.json");
const OPENCODE_CONFIG_PATH = path.join(os.homedir(), ".config", "opencode", "opencode.json");
const OPENCODE_AUTH_PATH = path.join(os.homedir(), ".local", "share", "opencode", "auth.json");

function readJson(filePath) {
  try {
    const content = fs.readFileSync(filePath, "utf-8");
    return JSON.parse(content);
  } catch {
    return null;
  }
}

function writeJson(filePath, data) {
  const dir = path.dirname(filePath);
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2) + "\n", "utf-8");
}

async function promptApiKey() {
  const { apiKey } = await inquirer.prompt([
    {
      type: "password",
      name: "apiKey",
      message: "Enter your API key:",
      mask: "*",
      validate: (val) => (val.length > 0 ? true : "API key cannot be empty"),
    },
  ]);
  return apiKey;
}

// ── Connect to Claude Code (CLI) ──
async function connectClaudeCode() {
  console.log(chalk.cyan("\n⚡ Connect to Claude Code (CLI)\n"));

  const apiKey = await promptApiKey();
  const settings = readJson(CLAUDE_SETTINGS_PATH) || {};

  settings.env = settings.env || {};
  settings.env.ANTHROPIC_BASE_URL = PROXY_BASE_URL;
  settings.env.ANTHROPIC_API_KEY = apiKey;

  writeJson(CLAUDE_SETTINGS_PATH, settings);
  console.log(chalk.green(`\n✔ Claude Code settings updated`));
  console.log(chalk.gray(`  → ${CLAUDE_SETTINGS_PATH}`));
  console.log(chalk.gray(`  → Base URL: ${PROXY_BASE_URL}`));
}


// ── Connect to OpenCode ──
async function connectOpenCode() {
  console.log(chalk.cyan("\n⚡ Connect to OpenCode\n"));

  const apiKey = await promptApiKey();

  // Write auth credentials
  const auth = readJson(OPENCODE_AUTH_PATH) || {};
  auth.anthropic = { type: "api", key: apiKey };
  writeJson(OPENCODE_AUTH_PATH, auth);

  // Write config with proxy baseURL
  const config = readJson(OPENCODE_CONFIG_PATH) || {};
  config["$schema"] = "https://opencode.ai/config.json";
  config.provider = config.provider || {};
  config.provider.anthropic = config.provider.anthropic || {};
  config.provider.anthropic.options = config.provider.anthropic.options || {};
  config.provider.anthropic.options.baseURL = `${PROXY_BASE_URL}/v1`;
  writeJson(OPENCODE_CONFIG_PATH, config);

  console.log(chalk.green(`\n✔ OpenCode settings updated`));
  console.log(chalk.gray(`  → Auth: ${OPENCODE_AUTH_PATH}`));
  console.log(chalk.gray(`  → Config: ${OPENCODE_CONFIG_PATH}`));
  console.log(chalk.gray(`  → Base URL: ${PROXY_BASE_URL}/v1`));
}

// ── Connect (sub-menu) ──
async function connect() {
  const { target } = await inquirer.prompt([
    {
      type: "list",
      name: "target",
      message: "Connect to:",
      choices: [
        { name: "🟠 Claude Code", value: "claude" },
        { name: "🔵 OpenCode", value: "opencode" },
        { name: "↩  Back", value: "back" },
      ],
    },
  ]);

  switch (target) {
    case "claude":
      await connectClaudeCode();
      break;
    case "opencode":
      await connectOpenCode();
      break;
  }
}

// ── Reset ──
async function resetClaudeCode() {
  const settings = readJson(CLAUDE_SETTINGS_PATH);
  if (settings?.env) {
    delete settings.env.ANTHROPIC_BASE_URL;
    delete settings.env.ANTHROPIC_API_KEY;
    if (Object.keys(settings.env).length === 0) delete settings.env;
    writeJson(CLAUDE_SETTINGS_PATH, settings);
    console.log(chalk.green("✔ Claude Code settings reset"));
  } else {
    console.log(chalk.yellow("⚠ No Claude Code proxy settings found"));
  }
}

async function resetOpenCode() {
  let found = false;

  const config = readJson(OPENCODE_CONFIG_PATH);
  if (config?.provider?.anthropic?.options?.baseURL) {
    found = true;
    delete config.provider.anthropic.options.baseURL;
    if (Object.keys(config.provider.anthropic.options).length === 0) delete config.provider.anthropic.options;
    if (Object.keys(config.provider.anthropic).length === 0) delete config.provider.anthropic;
    if (Object.keys(config.provider).length === 0) delete config.provider;
    writeJson(OPENCODE_CONFIG_PATH, config);
  }

  const auth = readJson(OPENCODE_AUTH_PATH);
  if (auth?.anthropic) {
    found = true;
    delete auth.anthropic;
    writeJson(OPENCODE_AUTH_PATH, auth);
  }

  if (found) {
    console.log(chalk.green("✔ OpenCode settings reset"));
  } else {
    console.log(chalk.yellow("⚠ No OpenCode proxy settings found"));
  }
}

async function reset() {
  const { target } = await inquirer.prompt([
    {
      type: "list",
      name: "target",
      message: "Reset settings for:",
      choices: [
        { name: "🟠 Claude Code", value: "claude" },
        { name: "🔵 OpenCode", value: "opencode" },
        { name: "↩  Back", value: "back" },
      ],
    },
  ]);

  if (target === "back") return;

  const { confirm } = await inquirer.prompt([
    {
      type: "confirm",
      name: "confirm",
      message: `Reset ${target === "claude" ? "Claude Code" : "OpenCode"} settings?`,
      default: false,
    },
  ]);

  if (!confirm) return;

  if (target === "claude") await resetClaudeCode();
  else await resetOpenCode();
}

async function changeToken() {
  const { target } = await inquirer.prompt([
    {
      type: "list",
      name: "target",
      message: "Change token for:",
      choices: [
        { name: "🟠 Claude Code", value: "claude" },
        { name: "🔵 OpenCode", value: "opencode" },
        { name: "↩  Back", value: "back" },
      ],
    },
  ]);

  if (target === "back") return;

  const apiKey = await promptApiKey();

  if (target === "claude") {
    const settings = readJson(CLAUDE_SETTINGS_PATH) || {};
    settings.env = settings.env || {};
    settings.env.ANTHROPIC_API_KEY = apiKey;
    writeJson(CLAUDE_SETTINGS_PATH, settings);
    console.log(chalk.green("✔ Claude Code API key updated"));
  } else {
    const auth = readJson(OPENCODE_AUTH_PATH) || {};
    auth.anthropic = { type: "api", key: apiKey };
    writeJson(OPENCODE_AUTH_PATH, auth);
    console.log(chalk.green("✔ OpenCode API key updated"));
  }
}

// ── Help ──
function showHelp() {
  console.log(`
${chalk.bold.cyan("apipod-cli")} — API Proxy Connector for Claude

${chalk.bold("Commands:")}
  ${chalk.yellow("Connect")}                 Configure Claude Code or OpenCode to use the proxy
  ${chalk.yellow("Reset")}                    Remove proxy settings
  ${chalk.yellow("Change Token")}             Update the API key
  ${chalk.yellow("Help")}                     Show this help message
  ${chalk.yellow("Exit")}                     Quit the CLI

${chalk.bold("Proxy URL:")} ${chalk.gray(PROXY_BASE_URL)}

${chalk.bold("Config files:")}
  Claude Code: ${chalk.gray(CLAUDE_SETTINGS_PATH)}
  OpenCode:    ${chalk.gray(OPENCODE_CONFIG_PATH)}
`);
}

// ── Main Menu ──
async function main() {
  console.log(chalk.bold.cyan("\n  apipod-cli") + chalk.gray(" v1.0.0\n"));

  while (true) {
    const { action } = await inquirer.prompt([
      {
        type: "list",
        name: "action",
        message: "What would you like to do?",
        choices: [
          { name: "⚡ Connect", value: "connect" },
          { name: "🔄 Reset", value: "reset" },
          { name: "🔑 Change Token", value: "token" },
          { name: "❓ Help", value: "help" },
          { name: "👋 Exit", value: "exit" },
        ],
      },
    ]);

    switch (action) {
      case "connect":
        await connect();
        break;
      case "reset":
        await reset();
        break;
      case "token":
        await changeToken();
        break;
      case "help":
        showHelp();
        break;
      case "exit":
        console.log(chalk.gray("\nGoodbye! 👋\n"));
        process.exit(0);
    }

    console.log("");
  }
}

main().catch((err) => {
  if (err.name === "ExitPromptError") process.exit(0);
  console.error(chalk.red("Error:"), err.message);
  process.exit(1);
});
