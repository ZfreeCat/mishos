package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"

	"mishosapi/services"
	"mishosapi/tvproviders"
)

// Start runs the MCP (Model Context Protocol) SSE server in a background
// goroutine. It lets AI assistants query and manage the user's Mishos library.
//   - endpoint:  http://<host>:8280/mcp
//   - auth:      Authorization: Bearer <MISHOS_MCP_TOKEN> (empty token = no auth)
//   - operating user: MISHOS_MCP_USER_ID (default: 1)
func Start() error {
	port := strings.TrimSpace(os.Getenv("MISHOS_MCP_PORT"))
	if port == "" {
		port = "8280"
	}

	token := strings.TrimSpace(os.Getenv("MISHOS_MCP_TOKEN"))

	userId := uint(1)
	if v := strings.TrimSpace(os.Getenv("MISHOS_MCP_USER_ID")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			userId = uint(n)
		}
	}

	s := server.NewMCPServer("mishos", "1.0.0", server.WithToolCapabilities(true))

	s.AddTool(
		mcp.NewTool(
			"search_shows",
			mcp.WithDescription("Search movies and TV shows on TMDB"),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, _ := req.Params.Arguments["query"].(string)

			client := tvproviders.TmdbClient{}
			results, err := client.Search(query, "multi")
			if err != nil {
				return toolResult("error: " + err.Error()), nil
			}

			rows := []map[string]any{}
			for _, item := range results {
				rows = append(rows, map[string]any{
					"tmdbId":    item.ID,
					"mediaType": item.MediaType,
					"name":      item.Name,
					"overview":  item.Overview,
					"date":      firstNonEmpty(item.FirstAirDate, item.ReleaseDate),
				})
			}
			out, _ := json.MarshalIndent(rows, "", "  ")
			return toolResult(string(out)), nil
		},
	)

	s.AddTool(
		mcp.NewTool(
			"get_my_ratings",
			mcp.WithDescription("List the user's TMDB ratings (watched items)"),
			mcp.WithString("mediaType", mcp.Description("all | tv | movie (default all)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			mediaType, _ := req.Params.Arguments["mediaType"].(string)
			if mediaType == "" {
				mediaType = "all"
			}

			tmdbService := services.TmdbService{}
			ratings, err := tmdbService.GetMyRatings(userId, mediaType)
			if err != nil {
				return toolResult("error: " + err.Error()), nil
			}

			out, _ := json.MarshalIndent(ratings, "", "  ")
			return toolResult(string(out)), nil
		},
	)

	s.AddTool(
		mcp.NewTool(
			"get_upcoming",
			mcp.WithDescription("List upcoming episodes of followed shows and movie premieres"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			watchlistService := services.WatchListService{}
			upcoming, err := watchlistService.GetUpcoming(userId)
			if err != nil {
				return toolResult("error: " + err.Error()), nil
			}

			out, _ := json.MarshalIndent(upcoming, "", "  ")
			return toolResult(string(out)), nil
		},
	)

	s.AddTool(
		mcp.NewTool(
			"get_watchlist",
			mcp.WithDescription("List the user's watchlist (unwatched episodes)"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			watchlistService := services.WatchListService{}
			unwatched, err := watchlistService.GetUnwatched(userId)
			if err != nil {
				return toolResult("error: " + err.Error()), nil
			}

			out, _ := json.MarshalIndent(unwatched, "", "  ")
			return toolResult(string(out)), nil
		},
	)

	s.AddTool(
		mcp.NewTool(
			"import_show",
			mcp.WithDescription("Import a movie/TV show from TMDB into the local library"),
			mcp.WithNumber("tmdbId", mcp.Required(), mcp.Description("TMDB id")),
			mcp.WithString("mediaType", mcp.Required(), mcp.Description("tv or movie")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tmdbId, _ := req.Params.Arguments["tmdbId"].(float64)
			mediaType, _ := req.Params.Arguments["mediaType"].(string)

			tmdbService := services.TmdbService{}
			showId, err := tmdbService.ImportShow(int(tmdbId), mediaType)
			if err != nil {
				return toolResult("error: " + err.Error()), nil
			}

			return toolResult("imported, local show id = " + strconv.FormatUint(uint64(showId), 10)), nil
		},
	)

	s.AddTool(
		mcp.NewTool(
			"follow_show",
			mcp.WithDescription("Import a movie/TV show from TMDB and follow it"),
			mcp.WithNumber("tmdbId", mcp.Required(), mcp.Description("TMDB id")),
			mcp.WithString("mediaType", mcp.Required(), mcp.Description("tv or movie")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tmdbId, _ := req.Params.Arguments["tmdbId"].(float64)
			mediaType, _ := req.Params.Arguments["mediaType"].(string)

			tmdbService := services.TmdbService{}
			added, followed, err := tmdbService.EnsureShowFollowed(userId, int(tmdbId), mediaType)
			if err != nil {
				return toolResult("error: " + err.Error()), nil
			}

			return toolResult("followed (added=" + strconv.Itoa(added) + ", followed=" + strconv.Itoa(followed) + ")"), nil
		},
	)

	sse := server.NewSSEServer(s,
		server.WithSSEEndpoint("/mcp"),
		server.WithMessageEndpoint("/mcp/message"),
		server.WithUseFullURLForMessageEndpoint(false),
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		sse.ServeHTTP(w, r)
	})

	addr := ":" + port
	log.Info().Str("addr", addr).Str("endpoint", "/mcp").Msg("mcp server listening")
	return http.ListenAndServe(addr, handler)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// toolResult wraps a plain-text reply into an MCP tool result (mcp-go v0.20 API).
func toolResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: text},
		},
	}
}

