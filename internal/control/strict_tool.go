package control

import (
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// addStrictTool derives the public schema from the same typed Go input used by
// the official MCP SDK. Its default object schema otherwise permits extra
// keys; silently ignoring root_pool, project or other unsupported client
// options would execute a different native intent than the client requested.
// Reject unknown top-level keys without changing typed nested map fields.
func addStrictTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	if tool.InputSchema != nil {
		panic(fmt.Sprintf("strict tool %q must derive its input schema", tool.Name))
	}
	schema, err := jsonschema.ForType(reflect.TypeFor[In](), nil)
	if err != nil || schema == nil || schema.Type != "object" {
		panic(fmt.Sprintf("strict schema for %q: %v", tool.Name, err))
	}
	// 'not: {}' matches no JSON value and is the draft-neutral false schema.
	// The SDK validates it BEFORE decoding or entering any native handler.
	schema.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}}
	tool.InputSchema = schema
	mcp.AddTool(server, tool, handler)
}
