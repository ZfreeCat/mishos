package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"mishosapi/config"
	"mishosapi/db"
	mcpServer "mishosapi/mcp"
	"mishosapi/migrations"
	"mishosapi/server"
	"mishosapi/tasks"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	config.LoadEnv()
	config.InitTimezone()

	db.Init()
	migrations.RunMigrationIfEnv()
}

func main() {
	taskRunner := tasks.TaskRunner{}
	taskRunner.InitializeTasks()

	// MCP server for AI assistants (SSE on :8280/mcp by default)
	go func() {
		if err := mcpServer.Start(); err != nil {
			log.Error().Err(err).Msg("mcp server stopped")
		}
	}()

	log.Info().Msg("server starting...")
	server.Start()
}
