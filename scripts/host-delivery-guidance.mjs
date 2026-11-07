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

function renderProtocol(protocol) {
  return [
    `## ${protocol.title}`, "", `适用模式：${protocol.modes.map(mode => labels[mode]).join("、")}；媒体：${protocol.media.map(kind => labels[kind]).join("、")}。`, "",
    `前提：${protocol.requirement}`, "", `动作：${protocol.action}`, "", `确认：${protocol.confirmation}`, "", `依据：${protocol.evidence}`,
    ...protocol.sources.map(url => `- 官方资料：${url}`), ""
  ];
}

export function renderHostDeliveryGuide(source) {
  const shared = source.protocols.find(row => row.id === source.defaultProtocol);
  if (!shared) throw new Error("Missing shared delivery protocol");
  const lines = [
    "# 当前客户端的结果交付", "",
    "当前会话尚无明确交付方式，或文件已生成但未出现在对话时，只读本页通用规则和 `delivery/<已绑定的 host>.md`；已知且仍有效的协议不重读，不读其他客户端文件。它不是生成前的联网检查，也不是实机验收清单。宿主 ID 入口见 [短索引](desktop-hosts.md)。", "",
    "## 先用已有宿主能力", "",
    "- 宿主 ID 沿用本任务绑定值；桌面、IDE、CLI 和远程模式不能互相代替。只依据当前应用上下文及现有工具声明选择交付方式，不扫描配置、安装插件或试发付费任务。",
    "- 普通生成不为此额外联网或增加发现步骤。验收未知不等于无法交付；仅当前宿主已明确不能接收所需文件或附件字节不可访问时，在提交前说明限制并停止。",
    "- 图片、视频、配音、音效和音乐使用执行器已校验的实际原生文件。将 `downloaded_paths` 或 `artifact.path` 中的绝对路径传给当前宿主支持的动作，不重命名、转换、上传或重复下载以制造可展示形式。",
    "- GIF 按原始 `.gif` 文件交付；当前预览工具不支持 GIF，但原生文件附件支持时，使用原始文件附件。不能用静态首帧、PNG/JPEG、改后缀、HTML 或视频替代。若用户明确要求在对话中播放，附件交付不能冒充播放验收；宿主两种方式均不支持时保留文件和原任务并说明限制，不重新生成。",
    "- 显式 SVG 仅交付执行器已校验的原始 `.svg` 文件；宿主未声明安全 SVG 预览时使用原生文件附件，不插入 HTML、不运行内容、不转成 PNG。原文件附件交付不等于渲染验收。",
    "- 显式视频尾帧是同任务第二份产物：索引 0 为视频，索引 1 为尾帧图片；逐份交付并确认，单份完成不代表整个任务完成。",
    "- 录音转写和 Jev 是文本结果：直接展示校验后的 `result.text` 或结构化答案，不要求附件、图片预览或媒体交付工具。", "",
    "各宿主文件只含该宿主的条件协议，不是客户端能力承诺。核对当前模式及前提；专用协议不适用时，使用本页的通用原生协议，不能虚构工具。目录、安装或 init 成功不证明交付，文档不改变实机验收状态。", "",
    ...renderProtocol(shared)
  ];
  lines.push("## 区分交付动作、交付记录和播放验收", "",
    "文件已校验、交付动作已发出、宿主已完成文件交接、用户确实打开／播放，是不同事实。只按当前协议的实际成功信号描述；发出一个路径不创造附件成功回执。宿主按其明确协议接收媒体项后，无须再要求用户每次回复确认；没有可确定的交接结果时，保留待交付并说明文件已准备好，不能预先执行 `delivered`。", "",
    "异步任务仅在实际交接后执行原记录的 `delivered`；同步音频不调用该命令。若交付发生在最终回复，不能在该回复发出前先确认；可在支持中间媒体消息的宿主先发送，再按实际协议记账。用户明确确认已收到也是可用依据，但不把确认作为每次正常交付的额外步骤。打开／播放成功属于单独的实机验收，不能从工具接收推断。", "",
    "交付失败保留原任务及原文件：异步用原记录 `resume`，同步音频用 `audio-verify` 校验后重交；不重生、不自动换模型。远程路径不能冒充本地可访问路径，终端图片输入、给模型看图或打印 Markdown 也不能冒充用户收到了音视频。", "",
    `资料核对日期：${source.reviewedAt}；本页不更改实机验收状态。`, "");
  return lines.join("\n");
}

export function renderSelectedHostDelivery(source, hostID) {
  const host = source.hosts.find(row => row.id === hostID);
  if (!host) throw new Error("Unknown delivery host; never fall back to another client");
  const lines = [
    `# ${hostID} 结果交付`, "",
    "沿用当前宿主绑定和 [通用交付规则](../host-delivery.md)，仅使用符合当前模式和前提的协议。", ""
  ];
  for (const id of host.protocols) {
    const protocol = source.protocols.find(row => row.id === id);
    if (!protocol) throw new Error("Unknown delivery protocol");
    if (id !== source.defaultProtocol) lines.push(...renderProtocol(protocol));
  }
  if (host.protocols.every(id => id === source.defaultProtocol)) {
    lines.push("本宿主没有额外专用协议；仅在当前会话明确支持时使用通用规则中的原生交付方式。", "");
  }
  return lines.join("\n");
}
