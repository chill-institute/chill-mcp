# Contributing

```bash
mise install
mise run hooks   # pre-commit formats staged Go; pre-push runs verify
mise run verify
mise run smoke
```

When changing a tool, update its description and annotations, the protocol
tests in `internal/server/server_test.go`, and the [tool table](./README.md#use).
Add missing procedure names or validators to `chill-cli` first.
