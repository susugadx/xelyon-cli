#!/usr/bin/env node
"use strict";

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { execFileSync } = require("node:child_process");
const {
  downloadFile,
  fetchJSON,
  fetchText,
  findAssets,
  releaseApiURL,
  resolvePlatform,
  verifyChecksum,
  vendorDir,
} = require("./download");

async function main() {
  if (process.env.XELYON_NPM_SKIP_DOWNLOAD === "1") {
    return;
  }

  const pkg = require("../package.json");
  const version = process.env.XELYON_RELEASE_VERSION || pkg.version;
  if (version === "0.0.0" && !process.env.XELYON_RELEASE_VERSION) {
    console.warn("Skipping xelyon binary download for development package version 0.0.0.");
    return;
  }

  const target = resolvePlatform();
  const release = await fetchJSON(releaseApiURL(version));
  const assets = findAssets(release, target);
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), "xelyon-npm-"));
  try {
    const archive = path.join(tmp, assets.assetName);
    await downloadFile(assets.assetURL, archive);
    const checksumsText = await fetchText(assets.checksumURL);
    verifyChecksum(archive, checksumsText, assets.assetName);

    const extractDir = path.join(tmp, "extract");
    fs.mkdirSync(extractDir);
    if (assets.assetName.endsWith(".zip")) {
      execFileSync("powershell", ["-NoProfile", "-Command", `Expand-Archive -LiteralPath '${archive.replace(/'/g, "''")}' -DestinationPath '${extractDir.replace(/'/g, "''")}' -Force`], { stdio: "inherit" });
    } else {
      execFileSync("tar", ["-xzf", archive, "-C", extractDir], { stdio: "inherit" });
    }

    const binary = findBinary(extractDir, target.exe);
    if (!binary) {
      throw new Error(`${target.exe} not found in release archive`);
    }
    fs.mkdirSync(vendorDir(), { recursive: true });
    const destination = path.join(vendorDir(), target.exe);
    fs.copyFileSync(binary, destination);
    fs.chmodSync(destination, 0o755);
  } finally {
    fs.rmSync(tmp, { recursive: true, force: true });
  }
}

function findBinary(root, name) {
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    const full = path.join(root, entry.name);
    if (entry.isDirectory()) {
      const found = findBinary(full, name);
      if (found) {
        return found;
      }
    } else if (entry.isFile() && entry.name === name) {
      return full;
    }
  }
  return "";
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
