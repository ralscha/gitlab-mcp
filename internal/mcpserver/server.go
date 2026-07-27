package mcpserver

import (
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"gitlab-mcp/internal/config"
	"gitlab-mcp/internal/gitlab"
)

var Version string

func ServerVersion() string {
	if Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func NewServer(cfg *config.Config, client *gitlab.Client) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "gitlab-mcp", Version: ServerVersion()}, nil)
	registerReadTools(server, client)
	if cfg.IsReadWrite() {
		registerWriteTools(server, client)
	}
	return server
}
