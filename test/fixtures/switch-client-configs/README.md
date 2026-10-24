# Switch desktop writer fixtures

These four synthetic JSON files are byte-for-byte copies of
`Pure_Tokens_Switch/tests/fixtures/client-configs/{claude-desktop,grok-build}/{macos,windows}.json`
reviewed on 2026-10-24 against Switch `878bf024` plus its working tree.
All credentials are fixture literals, not user secrets.

The Go tests consume these committed copies without Switch, a running client,
network access or an environment variable. They cover Claude Desktop's
default-first model list and legacy shape, Grok Build's exact and legacy display
names, invalid active selections and read-only behavior. Path-shaped fixtures
from both platforms do not prove Windows execution or real-host attachment
delivery. Refresh the copies and their provenance together when writers change.
