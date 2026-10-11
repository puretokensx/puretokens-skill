// Development-only source validation and rendering. No client runtime or API calls.
import { readFile } from "node:fs/promises";
import path from "node:path";

const topFields = ["$schema", "schemaVersion", "reviewedAt", "scope", "defaultProtocol", "ordinaryGenerationPreflight", "unknownMeansUnsupported", "nativeFileRequired", "protocols", "hosts"];
const protocolFields = ["id", "title", "modes", "media", "requirement", "action", "confirmation", "evidenceKind", "evidence", "sources"];
const modes = ["desktop-local", "ide-local", "cli-local", "remote"];
const media = ["image", "video", "audio"];
const labels = { "desktop-local": "本地桌面", "ide-local": "本地 IDE", "cli-local": "本地 CLI／终端", remote: "远程执行", image: "图片", video: "视频", audio: "音频" };

function exactFields(value, expected) {
  return value && typeof value === "object" && !Array.isArray(value) &&
    Object.keys(value).sort().join() === [...expected].sort().join();
}

function listOf(value, allowed) {
  return Array.isArray(value) && value.length > 0 && new Set(value).size === value.length &&
    value.every(item => allowed.includes(item));
}

function prose(value) {
  return typeof value === "string" && value.trim().length > 0 && value.length <= 1500 && !/[\r\n]/.test(value);
}

