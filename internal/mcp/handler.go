package mcp

import (
	"context"
	"encoding/json"
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

	o := Test123Output{
		Output: "Test123 received input: " + args.Input,
	}

	w.SetStructured(o)

	// ChatGPT's MCP client doesn't seem to like it if structured content is not accompanied by text content.
	bytes, err := json.Marshal(o)
	if err != nil {
		w.SetError(true)
		w.AppendText("error marshalling output: " + err.Error())
	}
	w.AppendText(string(bytes))

	return nil
}

func NewMCPServerCapabilities() mcpservice.ServerCapabilities {
	tools := mcpservice.NewToolsContainer(
		mcpservice.NewToolWithOutput("test_123", test123,
			mcpservice.WithToolDescription("Use this tool when the user asks you to test the ChatGPT App."),
			mcpservice.WithToolMeta(map[string]any{
				"openai/outputTemplate":          "ui://widget/form.v1.html",
				"openai/toolInvocation/invoking": "Displaying the tester tool.",
				"openai/toolInvocation/invoked":  "Displayed the tester tool.",
			}),
		),
	)

	resources := mcpservice.NewResourcesContainer()
	resources.AddResource(mcpservice.TextResource("ui://widget/form.v1.html", `
<div>
	<style type="text/css">
		html, body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			padding: 0;
			margin: 0;
			display: flex;
			flex-direction: column;
		}
		form {
			background-color: #f5f5f5;
			display: flex;
			flex-direction: column;

			fieldset {
				display: flex;
				flex-direction: column;
				gap: 0.5em;
				padding: 0.5em 1em;

				label {
					display: flex;
					flex-direction: row;
					gap: 0.5em;

					input {
						flex: 1 0 auto;
					}
				}
				
				button {
					display: block;
				}
			}
		}
	</style>
	<form>
		<fieldset disabled>
			<label>
				<span>Input:</span>
				<input type="text" id="input" placeholder="Enter input" style="width: 300px;">
			</label>
			<button type="submit" onclick="invokeTool()">Invoke Tool</button>
		</fieldset>
	</form>
	<script type="module">
		window.addEventListener("openai:set_globals", e => {
			const toolOutput = window.openai?.toolOutput ?? { output: "" };
	
			document.getElementById("input").value = toolOutput.output;
		});
	</script>
</div>
	`, mcpservice.WithName("Tester Tool UI"), mcpservice.WithMimeType("text/html+skybridge")))

	// Use string concatenation to safely include fenced code block without confusing the Go parser.
	detailedInstructions := `<TODO>`

	return mcpservice.NewServer(
		mcpservice.WithServerInfo(
			mcpservice.StaticServerInfo("Example ChatGPT App", "0.0.1", mcpservice.WithServerInfoTitle("Example ChatGPT App")),
		),
		mcpservice.WithProtocolVersion(mcpservice.StaticProtocolVersion("2025-06-18")),
		mcpservice.WithInstructions(mcpservice.StaticInstructions(detailedInstructions)),
		mcpservice.WithToolsCapability(tools),
		mcpservice.WithResourcesCapability(resources),
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
