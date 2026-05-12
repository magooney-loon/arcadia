package main

import (
	"os"
	"sync/atomic"

	"github.com/enviodev/hypersync-client-go/utils"
	"github.com/ethereum/go-ethereum/common"
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
	ArcRPCPrimary    = "https://rpc.testnet.arc.network"
	ArcRPCBlockdaemon = "https://rpc.blockdaemon.testnet.arc.network"
	ArcRPCDRPC       = "https://rpc.drpc.testnet.arc.network"
	ArcRPCQuickNode  = "https://rpc.quicknode.testnet.arc.network"

	// WebSocket endpoints
	ArcWSS           = "wss://rpc.testnet.arc.network"
	ArcWSSDrpc       = "wss://rpc.drpc.testnet.arc.network"
	ArcWSSQuickNode  = "wss://rpc.quicknode.testnet.arc.network"

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
	AddrPermit2       = common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3")
	AddrMulticall3    = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
	AddrCreate2Factory = common.HexToAddress("0x4e59b44847b379578588920cA78FbF26c0B4956C")

	// ERC-8004 Agent Registry — TBD: confirm from Arc docs/Discord
	AddrAgentRegistry = common.HexToAddress("0x0000000000000000000000000000000000000000")

	// ERC-8183 Job Escrow — TBD: confirm from Arc docs/Discord
	AddrJobEscrow = common.HexToAddress("0x0000000000000000000000000000000000000000")
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

	// CCTP v2 — TokenMessengerV2
	// DepositForBurn(uint64,address,uint256,address,bytes32,uint32,bytes32,bytes32,uint256,uint32,bytes)
	TopicDepositForBurn = common.HexToHash("0x2fa9ca894982930190727e75500a97d8dc500233a5065e0f3126c48fbe0343c0")

	// CCTP v2 — MessageTransmitterV2
	// MessageReceived(address,uint32,uint64,bytes32,bytes)
	TopicMessageReceived = common.HexToHash("0x58200b4c34ae05ee816d710053fff3ad1bcea173d0113462f6fd5162ab9adca5")

	// ERC-8004 — TBD: verify against deployed ABI
	TopicAgentRegistered = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")

	// ERC-8183 — TBD: verify against deployed ABI
	TopicJobCreated   = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	TopicJobDelivered = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	TopicJobSettled   = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
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
