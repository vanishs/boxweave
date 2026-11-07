# Product definition

BoxWeave is a minimal, agent-first controller for Linux containers managed through Incus on one host at each layer. A managed container is a **Box**. The program is **BoxWeave**. Those are product-facing names; implementation names should follow standard Incus and Caddy terminology.

A BoxWeave instance regards its own execution environment as its host. When installed inside an appropriately provisioned Box, it becomes a fully independent management layer for its own child Boxes. The nesting depth is not restricted by BoxWeave; support depends on Incus, kernel, namespaces, networking, and resources.

Operators use a secure Streamable HTTP MCP management endpoint. A limited local CLI supports installation, lifecycle maintenance, initial super-administrator credential setup and emergency recovery. Application traffic published through Caddy or port forwarding is a separate data plane, **not** an additional management API.

The project aims for correctness, bounded authority, repeatability and clear failure states, rather than feature parity with the former Containarium wrapper.
