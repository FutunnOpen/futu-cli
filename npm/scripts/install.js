#!/usr/bin/env node

const { createHash } = require("node:crypto");
const { chmodSync, copyFileSync, createWriteStream, mkdirSync, readFileSync, rmSync } = require("node:fs");
const { get } = require("node:https");
const { get: getHTTP } = require("node:http");
const { tmpdir } = require("node:os");
const { join } = require("node:path");
const { spawnSync } = require("node:child_process");

const product = "futu";
const binary = "futu";
const envPrefix = "FUTU_CLI";
const checksumAsset = "futu_checksums.txt";
const packageJson = require("../../package.json");
const defaultReleaseBase = "https://github.com/FutunnOpen/futu-cli";
const releaseBase = (process.env[`${envPrefix}_RELEASE_BASE`] || defaultReleaseBase).replace(/\/+$/, "");
const version = process.env[`${envPrefix}_VERSION`] || `v${packageJson.version}`;

function platformSuffix() {
  const arch = process.arch === "x64" ? "amd64" : process.arch;
  if (process.platform === "darwin" && (arch === "amd64" || arch === "arm64")) {
    return { suffix: `darwin_${arch}`, ext: "tar.gz" };
  }
  if (process.platform === "linux" && (arch === "amd64" || arch === "arm64")) {
    const libc = process.env[`${envPrefix}_LIBC`] === "glibc" ? "" : "musl_";
    return { suffix: `linux_${libc}${arch}`, ext: "tar.gz" };
  }
  if (process.platform === "win32" && arch === "amd64") {
    return { suffix: "windows_amd64", ext: "zip" };
  }
  throw new Error(`unsupported platform: ${process.platform}/${process.arch}`);
}

function download(url, destination) {
  const client = url.startsWith("http://") ? getHTTP : get;
  return new Promise((resolve, reject) => {
    client(url, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        download(response.headers.location, destination).then(resolve, reject);
        return;
      }
      if (response.statusCode < 200 || response.statusCode >= 300) {
        reject(new Error(`download ${url} failed: ${response.statusCode}`));
        return;
      }
      const file = createWriteStream(destination);
      response.pipe(file);
      file.on("finish", () => file.close(resolve));
      file.on("error", reject);
    }).on("error", reject);
  });
}

function checksum(path) {
  return createHash("sha256").update(readFileSync(path)).digest("hex");
}

async function main() {
  const selected = platformSuffix();
  const asset = `${binary}_${version}_${selected.suffix}.${selected.ext}`;
  const workDir = join(tmpdir(), `${product}-npm-${Date.now()}`);
  const archive = join(workDir, asset);
  const checksumFile = join(workDir, checksumAsset);
  const unpackDir = join(workDir, "unpack");
  const vendorDir = join(__dirname, "..", "vendor");

  mkdirSync(workDir, { recursive: true });
  mkdirSync(unpackDir, { recursive: true });
  mkdirSync(vendorDir, { recursive: true });

  try {
    await download(`${releaseBase}/releases/download/${version}/${asset}`, archive);
    await download(`${releaseBase}/releases/download/${version}/${checksumAsset}`, checksumFile);
    const line = readFileSync(checksumFile, "utf8").split(/\r?\n/).find((item) => item.trim().endsWith(` ${asset}`));
    if (!line) throw new Error(`checksum for ${asset} not found`);
    const expected = line.trim().split(/\s+/)[0].toLowerCase();
    const actual = checksum(archive).toLowerCase();
    if (expected !== actual) throw new Error(`checksum mismatch for ${asset}`);

    const extract = selected.ext === "zip"
      ? spawnSync("powershell", ["-NoProfile", "-Command", "Expand-Archive", "-Force", archive, unpackDir], { stdio: "inherit" })
      : spawnSync("tar", ["-xzf", archive, "-C", unpackDir], { stdio: "inherit" });
    if (extract.status !== 0) throw new Error(`failed to extract ${asset}`);

    const exe = process.platform === "win32" ? `${binary}.exe` : binary;
    const source = join(unpackDir, exe);
    const target = join(vendorDir, exe);
    copyFileSync(source, target);
    if (process.platform !== "win32") chmodSync(target, 0o755);
    const verify = spawnSync(target, ["version"], { encoding: "utf8" });
    if (verify.status !== 0) throw new Error(`failed to verify ${exe}`);
    if (!`${verify.stdout}${verify.stderr}`.includes(version)) {
      throw new Error(`${exe} version output does not contain ${version}`);
    }
  } finally {
    rmSync(workDir, { recursive: true, force: true });
  }
}

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});
