# chill-mcp

Model Context Protocol server for [chill.institute](https://chill.institute).
Agents search release indexers, browse the movie and TV catalogs, and send
downloads to put.io with the user's own account. Every tool is one validated
call to the hosted v4 API through [chill-cli](https://github.com/chill-institute/chill-cli)'s
public client.

No OAuth, no sessions. The hosted endpoint is a stateless Streamable HTTP
server that forwards the caller's chill.institute bearer to the API. The stdio
mode reuses a local `chilly` profile.

## Hosted

Add it to any client that sends custom headers:

```json
{
  "mcpServers": {
    "chill": {
      "type": "http",
      "url": "https://mcp.chill.institute/mcp",
      "headers": { "Authorization": "Bearer <token from https://chill.institute/token>" }
    }
  }
}
```

```sh
claude mcp add --transport http chill https://mcp.chill.institute/mcp \
  --header "Authorization: Bearer <token>"
```

Clients that only accept OAuth remote servers (Claude.ai and ChatGPT
connectors) cannot use the hosted endpoint; use stdio.

## Local stdio

```sh
go install github.com/chill-institute/chill-mcp/cmd/chill-mcp@latest
chilly auth login
```

```json
{ "mcpServers": { "chill": { "command": "chill-mcp", "args": ["stdio"] } } }
```

`--profile`, `--config`, and `--api-url` mirror `chilly`.

## Tools

| Tool | Procedure | Notes |
| --- | --- | --- |
| `search_releases` | `UserService/Search` | `query`, optional `indexer_id` |
| `list_movies` | `UserService/GetMovies` | user's movie source and sort |
| `list_tv_shows` | `UserService/GetTVShows` | optional provider `source` |
| `get_transfer` | `UserService/GetTransfer` | `id` |
| `whoami` | `UserService/GetUserProfile` | |
| `add_transfer` | `UserService/AddTransfer` | `url`, optional `movie_source` or `tv_source`, `dry_run` |

Read-only tools carry `readOnlyHint`; `add_transfer` is marked destructive so
clients prompt. Input is validated locally before any request is built.

## Run

```sh
mise install
mise run verify
mise run smoke
CHILL_LISTEN_HOST=127.0.0.1 go run ./cmd/chill-mcp http
```

| Variable | Default | Purpose |
| --- | --- | --- |
| `CHILL_LISTEN_HOST` | `127.0.0.1` | bind address; the image sets `0.0.0.0` |
| `CHILL_LISTEN_PORT` | `7100` | listen port |
| `CHILL_ENGINE_BASE_URL` | `https://api.chill.institute` | hosted API base |

`GET /health` reports liveness without contacting the API. `chill-mcp health`
probes it from inside the distroless image. Requests to `/mcp` without a
well-formed bearer get `401` and `WWW-Authenticate: Bearer`; the API remains
the authority on whether a token is valid. The process logs no headers,
tokens, or response bodies.

Every push to `main` publishes `ghcr.io/chill-institute/chill-mcp` with an
immutable `<sha>-<run>-<attempt>` tag and a moving `main` tag. Production
deploys pin the digest; see [chill-engine deployment](https://github.com/chill-institute/chill-engine/blob/main/docs/DEPLOYMENT.md).
