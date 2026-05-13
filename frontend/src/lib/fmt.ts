export const EXPLORER = 'https://testnet.arcscan.app';

export function explorerAddr(a?: string | null): string {
	return `${EXPLORER}/address/${a ?? ''}`;
}

export function explorerTx(h?: string | null): string {
	return `${EXPLORER}/tx/${h ?? ''}`;
}

export function explorerBlock(n: number): string {
	return `${EXPLORER}/block/${n}`;
}

export const DOMAIN_NAMES: Record<number, string> = {
	0: 'Ethereum',
	1: 'Avalanche',
	2: 'Optimism',
	6: 'Polygon',
	10: 'Arbitrum',
	12: 'Solana',
	23: 'Base',
	26: 'Arc'
};

export const SIGHASH: Record<string, string> = {
	'0xa9059cbb': 'transfer',
	'0x095ea7b3': 'approve',
	'0x23b872dd': 'transferFrom',
	'0x38ed1739': 'swap',
	'0x7ff36ab5': 'swapETH',
	'0x3593564c': 'execute',
	'0x5ae401dc': 'multicall'
};

export function addr(a?: string | null): string {
	if (!a) return '—';
	return a.slice(0, 6) + '…' + a.slice(-4);
}

export function hash(h?: string | null): string {
	if (!h) return '—';
	return h.slice(0, 10) + '…' + h.slice(-6);
}

export function usdc(s?: string | number | null, decimals = 2): string {
	if (s == null || s === '') return '—';
	const n = typeof s === 'number' ? s : parseFloat(s as string);
	if (isNaN(n)) return '—';
	if (Math.abs(n) >= 1_000_000) return '$' + (n / 1_000_000).toFixed(2) + 'M';
	if (Math.abs(n) >= 1_000) return '$' + (n / 1_000).toFixed(1) + 'K';
	return '$' + n.toFixed(decimals);
}

export function num(n?: number | null): string {
	if (n == null) return '—';
	if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
	if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
	return n.toLocaleString('en-US');
}

export function pct(n?: number | null): string {
	if (n == null) return '—';
	return n.toFixed(1) + '%';
}

export function tps(n?: number | null): string {
	if (n == null) return '—';
	return n.toFixed(1);
}

export function ms(n?: number | null): string {
	if (n == null) return '—';
	return Math.round(n) + 'ms';
}

// Age from unix timestamp (seconds)
export function tsAge(timestamp?: number | null): string {
	if (!timestamp) return '—';
	const s = Math.round((Date.now() - timestamp * 1000) / 1000);
	if (s <= 1) return 'now';
	if (s < 60) return `${s}s`;
	if (s < 3600) return `${Math.floor(s / 60)}m`;
	if (s < 86400) return `${Math.floor(s / 3600)}h`;
	return `${Math.floor(s / 86400)}d`;
}

// Age from block number (approximate, Arc L1 ~380ms/block)
export function blockAge(
	blockNum?: number | null,
	latestBlock?: number | null,
	avgMs = 380
): string {
	if (blockNum == null) return '—';
	if (!latestBlock) return `#${blockNum}`;
	const s = Math.round(((latestBlock - blockNum) * avgMs) / 1000);
	if (s <= 1) return 'now';
	if (s < 60) return `${s}s`;
	if (s < 3600) return `${Math.floor(s / 60)}m`;
	if (s < 86400) return `${Math.floor(s / 3600)}h`;
	return `${Math.floor(s / 86400)}d`;
}

export function domainName(id?: number | null): string {
	if (id == null) return '?';
	return DOMAIN_NAMES[id] ?? `chain-${id}`;
}

export function methodName(sig?: string | null): string {
	if (!sig) return '—';
	return SIGHASH[sig] ?? sig.slice(0, 10);
}

export function fxBadge(status: string): string {
	const m: Record<string, string> = {
		created: 'muted',
		taker_funded: 'warn',
		maker_funded: 'warn',
		settled: 'ok',
		cancelled: 'err'
	};
	return m[status] ?? 'muted';
}

export function jobBadge(status: string): string {
	const m: Record<string, string> = {
		created: 'muted',
		accepted: 'info',
		delivered: 'warn',
		settled: 'ok',
		disputed: 'err'
	};
	return m[status] ?? 'muted';
}
