"use strict";

const assert = require("node:assert/strict");
const dns = require("node:dns");
const { EventEmitter } = require("node:events");
const fs = require("node:fs");
const https = require("node:https");
const os = require("node:os");
const path = require("node:path");
const { Readable } = require("node:stream");
const test = require("node:test");
const {
  downloadFile,
  fetchText,
  findAssets,
  parseChecksum,
  resolvePlatform,
  sha256File,
  verifyChecksum,
} = require("../scripts/download");

test("resolvePlatform maps supported npm platforms to release asset coordinates", () => {
  assert.deepEqual(resolvePlatform("linux", "x64"), { os: "linux", arch: "amd64", extension: "tar.gz", exe: "xelyon" });
  assert.deepEqual(resolvePlatform("darwin", "arm64"), { os: "darwin", arch: "arm64", extension: "tar.gz", exe: "xelyon" });
  assert.deepEqual(resolvePlatform("win32", "x64"), { os: "windows", arch: "amd64", extension: "zip", exe: "xelyon.exe" });
  assert.throws(() => resolvePlatform("win32", "arm64"), /Unsupported platform/);
});

test("findAssets selects matching binary and checksums", () => {
  const release = {
    assets: [
      { name: "xelyon-cli_1.2.3_linux_amd64.tar.gz", browser_download_url: "https://example.test/linux" },
      { name: "xelyon-cli_1.2.3_darwin_arm64.tar.gz", browser_download_url: "https://example.test/darwin" },
      { name: "checksums.txt", browser_download_url: "https://example.test/checksums" },
    ],
  };
  assert.deepEqual(findAssets(release, resolvePlatform("linux", "x64")), {
    assetName: "xelyon-cli_1.2.3_linux_amd64.tar.gz",
    assetURL: "https://example.test/linux",
    checksumURL: "https://example.test/checksums",
  });
});

test("checksum parser and verifier reject mismatches", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xelyon-npm-test-"));
  const file = path.join(dir, "asset.tar.gz");
  fs.writeFileSync(file, "payload");
  const sum = sha256File(file);
  verifyChecksum(file, `${sum}  asset.tar.gz\n`, "asset.tar.gz");
  assert.equal(parseChecksum(`${sum}  asset.tar.gz\n`, "asset.tar.gz"), sum);
  assert.throws(() => verifyChecksum(file, `${"0".repeat(64)}  asset.tar.gz\n`, "asset.tar.gz"), /Checksum mismatch/);
});

test("downloadFile supports local fixture URLs", async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xelyon-npm-download-"));
  const source = path.join(dir, "source.bin");
  const dest = path.join(dir, "dest.bin");
  fs.writeFileSync(source, "fixture");
  await downloadFile(new URL(`file://${source}`).toString(), dest);
  assert.equal(fs.readFileSync(dest, "utf8"), "fixture");
});

test("downloadFile follows HTTPS redirects and writes the final payload", async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xelyon-npm-redirect-"));
  const dest = path.join(dir, "asset.bin");

  await withMockHTTPS({
    "/redirect": { statusCode: 302, headers: { location: "/asset" } },
    "/asset": { statusCode: 200, body: "redirected payload" },
  }, async () => {
    await downloadFile("https://downloads.example.test/redirect", dest);
  });

  assert.equal(fs.readFileSync(dest, "utf8"), "redirected payload");
});

test("downloadFile rejects HTTPS redirect loops without deleting an existing destination", async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xelyon-npm-redirect-loop-"));
  const dest = path.join(dir, "asset.bin");
  fs.writeFileSync(dest, "existing payload");

  await withMockHTTPS({
    "/loop": { statusCode: 302, headers: { location: "/loop" } },
  }, async () => {
    await assert.rejects(() => downloadFile("https://downloads.example.test/loop", dest), /Too many redirects/);
  });

  assert.equal(fs.readFileSync(dest, "utf8"), "existing payload");
});

test("downloadFile rejects oversized HTTPS responses without deleting an existing destination", async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xelyon-npm-large-download-"));
  const dest = path.join(dir, "asset.bin");
  fs.writeFileSync(dest, "existing payload");

  await withMockHTTPS({
    "/large": { statusCode: 200, headers: { "content-length": String(1024 * 1024 * 1024) } },
  }, async () => {
    await assert.rejects(() => downloadFile("https://downloads.example.test/large", dest), /Download too large/);
  });

  assert.equal(fs.readFileSync(dest, "utf8"), "existing payload");
});

test("remote downloads reject private addresses from DNS array lookups", async () => {
  await withMockDNSLookup([
    { address: "203.0.113.10", family: 4 },
    { address: "127.0.0.1", family: 4 },
  ], async () => {
    await withMockHTTPS({
      "/checksums.txt": { statusCode: 200, body: "ignored", lookupOptions: { all: true } },
    }, async () => {
      await assert.rejects(() => fetchText("https://downloads.example.test/checksums.txt"), /Refusing private or local IP address/);
    });
  });
});

