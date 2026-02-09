# Integration Guide

## 1) Validate toolchain

```bash
go version
cargo --version
sui --version
```

## 2) Build and test all modules

```bash
cd goserver && go test ./... -v
cd ../rustcli && cargo test --all
cd ../contract && sui move build
```

## 3) Run cryptography flow

```bash
cd rustcli
cargo build --release
./target/release/lazarus-vault encrypt-and-store \
  --file ../test_will.txt \
  --publisher https://publisher.walrus-testnet.walrus.space \
  --epochs 1
```

Use returned `blob_id` + `decryption_key`:

```bash
./target/release/lazarus-vault decrypt \
  --blob-id <blob_id> \
  --decryption-key <decryption_key> \
  --publisher https://publisher.walrus-testnet.walrus.space \
  --output ./decrypted.txt
```

## 4) Run Go benchmark sample

```bash
cd ../goserver
go run . --config config.openclaw.example.json --sentinel-benchmark benchmark_cases.example.json
```

## 5) Optional one-shot demo

```bash
cd ..
./demo.sh
```

## Notes

- Move modules include governance and enhanced audit storage, but production anti-sybil controls are still recommended.
- Sentinel guard now consumes behavioral policy signals in addition to keyword-based risk tags.
