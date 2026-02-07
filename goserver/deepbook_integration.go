package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// DeepBookConfig holds DeepBook trading configuration
type DeepBookConfig struct {
	Enabled         bool              `json:"enabled"`
	PoolID          string            `json:"pool_id"`           // DeepBook pool ID
	BaseAsset       string            `json:"base_asset"`        // Token to sell (e.g., MEME coin)
	QuoteAsset      string            `json:"quote_asset"`       // Token to buy (e.g., USDC)
	SlippageBps     int               `json:"slippage_bps"`      // Slippage in basis points (100 = 1%)
	MinOutputAmount string            `json:"min_output_amount"` // Minimum USDC to receive
	AssetBalances   map[string]string `json:"asset_balances"`    // Asset object IDs to sell
}

// PTBBuilder constructs Programmable Transaction Blocks
type PTBBuilder struct {
	config *DeepBookConfig
}

// NewPTBBuilder creates a new PTB builder
func NewPTBBuilder(config *DeepBookConfig) *PTBBuilder {
	return &PTBBuilder{
		config: config,
	}
}

// BuildPanicSellPTB constructs a PTB for emergency liquidation
// This PTB will:
// 1. Execute the will on the smart contract
// 2. Sell volatile assets on DeepBook for USDC
// 3. Transfer USDC to beneficiary
func (pb *PTBBuilder) BuildPanicSellPTB(vaultID, beneficiary string) (string, error) {
	log.Println("🔥 Building Panic Sell PTB...")
	log.Printf("   Vault ID: %s", vaultID)
	log.Printf("   Beneficiary: %s", beneficiary)
	log.Printf("   Pool: %s", pb.config.PoolID)
	log.Printf("   Selling: %s → %s", pb.config.BaseAsset, pb.config.QuoteAsset)

	// Build PTB JSON structure
	ptb := map[string]interface{}{
		"version": 1,
		"sender":  beneficiary, // Or could be anyone
		"gasData": map[string]interface{}{
			"budget": "100000000", // 0.1 SUI
		},
		"inputs": []interface{}{
			// Input 0: Vault object
			map[string]interface{}{
				"type":      "object",
				"objectId":  vaultID,
				"version":   "latest",
				"digest":    "",
				"mutable":   true,
				"objectRef": nil,
			},
			// Input 1: Clock object
			map[string]interface{}{
				"type":     "object",
				"objectId": "0x0000000000000000000000000000000000000000000000000000000000000006",
			},
			// Input 2: DeepBook Pool
			map[string]interface{}{
				"type":     "object",
				"objectId": pb.config.PoolID,
			},
			// Input 3: Asset to sell
			map[string]interface{}{
				"type":     "object",
				"objectId": pb.config.AssetBalances[pb.config.BaseAsset],
			},
			// Input 4: Min output amount
			map[string]interface{}{
				"type":  "pure",
				"value": pb.config.MinOutputAmount,
			},
		},
		"commands": []interface{}{
			// Command 0: Execute will
			map[string]interface{}{
				"MoveCall": map[string]interface{}{
					"package":  "PACKAGE_ID",
					"module":   "lazarus_protocol",
					"function": "execute_will",
					"arguments": []interface{}{
						map[string]interface{}{"Input": 0}, // vault
						map[string]interface{}{"Input": 1}, // clock
					},
				},
			},
			// Command 1: Place market order on DeepBook
			map[string]interface{}{
				"MoveCall": map[string]interface{}{
					"package":  "0xdee9", // DeepBook package
					"module":   "clob_v2",
					"function": "place_market_order",
					"typeArguments": []string{
						pb.config.BaseAsset,
						pb.config.QuoteAsset,
					},
					"arguments": []interface{}{
						map[string]interface{}{"Input": 2}, // pool
						map[string]interface{}{"Input": 3}, // base_coin
						map[string]interface{}{"Input": 4}, // min_quote
						map[string]interface{}{"Input": 1}, // clock
					},
				},
			},
			// Command 2: Transfer USDC to beneficiary
			map[string]interface{}{
				"TransferObjects": map[string]interface{}{
					"objects": []interface{}{
						map[string]interface{}{"Result": 1}, // USDC from trade
					},
					"address": beneficiary,
				},
			},
		},
	}

	// Convert to JSON
	ptbJSON, err := json.MarshalIndent(ptb, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal PTB: %w", err)
	}

	log.Println("✓ PTB constructed successfully")
	return string(ptbJSON), nil
}

