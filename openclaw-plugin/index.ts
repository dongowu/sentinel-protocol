/**
 * Sentinel Guard — OpenClaw Plugin
 *
 * Registers a `sentinel_gate` agent tool that calls the Sentinel proxy
 * to evaluate actions before execution. Also injects safety rules via
 * an agent:bootstrap hook.
 */

import { Type } from "@sinclair/typebox";
import path from "path";
import { fileURLToPath } from "url";

const DEFAULT_SENTINEL_URL = "http://127.0.0.1:18080";
const FETCH_TIMEOUT_MS = 10_000; // 10s timeout for Sentinel proxy calls

interface SentinelGateResponse {
  decision: string;
  score: number;
  tags: string[];
  record_hash: string;
  proof_index: number;
  token?: { id: string; expires_at: string };
  challenge_id?: string;
}

async function callSentinelGate(
  sentinelUrl: string,
  action: string,
  prompt: string,
  agentId?: string
): Promise<SentinelGateResponse> {
  const body: Record<string, string> = { action, prompt };
  if (agentId) body.agent_id = agentId;

  const res = await fetch(`${sentinelUrl}/sentinel/gate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal: AbortSignal.timeout(FETCH_TIMEOUT_MS),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Sentinel gate returned ${res.status}: ${text}`);
  }

  return res.json() as Promise<SentinelGateResponse>;
}

async function callSentinelStatus(sentinelUrl: string): Promise<unknown> {
  const res = await fetch(`${sentinelUrl}/sentinel/status`, {
    signal: AbortSignal.timeout(FETCH_TIMEOUT_MS),
  });
  if (!res.ok) throw new Error(`Sentinel status returned ${res.status}`);
  return res.json();
}

async function callSentinelApprovalConfirm(
  sentinelUrl: string,
  challengeId: string,
  approved: boolean,
  decidedBy: string
): Promise<unknown> {
  const res = await fetch(`${sentinelUrl}/sentinel/approval/confirm`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      challenge_id: challengeId,
      approved,
      decided_by: decidedBy,
    }),
    signal: AbortSignal.timeout(FETCH_TIMEOUT_MS),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Sentinel approval confirm returned ${res.status}: ${text}`);
  }
  return res.json();
}

export default function register(api: any) {
  const cfg = api.config?.plugins?.entries?.["sentinel-guard"]?.config ?? {};
  const sentinelUrl = cfg.sentinelUrl || DEFAULT_SENTINEL_URL;

  // ── Tool 1: sentinel_gate ─────────────────────────────────────────
  // The agent calls this BEFORE executing any risky action.
  api.registerTool({
    name: "sentinel_gate",
    description: `Evaluate an action through the Sentinel security gate BEFORE executing it.
Call this tool whenever you are about to perform a potentially risky action such as:
- Running shell commands (action: "EXEC")
- File system operations (action: "FS")
- Browser automation (action: "BROWSER")
- Wallet / blockchain transactions (action: "WALLET")
- Network requests (action: "NETWORK")
- Code editing with side effects (action: "CODE_EDITING")

