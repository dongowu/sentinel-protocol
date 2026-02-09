#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <tx_digest> <audit_log_path> [hash_cli_path]" >&2
  exit 1
fi

TX_DIGEST="$1"
AUDIT_LOG="$2"
HASH_CLI="${3:-./rustcli/target/release/lazarus-vault}"

if [ ! -f "$AUDIT_LOG" ]; then
  echo "FAIL: audit log not found: $AUDIT_LOG"
  exit 2
fi

TX_JSON=$(sui client tx-block "$TX_DIGEST" --json)
TX_STATUS=$(echo "$TX_JSON" | jq -r '.effects.V2.status // .effects.status.status // empty')
if [ "$TX_STATUS" != "Success" ]; then
  echo "FAIL: on-chain tx not successful ($TX_STATUS)"
  exit 3
fi

ENTRY=$(jq -c "select(.tx_digest == \"$TX_DIGEST\")" "$AUDIT_LOG" | head -n 1)
if [ -z "$ENTRY" ]; then
  echo "FAIL: no local audit record matched tx_digest=$TX_DIGEST"
  exit 4
fi

ACTION=$(echo "$ENTRY" | jq -r '.action')
PROMPT=$(echo "$ENTRY" | jq -r '.prompt')
SCORE=$(echo "$ENTRY" | jq -r '.score')
TAGS=$(echo "$ENTRY" | jq -r '.tags | join(",")')
DECISION=$(echo "$ENTRY" | jq -r '.decision')
REASON=$(echo "$ENTRY" | jq -r '.reason')
TIMESTAMP=$(echo "$ENTRY" | jq -r '.timestamp')
REC_HASH=$(echo "$ENTRY" | jq -r '.record_hash')

REHASH=$($HASH_CLI hash-audit \
  --action "$ACTION" \
  --prompt "$PROMPT" \
  --score "$SCORE" \
  --tags "$TAGS" \
  --decision "$DECISION" \
  --reason "$REASON" \
  --timestamp "$TIMESTAMP" | jq -r '.record_hash')

if [ "$REHASH" != "$REC_HASH" ]; then
  echo "FAIL: local record hash mismatch"
  echo "  expected: $REC_HASH"
  echo "  recomputed: $REHASH"
  exit 5
fi

echo "PASS"
echo "- tx_digest: $TX_DIGEST"
echo "- onchain_status: $TX_STATUS"
echo "- record_hash: $REC_HASH"
echo "- decision: $DECISION"
