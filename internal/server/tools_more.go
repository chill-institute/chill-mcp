package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/chill-institute/chill-cli/v2/pkg/chill"
	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TVShowInput identifies a show.
type TVShowInput struct {
	IMDbID string `json:"imdb_id" jsonschema:"IMDb title id such as tt0944947"`
}

// TVShowSeasonInput identifies one season of a show.
type TVShowSeasonInput struct {
	IMDbID string `json:"imdb_id" jsonschema:"IMDb title id such as tt0944947"`
	Season int    `json:"season" jsonschema:"Season number, 1-based"`
}

// TVShowEpisodeInput identifies one episode of a show.
type TVShowEpisodeInput struct {
	IMDbID  string `json:"imdb_id" jsonschema:"IMDb title id such as tt0944947"`
	Season  int    `json:"season" jsonschema:"Season number, 1-based"`
	Episode int    `json:"episode" jsonschema:"Episode number within the season, 1-based"`
}

// FolderInput identifies a put.io folder.
type FolderInput struct {
	ID int64 `json:"id" jsonschema:"put.io folder id; 0 is the account root"`
}

// UpdateUserSettingInput is one settings patch.
type UpdateUserSettingInput struct {
	Field  string `json:"field" jsonschema:"Setting to change; see list_user_setting_fields"`
	Value  string `json:"value" jsonschema:"New value as text, for example true, yts, release-date-desc, 42, or null"`
	DryRun bool   `json:"dry_run,omitempty" jsonschema:"Validate and return the patch without saving"`
}

