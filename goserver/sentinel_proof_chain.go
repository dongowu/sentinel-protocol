package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ProofEntry represents a single link in the tamper-evident proof chain.
type ProofEntry struct {
	Index      int       `json:"index"`
	RecordHash string    `json:"record_hash"`
	PrevHash   string    `json:"prev_hash"`
	ChainHash  string    `json:"chain_hash"`
	Timestamp  time.Time `json:"timestamp"`
	Action     string    `json:"action"`
	Decision   string    `json:"decision"`
}

// MerkleBatch groups proof entries into a Merkle-tree-anchored batch for Walrus storage.
type MerkleBatch struct {
	BatchID    int          `json:"batch_id"`
	MerkleRoot string       `json:"merkle_root"`
	Entries    []ProofEntry `json:"entries"`
	WalrusCID  string       `json:"walrus_cid,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

// ProofChain maintains a tamper-evident chain of audit records with Merkle tree
// batching and Walrus storage integration.
type ProofChain struct {
	entries           []ProofEntry
	batches           []MerkleBatch
	batchSize         int
	walrusPublisherURL string
	mu                sync.RWMutex
}

// NewProofChain creates a new proof chain with the given batch size and Walrus publisher URL.
func NewProofChain(batchSize int, walrusURL string) *ProofChain {
	if batchSize <= 0 {
		batchSize = 10
	}
	return &ProofChain{
		entries:           make([]ProofEntry, 0),
		batches:           make([]MerkleBatch, 0),
		batchSize:         batchSize,
		walrusPublisherURL: walrusURL,
	}
}

// Append adds an audit record to the proof chain. When the pending entries reach
// batchSize, they are automatically batched into a MerkleBatch and uploaded to Walrus.
func (pc *ProofChain) Append(rec *AuditRecord) *ProofEntry {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	index := len(pc.entries)
	prevHash := "0x0"
	if index > 0 {
		prevHash = pc.entries[index-1].ChainHash
	}

	entry := ProofEntry{
		Index:      index,
		RecordHash: rec.RecordHash,
		PrevHash:   prevHash,
		ChainHash:  computeChainHash(index, rec.RecordHash, prevHash),
		Timestamp:  rec.Timestamp,
		Action:     rec.Action,
		Decision:   rec.Decision,
	}
	pc.entries = append(pc.entries, entry)

	// Check if we have enough pending entries to form a batch.
	batchedCount := 0
	for _, b := range pc.batches {
		batchedCount += len(b.Entries)
	}
	pending := pc.entries[batchedCount:]
	if len(pending) >= pc.batchSize {
		pc.flushBatch(pending)
	}

	return &entry
}

// flushBatch creates a MerkleBatch from the given entries and uploads to Walrus.
// Must be called with pc.mu held.
func (pc *ProofChain) flushBatch(entries []ProofEntry) {
	batchEntries := make([]ProofEntry, len(entries))
	copy(batchEntries, entries)

	batch := MerkleBatch{
		BatchID:    len(pc.batches),
		MerkleRoot: computeMerkleRoot(batchEntries),
		Entries:    batchEntries,
		CreatedAt:  time.Now().UTC(),
	}

	if pc.walrusPublisherURL != "" {
		if cid, err := uploadToWalrus(pc.walrusPublisherURL, &batch); err == nil {
			batch.WalrusCID = cid
		}
	}

	pc.batches = append(pc.batches, batch)
}

// computeChainHash produces a SHA-256 digest of Index|RecordHash|PrevHash.
func computeChainHash(index int, recordHash, prevHash string) string {
	data := fmt.Sprintf("%d|%s|%s", index, recordHash, prevHash)
	sum := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(sum[:])
}

// computeMerkleRoot builds a binary Merkle tree over the ChainHash values of the
// given entries and returns the hex-encoded root.
func computeMerkleRoot(entries []ProofEntry) string {
	if len(entries) == 0 {
		return "0x0"
	}

	// Leaf layer: SHA-256 of each entry's ChainHash.
	hashes := make([][]byte, len(entries))
	for i, e := range entries {
		h := sha256.Sum256([]byte(e.ChainHash))
		hashes[i] = h[:]
	}

	// Build tree bottom-up.
	for len(hashes) > 1 {
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}
		next := make([][]byte, 0, len(hashes)/2)
		for i := 0; i < len(hashes); i += 2 {
			combined := append(hashes[i], hashes[i+1]...)
			h := sha256.Sum256(combined)
			next = append(next, h[:])
		}
		hashes = next
	}

	return "0x" + hex.EncodeToString(hashes[0])
}

// uploadToWalrus POSTs the JSON-marshaled batch to the Walrus publisher and returns
// the blob ID / CID from the response.
func uploadToWalrus(walrusURL string, batch *MerkleBatch) (string, error) {
	body, err := json.Marshal(batch)
	if err != nil {
		return "", fmt.Errorf("marshal batch: %w", err)
	}

	url := walrusURL + "/v1/blobs?epochs=5"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("walrus upload: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read walrus response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("walrus returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// The Walrus publisher may return the blob ID in various envelope shapes.
	// Try the known response formats.
	var newlyCreated struct {
		NewlyCreated struct {
			BlobObject struct {
				BlobID string `json:"blobId"`
			} `json:"blobObject"`
		} `json:"newlyCreated"`
	}
	if err := json.Unmarshal(respBody, &newlyCreated); err == nil && newlyCreated.NewlyCreated.BlobObject.BlobID != "" {
		return newlyCreated.NewlyCreated.BlobObject.BlobID, nil
	}

	var alreadyCertified struct {
		AlreadyCertified struct {
			BlobID string `json:"blobId"`
		} `json:"alreadyCertified"`
	}
	if err := json.Unmarshal(respBody, &alreadyCertified); err == nil && alreadyCertified.AlreadyCertified.BlobID != "" {
		return alreadyCertified.AlreadyCertified.BlobID, nil
	}

	// Fallback: try a flat blob_id or cid field.
	var flat struct {
		BlobID string `json:"blob_id"`
		CID    string `json:"cid"`
	}
	if err := json.Unmarshal(respBody, &flat); err == nil {
		if flat.BlobID != "" {
			return flat.BlobID, nil
		}
		if flat.CID != "" {
			return flat.CID, nil
		}
	}

	return "", fmt.Errorf("could not extract blob ID from walrus response: %s", string(respBody))
}

// GetLatestProof returns the most recent proof entry, or nil if the chain is empty.
func (pc *ProofChain) GetLatestProof() *ProofEntry {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if len(pc.entries) == 0 {
		return nil
	}
	entry := pc.entries[len(pc.entries)-1]
	return &entry
}

// GetLatestBatch returns the most recent Merkle batch, or nil if none exist yet.
func (pc *ProofChain) GetLatestBatch() *MerkleBatch {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	if len(pc.batches) == 0 {
		return nil
	}
	batch := pc.batches[len(pc.batches)-1]
	return &batch
}

// VerifyChain validates the integrity of every link in the proof chain by
// recomputing each ChainHash from its inputs.
func (pc *ProofChain) VerifyChain() bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	for i, entry := range pc.entries {
		expectedPrev := "0x0"
		if i > 0 {
			expectedPrev = pc.entries[i-1].ChainHash
		}
		if entry.PrevHash != expectedPrev {
			return false
		}
		expected := computeChainHash(entry.Index, entry.RecordHash, entry.PrevHash)
		if entry.ChainHash != expected {
			return false
		}
	}
	return true
}

// Len returns the total number of proof entries in the chain.
func (pc *ProofChain) Len() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return len(pc.entries)
}
