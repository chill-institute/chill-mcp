package server

import (
	"context"
	"encoding/json"
	"errors"
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

// Handler serves the stateless Streamable HTTP endpoint and the health route.
func (server *Server) Handler() http.Handler {
	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server.mcp }, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
	})
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+HealthPath, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "no-store")
		_, _ = writer.Write([]byte(`{"status":"ok"}` + "\n"))
	})
	gated := RequireBearer(streamable)
	mux.Handle("POST "+MCPPath+"{$}", gated)
	mux.Handle("POST "+LegacyMCPPath, gated)
	return mux
}
