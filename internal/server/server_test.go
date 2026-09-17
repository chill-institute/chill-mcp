package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type recordedCall struct {
	Procedure string
	Auth      string
	Body      map[string]any
}

type fakeEngine struct {
	*httptest.Server
	mu    sync.Mutex
	calls []recordedCall
	fail  func(procedure string) (int, string)
}

func newFakeEngine(t *testing.T) *fakeEngine {
	t.Helper()
	engine := &fakeEngine{}
	engine.Server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		procedure := strings.TrimPrefix(request.URL.Path, "/v4/")
		var body map[string]any
		_ = json.NewDecoder(request.Body).Decode(&body)
		engine.mu.Lock()
		engine.calls = append(engine.calls, recordedCall{Procedure: procedure, Auth: request.Header.Get("Authorization"), Body: body})
		engine.mu.Unlock()
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Request-Id", "req-1")
		if engine.fail != nil {
			if status, payload := engine.fail(procedure); status != 0 {
				writer.WriteHeader(status)
				_, _ = writer.Write([]byte(payload))
				return
			}
		}
		_, _ = writer.Write([]byte(`{"procedure":"` + procedure + `","client":"` + request.Header.Get("X-Chill-Client") + `"}`))
	}))
	t.Cleanup(engine.Close)
	return engine
}

func (engine *fakeEngine) last(t *testing.T) recordedCall {
	t.Helper()
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if len(engine.calls) == 0 {
		t.Fatal("engine received no calls")
	}
	return engine.calls[len(engine.calls)-1]
}

