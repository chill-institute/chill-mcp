# chill-mcp

![chill.institute mcp](https://chill.institute/banner.png)

`chill-mcp` is the [Model Context Protocol](https://modelcontextprotocol.io)
server for [chill.institute](https://chill.institute). Agents search release
indexers, browse the movie and TV catalogs, and send downloads to put.io with
your own account. Each tool is one validated call to the hosted API through the
public [chill-cli](https://github.com/chill-institute/chill-cli) client.

No OAuth, no sessions. The hosted endpoint is stateless Streamable HTTP that
forwards your chill.institute bearer to the API. The stdio mode reuses a local
`chilly` profile.

## Install

Hosted, for any client that sends custom headers:

```bash
claude mcp add --transport http chill https://mcp.chill.institute \
  --header "Authorization: Bearer <token>"
```

```json
{
  "mcpServers": {
    "chill": {
      "type": "http",
      "url": "https://mcp.chill.institute",
      "headers": { "Authorization": "Bearer <token>" }
    }
  }
}
```

Local stdio, reusing your `chilly` login:

```bash
go install github.com/chill-institute/chill-mcp/cmd/chill-mcp@latest
```

```json
{ "mcpServers": { "chill": { "command": "chill-mcp", "args": ["stdio"] } } }
```

`--profile`, `--config`, and `--api-url` mirror `chilly`.

## Sign In

Open <https://chill.institute/auth/setup-token> in a signed-in browser and copy
the token. Use it as the bearer above, or run `chilly auth login` for stdio.
Treat the token as a password: it lives in your client config and stays valid
until you sign out of chill.institute or replace it.

Clients that only accept OAuth remote servers, such as Claude.ai and ChatGPT
connectors, cannot use the hosted endpoint. Use stdio there.

## Use

| Tool | Does | Input |
| --- | --- | --- |
| `search_releases` | Search indexers with your saved filters | `query`, optional `indexer_id` |
| `list_movies` | Movie catalog for your source and sort | |
| `list_tv_shows` | TV catalog for your provider | optional `source` |
| `get_tv_show` | One show with its seasons | `imdb_id` |
| `get_tv_show_season` | Episodes of one season | `imdb_id`, `season` |
| `find_episode_download` | Best release for one episode | `imdb_id`, `season`, `episode` |
| `find_season_downloads` | Season pack plus per-episode releases | `imdb_id`, `season` |
| `list_indexers` | Your indexers with health | |
| `get_download_folder` | Where new transfers land | |
| `browse_folder` | Files and subfolders of one put.io folder | `id` |
| `get_transfer` | One put.io transfer | `id` |
| `whoami` | Your account profile | |
| `get_user_settings` | Search, catalog, and download settings | |
| `list_user_setting_fields` | Fields `update_user_setting` accepts | |
| `update_user_setting` | Change one setting | `field`, `value`, `dry_run` |
| `add_transfer` | Send a release to put.io | `url`, optional `movie_source` or `tv_source`, `dry_run` |

Read-only tools carry `readOnlyHint`. `add_transfer` and `update_user_setting`
are marked as mutations so clients ask first; pass `dry_run: true` to see the
exact request without sending it. Input is validated locally before any request
is built. Results are the API's JSON, returned as structured content.

## Develop

```bash
mise install
mise run hooks
mise run verify
mise run smoke
CHILL_LISTEN_HOST=127.0.0.1 go run ./cmd/chill-mcp http
```

| Variable | Default | Purpose |
| --- | --- | --- |
| `CHILL_LISTEN_HOST` | `127.0.0.1` | bind address; the image binds `0.0.0.0` |
| `CHILL_LISTEN_PORT` | `7100` | listen port |
| `CHILL_ENGINE_BASE_URL` | `https://api.chill.institute` | hosted API base |

`GET /health` reports liveness without contacting the API. `chill-mcp health`
probes it from inside the image. The MCP endpoint is the site root, with `/mcp`
as an alias. Requests to it without a well-formed bearer get `401` with
`WWW-Authenticate: Bearer`; the API decides whether a token is valid. The process logs no headers, tokens, or response bodies.

Every push to `main` cuts a semantic-release version and GitHub release, then
publishes `ghcr.io/chill-institute/chill-mcp` as `:X.Y.Z`, `:X.Y`, `:X`,
`:latest`, `:main`, and `:sha-<commit>`. The release notes carry the image
digest, and that digest is deployed to `mcp.chill.institute` through the
hosting repository's deploy workflow before the run finishes. Each published
digest carries build provenance:

```bash
gh attestation verify oci://ghcr.io/chill-institute/chill-mcp:X.Y.Z --owner chill-institute
```

[Architecture](./docs/ARCHITECTURE.md) · [Contributing](./CONTRIBUTING.md) ·
[Security](./SECURITY.md) · [MIT License](./LICENSE)
