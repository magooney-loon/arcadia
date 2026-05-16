import { getApiUrl } from '../stores/config.svelte.js';

export function formatTimestamp(timestamp: string): string {
	try {
		return new Date(timestamp).toLocaleString();
	} catch {
		return timestamp;
	}
}

// Default REST timeout. Long enough for slow snapshot/wallet queries
// during a heavy indexer sync (SQLite writer holds the lock briefly),
// short enough that a stalled request doesn't starve the browser's
// per-host connection pool — which is what makes the app feel frozen
// (it also blocks SvelteKit's lazy-loaded route chunks).
const DEFAULT_TIMEOUT_MS = 15_000;

// apiFetch is the single REST entry point for every CRUD client. It:
//   - prepends the configured API base URL
//   - aborts the request after `timeoutMs` so the connection slot is
//     freed (otherwise stalled fetches accumulate and block navigation)
//   - normalizes errors: timeouts and HTTP failures become Error
//     instances with a readable message that DataState renders cleanly
export async function apiFetch<T>(
	path: string,
	init: RequestInit = {},
	timeoutMs = DEFAULT_TIMEOUT_MS
): Promise<T> {
	const ctrl = new AbortController();
	const id = setTimeout(() => ctrl.abort(), timeoutMs);
	try {
		const res = await fetch(`${getApiUrl()}${path}`, { ...init, signal: ctrl.signal });
		if (!res.ok) {
			throw new Error(`${path}: HTTP ${res.status}`);
		}
		return (await res.json()) as T;
	} catch (e) {
		if (e instanceof DOMException && e.name === 'AbortError') {
			throw new Error(`request timed out (${timeoutMs / 1000}s) — server is busy`);
		}
		throw e;
	} finally {
		clearTimeout(id);
	}
}
