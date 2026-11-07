# Product definition

BoxWeave is a minimal, agent-first controller for Linux containers managed through Incus on one host at each layer. A managed container is a **Box**. The program is **BoxWeave**. Those are product-facing names; implementation names should follow standard Incus and Caddy terminology.

A BoxWeave instance regards its own execution environment as its host. When installed inside an appropriately provisioned Box, it becomes an independent controller that manages **only Boxes it directly owns**. The outer controller treats the containing Box like any other Box, not as a nested management tree. There is no BoxWeave-imposed depth limit; actual support depends on Incus, kernel, namespaces, networking, and resources.

Full nested compatibility primarily enables safe development, installation and testing of new BoxWeave versions **inside existing Boxes**. It also keeps the option of future environment subleasing open, without implementing a tenant/subleasing platform or any cross-layer orchestration in v1.

Operators use an authenticated Streamable HTTP MCP management endpoint bound to the local loopback interface by default; any public HTTPS exposure uses a separately operator-managed proxy or tunnel. A limited local CLI supports installation, lifecycle maintenance, initial super-administrator credential setup and emergency recovery. Application traffic published through Caddy or port forwarding is a separate data plane, **not** an additional management API.

Ordinary Linux Boxes need no installed BoxWeave guest client, resident management Agent, or SSH service: the host Incus daemon provides native lifecycle and Exec. Guest Exec defaults to root *inside the owned Box*, not the host. The project aims for correctness, bounded authority, repeatability and clear failure states, rather than feature parity with the former Containarium wrapper.
