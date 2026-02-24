---
name: sentinel-bootstrap
description: Injects Sentinel Guard security rules into the agent's system prompt during bootstrap, ensuring the agent evaluates all risky actions through the sentinel_gate tool before execution.
metadata:
  openclaw:
    emoji: "\u{1F6E1}"
    events:
      - agent:bootstrap
---

# Sentinel Guard Bootstrap Hook

Injects security policy rules into the agent's system prompt so the agent
calls `sentinel_gate` before any risky action.
