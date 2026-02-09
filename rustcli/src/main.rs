use aes_gcm::{
    aead::{Aead, KeyInit, OsRng},
    Aes256Gcm, Nonce,
};
use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use ed25519_dalek::{Signer, SigningKey, VerifyingKey};
use rand::RngCore;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs;
use std::path::PathBuf;

/// Lazarus Vault - Zero-knowledge encryption for Sentinel Protocol
#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Command to execute
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand, Debug)]
enum Commands {
    /// Encrypt a file and store it on Walrus Protocol
    EncryptAndStore {
        /// Path to the file to encrypt
        #[arg(short, long)]
        file: PathBuf,

        /// Walrus Publisher URL (e.g., https://publisher.walrus-testnet.walrus.space)
        #[arg(short, long)]
        publisher: String,

        /// Number of epochs to store (optional, defaults to 1)
        #[arg(short, long, default_value = "1")]
        epochs: u64,
    },

    /// Download and decrypt a blob from Walrus Protocol
    Decrypt {
        /// Walrus blob ID
        #[arg(long)]
        blob_id: String,

        /// Hex-encoded decryption key (key||nonce)
        #[arg(long)]
        decryption_key: String,

        /// Walrus Publisher/Reader URL
        #[arg(short, long)]
        publisher: String,

        /// Output file path for decrypted plaintext
        #[arg(short, long)]
        output: PathBuf,
    },

    /// Compute deterministic audit hash for Sentinel records
    HashAudit {
        #[arg(long)]
        action: String,
        #[arg(long)]
        prompt: String,
        #[arg(long)]
        score: u8,
        #[arg(long)]
        tags: String,
        #[arg(long)]
        decision: String,
        #[arg(long)]
        reason: String,
        #[arg(long)]
        timestamp: String,
    },

    /// Sign an audit hash with ed25519 private key (hex 32-byte seed)
    SignAudit {
        #[arg(long)]
        record_hash: String,
        #[arg(long)]
        private_key: String,
    },
}

/// Output structure containing encryption and storage metadata
#[derive(Serialize, Deserialize, Debug)]
struct EncryptionOutput {
    /// Walrus blob ID where encrypted data is stored
    blob_id: String,
    /// Base64-encoded decryption key (256-bit AES key + 96-bit nonce)
    decryption_key: String,
    /// SHA-256 checksum of the original plaintext file
    checksum: String,
    /// Size of the original file in bytes
    original_size: usize,
    /// Size of the encrypted blob in bytes
    encrypted_size: usize,
}

