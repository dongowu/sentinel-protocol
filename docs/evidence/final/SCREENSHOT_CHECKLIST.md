# Submission Screenshot Checklist (Final)

Generated on: 2026-02-24

Use this checklist to capture the human-verifiable demo evidence required by `docs/SUBMISSION.md`.

## Required screenshots / clips

- [ ] `01-block-injection.png`  
  Show `POST /sentinel/gate` with injection prompt returning `"decision": "BLOCK"` (source: `openclaw-cli/02-gate-block.json`).
- [ ] `02-require-approval.png`  
  Show wallet action returning `"decision": "REQUIRE_APPROVAL"` with `challenge_id` (source: `openclaw-cli/03-gate-require-approval.json`).
- [ ] `03-approval-confirmed.png`  
  Show `POST /sentinel/approval/confirm` approved response with issued token (source: `openclaw-cli/04-approval-confirm.json`).
- [ ] `04-kill-switch-armed.png`  
  Show `POST /sentinel/kill-switch/arm` response and `GET /sentinel/status` armed=true (sources: `openclaw-cli/05-kill-switch-arm.json`, `openclaw-cli/06-status-after-arm.json`).
- [ ] `05-proof-latest.png`  
  Show `GET /sentinel/proof/latest` with `chain_valid=true` and latest proof entry (source: `openclaw-cli/07-proof-latest.json`).
- [ ] `06-openclaw-plugin-status.png`  
  Show plugin/tool integration status (e.g. `openclaw sentinel status`) (source: `openclaw-cli/09-status-final.json`).
- [ ] `07-onchain-anchor-tx.png`  
  Show tx digest lookup result (Sui explorer or `sui client tx-block <digest>`).

## Recommended folder structure

Place all screenshots/clips in:

- `docs/evidence/final/media/`

Suggested clip names:

- `demo-block-flow.mp4`
- `demo-approval-flow.mp4`
- `demo-kill-switch.mp4`

## Notes

- Keep terminal prompt visible in screenshots.
- Include timestamp where possible.
- Keep the same endpoint host (`127.0.0.1:18080`) across captures for consistency.
