package server

import (
	"context"
	"strings"

	"github.com/chill-institute/chill-cli/v2/pkg/chill"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	boolTrue  = true
	boolFalse = false
)

func readOnly(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true, DestructiveHint: &boolFalse, OpenWorldHint: &boolTrue}
}

// SearchReleasesInput is the search_releases argument set.
type SearchReleasesInput struct {
	Query     string `json:"query" jsonschema:"Search text, for example a title and year"`
	IndexerID string `json:"indexer_id,omitempty" jsonschema:"Optional indexer id from the user's settings; omit to search every enabled indexer"`
}

// ListTVShowsInput is the list_tv_shows argument set.
type ListTVShowsInput struct {
	Source string `json:"source,omitempty" jsonschema:"Optional provider override: all-providers, netflix, hbo-max, apple-tv-plus, prime-video, disney-plus, hulu, paramount-plus, amc-plus, or peacock"`
}

// GetTransferInput is the get_transfer argument set.
type GetTransferInput struct {
	ID int64 `json:"id" jsonschema:"Transfer id returned by add_transfer"`
}

// AddTransferInput is the add_transfer argument set.
type AddTransferInput struct {
	URL         string `json:"url" jsonschema:"Magnet link or absolute http(s) URL of the release to download"`
	MovieSource string `json:"movie_source,omitempty" jsonschema:"Optional movie catalog origin: imdb-moviemeter, imdb-top-250, yts, rotten-tomatoes, or trakt"`
	TVSource    string `json:"tv_source,omitempty" jsonschema:"Optional TV catalog origin: all-providers, netflix, hbo-max, apple-tv-plus, prime-video, disney-plus, hulu, paramount-plus, amc-plus, or peacock"`
	DryRun      bool   `json:"dry_run,omitempty" jsonschema:"Validate and return the exact request without sending it to put.io"`
}

// EmptyInput is used by tools without arguments.
type EmptyInput struct{}

// DryRunResult is returned by add_transfer when dry_run is set.
type DryRunResult struct {
	DryRun    bool           `json:"dry_run"`
	Procedure string         `json:"procedure"`
	Request   map[string]any `json:"request"`
}

func (server *Server) registerTools() {
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "search_releases",
		Description: "Search release indexers with the user's saved filters. Returns release rows with size, seeders, and the URL to pass to add_transfer.",
		Annotations: readOnly("Search releases"),
	}, server.searchReleases)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "list_movies",
		Description: "List the movie catalog for the user's configured movie source and sort.",
		Annotations: readOnly("List movies"),
	}, server.listMovies)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "list_tv_shows",
		Description: "List the TV show catalog for the user's configured provider, or an explicit provider source.",
		Annotations: readOnly("List TV shows"),
	}, server.listTVShows)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "get_transfer",
		Description: "Show one put.io transfer by id, including status, progress, and file id once finished.",
		Annotations: readOnly("Get transfer"),
	}, server.getTransfer)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "whoami",
		Description: "Show the authenticated chill.institute account profile.",
		Annotations: readOnly("Who am I"),
	}, server.whoami)
	mcp.AddTool(server.mcp, &mcp.Tool{
		Name:        "add_transfer",
		Description: "Send a release to the user's put.io account through chill.institute. Use dry_run=true to preview the request without downloading.",
		Annotations: &mcp.ToolAnnotations{Title: "Add transfer", ReadOnlyHint: false, IdempotentHint: false, DestructiveHint: &boolTrue, OpenWorldHint: &boolTrue},
	}, server.addTransfer)
}

func (server *Server) searchReleases(ctx context.Context, req *mcp.CallToolRequest, input SearchReleasesInput) (*mcp.CallToolResult, any, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return nil, nil, toolError(&chill.ValidationError{Code: "missing_query", Message: "query is required"})
	}
	body := map[string]any{"query": query}
	if strings.TrimSpace(input.IndexerID) != "" {
		indexerID, err := chill.NormalizeIndexerID(input.IndexerID)
		if err != nil {
			return nil, nil, toolError(err)
		}
		body["indexer_id"] = indexerID
	}
	result, err := server.call(ctx, req, chill.ProcedureUserSearch, body)
	return nil, result, err
}

func (server *Server) listMovies(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
	result, err := server.call(ctx, req, chill.ProcedureUserGetMovies, map[string]any{})
	return nil, result, err
}

func (server *Server) listTVShows(ctx context.Context, req *mcp.CallToolRequest, input ListTVShowsInput) (*mcp.CallToolResult, any, error) {
	source, err := chill.NormalizeTVShowsSource(input.Source, true)
	if err != nil {
		return nil, nil, toolError(err)
	}
	body := map[string]any{}
	if source != "" {
		body["source"] = source
	}
	result, err := server.call(ctx, req, chill.ProcedureUserGetTVShows, body)
	return nil, result, err
}

func (server *Server) getTransfer(ctx context.Context, req *mcp.CallToolRequest, input GetTransferInput) (*mcp.CallToolResult, any, error) {
	if input.ID <= 0 {
		return nil, nil, toolError(&chill.ValidationError{Code: "invalid_transfer_id", Message: "id must be a positive integer"})
	}
	result, err := server.call(ctx, req, chill.ProcedureUserGetTransfer, map[string]any{"id": input.ID})
	return nil, result, err
}

func (server *Server) whoami(ctx context.Context, req *mcp.CallToolRequest, _ EmptyInput) (*mcp.CallToolResult, any, error) {
	result, err := server.call(ctx, req, chill.ProcedureUserGetUserProfile, map[string]any{})
	return nil, result, err
}

func (server *Server) addTransfer(ctx context.Context, req *mcp.CallToolRequest, input AddTransferInput) (*mcp.CallToolResult, any, error) {
	body, err := buildAddTransferRequest(input)
	if err != nil {
		return nil, nil, toolError(err)
	}
	if input.DryRun {
		return nil, DryRunResult{DryRun: true, Procedure: chill.ProcedureUserAddTransfer, Request: body}, nil
	}
	result, err := server.call(ctx, req, chill.ProcedureUserAddTransfer, body)
	return nil, result, err
}

func buildAddTransferRequest(input AddTransferInput) (map[string]any, error) {
	movieSource := strings.TrimSpace(input.MovieSource)
	tvSource := strings.TrimSpace(input.TVSource)
	if movieSource != "" && tvSource != "" {
		return nil, &chill.ValidationError{Code: "ambiguous_transfer_source", Message: "use either movie_source or tv_source, not both"}
	}
	transferURL, err := chill.NormalizeTransferURL(input.URL)
	if err != nil {
		return nil, err
	}
	body := map[string]any{"url": transferURL}
	if movieSource != "" {
		source, err := chill.NormalizeMovieSource(movieSource)
		if err != nil {
			return nil, err
		}
		body["catalogOrigin"] = map[string]any{"moviesSource": source}
	}
	if tvSource != "" {
		source, err := chill.NormalizeTVShowsSource(tvSource, false)
		if err != nil {
			return nil, err
		}
		body["catalogOrigin"] = map[string]any{"tvShowsSource": source}
	}
	return body, nil
}