#[derive(Serialize, Deserialize, Debug)]
struct AuditHashOutput {
    record_hash: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct AuditSignOutput {
    record_hash: String,
    signature: String,
    public_key: String,
}

#[derive(Serialize, Deserialize, Debug)]
struct CanonicalAuditRecord {
    action: String,
    prompt: String,
    score: u8,
    tags: Vec<String>,
    decision: String,
    reason: String,
    timestamp: String,
}

/// Walrus API response structure
#[derive(Deserialize, Debug)]
struct WalrusResponse {
    #[serde(rename = "newlyCreated")]
    newly_created: Option<WalrusBlob>,
    #[serde(rename = "alreadyCertified")]
    already_certified: Option<WalrusBlob>,
}

#[derive(Deserialize, Debug)]
struct WalrusBlob {
    #[serde(rename = "blobObject")]
    blob_object: BlobObject,
}

#[derive(Deserialize, Debug)]
struct BlobObject {
    #[serde(rename = "blobId")]
    blob_id: String,
}

impl WalrusResponse {
    fn get_blob_id(&self) -> Option<String> {
        if let Some(ref created) = self.newly_created {
            Some(created.blob_object.blob_id.clone())
        } else if let Some(ref certified) = self.already_certified {
            Some(certified.blob_object.blob_id.clone())
        } else {
            None
        }
    }
}

fn main() -> Result<()> {
    let args = Args::parse();

    match args.command {
        Commands::EncryptAndStore {
            file,
            publisher,
            epochs,
        } => {
            encrypt_and_store(&file, &publisher, epochs)?;
        }
        Commands::Decrypt {
            blob_id,
            decryption_key,
            publisher,
            output,
        } => {
            decrypt_blob(&blob_id, &decryption_key, &publisher, &output)?;
        }
        Commands::HashAudit {
            action,
            prompt,
            score,
            tags,
            decision,
            reason,
            timestamp,
        } => {
            hash_audit(action, prompt, score, tags, decision, reason, timestamp)?;
        }
        Commands::SignAudit {
            record_hash,
            private_key,
        } => {
            sign_audit(&record_hash, &private_key)?;
        }
    }

    Ok(())
}

/// Download ciphertext from Walrus and decrypt to a local file.
fn decrypt_blob(
    blob_id: &str,
    decryption_key: &str,
    publisher_url: &str,
    output: &PathBuf,
) -> Result<()> {
    eprintln!("[1/4] Downloading encrypted blob: {}", blob_id);
    let ciphertext = download_from_walrus(blob_id, publisher_url)?;
    eprintln!("       Ciphertext size: {} bytes", ciphertext.len());

    eprintln!("[2/4] Decoding decryption key...");
    let (key, nonce) = decode_decryption_key(decryption_key)?;

    eprintln!("[3/4] Decrypting blob...");
    let cipher = Aes256Gcm::new_from_slice(&key).context("Failed to create AES-256-GCM cipher")?;
    let nonce_obj = Nonce::from_slice(&nonce);
    let plaintext = cipher
        .decrypt(nonce_obj, ciphertext.as_ref())
        .map_err(|e| anyhow::anyhow!("Decryption failed: {}", e))?;

    eprintln!("[4/4] Writing decrypted file: {}", output.display());
    fs::write(output, &plaintext)
        .with_context(|| format!("Failed to write decrypted output: {}", output.display()))?;

    eprintln!("\n✓ Success! Blob decrypted to {}", output.display());
    Ok(())
}

fn download_from_walrus(blob_id: &str, publisher_url: &str) -> Result<Vec<u8>> {
    let runtime = tokio::runtime::Runtime::new()?;
    runtime.block_on(async { download_from_walrus_async(blob_id, publisher_url).await })
}

async fn download_from_walrus_async(blob_id: &str, publisher_url: &str) -> Result<Vec<u8>> {
    let client = reqwest::Client::new();
    let base = publisher_url.trim_end_matches('/');
    let candidates = [
        format!("{}/v1/{}", base, blob_id),
        format!("{}/v1/blobs/{}", base, blob_id),
    ];

    let mut last_err = String::new();
    for url in candidates {
        let response = client.get(&url).send().await;
        match response {
            Ok(resp) if resp.status().is_success() => {
                let bytes = resp
                    .bytes()
                    .await
                    .with_context(|| format!("Failed reading response body from {}", url))?;
                return Ok(bytes.to_vec());
            }
            Ok(resp) => {
                last_err = format!("{} -> status {}", url, resp.status());
            }
            Err(e) => {
                last_err = format!("{} -> {}", url, e);
            }
        }
    }

    anyhow::bail!(
        "Failed to download blob {} from Walrus: {}",
        blob_id,
        last_err
    )
}

fn decode_decryption_key(encoded: &str) -> Result<([u8; 32], [u8; 12])> {
    let raw = hex::decode(encoded.trim_start_matches("0x"))
        .context("decryption_key must be a hex string")?;
    if raw.len() != 44 {
        anyhow::bail!("decryption_key must be 44 bytes (88 hex chars)");
    }

    let mut key = [0u8; 32];
    let mut nonce = [0u8; 12];
    key.copy_from_slice(&raw[0..32]);
    nonce.copy_from_slice(&raw[32..44]);
    Ok((key, nonce))
}

/// Main encryption and storage workflow
fn encrypt_and_store(file_path: &PathBuf, publisher_url: &str, epochs: u64) -> Result<()> {
    // Step 1: Read the file
    eprintln!("[1/5] Reading file: {}", file_path.display());
    let plaintext = fs::read(file_path)
        .with_context(|| format!("Failed to read file: {}", file_path.display()))?;

    if plaintext.is_empty() {
        anyhow::bail!("File is empty");
    }

    let original_size = plaintext.len();
    eprintln!("       File size: {} bytes", original_size);

    // Step 2: Calculate checksum of original file
    eprintln!("[2/5] Computing SHA-256 checksum...");
    let checksum = compute_checksum(&plaintext);
    eprintln!("       Checksum: {}", checksum);

    // Step 3: Encrypt the file (zero-knowledge)
    eprintln!("[3/5] Encrypting file with AES-256-GCM...");
    let (ciphertext, key, nonce) = encrypt_data(&plaintext)?;
    let encrypted_size = ciphertext.len();
    eprintln!("       Encrypted size: {} bytes", encrypted_size);

    // Step 4: Upload to Walrus Protocol
    eprintln!("[4/5] Uploading to Walrus Protocol...");
    eprintln!("       Publisher: {}", publisher_url);
    let blob_id = upload_to_walrus(&ciphertext, publisher_url, epochs)?;
    eprintln!("       Blob ID: {}", blob_id);

    // Step 5: Generate output with decryption key
    eprintln!("[5/5] Generating output...");
    let decryption_key = encode_decryption_key(&key, &nonce);

    let output = EncryptionOutput {
        blob_id,
        decryption_key,
        checksum,
        original_size,
        encrypted_size,
    };

    // Output JSON to STDOUT (this is what the Go daemon will capture)
    let json_output = serde_json::to_string_pretty(&output)?;
    println!("{}", json_output);

    eprintln!("\n✓ Success! File encrypted and stored on Walrus Protocol.");
    eprintln!("⚠ CRITICAL: Save the 'decryption_key' securely. It cannot be recovered!");

    Ok(())
}

/// Encrypts data using AES-256-GCM with a randomly generated key
///
/// Returns: (ciphertext, key, nonce)
fn encrypt_data(plaintext: &[u8]) -> Result<(Vec<u8>, [u8; 32], [u8; 12])> {
    // Generate random 256-bit key
    let mut key = [0u8; 32];
    OsRng.fill_bytes(&mut key);

    // Generate random 96-bit nonce (12 bytes for GCM)
    let mut nonce_bytes = [0u8; 12];
    OsRng.fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);

