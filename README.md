# chill-mcp

![chill.institute mcp](https://chill.institute/banner.png)

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

| Tool | Does | Input |
| --- | --- | --- |
| `search_releases` | Search indexers with your saved filters | `query`, `indexer_id`? |
| `list_movies` | Movie catalog for your source and sort | |
| `list_tv_shows` | TV catalog for a provider | `source`? |
| `get_tv_show` | One show and its seasons | `imdb_id` |
| `get_tv_show_season` | Episodes of a season | `imdb_id`, `season` |
| `find_episode_download` | Best release for an episode | `imdb_id`, `season`, `episode` |
| `find_season_downloads` | Season pack and per-episode releases | `imdb_id`, `season` |
| `list_indexers` | Your indexers and their health | |
| `get_download_folder` | Where new transfers land | |
| `browse_folder` | Contents of a put.io folder | `id` |
| `get_transfer` | One put.io transfer | `id` |
| `whoami` | Your account profile | |
| `get_user_settings` | Search, catalog, and download settings | |
| `list_user_setting_fields` | Fields `update_user_setting` accepts | |
| `update_user_setting` | Change one setting | `field`, `value`, `dry_run`? |
| `add_transfer` | Send a release to put.io | `url`, `movie_source`? or `tv_source`?, `dry_run`? |

`?` marks optional input. Mutations are annotated so clients ask first, and
`dry_run: true` returns the request unsent. Results are the API's JSON.

## Develop

See [CONTRIBUTING.md](./CONTRIBUTING.md). For `http`, `CHILL_LISTEN_HOST` and
`CHILL_LISTEN_PORT` set the bind address ([defaults](./cmd/chill-mcp/main.go)),
and `CHILL_ENGINE_BASE_URL` sets the API base (`https://api.chill.institute`
when unset). Verify a release image with
`gh attestation verify oci://ghcr.io/chill-institute/chill-mcp:X.Y.Z --owner chill-institute`.

[Architecture](./docs/architecture.md) · [Security](./SECURITY.md) ·
[MIT License](./LICENSE)