func newTestServer(t *testing.T, engine *fakeEngine, token TokenSource) *Server {
	t.Helper()
	server, err := New(Options{
		Version: "test",
		API:     rpc.NewClient(engine.URL, engine.Client(), rpc.WithClientName(ClientName)),
		Token:   token,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return server
}

func connectInMemory(t *testing.T, server *Server) *mcp.ClientSession {
	t.Helper()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.MCP().Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func staticToken(token string) TokenSource {
	return func(context.Context, *mcp.CallToolRequest) (string, error) { return token, nil }
}

func TestToolListMatchesContract(t *testing.T) {
	t.Parallel()
	session := connectInMemory(t, newTestServer(t, newFakeEngine(t), staticToken("tok")))
	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	want := map[string]bool{"search_releases": true, "list_movies": true, "list_tv_shows": true, "get_transfer": true, "whoami": true, "add_transfer": false, "get_tv_show": true, "get_tv_show_season": true, "find_episode_download": true, "find_season_downloads": true, "list_indexers": true, "get_download_folder": true, "browse_folder": true, "get_user_settings": true, "list_user_setting_fields": true, "update_user_setting": false}
	if len(result.Tools) != len(want) {
		t.Fatalf("tool count = %d, want %d", len(result.Tools), len(want))
	}
	for _, tool := range result.Tools {
		readOnly, ok := want[tool.Name]
		if !ok {
			t.Fatalf("unexpected tool %q", tool.Name)
		}
		if tool.Annotations == nil || tool.Annotations.ReadOnlyHint != readOnly {
			t.Fatalf("%s read-only hint = %v, want %v", tool.Name, tool.Annotations, readOnly)
		}
		if tool.Description == "" {
			t.Fatalf("%s has no description", tool.Name)
		}
	}
}

func TestToolsForwardValidatedRequests(t *testing.T) {
	t.Parallel()
	engine := newFakeEngine(t)
	session := connectInMemory(t, newTestServer(t, engine, staticToken("secret-token")))

	cases := []struct {
		tool      string
		args      map[string]any
		procedure string
		body      map[string]any
	}{
		{"search_releases", map[string]any{"query": " dune ", "indexer_id": "yts"}, "chill.v4.UserService/Search", map[string]any{"query": "dune", "indexer_id": "yts"}},
		{"list_movies", map[string]any{}, "chill.v4.UserService/GetMovies", map[string]any{}},
		{"list_tv_shows", map[string]any{"source": "hbo max"}, "chill.v4.UserService/GetTVShows", map[string]any{"source": "TV_SHOWS_SOURCE_HBO_MAX"}},
		{"get_transfer", map[string]any{"id": 42}, "chill.v4.UserService/GetTransfer", map[string]any{"id": float64(42)}},
		{"whoami", map[string]any{}, "chill.v4.UserService/GetUserProfile", map[string]any{}},
		{"add_transfer", map[string]any{"url": "magnet:?xt=urn:btih:abc", "movie_source": "yts"}, "chill.v4.UserService/AddTransfer", map[string]any{"url": "magnet:?xt=urn:btih:abc", "catalogOrigin": map[string]any{"moviesSource": "MOVIES_SOURCE_YTS"}}},
	}
	for _, tc := range cases {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
		if err != nil {
			t.Fatalf("%s: CallTool() error = %v", tc.tool, err)
		}
		if result.IsError {
			t.Fatalf("%s: tool error %v", tc.tool, result.Content)
		}
		call := engine.last(t)
		if call.Procedure != tc.procedure {
			t.Fatalf("%s: procedure = %q, want %q", tc.tool, call.Procedure, tc.procedure)
		}
		if call.Auth != "Bearer secret-token" {
			t.Fatalf("%s: authorization = %q", tc.tool, call.Auth)
		}
		gotBody, _ := json.Marshal(call.Body)
		wantBody, _ := json.Marshal(tc.body)
		if string(gotBody) != string(wantBody) {
			t.Fatalf("%s: body = %s, want %s", tc.tool, gotBody, wantBody)
		}
		structured, _ := json.Marshal(result.StructuredContent)
		if !strings.Contains(string(structured), `"client":"mcp"`) {
			t.Fatalf("%s: structured content = %s", tc.tool, structured)
		}
	}
}

func TestAddTransferDryRunNeverCallsEngine(t *testing.T) {
	t.Parallel()
	engine := newFakeEngine(t)
	session := connectInMemory(t, newTestServer(t, engine, staticToken("tok")))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "add_transfer", Arguments: map[string]any{"url": "https://example.com/a.torrent", "tv_source": "netflix", "dry_run": true}})
	if err != nil || result.IsError {
		t.Fatalf("CallTool() = %v, %v", result, err)
	}
	engine.mu.Lock()
	calls := len(engine.calls)
	engine.mu.Unlock()
	if calls != 0 {
		t.Fatalf("engine calls = %d, want 0", calls)
	}
	payload, _ := json.Marshal(result.StructuredContent)
	if !strings.Contains(string(payload), `"dry_run":true`) || !strings.Contains(string(payload), `"tvShowsSource":"TV_SHOWS_SOURCE_NETFLIX"`) {
		t.Fatalf("dry run payload = %s", payload)
	}
}

func TestToolErrorsStayInsideResults(t *testing.T) {
	t.Parallel()
	engine := newFakeEngine(t)
	engine.fail = func(procedure string) (int, string) {
		if strings.HasSuffix(procedure, "GetUserProfile") {
			return 401, `{"code":"invalid_auth_token","message":"token revoked","request_id":"req-9"}`
		}
		return 0, ""
	}
	session := connectInMemory(t, newTestServer(t, engine, staticToken("tok")))

	cases := []struct {
		tool string
		args map[string]any
		want string
	}{
		{"search_releases", map[string]any{"query": "  "}, "missing_query"},
		{"search_releases", map[string]any{"query": "x", "indexer_id": "../etc"}, "invalid_indexer_id"},
		{"add_transfer", map[string]any{"url": "ftp://x/y"}, "invalid_url"},
		{"add_transfer", map[string]any{"url": "magnet:?xt=a", "movie_source": "yts", "tv_source": "hulu"}, "ambiguous_transfer_source"},
		{"get_transfer", map[string]any{"id": 0}, "invalid_transfer_id"},
		{"whoami", map[string]any{}, "auth_error: token revoked (request req-9)"},
	}
	for _, tc := range cases {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
		if err != nil {
			t.Fatalf("%s: protocol error = %v", tc.tool, err)
		}
		if !result.IsError {
			t.Fatalf("%s: expected tool error", tc.tool)
		}
		text, _ := result.Content[0].(*mcp.TextContent)
		if text == nil || !strings.Contains(text.Text, tc.want) {
			t.Fatalf("%s: error content = %#v, want %q", tc.tool, result.Content[0], tc.want)
		}
	}
}

func TestHTTPRequiresBearerAndForwardsIt(t *testing.T) {
	t.Parallel()
	engine := newFakeEngine(t)
	handler := newTestServer(t, engine, HTTPTokenSource).Handler()
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)

	health, err := http.Get(web.URL + HealthPath)
	if err != nil || health.StatusCode != http.StatusOK {
		t.Fatalf("health = %v, %v", health, err)
	}
	_ = health.Body.Close()

	for _, header := range []string{"", "Basic abc", "Bearer", "Bearer bad token"} {
		request, _ := http.NewRequest(http.MethodPost, web.URL+MCPPath, strings.NewReader(`{}`))
		if header != "" {
			request.Header.Set("Authorization", header)
		}
		response, err := web.Client().Do(request)
		if err != nil {
			t.Fatalf("request error = %v", err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized || !strings.HasPrefix(response.Header.Get("WWW-Authenticate"), "Bearer") {
			t.Fatalf("header %q: status = %d, www-authenticate = %q", header, response.StatusCode, response.Header.Get("WWW-Authenticate"))
		}
	}

	for _, path := range []string{MCPPath, LegacyMCPPath} {
		client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
		httpClient := &http.Client{Transport: bearerTransport{token: "user-token", next: web.Client().Transport}}
		session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: web.URL + path, HTTPClient: httpClient, DisableStandaloneSSE: true}, nil)
		if err != nil {
			t.Fatalf("%s: Connect() error = %v", path, err)
		}
		t.Cleanup(func() { _ = session.Close() })
		if session.ID() != "" {
			t.Fatalf("%s: session id = %q, want stateless", path, session.ID())
		}
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "whoami", Arguments: map[string]any{}})
		if err != nil || result.IsError {
			t.Fatalf("%s: CallTool() = %v, %v", path, result, err)
		}
		if call := engine.last(t); call.Auth != "Bearer user-token" {
			t.Fatalf("%s: forwarded authorization = %q", path, call.Auth)
		}
	}

	for _, path := range []string{"/other", "/mcp/extra"} {
		request, _ := http.NewRequest(http.MethodPost, web.URL+path, strings.NewReader(`{}`))
		request.Header.Set("Authorization", "Bearer user-token")
		response, err := web.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("%s: status = %d, want 404", path, response.StatusCode)
		}
	}
	get, err := http.Get(web.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	_ = get.Body.Close()
	if get.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET / status = %d, want 405", get.StatusCode)
	}
}

type bearerTransport struct {
	token string
	next  http.RoundTripper
}

func (transport bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	cloned := request.Clone(request.Context())
	cloned.Header.Set("Authorization", "Bearer "+transport.token)
	next := transport.next
	if next == nil {
		next = http.DefaultTransport
	}
	return next.RoundTrip(cloned)
}