    // Create cipher instance
    let cipher = Aes256Gcm::new_from_slice(&key).context("Failed to create AES-256-GCM cipher")?;

    // Encrypt the data
    let ciphertext = cipher
        .encrypt(nonce, plaintext)
        .map_err(|e| anyhow::anyhow!("Encryption failed: {}", e))?;

    Ok((ciphertext, key, nonce_bytes))
}

/// Computes SHA-256 checksum of data
fn compute_checksum(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    hex::encode(hasher.finalize())
}

/// Encodes the decryption key and nonce as a single base64 string
///
/// Format: base64(key || nonce) where key is 32 bytes and nonce is 12 bytes
fn encode_decryption_key(key: &[u8; 32], nonce: &[u8; 12]) -> String {
    let mut combined = Vec::with_capacity(44);
    combined.extend_from_slice(key);
    combined.extend_from_slice(nonce);
    hex::encode(combined)
}

/// Uploads encrypted data to Walrus Protocol
///
/// Returns the blob ID on success
fn upload_to_walrus(data: &[u8], publisher_url: &str, epochs: u64) -> Result<String> {
    let runtime = tokio::runtime::Runtime::new()?;
    runtime.block_on(async { upload_to_walrus_async(data, publisher_url, epochs).await })
}

async fn upload_to_walrus_async(data: &[u8], publisher_url: &str, epochs: u64) -> Result<String> {
    let client = reqwest::Client::new();

    // Construct the Walrus store endpoint
    let store_url = format!("{}/v1/store?epochs={}", publisher_url, epochs);

    // Send PUT request with the encrypted data
    let response = client
        .put(&store_url)
        .header("Content-Type", "application/octet-stream")
        .body(data.to_vec())
        .send()
        .await
        .context("Failed to send request to Walrus Publisher")?;

    // Check response status
    if !response.status().is_success() {
        let status = response.status();
        let error_text = response
            .text()
            .await
            .unwrap_or_else(|_| "Unknown error".to_string());
        anyhow::bail!(
            "Walrus upload failed with status {}: {}",
            status,
            error_text
        );
    }

    // Parse response JSON
    let walrus_response: WalrusResponse = response
        .json()
        .await
        .context("Failed to parse Walrus response")?;

    // Extract blob ID
    walrus_response
        .get_blob_id()
        .ok_or_else(|| anyhow::anyhow!("Walrus response missing blob ID"))
}

