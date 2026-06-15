#!/usr/bin/env node
"use strict";

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const exe = process.platform === "win32" ? "xelyon.exe" : "xelyon";
const binary = path.join(__dirname, "..", "vendor", exe);

if (!fs.existsSync(binary)) {
  console.error("xelyon binary is missing. Reinstall the package or run `npm rebuild xelyon`.");
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
