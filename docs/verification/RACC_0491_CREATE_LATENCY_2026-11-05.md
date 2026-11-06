# Issue #88 creation-latency investigation: sanitized public summary

**Date:** 2026-11-05, owner timezone. **State: PARTIAL; original requests remain UNKNOWN.**

A separate Finder Agent performed read-only inspection of the preserved original independent Verifier failure scene. Original native identity still matched; no create was replayed, no claim unlocked and no target started, stopped or deleted. Guest package history and a narrow Provider journal selection showed provisioning continuing after the original harness deadline. This establishes continuing side effects, not an acknowledged original response.

The already confirmed harness timeout clamp was fixed before this investigation. A different newly owned child had previously completed four acknowledged lifecycle steps; that historical PARTIAL result does not resolve either original UNKNOWN request or accept the changed candidate.

## Pinned public-source findings

The source investigation resolved upstream tag v0.128.0 to commit `467e8f0c0bf2a524c36ae4e1da1e23e1a1f7a08f` and compared the installed MCP executable digest against its official release asset. A running daemon observation was kept separate from CLI and MCP observations; BoxWeave's minimum baseline was not used as runtime proof. The MCP executable exposes no version flag, so its authenticated runtime version handshake remains unverified.

- The [MCP HTTP client](https://github.com/FootprintAI/Containarium/blob/467e8f0c0bf2a524c36ae4e1da1e23e1a1f7a08f/internal/mcp/client.go#L80) has a separate 120-second deadline. Its create request synchronously POSTs the container route.
- The [tool request construction](https://github.com/FootprintAI/Containarium/blob/467e8f0c0bf2a524c36ae4e1da1e23e1a1f7a08f/internal/mcp/tools.go#L2197) does not request async behavior; the server's synchronous path waits through provisioning despite async success guidance.
- BoxWeave's [HTTP relay](../../bwmini/remote_mcp.py) independently bounds the Gateway subprocess to 45 seconds. This is a separate source-only transport limitation; no authenticated HTTP create was executed in this investigation.

Increasing the developer harness budget cannot remove these other deadlines. However, the original Provider response, cancellation propagation and eventual Gateway completion were not captured. SSH collector errors are correlated log events, not a confirmed latency root cause. Original symptom ownership remains unresolved.

## Evidence, safety and next check

Detailed environment identifiers, stage timestamps, original evidence paths and hashes are retained in the originating private evidence and a local private developer report. They are intentionally omitted from this public document. An attempted broad metadata read was rejected by automatic approval review because the file might contain credentials; it was not retried or needed. A proposed detailed GitHub comment was likewise rejected; only a sanitized status summary was posted.

[Issue #88](https://github.com/vanishs/boxweave/issues/88) remains OPEN. On a new uniquely owned disposable target, capture the original response and independent native state with stage timing, characterize each deadline separately, preserve any UNKNOWN intent, and refuse replay. Do not change either original incarnation to obtain a clean result. See the [release-gate checkpoint](RACC_0491_RELEASE_GATE_2026-11-05.md) for remaining acceptance blockers.
