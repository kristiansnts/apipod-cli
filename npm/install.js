const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const https = require("https");
const http = require("http");

const VERSION = require("./package.json").version;
const REPO = "rpay/apipod-cli";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

function getBinaryName() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!platform || !arch) {
    console.error(`Unsupported platform: ${process.platform}/${process.arch}`);
    process.exit(1);
  }

  let name = `apipod-cli-${platform}-${arch}`;
  if (process.platform === "win32") {
    name += ".exe";
  }
  return name;
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const client = url.startsWith("https") ? https : http;
    client.get(url, { headers: { "User-Agent": "apipod-cli-installer" } }, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        return download(res.headers.location, dest).then(resolve).catch(reject);
      }
      if (res.statusCode !== 200) {
        return reject(new Error(`Download failed: HTTP ${res.statusCode}`));
      }
      const file = fs.createWriteStream(dest);
      res.pipe(file);
      file.on("finish", () => {
        file.close();
        resolve();
      });
    }).on("error", reject);
  });
}

async function main() {
  const binaryName = getBinaryName();
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${binaryName}`;
  const binDir = path.join(__dirname, "bin");
  const dest = path.join(binDir, process.platform === "win32" ? "apipod-cli.exe" : "apipod-cli");

  fs.mkdirSync(binDir, { recursive: true });

  console.log(`Downloading apipod-cli v${VERSION} for ${process.platform}/${process.arch}...`);
  console.log(`  ${url}`);

  try {
    await download(url, dest);
    fs.chmodSync(dest, 0o755);
    console.log(`✓ Installed apipod-cli to ${dest}`);
  } catch (err) {
    console.error(`✗ Failed to download: ${err.message}`);
    console.error("  You can build from source: go install github.com/rpay/apipod-cli/cmd/apipod-cli@latest");
    process.exit(1);
  }
}

main();
