# Web navigation and behavior comparison (R-06)

Source mapping of old `server/webui/*` / `web/src/*` and new `web/src/main.tsx`, `web.py` and components. **U**: no new browser was executed for this document; previous historical reports remain tied to their old SHAs. The [routine fixture runner](../../scripts/validate.py) contains seven distinct browser flows.

| Legacy screen/workflow | New UI entry | Required behavior / rejection | Existing fixture check |
| --- | --- | --- | --- |
| Overview/Boxes | **Projects**, **Environments** | Bind/edit/unbind/rebind preserve receipt and Graph, exact target, stale revision | `browser_check.py`, `binding_browser_check.py` |
| Login/admin | Operator sign in, local CLI init/reset | Session expiration, CSRF/Origin/401, reset revokes sessions, no credential output | `browser_check.py` + unit auth |
| Box operations | **Environments** | Create/start/stop/delete single action with plan and durable receipt; no unknown auto retry | `lifecycle_browser_check.py` |
| Monitoring/history | Projects receipt/history, environment observed state | HTTP != MCP success; explicit unknown; refresh never re-executes | `history_browser_check.py` |
| Tenants/Network | **Network & ports** | Native member + capability/Guard/revision, TCP/UDP audience and permission errors | `access_browser_check.py`; real packet U |
| Prompt Graph | **Prompt Graph** | Dirty draft guard, revision conflict, safe copy/export, no execution | `graph_browser_check.py` |
| Model panel | **Model setup** | ZCode native settings handoff, public fields, inference not presumed | `model_browser_check.py` |
| Client generator | **Client setup** | Four public descriptors and hash-gated local script; no server enrollment | `binding_browser_check.py` + client test |
| Proxy domain editor | No generic selected v2 equivalent | Approved exclusion, not hidden or implicitly deployed | [SCOPE](../product/SCOPE.md) |
| Mobile | Responsive CSS | Test narrow viewport, accessible control/error message, console failures | Manual/Chromium viewport evidence needed |

The actual source navbar exposes Projects, Prompt Graph, Environments, Model setup, Network & ports and Client setup. When Graph edits are dirty, navigation is gated. Unbind does not cancel already dispatched work. This source reachability does not prove full historic screenshot/field parity or a real remote action.

**Next Agent**: Run `.venv/bin/python scripts/validate.py --browser` from the approved non-root Linux environment; record script exit, console output, screenshots and fixture marker separately for each of seven flows. Inspect 375px/mobile, tab switching, login/reset, unavailable Provider state, failure text and refresh. Compare real legacy controls only within selected scope; expand assertions when a concrete user goal lacks coverage.
