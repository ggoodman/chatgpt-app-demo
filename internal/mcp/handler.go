package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ggoodman/mcp-server-go/auth"
	"github.com/ggoodman/mcp-server-go/mcpservice"
	"github.com/ggoodman/mcp-server-go/sessions"
	"github.com/ggoodman/mcp-server-go/sessions/redishost"
	"github.com/ggoodman/mcp-server-go/streaminghttp"
)

type Test123Params struct {
	// Define input parameters here
	Input string `json:"input"`
}

type Test123Output struct {
	// Define output structure here
	Output string `json:"output"`
}

func test123(ctx context.Context, session sessions.Session, w mcpservice.ToolResponseWriterTyped[Test123Output], r *mcpservice.ToolRequest[Test123Params]) error {
	args := r.Args()

	w.SetStructured(Test123Output{
		Output: "Test123 received input: " + args.Input,
	})

	return nil
}

func NewMCPServerCapabilities() mcpservice.ServerCapabilities {
	tools := mcpservice.NewToolsContainer(
		mcpservice.NewToolWithOutput("test_123", test123,
			mcpservice.WithToolDescription("Use this tool when the user asks you to test the ChatGPT App."),
			mcpservice.WithToolMeta(map[string]any{
				// "openai/outputTemplate":          "ui://widget/tester.html",
				// "openai/toolInvocation/invoking": "Displaying the tester tool.",
				// "openai/toolInvocation/invoked":  "Displayed the tester tool.",
			}),
		),
	)

	// Use string concatenation to safely include fenced code block without confusing the Go parser.
	detailedInstructions := `<TODO>`

	return mcpservice.NewServer(
		mcpservice.WithServerInfo(
			mcpservice.StaticServerInfo("Example ChatGPT App", "0.0.1", mcpservice.WithServerInfoTitle("Example ChatGPT App")),
		),
		mcpservice.WithProtocolVersion(mcpservice.StaticProtocolVersion("2025-06-18")),
		mcpservice.WithInstructions(mcpservice.StaticInstructions(detailedInstructions)),
		mcpservice.WithToolsCapability(tools),
	)
}

func NewMCPHandler(ctx context.Context, log *slog.Logger, serverUrl string, authIssuerUrl string, redisUrl string, redisKeyPrefix string) (http.Handler, error) {
	redisHost, err := redishost.New(redisUrl, redishost.WithKeyPrefix(redisKeyPrefix), redishost.WithStreamMaxLen(20))
	if err != nil {
		return nil, fmt.Errorf("error instantiating redis host: %w", err)
	}

	auth, err := auth.NewFromDiscovery(ctx, authIssuerUrl, serverUrl)
	if err != nil {
		return nil, fmt.Errorf("error configuring auth: %w", err)
	}

	srv := NewMCPServerCapabilities()

	return streaminghttp.New(ctx, serverUrl, redisHost, srv, auth,
		streaminghttp.WithServerName("Example ChatGPT App"),
		streaminghttp.WithLogger(log),
		streaminghttp.WithVerboseRequestLogging(true),
	)
}
