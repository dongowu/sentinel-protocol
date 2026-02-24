/**
 * Sentinel Guard — agent:bootstrap hook
 *
 * Injects mandatory security rules into the agent's system prompt so
 * the agent always calls sentinel_gate before risky actions.
 */

import type { HookHandler } from "openclaw/hooks";

const SENTINEL_RULES = `
## Sentinel Guard — Mandatory Security Policy

You have access to the \`sentinel_gate\` tool. You MUST call it BEFORE performing
any of the following action categories:

| Category | When to call |
|----------|-------------|
| EXEC | Before running any shell command |
| FS | Before writing, deleting, or modifying files outside your workspace |
| BROWSER | Before browser automation or web scraping |
| WALLET | Before any blockchain transaction or wallet operation |
| NETWORK | Before making external API calls or network requests |
| CODE_EDITING | Before code editing operations with side effects |

### How to use sentinel_gate

1. Call \`sentinel_gate\` with \`action\` (category) and \`prompt\` (what you intend to do).
2. Read the \`decision\` field in the response:
   - **ALLOW**: You may proceed. A one-time token was issued.
   - **REQUIRE_APPROVAL**: Do NOT proceed yet. Tell the user a human approval is needed and provide the challenge_id.
   - **BLOCK**: Do NOT proceed. The action was denied. Explain why to the user.
   - **TRIGGER_KILL_SWITCH**: The system is in emergency mode. ALL actions are blocked.
3. If BLOCKED, do NOT attempt the action. Suggest a safer alternative.
4. You can check the system status anytime with \`sentinel_status\`.

### Examples

Before running \`rm -rf /tmp/data\`:
→ Call \`sentinel_gate(action="EXEC", prompt="rm -rf /tmp/data")\`

Before sending 100 USDC:
→ Call \`sentinel_gate(action="WALLET", prompt="transfer 100 USDC to 0xabc...")\`

This policy is NON-NEGOTIABLE. Skipping sentinel_gate for risky actions is a security violation.
`.trim();

const handler: HookHandler = async (event) => {
  // Inject Sentinel rules into the bootstrap context
  if (event.type === "agent:bootstrap" && event.context?.files) {
    // Add Sentinel rules as a virtual file in the bootstrap
    event.context.files.push({
      path: "SENTINEL_GUARD.md",
      content: SENTINEL_RULES,
    });
  }
  return event;
};

export default handler;
