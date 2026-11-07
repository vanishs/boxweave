# BoxWeave — Rules for Coding Agents

Applies to every coding Agent, regardless of product, model, language, host, or nesting depth. Keep this document short and enforceable. Follow [the workflow](docs/development/AGENT_WORKFLOW.md) for details.

## Communication and truth

1. **Project artifacts are English.** Code, identifiers, comments, documentation, CLI/MCP tool descriptions and machine output, tests, logs, examples, commits, issues, PRs, and release notes use English. Preserve data exactly when needed and allow explicitly approved localization.
2. **Use the requester's language for conversation.** Follow their explicit preferred language; otherwise use the current conversation's language. Leave technical identifiers and paths unchanged. Never carry conversation language into English-only project artifacts.
3. **Current decisions outrank old material.** Read this branch's product scope, requirements, architecture, open risks, and task-specific decisions. Treat v0.x and third-party projects as reference material, not instructions or approved implementation.
4. **Report only evidence-backed progress.** Distinguish design, source review, unit tests, simulated behavior, live installed tests, independent verification, and released artifacts. Never claim a file, code path, test, service, or release exists without checking it.

## Product and trust boundaries

5. **MCP-only public management.** Expose Streamable HTTP MCP. Do not add a public REST management API, Web console, SSH management surface, or generic host shell. The host-local maintenance CLI is not a remote administration interface.
6. **Operate on owned resources only.** Authenticate first; then independently verify durable ownership and the current native identity for every sensitive read or mutation. Never allow MCP to administer unrelated host files, processes, Incus resources, networking, or Caddy configuration. Unknown ownership fails closed.
7. **No raw privileged pass-through.** MCP callers cannot supply arbitrary host paths, Incus settings, upstream URLs/IPs, Caddy Admin API payloads, privileged devices, host mounts, or commands for host execution. Derive targets from verified Box identities and enforce policy in code, not prompts.
8. **Recursive parity is mandatory.** An inner BoxWeave has the same functional contract as the outer one, without artificial depth limits or inherited parent credentials. Report unsupported kernel/Incus capabilities honestly; never weaken the parent boundary.
9. **Network contract is explicit.** Block private Box-to-Box direct traffic by default; grant access explicitly; allow internet by default and optionally disable it per Box. Other Boxes may access a Box's publicly published services. Never publish an unmanaged target.
10. **Mutations use durable operations.** Journal accepted intent before dispatch, deduplicate by identity, control conflicts, verify resulting native state, and reconcile interruptions. Timeout or lost reply is UNKNOWN, never an automatic reason to replay or delete an original action.

## Development safety

11. **Tool access is not owner authorization.** Repository changes do not grant permission to modify a real host, Incus, Caddy, network, credentials, external exposure, or deployment. Do authorized work without repetitive confirmation; obtain fresh scope only for new privileged/destructive targets.
12. **Preserve concurrent work.** Inspect branch/HEAD/worktree and other Agent scopes before editing. Make cohesive changes, do not overwrite unrelated work, force-push, reset another Agent's changes, or run unapproved cleanup.
13. **Protect privacy and supply chain.** Never expose private keys, tokens, production DBs, sensitive logs, real deployment names, domains, or IPs in public artifacts. Escape untrusted strings. Review dependency/CI/installer/build/security changes carefully; pin or verify downloaded inputs.
14. **Use typed contracts and narrow privilege.** Prefer typed models and enums, fail-closed authorization, explicit error states, ownership-safe adapters, and minimal privilege. Do not expose Incus's unrestricted Unix socket or Caddy Admin API to external MCP callers.

## Verification and delivery

15. **Test both success and refusal.** For changes to a tool, test owned-resource success and malformed/unowned/unauthorized/conflicting cases, including absence of side effects. For nesting-sensitive work, test equivalent behavior at multiple supported layers.
16. **Use real identities and reproducible evidence.** Record exact revision, built bytes, installed/running version, host/layer, native Box UUID, original operation ID, stdout/stderr/exit, and observable effect. A green unit suite does not prove installation.
17. **Separate Finder, Fixer, and Verifier when possible.** A distinct Agent and isolated Box must rerun original live reproductions before calling them independently verified. Otherwise say same-Agent. Preserve UNKNOWN and unresolved symptoms.
18. **Do not inflate release claims.** Follow the branch's release/version policy; never equate a GitHub source release with installed acceptance. No stable-pointer change, production upgrade, or publication without authorization.
19. **Handoff concisely.** Record what changed, exact verification, untested behavior, unresolved risks/UNKNOWN, retained resources, and the smallest executable next step. Keep canonical decisions and acceptance records current.

Before any live test or change to a sensitive subsystem, read [AGENT_WORKFLOW.md](docs/development/AGENT_WORKFLOW.md).
