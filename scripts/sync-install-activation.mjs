import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const registry = JSON.parse(await readFile(path.join(root, "references/install-activation.json"), "utf8"));
const hosts = JSON.parse(await readFile(path.join(root, "references/host-support.json"), "utf8")).supported.map(host => host.id);
if (registry.schemaVersion !== 1 || !registry.default || !registry.hosts) throw new Error("Invalid installation activation registry");
for (const host of Object.keys(registry.hosts)) if (!hosts.includes(host)) throw new Error(`Unknown activation host ${host}`);
const resolved = host => ({ ...registry.default, ...(registry.hosts[host] ?? {}) });
for (const host of [...hosts, ""]) {
  const value = resolved(host);
  if (typeof value.activation !== "string" || !/^[\x20-\x7e]+$/.test(value.activation) || !["implementation_available", "unsupported"].includes(value.apiSupport) || !Array.isArray(value.examples) || value.examples.length > 2 || value.examples.some(example => typeof example !== "string" || !/^[\x20-\x7e]+$/.test(example)) || value.apiSupport === "unsupported" && value.examples.length) throw new Error(`Invalid activation guidance ${host}`);
}
const shellQuote = value => `'${value.replaceAll("'", `'"'"'`)}'`;
const psQuote = value => `'${value.replaceAll("'", "''")}'`;
function lines(value) {
  const result = ["Next: " + value.activation];
  if (value.apiSupport === "unsupported") result.push("API tasks: unsupported in this host; no credential adapter is available. Installing instructions does not enable generation.");
  else {
    result.push("Available instructions: images, video, audio/music, Jev evaluation, embeddings, reranking, models and balance. Connection and task results remain separate checks.");
    for (const example of value.examples) result.push("Example (not executed): " + example);
  }
  result.push("Help: Ask how to use Pure Tokens Skills to read the local usage guide without a network request.");
  return result;
}
const shell = ["# generated:install-activation", "installation_summary() {", "  case \"$1\" in"];
const ps = ["# generated:install-activation", "function Show-InstallationSummary([string]$RequestedHost) {", "  switch ($RequestedHost) {"];
for (const host of [...Object.keys(registry.hosts), ""]) {
  shell.push(`    ${host || "*"})`);
  shell.push(`      printf '%s\\n' ${lines(resolved(host)).map(shellQuote).join(" ")}`);
  shell.push("      ;;");
  ps.push(`    ${host ? psQuote(host) : "default"} {`);
  for (const line of lines(resolved(host))) ps.push(`      Write-Output ${psQuote(line)}`);
  ps.push("    }");
}
shell.push("  esac", "}", "# /generated:install-activation");
ps.push("  }", "}", "# /generated:install-activation");
let changed = false;
for (const [file, content] of [["runtime/puretokens-skill-install.sh", shell.join("\n")], ["runtime/puretokens-skill-install.ps1", ps.join("\n")]]) {
  const location = path.join(root, file);
  const previous = await readFile(location, "utf8");
  const pattern = /# generated:install-activation[\s\S]*?# \/generated:install-activation/;
  if (!pattern.test(previous)) throw new Error(`Missing generated activation block in ${file}`);
  const next = previous.replace(pattern, content);
  if (next !== previous) {
    if (process.argv.includes("--write")) await writeFile(location, next);
    else { changed = true; console.error(`${file}: generated installation guidance is stale`); }
  }
}
if (changed) process.exitCode = 1;
