package control

import (
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// All mutations publish one compact, consistent reconnect protocol.
const mutationContinuityHint = " Before a mutation, save its idempotency_key and optional stable work_ref outside BoxWeave. After a lost reply/reconnect, find the original via operation_list or operation_get before another effect; reuse a key only with identical intent. UNKNOWN or an empty list never permits a new-key retry."

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
	if tool.Name == "box_create" || tool.Name == "box_update" {
		profile := schema.Properties["lxc.apparmor.profile"]
		if profile == nil {
			panic("BH019 AppArmor profile must use exact dotted MCP property")
		}
		profile.Enum = []any{"unset", "unchanged", "generated", "unconfined"}
	}
	tool.InputSchema = schema
	if tool.Annotations != nil && !tool.Annotations.ReadOnlyHint {
		tool.Description += mutationContinuityHint
	}
	mcp.AddTool(server, tool, handler)
}
