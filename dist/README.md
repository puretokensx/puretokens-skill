# Local platform release candidates

`npm run release:package` generates six platform archives, four installer scripts
and a release manifest under the ignored `dist/releases/` directory. Each
platform archive contains the current Skills and exactly one native executor.

Normal installation and update use the published stable release of
`puretokensx/puretokens-skill`, verify its manifest and checksums, and download only
the selected platform archive. A missing stable release is a distribution blocker;
installers must not fall back to `main`, a source archive or an unpublished draft.

Local archives are not published assets. Packaging a dirty checkout leaves
`sourceCommit: null`; such a draft cannot pass `release:verify -- --publishable`.
Distribution changes also require a new immutable version and the existing
catalog-freshness and real-host release acceptance gates.

The retired `puretokens-skill-install.zip` and
`puretokens-skill-install-payload.zip` migration bridges remain absent.
Superseded local version archives may be removed after confirming no active
packaging or verification uses them. Preserve the current candidate and its
manifest, independent migration/recovery evidence, and published immutable bytes.
