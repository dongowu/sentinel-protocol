#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 5 ]; then
  echo "Usage: $0 <package> <module> <function> <gas_budget> <arg1> [arg2 ...]" >&2
  exit 1
fi

PACKAGE="$1"
MODULE="$2"
FUNCTION="$3"
GAS_BUDGET="$4"
shift 4

TS="$(date +%Y%m%d_%H%M%S)"
OUT_DIR="./proposals"
mkdir -p "$OUT_DIR"
OUT_FILE="$OUT_DIR/proposal_${MODULE}_${FUNCTION}_${TS}.json"

# Air-gap flow: only produce unsigned tx bytes; execution requires external signer/hardware wallet.
TX_BYTES=$(sui client call \
  --package "$PACKAGE" \
  --module "$MODULE" \
  --function "$FUNCTION" \
  --args "$@" \
  --gas-budget "$GAS_BUDGET" \
  --serialize-unsigned-transaction | tail -n 1)

cat > "$OUT_FILE" <<EOF
{
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "package": "$PACKAGE",
  "module": "$MODULE",
  "function": "$FUNCTION",
  "args": ["$*"],
  "gas_budget": "$GAS_BUDGET",
  "unsigned_tx_bytes": "$TX_BYTES",
  "status": "PROPOSED_REQUIRES_EXTERNAL_SIGNATURE"
}
EOF

echo "Proposal written: $OUT_FILE"
echo "Next step: sign via hardware wallet, then execute with:"
echo "  sui client execute-signed-tx --tx-bytes <TX_BYTES> --signatures <SIGS>"
