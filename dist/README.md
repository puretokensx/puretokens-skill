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

Publish with `Publish verified stable release` using the matching immutable tag.
The workflow uploads all eleven assets to a draft, downloads and verifies them,
then promotes it to the latest stable release. A draft is visible only to
authorized maintainers and cannot serve ordinary installation. After promotion,
the workflow anonymously downloads all eleven assets through `releases/latest`
and verifies their bytes against the tagged source. Publication is complete only
after this public check passes.

The retired `puretokens-skill-install.zip` and
`puretokens-skill-install-payload.zip` migration bridges remain absent.
Superseded local version archives may be removed after confirming no active
packaging or verification uses them. Preserve the current candidate and its
manifest, independent migration/recovery evidence, and published immutable bytes.
