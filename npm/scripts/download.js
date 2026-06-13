"use strict";

const crypto = require("node:crypto");
const dns = require("node:dns");
const fs = require("node:fs");
const https = require("node:https");
const net = require("node:net");
const path = require("node:path");

const repo = "susugadx/xelyon-cli";
const userAgentHeaders = { "User-Agent": "xelyon-npm-installer" };
const maxRedirects = 5;
const maxTextBytes = 10 * 1024 * 1024;
const maxDownloadBytes = 512 * 1024 * 1024;

function resolvePlatform(platform = process.platform, arch = process.arch) {
  const osMap = {
    linux: "linux",
    darwin: "darwin",
    win32: "windows",
  };
  const archMap = {
    x64: "amd64",
    arm64: "arm64",
  };
  const os = osMap[platform];
  const goarch = archMap[arch];
  if (!os || !goarch || (os === "windows" && goarch !== "amd64")) {
    throw new Error(`Unsupported platform: ${platform}/${arch}`);
  }
  return { os, arch: goarch, extension: os === "windows" ? "zip" : "tar.gz", exe: os === "windows" ? "xelyon.exe" : "xelyon" };
}

function releaseApiURL(version) {
  if (!version || version === "latest") {
    return `https://api.github.com/repos/${repo}/releases/latest`;
  }
  const tag = version.startsWith("v") ? version : `v${version}`;
  return `https://api.github.com/repos/${repo}/releases/tags/${tag}`;
}

function findAssets(release, target) {
  const assets = Array.isArray(release.assets) ? release.assets : [];
  const suffix = `_${target.os}_${target.arch}.${target.extension}`;
  const asset = assets.find((entry) => typeof entry.name === "string" && entry.name.endsWith(suffix));
  const checksums = assets.find((entry) => entry.name === "checksums.txt");
  if (!asset || !checksums) {
    throw new Error(`Release asset not found for ${target.os}/${target.arch}`);
  }
  return {
    assetName: asset.name,
    assetURL: asset.browser_download_url,
    checksumURL: checksums.browser_download_url,
  };
}

function parseChecksum(checksumsText, assetName) {
  const line = checksumsText.split(/\r?\n/).find((value) => value.trim().endsWith(` ${assetName}`));
  if (!line) {
    throw new Error(`Checksum entry not found for ${assetName}`);
  }
  return line.trim().split(/\s+/)[0].toLowerCase();
}

function sha256File(file) {
  const hash = crypto.createHash("sha256");
  hash.update(fs.readFileSync(file));
  return hash.digest("hex");
}

function verifyChecksum(file, checksumsText, assetName) {
  const expected = parseChecksum(checksumsText, assetName);
  const actual = sha256File(file);
  if (actual !== expected) {
    throw new Error(`Checksum mismatch for ${assetName}: expected ${expected} actual ${actual}`);
  }
}

function fetchJSON(url) {
  return fetchText(url).then((text) => JSON.parse(text));
}

function fetchText(url, redirectCount = 0) {
  if (url.startsWith("file://")) {
    return fs.promises.readFile(new URL(url), "utf8");
  }
  return new Promise((resolve, reject) => {
    let client;
    let requestOptions;
    try {
      client = remoteClient(url);
      requestOptions = remoteRequestOptions(url);
    } catch (error) {
      reject(error);
      return;
    }
    const request = client.get(url, requestOptions, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        response.resume();
        let redirectURL;
        try {
          redirectURL = resolveRedirectURL(url, response.headers.location, redirectCount);
        } catch (error) {
          reject(error);
          return;
        }
        fetchText(redirectURL, redirectCount + 1).then(resolve, reject);
        return;
      }
      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`HTTP ${response.statusCode} for ${url}`));
        return;
      }
      if (contentLengthExceeds(response, maxTextBytes)) {
        response.resume();
        reject(new Error(`Response too large for ${url}`));
        return;
      }
      response.setEncoding("utf8");
      let body = "";
      let bodyBytes = 0;
      let settled = false;
      function rejectText(error) {
        if (settled) {
          return;
        }
        settled = true;
        response.destroy(error);
        reject(error);
      }
      response.on("error", rejectText);
      response.on("data", (chunk) => {
        bodyBytes += Buffer.byteLength(chunk);
        if (bodyBytes > maxTextBytes) {
          rejectText(new Error(`Response too large for ${url}`));
          return;
        }
        body += chunk;
      });
      response.on("end", () => {
        if (settled) {
          return;
        }
        settled = true;
        resolve(body);
      });
    });
    request.setTimeout(30000, () => {
      request.destroy(new Error(`Timeout fetching ${url}`));
    });
    request.on("error", reject);
  });
}

