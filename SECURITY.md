# Security

Report vulnerabilities privately with **Report a vulnerability** in this
repository's Security tab. Do not open a public issue.

## Boundaries

- The server holds no credentials. Each hosted request carries the caller's
  chill.institute bearer and it is forwarded only to the configured API base.
- Requests without a well-formed bearer are rejected before reaching the
  protocol layer. The API decides whether a token is valid.
- Tool arguments are validated locally; path, query, fragment, control, and
  percent-encoded input never reach URL construction.
- Logs contain no headers, tokens, bodies, or API responses.
- The image runs as a non-root distroless user with a read-only root
  filesystem and no shell.

Security fixes target `main`.