The gate returns a decision:
- ALLOW: safe to proceed (includes a one-time execution token)
- REQUIRE_APPROVAL: needs human approval first (includes a challenge_id)
- BLOCK: action is denied (do NOT proceed)
- TRIGGER_KILL_SWITCH: system is in emergency shutdown mode`,
    parameters: Type.Object({
      action: Type.String({
        description:
          'Action category: "EXEC", "FS", "BROWSER", "WALLET", "NETWORK", or "CODE_EDITING"',
      }),
      prompt: Type.String({
        description: "The specific action or command you intend to execute",
      }),
    }),
    async execute(_id: string, params: { action: string; prompt: string }) {
      try {
        const result = await callSentinelGate(
          sentinelUrl,
          params.action,
          params.prompt
        );

        let summary: string;
        switch (result.decision) {
          case "ALLOW":
            summary = `ALLOWED (risk score: ${result.score}). Token issued: ${result.token?.id ?? "none"}. You may proceed with the action.`;
            break;
          case "REQUIRE_APPROVAL":
            summary = `REQUIRES HUMAN APPROVAL (risk score: ${result.score}). Challenge ID: ${result.challenge_id}. Wait for approval before proceeding. Tags: ${result.tags?.join(", ")}`;
            break;
          case "BLOCK":
            summary = `BLOCKED (risk score: ${result.score}). Do NOT proceed with this action. Tags: ${result.tags?.join(", ")}`;
            break;
          case "TRIGGER_KILL_SWITCH":
            summary = `KILL SWITCH ACTIVE. ALL actions are blocked. Contact the operator.`;
            break;
          default:
            summary = `Unknown decision: ${result.decision}`;
        }

        return {
          content: [
            {
              type: "text",
              text: JSON.stringify(
                { summary, ...result },
                null,
                2
              ),
            },
          ],
        };
      } catch (err: any) {
        return {
          content: [
            {
              type: "text",
              text: `Sentinel gate error: ${err.message}. Defaulting to BLOCK — do NOT proceed.`,
            },
          ],
          isError: true,
        };
      }
    },
  });

  // ── Tool 2: sentinel_status ───────────────────────────────────────
  api.registerTool({
    name: "sentinel_status",
    description:
      "Check the current status of the Sentinel security system (kill switch state, proof chain, pending approvals).",
    parameters: Type.Object({}),
    async execute() {
      try {
        const status = await callSentinelStatus(sentinelUrl);
        return {
          content: [
            { type: "text", text: JSON.stringify(status, null, 2) },
          ],
        };
      } catch (err: any) {
        return {
          content: [
            {
              type: "text",
              text: `Sentinel status error: ${err.message}`,
            },
          ],
          isError: true,
        };
      }
    },
  });

  // ── Tool 3: sentinel_approval ─────────────────────────────────────
  api.registerTool({
    name: "sentinel_approval",
    description:
      "Approve or reject a pending Sentinel challenge. Use after sentinel_gate returns REQUIRE_APPROVAL and a human has made a decision.",
    parameters: Type.Object({
      challenge_id: Type.String({
        description: "The challenge ID from sentinel_gate response",
      }),
      approved: Type.Boolean({
        description: "true to approve, false to reject",
      }),
      decided_by: Type.String({
        description: 'Who made the decision (e.g. "human-operator")',
      }),
    }),
    async execute(
      _id: string,
      params: {
        challenge_id: string;
        approved: boolean;
        decided_by: string;
      }
    ) {
      try {
        const result = await callSentinelApprovalConfirm(
          sentinelUrl,
          params.challenge_id,
          params.approved,
          params.decided_by
        );
        return {
          content: [
            { type: "text", text: JSON.stringify(result, null, 2) },
          ],
        };
      } catch (err: any) {
        return {
          content: [
            {
              type: "text",
              text: `Sentinel approval error: ${err.message}`,
            },
          ],
          isError: true,
        };
      }
    },
  });

  // ── Hook: agent:bootstrap — inject Sentinel guard rules ───────────
  // This tells the agent about Sentinel before every session.
  if (api.registerPluginHooksFromDir) {
    try {
      const __dirname = path.dirname(fileURLToPath(import.meta.url));
      api.registerPluginHooksFromDir(api, path.join(__dirname, "hooks"));
    } catch {
      // Hook registration is optional; tools still work without it.
    }
  }

  // ── CLI: openclaw sentinel ────────────────────────────────────────
  if (api.registerCli) {
    api.registerCli(
      ({ program }: any) => {
        const cmd = program
          .command("sentinel")
          .description("Sentinel Guard security tools");

        cmd
          .command("status")
          .description("Show Sentinel system status")
          .action(async () => {
            try {
              const status = await callSentinelStatus(sentinelUrl);
              console.log(JSON.stringify(status, null, 2));
            } catch (err: any) {
              console.error(`Error: ${err.message}`);
              process.exit(1);
            }
          });

        cmd
          .command("gate")
          .description("Evaluate an action through the gate")
          .requiredOption("-a, --action <action>", "Action category")
          .requiredOption("-p, --prompt <prompt>", "Action prompt")
          .action(async (opts: { action: string; prompt: string }) => {
            try {
              const result = await callSentinelGate(
                sentinelUrl,
                opts.action,
                opts.prompt
              );
              console.log(JSON.stringify(result, null, 2));
            } catch (err: any) {
              console.error(`Error: ${err.message}`);
              process.exit(1);
            }
          });
      },
      { commands: ["sentinel"] }
    );
  }

  // ── Auto-reply command: /sentinel ─────────────────────────────────
  if (api.registerCommand) {
    api.registerCommand({
      name: "sentinel",
      description: "Show Sentinel Guard status (no AI needed)",
      handler: async () => {
        try {
          const status = await callSentinelStatus(sentinelUrl);
          const s = status as any;
          const ks = s.kill_switch || {};
          const pc = s.proof_chain || {};
          return {
            text: [
              `Sentinel Guard Status:`,
              `  Kill Switch: ${ks.armed ? "ARMED" : "disarmed"}`,
              `  Proof Chain: ${pc.chain_length ?? 0} entries, valid=${pc.chain_valid ?? "unknown"}`,
              `  Pending Approvals: ${s.pending_approvals ?? 0}`,
            ].join("\n"),
          };
        } catch (err: any) {
          return { text: `Sentinel offline: ${err.message}` };
        }
      },
    });
  }

  api.logger?.info?.("Sentinel Guard plugin loaded");
}
