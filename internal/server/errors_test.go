package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolErrorsNeverEchoTokensOrBodies(t *testing.T) {
	t.Parallel()
	engine := newFakeEngine(t)
	engine.fail = func(procedure string) (int, string) {
		switch {
		case strings.HasSuffix(procedure, "GetUserProfile"):
			return 500, `{"code":"boom","message":"internal","request_id":"req-5","echo":"Bearer super-secret-token"}`
		case strings.HasSuffix(procedure, "GetMovies"):
			return 502, `<html><body>bad gateway super-secret-token</body></html>`
		case strings.HasSuffix(procedure, "GetIndexers"):
			return 403, `{"code":"plan_required","message":"upgrade needed","request_id":"req-7"}`
		}
		return 0, ""
	}
	session := connectInMemory(t, newTestServer(t, engine, staticToken("super-secret-token")))

	cases := []struct {
		tool     string
		want     string
		mustMiss []string
	}{
		{"whoami", "boom: internal (request req-5)", []string{"super-secret-token", "echo"}},
		{"list_movies", "api_error: hosted API returned status 502 (request req-1)", []string{"super-secret-token", "<html>"}},
		{"list_indexers", "plan_required: upgrade needed (request req-7)", []string{"auth_error", "super-secret-token"}},
	}
	for _, tc := range cases {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.tool, Arguments: map[string]any{}})
		if err != nil || !result.IsError {
			t.Fatalf("%s: result = %v, err = %v", tc.tool, result, err)
		}
		text := result.Content[0].(*mcp.TextContent).Text
		if !strings.Contains(text, tc.want) {
			t.Fatalf("%s: text = %q, want %q", tc.tool, text, tc.want)
		}
		for _, miss := range tc.mustMiss {
			if strings.Contains(text, miss) {
				t.Fatalf("%s: text %q leaks %q", tc.tool, text, miss)
			}
		}
	}
}

func TestUpdateUserSettingRefusesWithoutCurrentSettings(t *testing.T) {
	t.Parallel()
	for name, payload := range map[string]struct {
		status int
		body   string
		want   string
	}{
		"null":    {200, `null`, "settings_unavailable"},
		"empty":   {200, `{}`, "settings_unavailable"},
		"failure": {500, `{"code":"down","message":"redis","request_id":"r"}`, "down: redis"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			engine := newFakeEngine(t)
			engine.fail = func(procedure string) (int, string) {
				if strings.HasSuffix(procedure, "GetUserSettings") {
					return payload.status, payload.body
				}
				return 0, ""
			}
			session := connectInMemory(t, newTestServer(t, engine, staticToken("tok")))
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "update_user_setting", Arguments: map[string]any{"field": "catalog-sort", "value": "popularity"}})
			if err != nil || !result.IsError {
				t.Fatalf("result = %v, err = %v", result, err)
			}
			if text := result.Content[0].(*mcp.TextContent).Text; !strings.Contains(text, payload.want) {
				t.Fatalf("text = %q, want %q", text, payload.want)
			}
			engine.mu.Lock()
			defer engine.mu.Unlock()
			for _, call := range engine.calls {
				if strings.HasSuffix(call.Procedure, "SaveUserSettings") {
					body, _ := json.Marshal(call.Body)
					t.Fatalf("SaveUserSettings was called with %s", body)
				}
			}
		})
	}
}
