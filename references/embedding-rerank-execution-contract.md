# Embedding and rerank execution contract

Candidate 0.19.5, 2026-10-18. This repository owns the native commands, installed
Skills and distribution. Gateway deployment, account eligibility, billing and
real-host acceptance remain separate evidence.

## Commands and authentication

```text
<managed-native-executor> embedding --host <current-host> --request <absolute-json-file>
<managed-native-executor> rerank --host <current-host> --request <absolute-json-file>
```

Only `--host` and `--request` are accepted. Resolve the executor relative to the
loaded Skill, not PATH or its install directory as a host identity. Validate
the complete request and explicit new output file before credential resolution
or networking. Use only the current host's verified Pure Tokens connection,
with its matching credential in executor memory; no key argument, environment
fallback, credential scan, browser, MCP, proxy or SDK transport.

Each command makes one synchronous POST with a 90-second total deadline, no
redirect, retry, polling, asynchronous task ID or result lookup. Inputs are
explicit strings, not implicit file/URL references. No database, vector-store
creation, file ingestion or automatic expansion into paid batches is provided.

## Requests and user artifacts

| Contract | Embedding | Rerank |
| --- | --- | --- |
| Skill | `puretokens-embedding` | `puretokens-rerank` |
| Fixed full API URL | `https://api.puretokensx.com/v1/embeddings` | `https://api.puretokensx.com/v1/rerank` |
| Reviewed exact IDs | `qwen3.7-text-embedding`, `qwen3.7-text-embedding-flash` | `qwen3.7-text-rerank` |
| Request fields | `model`, `input`, `output_file` | `model`, `query`, `documents`, optional `top_n`, `output_file` |
| API body | `model`, `input`, fixed `encoding_format=float` | `model`, `query`, `documents`, `top_n`, fixed `return_documents=false` |
| Array bound | 1–64 nonempty input strings | 1–64 candidate strings |
| Text bound | 32768 Unicode runes per item | 32768 runes per item, nonempty query up to 8192 runes |
| Output | New absolute `.jsonl` file | New absolute `.json` file |

Both request files are at most 1048576 bytes (1 MiB); responses are at most
2097152 bytes (2 MiB). `top_n` is an integer from 1 to the candidate count,
defaulting to that count. Local output_file never enters the API body. Output
parents must already exist and be writable; existing files are never overwritten
or deleted. These bounds are local resource controls, not token, price or
upstream capacity guarantees.

Embedding writes one `{index,dimensions,embedding}` object per JSONL line,
ordered by original zero-based input index. Entries, dimensions and finite
vector components are validated before success. Chat reports only item count,
actual dimensions and the user path, never the vector arrays.

Rerank writes `{results:[{rank,index,relevance_score}]}` with one-based rank,
zero-based original candidate index and the original finite score, in validated
relevance order. An optional safe receipt summary contains the same ordering;
no candidate text is returned. Scores are not probabilities, accuracy or
calibrated values across requests. Preserve original indexes and raw scores;
do not invent facts, normalize scores or claim global ranking across batches.

## Receipts and uncertainty

Success is `ok=true`, `command=embedding|rerank`,
`submission_outcome=accepted`, `next_step=done`, `output_verified=true`.
`artifact` contains `path`, `format` (`jsonl` or `json`), `item_count`,
`sha256`, `bytes`, plus `dimensions` for embedding. Rerank may include
`result.results` with the same rank/index/score projection as the file.

Failure follows evaluation: `ok`, `command`, `submission_outcome`,
`next_step`, `output_verified=false`, and applicable `failure_phase`,
`local_error_code`, `http_status`, `api_error_code`, `error_message`,
`next_action`. Typed support wrapping excludes inputs, vectors, query,
candidate text, scores, local paths, credentials and raw upstream errors.
Optional status/code fields are not fabricated.

`not_submitted` proves only this invocation did not submit; `rejected` means a
confirmed rejection; `unknown` preserves possible submission and billing when
the outcome cannot be validated. An accepted response whose output cannot be
saved or verified remains `accepted`, `failure_phase=content`,
`local_error_code=embedding_output_unavailable|rerank_output_unavailable`,
without `result` or `artifact`. Neither unknown nor accepted output failure
permits automatic resubmission or a no-charge claim. No receipt after process
interruption also does not prove no request. An informed explicit new attempt
can create another charge; there is no synchronous recovery or idempotency
guarantee.

Request schemas are [embedding](../schemas/embedding-request.schema.json) and
[rerank](../schemas/rerank-request.schema.json); the shared receipt is
[retrieval-receipt](../schemas/retrieval-receipt.schema.json). Artifact schemas
are [embedding rows](../schemas/embedding-output.schema.json) and
[rerank output](../schemas/rerank-output.schema.json). A schema or offline
fixture is not production or real-host acceptance.

## Explicit discovery and Switch guidance

`models --host <current-host> --request <absolute-filter-json>` accepts an
explicit `{"kind":"embedding"}` or `{"kind":"rerank"}`, optionally with exact
`model`. Each invocation makes one authenticated `GET
https://api.puretokensx.com/v1/models`, intersects with the reviewed exact IDs
above, and deduplicates. No operation/parameters filter or static fallback.
`matching_scope` is `reviewed_embedding_models_listed_by_api` or
`reviewed_rerank_models_listed_by_api`. This proves visibility only, not route
availability, permission to submit, price, output quality, billing or
real-host acceptance. It is not an ordinary execution preflight.

Switch guides users to select eligible model groups and apply the client chat
connection, then separately install the Skills and request the dedicated task
in that client. Dedicated model permission comes from the full selected-group
model union; it need not enter the chat-compatible subset. Users retain their
existing groups when adding access. Only an actual typed execution receipt can
establish this invocation's outcome; a copied prompt, configuration write,
chat success, init or directory visibility cannot do so. The paired Switch
source changes and this unpublished candidate do not establish a published
installer or deployed gateway.