function downloadFile(url, destination, redirectCount = 0) {
  if (url.startsWith("file://")) {
    fs.copyFileSync(new URL(url), destination);
    return Promise.resolve();
  }
  return new Promise((resolve, reject) => {
    let client;
    let requestOptions;
    let file;
    let settled = false;

    function rejectWithCleanup(error) {
      if (settled) {
        return;
      }
      settled = true;
      if (file) {
        file.destroy();
        fs.rmSync(destination, { force: true });
      }
      reject(error);
    }

    try {
      client = remoteClient(url);
      requestOptions = remoteRequestOptions(url);
    } catch (error) {
      reject(error);
      return;
    }
    const request = client.get(url, requestOptions, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        response.resume();
        let redirectURL;
        try {
          redirectURL = resolveRedirectURL(url, response.headers.location, redirectCount);
        } catch (error) {
          reject(error);
          return;
        }
        downloadFile(redirectURL, destination, redirectCount + 1).then(resolve, reject);
        return;
      }
      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`HTTP ${response.statusCode} for ${url}`));
        return;
      }
      if (contentLengthExceeds(response, maxDownloadBytes)) {
        response.resume();
        reject(new Error(`Download too large for ${url}`));
        return;
      }

      file = fs.createWriteStream(destination, { mode: 0o644 });
      let downloadedBytes = 0;
      response.on("error", rejectWithCleanup);
      response.on("data", (chunk) => {
        downloadedBytes += chunk.length;
        if (downloadedBytes > maxDownloadBytes) {
          rejectWithCleanup(new Error(`Download too large for ${url}`));
          response.destroy();
        }
      });
      file.on("error", rejectWithCleanup);
      response.pipe(file);
      file.on("finish", () => {
        file.close((error) => {
          if (error) {
            rejectWithCleanup(error);
            return;
          }
          if (settled) {
            return;
          }
          settled = true;
          resolve();
        });
      });
    });
    request.setTimeout(30000, () => {
      request.destroy(new Error(`Timeout downloading ${url}`));
    });
    request.on("error", rejectWithCleanup);
  });
}

function remoteClient(url) {
  const parsed = validateRemoteURL(url);
  if (parsed.protocol === "https:") {
    return https;
  }
  if (parsed.protocol === "http:") {
    throw new Error(`Refusing non-HTTPS URL: ${url}`);
  }
  throw new Error(`Unsupported URL: ${url}`);
}

function remoteRequestOptions(url) {
  validateRemoteURL(url);
  return {
    headers: userAgentHeaders,
    lookup: safeLookup,
  };
}

function validateRemoteURL(url) {
  let parsed;
  try {
    parsed = new URL(url);
  } catch (error) {
    throw new Error(`Unsupported URL: ${url}`);
  }
  if (parsed.protocol === "http:") {
    throw new Error(`Refusing non-HTTPS URL: ${url}`);
  }
  if (parsed.protocol !== "https:") {
    throw new Error(`Unsupported URL: ${url}`);
  }
  validateHostname(parsed.hostname, url);
  return parsed;
}

function resolveRedirectURL(baseURL, location, redirectCount) {
  if (redirectCount >= maxRedirects) {
    throw new Error(`Too many redirects for ${baseURL}`);
  }
  const redirectURL = new URL(location, baseURL).toString();
  validateRemoteURL(redirectURL);
  return redirectURL;
}

function contentLengthExceeds(response, maxBytes) {
  const value = Number(response.headers["content-length"]);
  return Number.isFinite(value) && value > maxBytes;
}