test("remote downloads reject IPv6 link-local addresses from DNS array lookups", async () => {
  await withMockDNSLookup([
    { address: "203.0.113.10", family: 4 },
    { address: "fe81::1", family: 6 },
  ], async () => {
    await withMockHTTPS({
      "/checksums.txt": { statusCode: 200, body: "ignored", lookupOptions: { all: true } },
    }, async () => {
      await assert.rejects(() => fetchText("https://downloads.example.test/checksums.txt"), /Refusing private or local IP address/);
    });
  });
});

test("remote downloads reject private IPv4-mapped IPv6 addresses from DNS array lookups", async () => {
  await withMockDNSLookup([
    { address: "203.0.113.10", family: 4 },
    { address: "::ffff:7f00:1", family: 6 },
  ], async () => {
    await withMockHTTPS({
      "/checksums.txt": { statusCode: 200, body: "ignored", lookupOptions: { all: true } },
    }, async () => {
      await assert.rejects(() => fetchText("https://downloads.example.test/checksums.txt"), /Refusing private or local IP address/);
    });
  });
});

test("downloadFile accepts public addresses from DNS array lookups", async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xelyon-npm-public-dns-"));
  const dest = path.join(dir, "asset.bin");

  await withMockDNSLookup([{ address: "203.0.113.10", family: 4 }], async () => {
    await withMockHTTPS({
      "/asset": { statusCode: 200, body: "public dns payload", lookupOptions: { all: true } },
    }, async () => {
      await downloadFile("https://downloads.example.test/asset", dest);
    });
  });

  assert.equal(fs.readFileSync(dest, "utf8"), "public dns payload");
});

test("downloadFile accepts public IPv4-mapped IPv6 addresses from DNS array lookups", async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "xelyon-npm-public-mapped-dns-"));
  const dest = path.join(dir, "asset.bin");

  await withMockDNSLookup([{ address: "::ffff:cb00:710a", family: 6 }], async () => {
    await withMockHTTPS({
      "/asset": { statusCode: 200, body: "public mapped payload", lookupOptions: { all: true } },
    }, async () => {
      await downloadFile("https://downloads.example.test/asset", dest);
    });
  });

  assert.equal(fs.readFileSync(dest, "utf8"), "public mapped payload");
});

test("remote downloads reject non-HTTPS URLs", async () => {
  await assert.rejects(() => fetchText("http://example.test/checksums.txt"), /Refusing non-HTTPS URL/);
  await assert.rejects(() => downloadFile("http://example.test/asset.tar.gz", path.join(os.tmpdir(), "xelyon-http-reject")), /Refusing non-HTTPS URL/);
  await assert.rejects(() => fetchText("https://localhost/checksums.txt"), /Refusing local hostname/);
  await assert.rejects(() => downloadFile("https://127.0.0.1/asset.tar.gz", path.join(os.tmpdir(), "xelyon-local-reject")), /Refusing private or local IP address/);
  await assert.rejects(() => fetchText("https://[::ffff:127.0.0.1]/checksums.txt"), /Refusing private or local IP address/);
  await assert.rejects(() => fetchText("https://[fe81::1]/checksums.txt"), /Refusing private or local IP address/);
});

async function withMockHTTPS(routes, fn) {
  const originalGet = https.get;
  https.get = (url, _options, onResponse) => {
    const request = new EventEmitter();
    request.setTimeout = () => request;
    request.destroy = (error) => {
      process.nextTick(() => request.emit("error", error));
    };

    process.nextTick(() => {
      const pathname = new URL(url).pathname;
      const route = routes[pathname];
      if (!route) {
        request.emit("error", new Error(`unexpected URL: ${url}`));
        return;
      }

      if (route.lookupOptions && _options.lookup) {
        _options.lookup(new URL(url).hostname, route.lookupOptions, (error) => {
          if (error) {
            request.emit("error", error);
            return;
          }
          respond(route, onResponse);
        });
        return;
      }

      respond(route, onResponse);
    });

    return request;
  };

  try {
    await fn();
  } finally {
    https.get = originalGet;
  }
}

function respond(route, onResponse) {
  const response = Readable.from(route.body ? [route.body] : []);
  response.statusCode = route.statusCode;
  response.headers = route.headers || {};
  onResponse(response);
}

async function withMockDNSLookup(result, fn) {
  const originalLookup = dns.lookup;
  dns.lookup = (_hostname, _options, callback) => {
    process.nextTick(() => {
      callback(null, result, Array.isArray(result) ? undefined : 4);
    });
  };
  try {
    await fn();
  } finally {
    dns.lookup = originalLookup;
  }
}
