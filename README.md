# chill-mcp

![chill.institute mcp](https://raw.githubusercontent.com/chill-institute/.github/main/assets/banner.png)

The [Model Context Protocol](https://modelcontextprotocol.io) server for
[chill.institute](https://chill.institute): search indexers, browse catalogs,
and send downloads to put.io, one validated API call per tool.

## Install

Get a token at <https://chill.institute/auth/chilly-token> and treat it as a
password. Signing out doesn't revoke it; removing chill.institute's access in
your put.io settings does, within a day. Clients that send custom headers use
`https://mcp.chill.institute` with that bearer:

```bash
claude mcp add --transport http chill https://mcp.chill.institute --header "Authorization: Bearer <token>"
```

OAuth-only clients, such as Claude.ai and ChatGPT connectors, need local stdio
with a `chilly auth login` profile (`--profile`, `--config`, and `--api-url` as
in `chilly`):

```bash
go install github.com/chill-institute/chill-mcp/cmd/chill-mcp@latest
# client config: { "mcpServers": { "chill": { "command": "chill-mcp", "args": ["stdio"] } } }
```

## Use

Use your client's MCP tool discovery for the installed server's tools, input
schemas, and descriptions. Definitions live in [internal/server](./internal/server/).
Mutations are annotated so clients ask first; `dry_run: true` previews the
request without sending it. Results are the API's JSON.

## Develop

See [CONTRIBUTING.md](./CONTRIBUTING.md). Configuration and defaults live in
[cmd/chill-mcp/main.go](./cmd/chill-mcp/main.go). Verify a release image with
`gh attestation verify oci://ghcr.io/chill-institute/chill-mcp:X.Y.Z --owner chill-institute`.

[Security](./SECURITY.md) · [MIT License](./LICENSE)