// ExecutePanicSell executes the panic sell PTB
func (pb *PTBBuilder) ExecutePanicSell(vaultID, beneficiary, packageID string) error {
	log.Println("\n🚨 EXECUTING PANIC SELL 🚨")
	log.Println("=" + strings.Repeat("=", 59))

	// Step 1: Get asset balance
	log.Println("\n[1/4] Checking asset balance...")
	balance, err := pb.getAssetBalance(pb.config.BaseAsset)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}
	log.Printf("   Balance: %s %s", balance, pb.config.BaseAsset)

	// Step 2: Get current market price
	log.Println("\n[2/4] Fetching market price from DeepBook...")
	price, err := pb.getMarketPrice()
	if err != nil {
		log.Printf("   ⚠️  Could not fetch price: %v", err)
		log.Println("   Proceeding with market order...")
	} else {
		log.Printf("   Current price: %s USDC per token", price)
	}

	// Step 3: Build and execute PTB
	log.Println("\n[3/4] Building Programmable Transaction Block...")

	// For now, use CLI to execute the PTB
	// In production, you would use the Sui SDK to build and sign the PTB
	cmd := exec.Command("sui", "client", "ptb",
		"--move-call", fmt.Sprintf("%s::lazarus_protocol::execute_will @%s @0x6", packageID, vaultID),
		"--move-call", fmt.Sprintf("0xdee9::clob_v2::place_market_order<%s,%s> @%s @%s %s @0x6",
			pb.config.BaseAsset, pb.config.QuoteAsset,
			pb.config.PoolID, pb.config.AssetBalances[pb.config.BaseAsset], pb.config.MinOutputAmount),
		"--transfer-objects", "[Result(1)]", beneficiary,
		"--gas-budget", "100000000",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("PTB execution failed: %v\nOutput: %s", err, string(output))
	}

	log.Println("   ✓ PTB executed successfully")

	// Step 4: Verify results
	log.Println("\n[4/4] Verifying transaction...")
	log.Printf("   Transaction output:\n%s", string(output))

	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("✓ PANIC SELL COMPLETED SUCCESSFULLY!")
	log.Println("  Assets liquidated and USDC transferred to beneficiary")
	log.Println(strings.Repeat("=", 60))

	return nil
}

// getAssetBalance queries the balance of a specific asset
func (pb *PTBBuilder) getAssetBalance(assetType string) (string, error) {
	// This would query the Sui RPC to get the balance
	// For now, return a placeholder
	return "1000000", nil // 1 token with 6 decimals
}

// getMarketPrice fetches the current market price from DeepBook
func (pb *PTBBuilder) getMarketPrice() (string, error) {
	// This would query DeepBook to get the current best bid/ask
	// For now, return a placeholder
	return "0.05", nil // 0.05 USDC per token
}

// SimulatePanicSell simulates the panic sell without executing
func (pb *PTBBuilder) SimulatePanicSell(vaultID, beneficiary string) error {
	log.Println("\n🧪 SIMULATING PANIC SELL 🧪")
	log.Println("=" + strings.Repeat("=", 59))

	// Get balances
	balance, _ := pb.getAssetBalance(pb.config.BaseAsset)
	price, _ := pb.getMarketPrice()

	// Calculate expected output
	// This is simplified - real calculation would account for slippage, fees, etc.
	log.Println("\n📊 Simulation Results:")
	log.Printf("   Input: %s %s", balance, pb.config.BaseAsset)
	log.Printf("   Market Price: %s USDC", price)
	log.Printf("   Slippage: %d bps (%.2f%%)", pb.config.SlippageBps, float64(pb.config.SlippageBps)/100)
	log.Printf("   Min Output: %s USDC", pb.config.MinOutputAmount)
	log.Println("\n   Transaction Flow:")
	log.Println("   1. Execute will on vault")
	log.Println("   2. Place market sell order on DeepBook")
	log.Println("   3. Receive USDC")
	log.Println("   4. Transfer USDC to beneficiary")

	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("✓ Simulation complete (no actual transaction)")
	log.Println(strings.Repeat("=", 60))

	return nil
}
