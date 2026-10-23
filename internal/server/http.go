package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPPath is the Streamable HTTP endpoint: the site root.
const MCPPath = "/"

// LegacyMCPPath is the original endpoint, kept as an alias for clients that
// were configured before the root became canonical.
const LegacyMCPPath = "/mcp"

// HealthPath reports process liveness without contacting Engine.
const HealthPath = "/health"

// MaxRequestBodyBytes bounds one MCP request body. Tool arguments are short
// strings, so this leaves ample room while keeping each request small.
const MaxRequestBodyBytes = 64 << 10

type bearerKey struct{}

// ErrMissingBearer is returned by HTTPTokenSource when a request carried no token.
var ErrMissingBearer = errors.New("auth_error: Authorization: Bearer <chill.institute token> header is required")

// HTTPTokenSource reads the bearer stored by RequireBearer.
func HTTPTokenSource(ctx context.Context, _ *mcp.CallToolRequest) (string, error) {
	token, _ := ctx.Value(bearerKey{}).(string)
	if token == "" {
		return "", ErrMissingBearer
	}
	return token, nil
}

// RequireBearer rejects requests without a well-formed bearer token and
// stores the token in the request context for HTTPTokenSource. Engine remains
// the authority on whether the token is valid.
func RequireBearer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		token, ok := bearerFromHeader(request.Header.Get("Authorization"))
		if !ok {
			writer.Header().Set("WWW-Authenticate", `Bearer realm="chill.institute", error="invalid_token"`)
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(writer).Encode(map[string]string{
				"error":   "unauthorized",
				"message": "send Authorization: Bearer <chill.institute token>",
			})
			return
		}
		next.ServeHTTP(writer, request.WithContext(context.WithValue(request.Context(), bearerKey{}, token)))
	})
}

func bearerFromHeader(value string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(value), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	if token == "" || len(token) > 4096 {
		return "", false
	}
	for _, r := range token {
		if r <= ' ' || r > '~' {
			return "", false
		}
	}
	return token, true
}

// rejectBatches answers JSON-RPC batch bodies with 400 before the SDK sees
// them. The SDK still accepts batches from pre-2025-06-18 clients, and in
// stateless mode each element would become its own concurrent Engine call.
func rejectBatches(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, MaxRequestBodyBytes))
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				http.Error(writer, fmt.Sprintf("request body exceeds %d bytes", tooLarge.Limit), http.StatusRequestEntityTooLarge)
				return
			}
			http.Error(writer, "failed to read body", http.StatusBadRequest)
			return
		}
		if trimmed := bytes.TrimLeft(body, " \t\r\n"); len(trimmed) > 0 && trimmed[0] == '[' {
			http.Error(writer, "JSON-RPC batching is not supported", http.StatusBadRequest)
			return
		}
		request.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(writer, request)
	})
}

// Handler serves the stateless Streamable HTTP endpoint and the health route.
func (server *Server) Handler() http.Handler {
	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server.mcp }, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
		MaxRequestBodyBytes:          MaxRequestBodyBytes,
	})
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+HealthPath, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "no-store")
		_, _ = writer.Write([]byte(`{"status":"ok"}` + "\n"))
	})
	gated := RequireBearer(rejectBatches(streamable))
	mux.Handle("POST "+MCPPath+"{$}", gated)
	mux.Handle("POST "+LegacyMCPPath, gated)
	return mux
}
