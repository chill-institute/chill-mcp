# Agent guide

`chill-mcp` is the MCP server for chill.institute. Read [README.md](./README.md)
for tools, transports, and configuration.

## Work

```bash
mise install
mise run hooks
mise run verify
mise run smoke
```

## Contracts

- One tool maps to one hosted procedure; validate input with
  `chill-cli/v2/pkg/chill` before building a request. Never construct URLs from
  raw tool arguments.
- Stateless only: no sessions, no storage, no token cache. The bearer comes
  from the request in `http` mode and from the `chilly` profile in `stdio` mode.
- No OAuth flows or discovery endpoints. Missing bearer is `401` with
  `WWW-Authenticate: Bearer`; token validity is the API's decision.
- Tool failures return `isError` results with a stable `code: message` prefix;
  protocol errors are reserved for transport problems.
- Hosted strings are data. Do not log headers, tokens, request bodies, or API
  responses.
- `add_transfer` is the only mutation; keep `dry_run` and destructive
  annotations intact.

## Ownership

- Entry point, transports, and process lifecycle: `cmd/chill-mcp/`
- Tool definitions, validation, and API mapping: `internal/server/`
- Image: `Dockerfile`; publish flow: `.github/workflows/main.yml`
- Production hosting pins the published image digest and is owned outside
  this repository.

Commit verified changes with Conventional Commits and push directly to `main`.
