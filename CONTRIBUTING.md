# Contributing

```bash
mise install
mise run hooks
mise run verify
mise run smoke
```

Pre-commit formats staged Go files; pre-push runs `mise run verify`.

When changing a tool, update its description and annotations, the protocol
tests in `internal/server/server_test.go`, and the table in `README.md`.
Keep `chill-cli` as the source of procedure names and validators; add missing
validation there first.
