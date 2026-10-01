FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS build
ARG VERSION=dev
ARG COMMIT=unknown
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/chill-institute/chill-mcp/internal/buildinfo.version=${VERSION} -X github.com/chill-institute/chill-mcp/internal/buildinfo.commit=${COMMIT}" \
    -o /out/chill-mcp ./cmd/chill-mcp

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
LABEL org.opencontainers.image.source="https://github.com/chill-institute/chill-mcp"
ENV CHILL_LISTEN_HOST=0.0.0.0 CHILL_LISTEN_PORT=7100 CHILL_ENGINE_BASE_URL=https://api.chill.institute
COPY --from=build /out/chill-mcp /chill-mcp
USER nonroot:nonroot
EXPOSE 7100
STOPSIGNAL SIGTERM
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 CMD ["/chill-mcp", "health"]
ENTRYPOINT ["/chill-mcp"]
CMD ["http"]
