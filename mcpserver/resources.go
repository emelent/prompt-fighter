package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"promptfighter/docs"
)

const (
	guideURI = "promptfighter://docs/player-guide"
	rulesURI = "promptfighter://docs/rules"
	botURI   = "promptfighter://bots/"
)

func staticResource(uri, mime, text string) mcp.ResourceHandler {
	return func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if req.Params.URI != uri {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: mime, Text: text}}}, nil
	}
}

func (h *handler) registerResources(s *mcp.Server) {
	s.AddResource(&mcp.Resource{
		URI: guideURI, Name: "player-guide", Title: "Prompt Fighter player guide", MIMEType: "text/markdown",
		Description: "How to write .pf fight scripts: language, frame costs, combat rules, limits and strategy.",
	}, staticResource(guideURI, "text/markdown", docs.PlayerGuide))
	s.AddResource(&mcp.Resource{
		URI: rulesURI, Name: "rules", Title: "Prompt Fighter engine rules", MIMEType: "text/markdown",
		Description: "The engine rules reference: tick order, frame data, resolution and the event log format.",
	}, staticResource(rulesURI, "text/markdown", docs.Rules))
	for _, b := range h.bots {
		uri := botURI + b.Name
		s.AddResource(&mcp.Resource{
			URI: uri, Name: "bot-" + b.Name, Title: "Sample bot: " + b.Name, MIMEType: "text/plain", Description: b.Summary,
		}, staticResource(uri, "text/plain", b.Source))
	}
}
