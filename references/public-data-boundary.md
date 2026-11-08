# Public data boundary

Public model brands and exact model IDs are product capabilities, not private
supplier identities. Preserve them and the parameters needed for correct calls.

## Catalog output and publication

`runtime/executor/public-catalog-projection.json` defines the reviewed public
schema structure shared by the native receipt projection and development-time
catalog/profile publication checks. Parameter names, operation names and the
declared dimension/format maps remain dynamic. Metadata, descriptions, examples,
unknown constraint families and arbitrary nested objects are not forwarded.

Receipt projection does not change the executor's full validation schema, fixed
request target, authentication or existing task recovery. The publication tools
stop on unknown structure instead of dropping a potentially important new
constraint. Review a new field and extend this contract with regression tests
before publishing it. Never print rejected values in diagnostics.

This is a structural boundary, not a promise to identify arbitrary private text
misplaced inside a legitimate model ID, enum or default. Private metadata must
also stay out of the server's public catalog. Import checks do not establish
live entitlement, paid generation or real-client delivery.

## Public acceptance records

Retain dates, scope, model IDs, safe response categories and pass/fail evidence.
Omit private supplier associations, implementation identities, deployment
digests and per-user timestamps or usage combinations unnecessary to explain
the result. Detailed incident evidence belongs in access-controlled operational
records, not this public repository or installed Skills.

`scripts/validate-publication-privacy.mjs` prevents known removed identities from
returning to public acceptance documents. It is a bounded regression check,
not a full secret scanner, semantic classifier or Git-history scrubber.

## Remaining protocol dependency

The current asynchronous audio submission path and task platform discriminator
contain a legacy implementation-specific marker. The executor, exact-model
profiles, schemas and executable contracts still need those literal values.
They remain visible in source and native packages. No compatibility exception
here means that their disclosure is fixed.

Complete removal requires a gateway-owned neutral submission route and neutral
task identity contract, with backwards compatibility for existing tasks. Before
changing this repository, confirm the deployed route and the status/content
behavior for both old and new tasks. Preserve original task ID checks, account
and Key binding, fixed-origin requests and no-resubmit behavior. Do not guess a
new route, conceal the literal or disable platform validation as a workaround.

This repair does not change the gateway, Switch, published releases or Git
history. Prior public copies cannot be recalled by editing the current tree.
