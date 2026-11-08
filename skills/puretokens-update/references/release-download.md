# Official release downloads

This is installation-only guidance, not an alternate inference transport.
The existing local command, HTTPS and directory permissions still apply.
No GitHub Token, new runtime, mirror or background service is required.

## First installation or old bootstrap

1. Make one anonymous GET to the fixed official metadata endpoint:
   https://api.github.com/repos/puretokensx/puretokens-skill/releases/latest
2. Require a stable tag of the form vMAJOR.MINOR.PATCH, draft=false and
   prerelease=false. Select exactly one uploaded asset named
   puretokens-skill-fetch.sh or puretokens-skill-fetch.ps1 for this OS.
3. Require its numeric asset ID, positive size (at most 32 MiB), and
   sha256 digest. Its url must be exactly the fixed repository API prefix
   https://api.github.com/repos/puretokensx/puretokens-skill/releases/assets/
   followed by that ID. Its browser_download_url must name the same
   repository, pinned tag and exact filename. Never execute metadata text.
4. Download that API asset URL once with Accept: application/octet-stream,
   allowing bounded HTTPS redirects only. Save to a unique temporary local
   file, verify both size and SHA-256 digest, then execute install with the
   already confirmed host/target where available. Never pipe a response to
   a shell. Clean this bootstrap file after completion.
5. A denied request, bad metadata, download failure or checksum mismatch
   stops before execution. Report the stage and safe error; do not read
   credentials, change permissions, install a runtime or try another source.

The API metadata and asset request each have a bounded 45-second timeout.
They are anonymous and never use gh authentication or local GitHub tokens.
The initial bootstrap cannot repair its own download before it runs.
Older hardcoded website prompts are not rewritten by this repository.

## Managed fetch downloads

The normal browser-download path remains unchanged. Only a complete HTTP
502, 503 or 504 permits a bounded alternate lookup and asset GET.

- Read official metadata at most once per invocation. For the initial
  latest manifest resolve latest; after a manifest is available resolve
  its exact tag, never a newer latest release. Cache only in the private
  temporary directory, removed when fetch exits.
- Validate the stable tag and unique uploaded asset, fixed repository URLs,
  size and digest as above. Fetch one asset with application/octet-stream,
  then retain all existing manifest, source commit, selector, archive and
  executor checks. A fallback manifest must match the API-selected tag.
- Each primary request keeps its existing 45-second budget, or 180 seconds
  for the archive. The one metadata lookup and one alternate request are
  separately bounded by the same limits; total failure time can therefore
  increase. There is no retry loop or guarantee of recovery.
- Permission, TLS, DNS, connection, timeout, local I/O, 401/403/404/429 and
  integrity failures stop. A failed alternate request also stops. Never
  switch versions, sources, hosts or credentials to get around a failure.
- Same-version integrity success still skips archive download, writes and
  init. No media POST or paid task is part of installation acceptance.