fn hash_audit(
    action: String,
    prompt: String,
    score: u8,
    tags: String,
    decision: String,
    reason: String,
    timestamp: String,
) -> Result<()> {
    let mut parsed_tags: Vec<String> = tags
        .split(',')
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty())
        .collect();
    parsed_tags.sort();

    let record = CanonicalAuditRecord {
        action,
        prompt,
        score,
        tags: parsed_tags,
        decision,
        reason,
        timestamp,
    };

    let canonical = serde_json::to_string(&record)?;
    let mut hasher = Sha256::new();
    hasher.update(canonical.as_bytes());
    let record_hash = format!("0x{}", hex::encode(hasher.finalize()));

    let output = AuditHashOutput { record_hash };
    println!("{}", serde_json::to_string_pretty(&output)?);
    Ok(())
}

fn sign_audit(record_hash: &str, private_key_hex: &str) -> Result<()> {
    let hash_bytes = hex::decode(record_hash.trim_start_matches("0x"))
        .context("record_hash must be a hex string")?;

    let sk_bytes =
        hex::decode(private_key_hex.trim_start_matches("0x")).context("private_key must be hex")?;
    if sk_bytes.len() != 32 {
        anyhow::bail!("private_key must be 32 bytes (64 hex chars)");
    }

    let mut key = [0u8; 32];
    key.copy_from_slice(&sk_bytes);
    let signing_key = SigningKey::from_bytes(&key);
    let verifying_key: VerifyingKey = signing_key.verifying_key();
    let sig = signing_key.sign(&hash_bytes);

    let output = AuditSignOutput {
        record_hash: record_hash.to_string(),
        signature: hex::encode(sig.to_bytes()),
        public_key: hex::encode(verifying_key.to_bytes()),
    };

    println!("{}", serde_json::to_string_pretty(&output)?);
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_encryption_decryption() {
        let plaintext = b"Hello, Sentinel Protocol!";
        let (ciphertext, key, nonce) = encrypt_data(plaintext).unwrap();

        // Verify ciphertext is different from plaintext
        assert_ne!(ciphertext.as_slice(), plaintext);

        // Decrypt and verify
        let cipher = Aes256Gcm::new_from_slice(&key).unwrap();
        let nonce_obj = Nonce::from_slice(&nonce);
        let decrypted = cipher.decrypt(nonce_obj, ciphertext.as_ref()).unwrap();

        assert_eq!(decrypted.as_slice(), plaintext);
    }

    #[test]
    fn test_checksum() {
        let data = b"test data";
        let checksum = compute_checksum(data);
        assert_eq!(checksum.len(), 64); // SHA-256 produces 64 hex characters
    }

    #[test]
    fn test_key_encoding() {
        let key = [0u8; 32];
        let nonce = [1u8; 12];
        let encoded = encode_decryption_key(&key, &nonce);

        // Should be 88 hex characters (44 bytes * 2)
        assert_eq!(encoded.len(), 88);
    }

    #[test]
    fn test_key_decoding_roundtrip() {
        let key = [7u8; 32];
        let nonce = [9u8; 12];
        let encoded = encode_decryption_key(&key, &nonce);

        let (decoded_key, decoded_nonce) = decode_decryption_key(&encoded).unwrap();
        assert_eq!(decoded_key, key);
        assert_eq!(decoded_nonce, nonce);
    }
}