function validateHostname(hostname, url) {
  const normalized = normalizeHostname(hostname);
  if (!normalized) {
    throw new Error(`Unsupported URL: ${url}`);
  }
  if (isLocalHostname(normalized)) {
    throw new Error(`Refusing local hostname: ${hostname}`);
  }
  if (net.isIP(normalized) && isBlockedIPAddress(normalized)) {
    throw new Error(`Refusing private or local IP address: ${hostname}`);
  }
}

function normalizeHostname(hostname) {
  return String(hostname || "")
    .trim()
    .replace(/^\[/, "")
    .replace(/\]$/, "")
    .replace(/\.$/, "")
    .toLowerCase();
}

function isLocalHostname(hostname) {
  return hostname === "localhost" || hostname.endsWith(".localhost") || hostname === "metadata.google.internal";
}

function safeLookup(hostname, options, callback) {
  const normalized = normalizeHostname(hostname);
  if (isLocalHostname(normalized)) {
    callback(new Error(`Refusing local hostname: ${hostname}`));
    return;
  }
  dns.lookup(hostname, options, (error, address, family) => {
    if (error) {
      callback(error);
      return;
    }
    let records;
    try {
      records = lookupAddressRecords(address, family, hostname);
    } catch (lookupError) {
      callback(lookupError);
      return;
    }
    const blockedRecord = records.find((record) => isBlockedIPAddress(record.address));
    if (blockedRecord) {
      callback(new Error(`Refusing private or local IP address for ${hostname}: ${blockedRecord.address}`));
      return;
    }
    if (Array.isArray(address)) {
      callback(null, address);
      return;
    }
    callback(null, address, family);
  });
}

function lookupAddressRecords(address, family, hostname) {
  if (Array.isArray(address)) {
    if (address.length === 0) {
      throw new Error(`DNS lookup returned no addresses for ${hostname}`);
    }
    return address.map((entry) => {
      if (!entry || typeof entry.address !== "string" || !net.isIP(entry.address)) {
        throw new Error(`Invalid DNS address for ${hostname}`);
      }
      return { address: entry.address, family: entry.family };
    });
  }
  if (typeof address !== "string" || !net.isIP(address)) {
    throw new Error(`Invalid DNS address for ${hostname}`);
  }
  return [{ address, family }];
}

function isBlockedIPAddress(address) {
  const normalized = normalizeHostname(address);
  const version = net.isIP(normalized);
  if (version === 4) {
    const parts = normalized.split(".").map((part) => Number(part));
    if (parts.length !== 4 || parts.some((part) => !Number.isInteger(part) || part < 0 || part > 255)) {
      return true;
    }
    const [a, b] = parts;
    return (
      a === 0 ||
      a === 10 ||
      a === 127 ||
      (a === 100 && b >= 64 && b <= 127) ||
      (a === 169 && b === 254) ||
      (a === 172 && b >= 16 && b <= 31) ||
      (a === 192 && b === 168)
    );
  }
  if (version === 6) {
    if (normalized === "::1" || normalized === "::") {
      return true;
    }
    if (isIPv6LinkLocalAddress(normalized)) {
      return true;
    }
    if (normalized.startsWith("fc") || normalized.startsWith("fd")) {
      return true;
    }
    const ipv4Mapped = normalized.match(/^::ffff:(\d+\.\d+\.\d+\.\d+)$/);
    if (ipv4Mapped) {
      return isBlockedIPAddress(ipv4Mapped[1]);
    }
  }
  return false;
}

function isIPv6LinkLocalAddress(address) {
  const firstSegment = address.split(":")[0];
  if (!/^[0-9a-f]{1,4}$/.test(firstSegment)) {
    return false;
  }
  const firstHextet = Number.parseInt(firstSegment, 16);
  return (firstHextet & 0xffc0) === 0xfe80;
}

function vendorDir() {
  return path.join(__dirname, "..", "vendor");
}

module.exports = {
  downloadFile,
  fetchJSON,
  fetchText,
  findAssets,
  parseChecksum,
  releaseApiURL,
  resolvePlatform,
  sha256File,
  verifyChecksum,
  vendorDir,
};
