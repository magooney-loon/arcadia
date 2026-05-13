package main

import (
	"os"
	"sync/atomic"

	"github.com/enviodev/hypersync-client-go/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// arcRPCPool is the ordered list of public Arc RPC endpoints used for rotation.
var arcRPCPool = []string{
	ArcRPCPrimary,
	ArcRPCBlockdaemon,
	ArcRPCDRPC,
	ArcRPCQuickNode,
}

var arcRPCIndex atomic.Uint64

// ── Arc Testnet network config ────────────────────────────────────────────────

const (
	ArcChainID = 5042002

	// HyperSync — also reachable via https://5042002.hypersync.xyz
	ArcHyperSyncURL = "https://arc-testnet.hypersync.xyz"

	// Public JSON-RPC endpoints (no auth required)
	ArcRPCPrimary     = "https://rpc.testnet.arc.network"
	ArcRPCBlockdaemon = "https://rpc.blockdaemon.testnet.arc.network"
	ArcRPCDRPC        = "https://rpc.drpc.testnet.arc.network"
	ArcRPCQuickNode   = "https://rpc.quicknode.testnet.arc.network"

	// WebSocket endpoints
	ArcWSS          = "wss://rpc.testnet.arc.network"
	ArcWSSDrpc      = "wss://rpc.drpc.testnet.arc.network"
	ArcWSSQuickNode = "wss://rpc.quicknode.testnet.arc.network"

	// Block explorer
	ArcExplorer = "https://testnet.arcscan.app"

	// CCTP domain ID for Arc
	ArcCCTPDomain = 26
)

// ArcNetworkID is the HyperSync internal key for the Arc Testnet client.
var ArcNetworkID = utils.NetworkID(ArcChainID)

// ── Contract addresses ────────────────────────────────────────────────────────

var (
	// Stablecoins
	AddrUSDC = common.HexToAddress("0x3600000000000000000000000000000000000000")
	AddrEURC = common.HexToAddress("0x89B50855Aa3bE2F677cD6303Cec089B5F319D72a")
	AddrUSYC = common.HexToAddress("0xe9185F0c5F296Ed1797AaE4238D26CCaBEadb86C")

	// USYC supporting contracts
	AddrUSYCEntitlements = common.HexToAddress("0xcc205224862c7641930c87679e98999d23c26113")
	AddrUSYCTeller       = common.HexToAddress("0x9fdF14c5B14173D74C08Af27AebFf39240dC105A")

	// CCTP v2
	AddrCCTPTokenMessenger     = common.HexToAddress("0x8FE6B999Dc680CcFDD5Bf7EB0974218be2542DAA")
	AddrCCTPMessageTransmitter = common.HexToAddress("0xE737e5cEBEEBa77EFE34D4aa090756590b1CE275")
	AddrCCTPTokenMinter        = common.HexToAddress("0xb43db544E2c27092c107639Ad201b3dEfAbcF192")
	AddrCCTPMessage            = common.HexToAddress("0xbaC0179bB358A8936169a63408C8481D582390C4")

	// Gateway
	AddrGatewayWallet = common.HexToAddress("0x0077777d7EBA4688BDeF3E311b846F25870A19B9")
	AddrGatewayMinter = common.HexToAddress("0x0022222ABE238Cc2C7Bb1f21003F0a260052475B")

	// StableFX
	AddrFxEscrow = common.HexToAddress("0x867650F5eAe8df91445971f14d89fd84F0C9a9f8")

	// Common Ethereum contracts deployed on Arc
	AddrPermit2        = common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3")
	AddrMulticall3     = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
	AddrCreate2Factory = common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C")

	// ERC-8004 agent registries on Arc testnet
	AddrAgentRegistry      = common.HexToAddress("0x8004A818BFB912233c491871b3d84c89A494BD9e")
	AddrReputationRegistry = common.HexToAddress("0x8004B663056A597Dffe9eCcC1965A193B7388713")
	AddrValidationRegistry = common.HexToAddress("0x8004Cb1BF31DAf7788923b405b754f57acEB4272")

	// ERC-8183 AgenticCommerce reference implementation on Arc testnet
	AddrAgenticCommerce = common.HexToAddress("0x0747EEf0706327138c69792bF28Cd525089e4583")
)

// KnownTokens maps contract address → symbol for the three Arc stablecoins.
var KnownTokens = map[common.Address]string{
	AddrUSDC: "USDC",
	AddrEURC: "EURC",
	AddrUSYC: "USYC",
}

// ── Event topics (keccak256 of event signatures) ──────────────────────────────

var (
	// ERC-20: Transfer(address indexed from, address indexed to, uint256 value)
	TopicTransfer = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	// ERC-20: Approval(address indexed owner, address indexed spender, uint256 value)
	TopicApproval = common.HexToHash("0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")

	// CCTP v2 — TokenMessengerV2 (verified from impl 0xf07c0ad1)
	// DepositForBurn(address indexed burnToken, uint256 amount, address indexed depositor,
	//   bytes32 mintRecipient, uint32 destinationDomain, bytes32 destinationTokenMessenger,
	//   bytes32 destinationCaller, uint256 maxFee, uint32 indexed minFinalityThreshold, bytes hookData)
	TopicDepositForBurn = crypto.Keccak256Hash([]byte("DepositForBurn(address,uint256,address,bytes32,uint32,bytes32,bytes32,uint256,uint32,bytes)"))

	// CCTP v2 — TokenMessengerV2
	// MintAndWithdraw(address indexed mintRecipient, uint256 amount, address indexed mintToken, uint256 feeCollected)
	TopicMintAndWithdraw = crypto.Keccak256Hash([]byte("MintAndWithdraw(address,uint256,address,uint256)"))

	// CCTP v2 — MessageTransmitterV2 (verified from impl 0xa849059b)
	// MessageReceived(address indexed caller, uint32 sourceDomain, bytes32 indexed nonce,
	//   bytes32 sender, uint32 indexed finalityThresholdExecuted, bytes messageBody)
	TopicMessageReceived = crypto.Keccak256Hash([]byte("MessageReceived(address,uint32,bytes32,bytes32,uint32,bytes)"))

	// GatewayWallet (verified from impl 0x44eeddc9)
	// Deposited(address indexed token, address indexed depositor, address indexed sender, uint256 value)
	TopicGatewayDeposited = crypto.Keccak256Hash([]byte("Deposited(address,address,address,uint256)"))

	// GatewayWallet — outbound bridge (USDC leaving Arc)
	// GatewayBurned(address indexed token, address indexed depositor, bytes32 indexed transferSpecHash,
	//   uint32 destinationDomain, bytes32 destinationRecipient, address signer,
	//   uint256 value, uint256 fee, uint256 fromAvailable, uint256 fromWithdrawing)
	TopicGatewayBurned = crypto.Keccak256Hash([]byte("GatewayBurned(address,address,bytes32,uint32,bytes32,address,uint256,uint256,uint256,uint256)"))

	// GatewayMinter (verified from impl 0x9ef4c7ad) — inbound bridge (USDC arriving on Arc)
	// AttestationUsed(address indexed token, address indexed recipient, bytes32 indexed transferSpecHash,
	//   uint32 sourceDomain, bytes32 sourceDepositor, bytes32 sourceSigner, uint256 value)
	TopicAttestationUsed = crypto.Keccak256Hash([]byte("AttestationUsed(address,address,bytes32,uint32,bytes32,bytes32,uint256)"))

	// ERC-8004 IdentityRegistry is ERC-721; agent registration mints a token.
	TopicAgentRegistered = TopicTransfer

	// ERC-8183 AgenticCommerce reference implementation.
	TopicJobCreated = crypto.Keccak256Hash([]byte("JobCreated(uint256,address,address,address,uint256,address)"))

	// FxEscrow (StableFX) — verified from implementation ABI at 0x721eAFa9C1e38DD7fFf81d30ea1a5500b37Cf658
	TopicTradeRecorded      = crypto.Keccak256Hash([]byte("TradeRecorded(uint256,bytes32)"))
	TopicMakerFunded        = crypto.Keccak256Hash([]byte("MakerFunded(uint256,address)"))
	TopicTakerFunded        = crypto.Keccak256Hash([]byte("TakerFunded(uint256,address)"))
	TopicTradeStatusChanged = crypto.Keccak256Hash([]byte("TradeStatusChanged(uint256,address,uint8)"))
	TopicFeesProcessed      = crypto.Keccak256Hash([]byte("FeesProcessed(uint256,uint256,uint256)"))

	// ERC-8183 AgenticCommerce job lifecycle — verified from impl 0xA316fd02827242D537F84730F8a37D0BA5fd351a
	// JobFunded(uint256 indexed jobId, address indexed client, uint256 amount)
	TopicJobFunded = crypto.Keccak256Hash([]byte("JobFunded(uint256,address,uint256)"))
	// JobSubmitted(uint256 indexed jobId, address indexed provider, bytes32 deliverable)
	TopicJobSubmitted = crypto.Keccak256Hash([]byte("JobSubmitted(uint256,address,bytes32)"))
	// JobCompleted(uint256 indexed jobId, address indexed evaluator, bytes32 reason)
	TopicJobCompleted = crypto.Keccak256Hash([]byte("JobCompleted(uint256,address,bytes32)"))
	// JobRejected(uint256 indexed jobId, address indexed rejector, bytes32 reason)
	TopicJobRejected = crypto.Keccak256Hash([]byte("JobRejected(uint256,address,bytes32)"))
	// PaymentReleased(uint256 indexed jobId, address indexed provider, uint256 amount)
	TopicPaymentReleased = crypto.Keccak256Hash([]byte("PaymentReleased(uint256,address,uint256)"))
	// JobExpired(uint256 indexed jobId)
	TopicJobExpired = crypto.Keccak256Hash([]byte("JobExpired(uint256)"))
)

// ── Environment variables ─────────────────────────────────────────────────────

// EnvioAPIToken returns the HyperSync API token from the environment.
// Get one at https://envio.dev
func EnvioAPIToken() string {
	return os.Getenv("ENVIO_API_TOKEN")
}

// NextRPCURL advances the round-robin counter and returns the next public Arc RPC endpoint.
// Called on each indexer restart so errors/rate-limits naturally rotate to the next provider.
func NextRPCURL() string {
	idx := arcRPCIndex.Add(1) - 1
	return arcRPCPool[idx%uint64(len(arcRPCPool))]
}