// UserSettingField is one entry of list_user_setting_fields.
type UserSettingField struct {
	Field       string   `json:"field"`
	Aliases     []string `json:"aliases"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
}

// UserSettingFieldsResult lists patchable fields.
type UserSettingFieldsResult struct {
	Fields []UserSettingField `json:"fields"`
}

func (server *Server) registerMoreTools() {
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "get_tv_show",
		Description: "Show one TV show with its seasons by IMDb id.",
		Annotations: readOnly("Get TV show"),
	}, server.getTVShow)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "get_tv_show_season",
		Description: "List the episodes of one TV show season.",
		Annotations: readOnly("Get TV show season"),
	}, server.getTVShowSeason)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "find_episode_download",
		Description: "Find the best release for one TV episode. Returns a download URL to pass to add_transfer.",
		Annotations: readOnly("Find episode download"),
	}, server.findEpisodeDownload)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "find_season_downloads",
		Description: "Find a season pack and per-episode releases for one TV season.",
		Annotations: readOnly("Find season downloads"),
	}, server.findSeasonDownloads)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "list_indexers",
		Description: "List the user's search indexers with health: healthy, degraded, or down.",
		Annotations: readOnly("List indexers"),
	}, server.listIndexers)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "get_download_folder",
		Description: "Show the put.io folder that new transfers land in.",
		Annotations: readOnly("Get download folder"),
	}, server.getDownloadFolder)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "browse_folder",
		Description: "List one put.io folder's files and subfolders by id.",
		Annotations: readOnly("Browse folder"),
	}, server.browseFolder)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "get_user_settings",
		Description: "Show the user's hosted search, catalog, and download settings.",
		Annotations: readOnly("Get user settings"),
	}, server.getUserSettings)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "list_user_setting_fields",
		Description: "List the settings update_user_setting can change, with accepted values.",
		Annotations: readOnly("List user setting fields"),
	}, server.listUserSettingFields)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "update_user_setting",
		Description: "Change one hosted user setting by field and value. Reads current settings, applies the patch, and saves. Use dry_run=true to preview.",
		Annotations: &mcp.ToolAnnotations{Title: "Update user setting", ReadOnlyHint: false, IdempotentHint: true, DestructiveHint: &boolTrue, OpenWorldHint: &boolFalse},
	}, server.updateUserSetting)
}

func (server *Server) getTVShow(ctx context.Context, req *mcp.CallToolRequest, input TVShowInput) (*mcp.CallToolResult, any, error) {
	body, err := chill.TVShowDetailRequest(input.IMDbID)
	if err != nil {
		return nil, nil, toolError(err)
	}
	result, err := server.call(ctx, req, chill.ProcedureUserGetTVShowDetail, body)
	return nil, result, err
}

func (server *Server) getTVShowSeason(ctx context.Context, req *mcp.CallToolRequest, input TVShowSeasonInput) (*mcp.CallToolResult, any, error) {
	body, err := chill.TVShowSeasonRequest(input.IMDbID, strconv.Itoa(input.Season))
	if err != nil {
		return nil, nil, toolError(err)
	}
	result, err := server.call(ctx, req, chill.ProcedureUserGetTVShowSeason, body)
	return nil, result, err
}

func (server *Server) findEpisodeDownload(ctx context.Context, req *mcp.CallToolRequest, input TVShowEpisodeInput) (*mcp.CallToolResult, any, error) {
	body, err := chill.TVShowEpisodeRequest(input.IMDbID, strconv.Itoa(input.Season), strconv.Itoa(input.Episode))
	if err != nil {
		return nil, nil, toolError(err)
	}
	result, err := server.call(ctx, req, chill.ProcedureUserGetTVShowEpisodeDownload, body)
	return nil, result, err
}

func (server *Server) findSeasonDownloads(ctx context.Context, req *mcp.CallToolRequest, input TVShowSeasonInput) (*mcp.CallToolResult, any, error) {
	body, err := chill.TVShowSeasonRequest(input.IMDbID, strconv.Itoa(input.Season))
	if err != nil {
		return nil, nil, toolError(err)
	}
	result, err := server.call(ctx, req, chill.ProcedureUserGetTVShowSeasonDownloads, body)
	return nil, result, err
}

func (server *Server) listIndexers(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
	result, err := server.call(ctx, req, chill.ProcedureUserGetIndexers, map[string]any{})
	return nil, result, err
}

func (server *Server) getDownloadFolder(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
	result, err := server.call(ctx, req, chill.ProcedureUserGetDownloadFolder, map[string]any{})
	return nil, result, err
}

func (server *Server) browseFolder(ctx context.Context, req *mcp.CallToolRequest, input FolderInput) (*mcp.CallToolResult, any, error) {
	id, err := chill.NormalizeFolderID(strconv.FormatInt(input.ID, 10))
	if err != nil {
		return nil, nil, toolError(err)
	}
	result, err := server.call(ctx, req, chill.ProcedureUserGetFolder, map[string]any{"id": id})
	return nil, result, err
}

func (server *Server) getUserSettings(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
	result, err := server.call(ctx, req, chill.ProcedureUserGetUserSettings, map[string]any{})
	return nil, result, err
}

func (server *Server) listUserSettingFields(_ context.Context, _ *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, UserSettingFieldsResult, error) {
	fields := chill.UserSettingsFields()
	out := UserSettingFieldsResult{Fields: make([]UserSettingField, 0, len(fields))}
	for _, field := range fields {
		out.Fields = append(out.Fields, UserSettingField{Field: field.Name(), Aliases: field.Aliases, Type: field.ValueType, Description: field.Description})
	}
	return nil, out, nil
}

func (server *Server) updateUserSetting(ctx context.Context, req *mcp.CallToolRequest, input UpdateUserSettingInput) (*mcp.CallToolResult, any, error) {
	patch, err := chill.NormalizeUserSettingsPatch(input.Field, input.Value)
	if err != nil {
		return nil, nil, toolError(err)
	}
	if input.DryRun {
		return nil, map[string]any{"dry_run": true, "procedure": chill.ProcedureUserSaveUserSettings, "patch": patch}, nil
	}
	token, err := server.token(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	current, err := server.api.Call(ctx, rpc.CallRequest{Procedure: chill.ProcedureUserGetUserSettings, Body: map[string]any{}, AuthMode: rpc.AuthUser, AuthToken: token})
	if err != nil {
		return nil, nil, toolError(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(current.Body, &settings); err != nil {
		return nil, nil, fmt.Errorf("decode current user settings: %w", err)
	}
	if strings.TrimSpace(string(current.Body)) == "" || settings == nil {
		settings = map[string]any{}
	}
	request := map[string]any{"settings": chill.ApplyUserSettingsPatch(settings, patch)}
	result, err := server.call(ctx, req, chill.ProcedureUserSaveUserSettings, request)
	return nil, result, err
}
