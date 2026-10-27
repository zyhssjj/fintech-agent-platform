import { execFileSync } from "node:child_process";
import { appendFileSync, existsSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const swagger = "../backend/docs/swagger.json";
const openapi = "src/base/api/openapi.generated.json";
const contracts = "src/base/api/contracts.ts";
const compat = resolve(root, "src/base/api/contracts.compat.ts");

execFileSync("npx", ["swagger2openapi", "-p", "-o", openapi, swagger], {
  cwd: root,
  stdio: "inherit",
  shell: process.platform === "win32",
});
execFileSync("npx", ["openapi-typescript", openapi, "-o", contracts], {
  cwd: root,
  stdio: "inherit",
  shell: process.platform === "win32",
});

if (existsSync(compat)) {
  appendFileSync(
    contracts,
    "\n// Compatibility exports retained until all business module schemas are fully described in backend Swagger.\n",
  );
  appendFileSync(contracts, readFileSync(compat));
}
