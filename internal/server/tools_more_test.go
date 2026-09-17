package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMoreToolsForwardValidatedRequests(t *testing.T) {
	t.Parallel()
	engine := newFakeEngine(t)
	session := connectInMemory(t, newTestServer(t, engine, staticToken("tok")))

	cases := []struct {
		tool      string
		args      map[string]any
		procedure string
		body      map[string]any
	}{
		{"get_tv_show", map[string]any{"imdb_id": " TT0944947 "}, "chill.v4.UserService/GetTVShowDetail", map[string]any{"imdbId": "tt0944947"}},
		{"get_tv_show_season", map[string]any{"imdb_id": "tt0944947", "season": 2}, "chill.v4.UserService/GetTVShowSeason", map[string]any{"imdbId": "tt0944947", "seasonNumber": float64(2)}},
		{"find_episode_download", map[string]any{"imdb_id": "tt0944947", "season": 2, "episode": 5}, "chill.v4.UserService/GetTVShowEpisodeDownload", map[string]any{"imdbId": "tt0944947", "seasonNumber": float64(2), "episodeNumber": float64(5)}},
		{"find_season_downloads", map[string]any{"imdb_id": "tt0944947", "season": 1}, "chill.v4.UserService/GetTVShowSeasonDownloads", map[string]any{"imdbId": "tt0944947", "seasonNumber": float64(1)}},
		{"list_indexers", map[string]any{}, "chill.v4.UserService/GetIndexers", map[string]any{}},
		{"get_download_folder", map[string]any{}, "chill.v4.UserService/GetDownloadFolder", map[string]any{}},
		{"browse_folder", map[string]any{"id": 0}, "chill.v4.UserService/GetFolder", map[string]any{"id": float64(0)}},
		{"get_user_settings", map[string]any{}, "chill.v4.UserService/GetUserSettings", map[string]any{}},
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
		gotBody, _ := json.Marshal(call.Body)
		wantBody, _ := json.Marshal(tc.body)
		if string(gotBody) != string(wantBody) {
			t.Fatalf("%s: body = %s, want %s", tc.tool, gotBody, wantBody)
		}
	}
}

func TestUpdateUserSettingReadsThenSaves(t *testing.T) {
	t.Parallel()
	engine := newFakeEngine(t)
	engine.fail = func(procedure string) (int, string) {
		if strings.HasSuffix(procedure, "GetUserSettings") {
			return 200, `{"search":{"sortBy":"SORT_BY_TITLE"},"catalog":{"moviesSource":"MOVIES_SOURCE_YTS"},"download":{"folderId":"7"},"ignored":true}`
		}
		return 0, ""
	}
	session := connectInMemory(t, newTestServer(t, engine, staticToken("tok")))

	dry, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "update_user_setting", Arguments: map[string]any{"field": "catalog-sort", "value": "rating-desc", "dry_run": true}})
	if err != nil || dry.IsError {
		t.Fatalf("dry run = %v, %v", dry, err)
	}
	engine.mu.Lock()
	calls := len(engine.calls)
	engine.mu.Unlock()
	if calls != 0 {
		t.Fatalf("dry run reached engine: %d calls", calls)
	}

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "update_user_setting", Arguments: map[string]any{"field": "catalog-sort", "value": "rating-desc"}})
	if err != nil || result.IsError {
		t.Fatalf("update = %v, %v", result, err)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if len(engine.calls) != 2 || !strings.HasSuffix(engine.calls[0].Procedure, "GetUserSettings") || !strings.HasSuffix(engine.calls[1].Procedure, "SaveUserSettings") {
		t.Fatalf("calls = %#v", engine.calls)
	}
	saved, _ := json.Marshal(engine.calls[1].Body)
	want := `{"settings":{"catalog":{"moviesSource":"MOVIES_SOURCE_YTS","sort":"CATALOG_SORT_RATING_DESC"},"download":{"folderId":"7"},"search":{"sortBy":"SORT_BY_TITLE"}}}`
	if string(saved) != want {
		t.Fatalf("saved = %s\nwant  = %s", saved, want)
	}
}

func TestMoreToolValidationErrors(t *testing.T) {
	t.Parallel()
	session := connectInMemory(t, newTestServer(t, newFakeEngine(t), staticToken("tok")))
	cases := []struct {
		tool string
		args map[string]any
		want string
	}{
		{"get_tv_show", map[string]any{"imdb_id": "nm0000001"}, "invalid_imdb_id"},
		{"get_tv_show_season", map[string]any{"imdb_id": "tt0944947", "season": 0}, "invalid_season_number"},
		{"find_episode_download", map[string]any{"imdb_id": "tt0944947", "season": 1, "episode": -1}, "invalid_episode_number"},
		{"browse_folder", map[string]any{"id": -5}, "invalid_folder_id"},
		{"update_user_setting", map[string]any{"field": "nope", "value": "1"}, "unsupported_user_settings_field"},
		{"update_user_setting", map[string]any{"field": "filter-nasty-results", "value": "maybe"}, "invalid_user_settings_value"},
	}
	for _, tc := range cases {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
		if err != nil {
			t.Fatalf("%s: protocol error = %v", tc.tool, err)
		}
		text, _ := result.Content[0].(*mcp.TextContent)
		if !result.IsError || text == nil || !strings.Contains(text.Text, tc.want) {
			t.Fatalf("%s: result = %#v, want error %q", tc.tool, result.Content, tc.want)
		}
	}
	fields, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_user_setting_fields", Arguments: map[string]any{}})
	if err != nil || fields.IsError {
		t.Fatalf("fields = %v, %v", fields, err)
	}
	payload, _ := json.Marshal(fields.StructuredContent)
	if !strings.Contains(string(payload), `"field":"catalog.sort"`) {
		t.Fatalf("fields payload = %s", payload)
	}
}
