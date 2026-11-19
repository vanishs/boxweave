package control

import (
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverContinuityInstructions appears in the official MCP initialize result.
// Keep the first paragraph independently useful if a client truncates hints.
const serverContinuityInstructions = "BoxWeave manages only registered Boxes in its configured Incus Project. " +
	"Mutations are durable asynchronous Operations; a receipt is not completion. " +
	"Use a distinct idempotency_key per logical mutation, and reuse the same key for exact retries. " +
	"After interruption, use operation_list and operation_get to check originals before any new effect. " +
	"UNKNOWN or missing records do not justify new-key retries. " +
	"Operation records track admitted actions and observed outcomes, not Agent task completion.\n" +
	"work_ref and agent_ref are untrusted correlation labels, not authorization or reliable Agent identities. " +
	"Separate authenticated Actors may inspect authorized same-Layer originals but cannot borrow another Actor's idempotency identity. " +
	"Use operation_output_get for guest Exec output and exit_code; Operation SUCCEEDED does not guarantee a zero guest exit. " +
	"Use box_get for observed native Box state; Running does not prove guest readiness. " +
	"box_activity_get offers bounded SQLite-only overview/issues/evidence after reconnect; cursor pages are live views, not immutable snapshots. " +
	"BoxWeave does not store Agent plans, conversation history, external Git checkpoints or unrecorded activity."

// Each mutating tool still includes a short reminder for clients that omit
// initialize instructions; the server enforces original-key/UNKNOWN safety.
const mutationContinuityHint = " One key per logical mutation; reuse only for exact retries. After an uncertain response, inspect the original Operation first."

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
	if tool.Name == "box_activity_get" {
		view := schema.Properties["view"]
		if view == nil {
			panic("Activity requires typed view field")
		}
		view.Enum = []any{"overview", "issues", "evidence"}
	}
	tool.InputSchema = schema
	if tool.Annotations != nil && !tool.Annotations.ReadOnlyHint {
		tool.Description += mutationContinuityHint
	}
	mcp.AddTool(server, tool, handler)
}
