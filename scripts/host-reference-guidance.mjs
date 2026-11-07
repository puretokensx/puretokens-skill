// Development-only rendering; installed files are never specialized at install time.
export function renderHostReferences(source, support, binding) {
  const ids = support.supported.map(host => host.id);
  const sections = new Map();
  for (const section of source.split(/^## /m).slice(1)) {
    const end = section.indexOf("\n");
    const id = section.slice(0, end).trim();
    if ((!ids.includes(id) && !["common", "unsupported"].includes(id)) || sections.has(id)) {
      throw new Error(`Unknown or duplicate host guidance section: ${id}`);
    }
    sections.set(id, section.slice(end + 1).trim());
  }
  if (!sections.has("common") || source.split("<!-- host-binding -->").length !== 2) {
    throw new Error("Host references need common diagnostics and one host-binding marker");
  }
  const index = [
    "# Host reference index", "", binding, "",
    "Only for a host-specific limitation or failure: read `hosts/<bound-host>.md` below. If the host is already bound, go directly to that file; do not read other clients or repeat this lookup during the same unchanged session. Ordinary requests need no extra preflight.", "",
    "Shared installation files never select the host. The current session or explicit user choice supplies `--host`; connection presence alone is not a routing signal. Do not trim shared files to one client or replace another client's configuration.", "",
    "For a local connection failure, see [sanitized diagnostics](host-errors.md). An unlisted host must stop; see [unsupported boundaries](unsupported-hosts.md), never substitute another client.", "",
    ...ids.map(id => `- [${id}](hosts/${id}.md)`), ""
  ].join("\n");
  const files = new Map([
    ["desktop-hosts.md", index],
    ["host-errors.md", `# Sanitized host diagnostics\n\n${sections.get("common").replace("<!-- host-binding -->", binding)}\n`],
    ["unsupported-hosts.md", `# Unsupported host boundaries\n\n${sections.get("unsupported") ?? "Unlisted hosts have no verified binding. Stop without borrowing another client's credentials."}\n`]
  ]);
  for (const host of support.supported) {
    const details = sections.get(host.id) ?? `${host.directorySelection}\n\n${host.guidance}`;
    files.set(`hosts/${host.id}.md`, [
      `# ${host.id}`, "", `Bound host: \`--host ${host.id}\`.`, "",
      details.replaceAll("](claude-mapped-connection.md)", "](../claude-mapped-connection.md)"), "",
      "Read [sanitized diagnostics](../host-errors.md) only after a local failure.", ""
    ].join("\n"));
  }
  return files;
}
