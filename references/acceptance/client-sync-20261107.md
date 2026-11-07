# Switch client compatibility review, 2026-11-07

## Scope and authorization

Request: support clients newly added to Switch. Only puretokens-skill is edited.
No production credentials read, client configuration writes, paid requests,
release, push or Switch/Web changes. Existing unrelated work is preserved.
Defaults remain gpt-image-2.6, MiniMax-H3 and qwen3.7-text-embedding.

## Evidence sources

- Switch checkout HEAD: 3a41ef525383fa923b2eb8869574a330cd6fb643. Registry: apps/desktop/src-tauri/src/client_registry.rs. Writers under crates/client-adapters/{qwen-code,codebuddy-code,copilot-cli,factory-droid,mimo-coder,paseo,antigravity-cli}.
- Ten configured macOS/Windows synthetic writer fixtures copied into runtime/executor/testdata/switch-cli-writers.json from tests/fixtures/client-configs. Fixtures contain synthetic tokens only, not user data. Paseo uses a synthetic profile based on the writer schema; it is not claimed as a captured real session.
- Qwen: https://raw.githubusercontent.com/QwenLM/qwen-code/main/docs/users/features/skills.md and packages/cli/src/config/settings.ts; Switch dotenv assignment-name parser.
- CodeBuddy Code: https://www.codebuddy.cn/docs/cli/skills
- Copilot CLI: https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-skills and use-byok-models.
- Factory: https://docs.factory.com/cli/configuration/skills.md and settings.md.
- MiMo v0.1.16, b8dbe084ed8e77be2f75a85a9c880031aed699c9: https://github.com/XiaomiMiMo/MiMo-Code/tree/v0.1.16 ; shared/src/global.ts and cli/src/{skill,config,auth,provider}.
- Paseo: https://paseo.sh/docs/custom-providers.md and https://raw.githubusercontent.com/getpaseo/paseo/main/packages/server/src/server/agent/providers/codex-app-server-agent.ts ; app-server custom model_provider and env_key are runtime values, not independent Codex config/auth writes.

## Implemented scope

Five native CLI entries: qwen-code, codebuddy-code, copilot-cli, factory-droid,
mimo-coder. Conditional sixth entry: paseo, local Codex engine only, with
explicit absolute PASEO_HOME and matching inherited profile endpoint/Key.
Filtered or absent inherited credentials stop; no fallback to Codex auth.
Current source registry grows from 21 to 27 hosts, not 27 client/OS acceptances.

Antigravity CLI stays unsupported: Switch uses process-only launch credentials,
and no current-session credential plus Skill discovery contract is verified.
DeepSeek Bot, Rakazo and OpenMausBot stay outside direct Skill support.
TRAE Work retains host ID trae and its existing same-host SQLite adapter.

## Offline coverage

- Each of six new host bindings runs the real command dispatcher against mock HTTP for image, video, audio, Jev, embedding and rerank. Exact ID/path, catalog visible/hidden/refreshed, POST 200/401/403, one submission only and sanitized receipts are checked.
- Audio native WAV bytes, typed Jev output, ordered vector JSONL and ranking JSON are checked. Image/video submission receipts are not native host attachment proof. Existing shared recovery/output tests remain separate.
- Wrong origin, dynamic/duplicate fields, multiple different Keys, project/environment overrides, symlinked configuration and unchanged source bytes are covered.
- Qwen unrelated dotenv assignments and Factory empty legacy configuration do not block. MiMo unrelated provider preferences do not block; substitutions and wellknown remote authentication stop.
- Shell isolated HOME tests locate, install all ten Skills plus one executor, verify manifests/inventories, reject invalid root overrides and preserve host boundaries.

Fresh targeted checks passed: go test ./... -run TestSwitchCLI -count=1
and node --test test/switch-cli-installation.test.mjs. Shared guidance generation
completed.

## Final local verification

- Environment: macOS arm64, Node v26.9.0, Go 1.26.0.
- `npm run executor:build`: exit 0, rebuilt darwin/linux/windows for amd64
  and arm64 from the final executor source with an isolated temporary Go cache.
- `npm run check`: exit 0. Validation, generated guidance, host records, source
  build proof and all six artifact hashes passed. Go full tests and vet passed.
  Node: 208 tests, 204 passed, 0 failed, 4 skipped because PowerShell is absent.
- Registry consistency: 27 source hosts across all ten Skill manifests.
- Installed host-guide link regression reproduced a missing
  `credential-adapters.md` link. Replaced it with self-contained bounded guidance;
  the regression now passes without adding duplicate maintainer documentation.
- Windows installer cases now cover all six new entries, inventory verification,
  selected root overrides and invalid-root rejection. These Windows cases were
  added for the existing Windows job and have not been executed locally.
- No stable version bump, release packaging, commit, push or publication. The
  working tree also contains prior multimedia changes, which were not staged.

## Not accepted

No real-client Skill loading, real Key group permission, billable dedicated API,
image opening, video/audio playback or native attachment delivery was tested.
macOS/Windows fixture formats do not constitute Windows process execution.
Paseo session environment availability remains an explicit real-host check.
Remote/WSL, alternative engines and unrepresented launch/session overrides
are not accepted. No version publication is authorized by these results.

Switch's own Skills support UI list has not been changed in this repository.
When separately coordinating that project, synchronize the exact supported IDs
and boundaries rather than promoting every configured chat client.
