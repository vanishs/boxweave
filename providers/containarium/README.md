# Containarium provider

**Status: initial implemented provider.**

The current Ubuntu installer:

- Reuses an existing Containarium service or invokes `https://containarium.dev/install.sh` on a compatible Ubuntu host.
- Downloads a Containarium `mcp-server` release and verifies its upstream `SHA256SUMS.txt`.
- Runs the provider MCP through `/usr/local/bin/boxweave-mcp` and the loopback Containarium REST API.
- Rotates a provider JWT and reconciles project-container SSH host keys via the trusted Incus API.

Containarium retains its **upstream** service name, container metadata (`user.containarium.*`), API settings, internal container naming rules, CLI and MCP tool names. BoxWeave acts as the adapter and installer; it is not a replacement for Containarium itself.

Support for another provider should be added as an independent adapter rather than replacing Containarium-specific constants globally.
