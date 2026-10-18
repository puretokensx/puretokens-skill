// Development-only generation. Installed Skills need no JavaScript runtime.
import { readFile } from "node:fs/promises";
import path from "node:path";

const fields = ["id", "skill", "label", "command", "kind", "operations", "lifecycle", "delivery", "recovery"];
const ids = ["image", "video", "speech", "transcribe", "sound", "music", "evaluate", "embedding", "rerank"];
const recoveries = {
  "task-record": "原任务 `resume`；仅实际附件交付后 `delivered`",
  "local-artifact": "仅已有文件 `audio-verify` 后重新附加；无任务查询",
  "none": "无任务查询或结果找回；不自动重发"
};

export async function loadOperationGuidance(root) {
  const read = async file => JSON.parse(await readFile(path.join(root, file), "utf8"));
  const source = await read("references/operation-guidance.json");
  const direct = await read("references/direct-api-execution-contract.json");
  for (const kind of ["audio", "evaluation"]) {
    if (direct[kind]?.deadlineSeconds !== 90 || direct[kind]?.maxRequests !== 1 ||
        !direct[kind]?.synchronous || !direct[kind]?.neverPoll) {
      throw new Error(`${kind}: synchronous operation summary needs contract review`);
    }
  }
  if (source.schemaVersion !== 1 || Object.keys(source).sort().join() !== "operations,schemaVersion" ||
      !Array.isArray(source.operations) || source.operations.map(row => row.id).join() !== ids.join()) {
    throw new Error("Operation guidance must declare each reviewed operation once");
  }
  const contracts = new Map();
  for (const skill of ["image", "video", "audio", "evaluate", "embedding", "rerank"]) {
    contracts.set(`puretokens-${skill}`, await read(`skills/puretokens-${skill}/references/execution-contract.json`));
  }
  for (const row of source.operations) {
    const contract = contracts.get(row.skill);
    if (!contract || Object.keys(row).some(key => !fields.includes(key)) ||
        typeof row.label !== "string" || !row.label.trim() || /[|\r\n`<>]/.test(row.label) ||
        !Array.isArray(row.operations)) throw new Error(`Invalid operation guidance: ${row.id}`);

    // Verify lifecycle against owned API contracts, not the prose being generated.
    if (row.id === "image" || row.id === "video") {
      if (contract.kind !== row.id || !contract.operations.status || !contract.operations.content ||
          !contract.result.sameTaskOnly || !contract.result.neverAutoResubmit ||
          row.operations.join() !== "generate,edit") throw new Error(`${row.id}: async contract differs`);
    } else if (row.id === "music") {
      if (contract.kind !== "audio" || contract.result.asynchronousMusic?.kind !== "music" ||
          !contract.result.asynchronousMusic.asynchronous || !contract.result.asynchronousMusic.neverAutomaticallyResubmits ||
          row.operations.join() !== "generate") throw new Error("music: async contract differs");
    } else if (row.id === "evaluate") {
      if (contract.kind !== "evaluate" || !contract.result.synchronous || !contract.result.neverPoll ||
          !contract.result.neverAutomaticallyRetries || row.operations.length !== 0) throw new Error("evaluate: sync contract differs");
    } else if (row.id === "embedding" || row.id === "rerank") {
      if (contract.kind !== row.id || !contract.result.synchronous || !contract.result.neverPoll ||
          !contract.result.neverAutomaticallyRetries || !contract.result.onePostPerInvocation ||
          contract.result.deadlineSeconds !== 90 || !contract.result.validatesRequestAndNewOutputBeforeCredentials ||
          contract.result.outputFormat !== (row.id === "embedding" ? "jsonl" : "json") ||
          row.operations.length !== 0) throw new Error(`${row.id}: sync file contract differs`);
    } else {
      const operation = row.id === "sound" ? "generate" : row.id;
      if (contract.kind !== "audio" || row.operations.join() !== operation ||
          !contract.result.synchronousOperations.includes(operation) || !contract.result.neverAutomaticallyRetries ||
          !contract.operations[operation]) throw new Error(`${row.id}: sync contract differs`);
    }
    const async = ["image", "video", "music"].includes(row.id);
    const text = ["transcribe", "evaluate"].includes(row.id);
    const file = ["embedding", "rerank"].includes(row.id);
    const command = async ? "submit" : row.id === "evaluate" || file ? row.id : "audio";
    if (row.lifecycle !== (async ? "async" : "sync") ||
        row.command !== command ||
        row.kind !== (async ? row.id : undefined) ||
        row.delivery !== (file ? "file" : text ? "text" : "attachment") ||
        row.recovery !== (async ? "task-record" : text || file ? "none" : "local-artifact")) {
      throw new Error(`${row.id}: command, lifecycle, recovery or delivery mismatch`);
    }
  }
  return source.operations;
}

export function renderOperationGuidance(rows) {
  const lines = [
    "| 用户需求 | 命令与请求 | 返回方式 | 恢复与交付 |",
    "| --- | --- | --- | --- |"
  ];
  for (const row of rows) {
    const fileRequest = { embedding: "model / input / output_file", rerank: "model / query / documents / top_n? / output_file" };
    const request = row.kind ? `kind=${row.kind}` : row.operations.length ? `operation=${row.operations.join(" / ")}` : fileRequest[row.id] ?? "model / state / questions";
    const mode = row.lifecycle === "async" ? "异步；先回执，再独立等待／下载" : "同步；一次请求最多90秒";
    lines.push(`| ${row.label} | \`${row.command}\`；\`${request}\` | ${mode} | ${recoveries[row.recovery]} |`);
  }
  lines.push("", "文件路径／下载回执不等于交付；附件须实际附加，文本须展示校验后的结果。未知提交不证明未处理或未扣费，均不自动重发。");
  if (rows.some(row => row.delivery === "file")) {
    lines.push("向量／重排按明确请求保存新的 JSONL／JSON 文件；仅验证成功后报告产物。向量不铺进聊天，重排摘要与文件排序一致；accepted 但 output_verified=false 仍停止，不自动重发。");
  }
  return lines.join("\n");
}

export function replaceGuidanceBlock(text, name, content) {
  const start = `<!-- generated:${name} -->`, end = `<!-- /generated:${name} -->`;
  if (text.split(start).length !== 2 || text.split(end).length !== 2 || text.indexOf(end) < text.indexOf(start)) {
    throw new Error(`Expected one ordered ${name} guidance block`);
  }
  return text.slice(0, text.indexOf(start)) + start + "\n" + content + "\n" + text.slice(text.indexOf(end));
}