export async function loadHostDelivery(root) {
  const read = async file => JSON.parse(await readFile(path.join(root, file), "utf8"));
  const [source, support] = await Promise.all([read("references/host-delivery.json"), read("references/host-support.json")]);
  if (!exactFields(source, topFields) || source.$schema !== "https://puretokensx.com/schemas/host-delivery.schema.json" ||
      source.schemaVersion !== 1 || !/^\d{4}-\d{2}-\d{2}$/.test(source.reviewedAt) ||
      source.scope !== "delivery_guidance_not_real_host_acceptance" || source.defaultProtocol !== "session-native" ||
      source.ordinaryGenerationPreflight !== false || source.unknownMeansUnsupported !== false || source.nativeFileRequired !== true ||
      !Array.isArray(source.protocols) || !Array.isArray(source.hosts)) {
    throw new Error("Host delivery guidance must preserve native files, conditional evidence and no generation preflight");
  }
  const ids = new Set();
  for (const protocol of source.protocols) {
    if (!exactFields(protocol, protocolFields) || !/^[a-z][a-z0-9-]*$/.test(protocol.id) || ids.has(protocol.id) ||
        !listOf(protocol.modes, modes) || !listOf(protocol.media, media) ||
        !["title", "requirement", "action", "confirmation", "evidence"].every(key => prose(protocol[key])) ||
        /[|`<>]/.test(protocol.title) || !["current_session_contract", "official_documentation"].includes(protocol.evidenceKind) ||
        !Array.isArray(protocol.sources) || new Set(protocol.sources).size !== protocol.sources.length) {
      throw new Error("Invalid or duplicate conditional host delivery protocol");
    }
    for (const link of protocol.sources) {
      let url;
      try { url = new URL(link); } catch { throw new Error("Host delivery sources must use public HTTPS documentation"); }
      if (url.protocol !== "https:" || url.username || url.password || /[\s<>]/.test(link)) {
        throw new Error("Host delivery sources must use public HTTPS documentation");
      }
    }
    if ((protocol.evidenceKind === "official_documentation") !== (protocol.sources.length > 0)) {
      throw new Error("Host delivery evidence must distinguish official documentation from the current session contract");
    }
    ids.add(protocol.id);
  }
  const fallback = source.protocols.find(row => row.id === source.defaultProtocol);
  if (!fallback || fallback.evidenceKind !== "current_session_contract" ||
      !modes.every(mode => fallback.modes.includes(mode)) || !media.every(kind => fallback.media.includes(kind))) {
    throw new Error("Host delivery needs the conditional current-session protocol for every mode and media kind");
  }
  const hostIDs = [];
  for (const host of source.hosts) {
    if (!exactFields(host, ["id", "protocols"]) || !/^[a-z][a-z0-9-]*$/.test(host.id) ||
        hostIDs.includes(host.id) || !listOf(host.protocols, [...ids]) || !host.protocols.includes(source.defaultProtocol)) {
      throw new Error("Invalid host delivery mapping or unknown protocol");
    }
    hostIDs.push(host.id);
  }
  const supported = support.supported.map(host => host.id).sort();
  if (hostIDs.sort().join() !== supported.join()) throw new Error("Host delivery IDs must match the supported host registry");
  return source;
}

export function renderHostDeliveryGuide(source) {
  const protocols = new Map(source.protocols.map(row => [row.id, row]));
  const lines = [
    "# 当前客户端的结果交付", "",
    "当前会话尚无明确交付方式，或遇到文件已生成但未出现在对话时，读取本页对应宿主／模式；已知且仍有效的协议不重读。它不是生成前的联网检查，也不是实机验收清单。", "",
    "## 先用已有宿主能力", "",
    "- 宿主 ID 沿用本任务绑定值；桌面、IDE、CLI 和远程模式不能互相代替。只依据当前应用上下文及现有工具声明选择交付方式，不扫描配置、安装插件或试发付费任务。",
    "- 普通生成不为此额外联网或增加发现步骤。验收未知不等于无法交付；仅当前宿主已明确不能接收所需文件或附件字节不可访问时，在提交前说明限制并停止。",
    "- 图片、视频、配音、音效和音乐使用执行器已校验的实际原生文件。将 `downloaded_paths` 或 `artifact.path` 中的绝对路径传给当前宿主支持的动作，不重命名、转换、上传或重复下载以制造可展示形式。",
    "- 录音转写和 Jev 是文本结果：直接展示校验后的 `result.text` 或结构化答案，不要求附件、图片预览或媒体交付工具。", "",
    "## 按宿主选择", "",
    "表中是条件协议，不是客户端能力承诺。先核对当前模式和协议前提；专用协议不适用时，依据当前会话的通用原生协议，不能把未验证写成不支持，也不能虚构不存在的工具。", "",
    "| 宿主 ID | 可检查的协议 |", "| --- | --- |"
  ];
  for (const host of source.hosts) {
    lines.push(`| \`${host.id}\` | ${host.protocols.map(id => protocols.get(id).title).join("；")} |`);
  }
  lines.push("", "Trae 目前缺少凭据适配器；这里列出交付协议不新增其 API 支持。其他客户端同样继续遵守原有凭据与执行模式边界。", "");
  for (const protocol of source.protocols) {
    lines.push(`## ${protocol.title}`, "", `适用模式：${protocol.modes.map(mode => labels[mode]).join("、")}；媒体：${protocol.media.map(kind => labels[kind]).join("、")}。`, "",
      `前提：${protocol.requirement}`, "", `动作：${protocol.action}`, "", `确认：${protocol.confirmation}`, "", `依据：${protocol.evidence}`);
    for (const url of protocol.sources) lines.push(`- 官方资料：${url}`);
    lines.push("");
  }
  lines.push("## 区分交付动作、交付记录和播放验收", "",
    "文件已校验、交付动作已发出、宿主已完成文件交接、用户确实打开／播放，是不同事实。只按当前协议的实际成功信号描述；发出一个路径不创造附件成功回执。宿主按其明确协议接收媒体项后，无须再要求用户每次回复确认；没有可确定的交接结果时，保留待交付并说明文件已准备好，不能预先执行 `delivered`。", "",
    "异步任务仅在实际交接后执行原记录的 `delivered`；同步音频不调用该命令。若交付发生在最终回复，不能在该回复发出前先确认；可在支持中间媒体消息的宿主先发送，再按实际协议记账。用户明确确认已收到也是可用依据，但不把确认作为每次正常交付的额外步骤。打开／播放成功属于单独的实机验收，不能从工具接收推断。", "",
    "交付失败保留原任务及原文件：异步用原记录 `resume`，同步音频用 `audio-verify` 校验后重交；不重生、不自动换模型。远程路径不能冒充本地可访问路径，终端图片输入、给模型看图或打印 Markdown 也不能冒充用户收到了音视频。", "",
    `资料核对日期：${source.reviewedAt}；本页不更改实机验收状态。`, "");
  return lines.join("\n");
}
