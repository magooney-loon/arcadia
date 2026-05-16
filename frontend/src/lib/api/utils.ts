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

// ── In-flight request tracking ──────────────────────────────────────
// Every `apiFetch` call registers its AbortController here. Calling
// `abortAll()` (e.g. on navigation) cancels them all so the browser
// frees the connection slots for SvelteKit's route chunk download.
interface TrackedController extends AbortController {
	_navAbort?: boolean;
}
const inFlight = new Set<TrackedController>();

/** Cancel every pending REST request and clear the pool. */
export function abortAll() {
	for (const ctrl of inFlight) {
		ctrl._navAbort = true;
		ctrl.abort();
	}
	inFlight.clear();
}

// apiFetch is the single REST entry point for every CRUD client. It:
//   - prepends the configured API base URL
//   - aborts the request after `timeoutMs` so the connection slot is
//     freed (otherwise stalled fetches accumulate and block navigation)
//   - tracks the request so `abortAll()` can cancel it on navigation
//   - normalizes errors: timeouts and HTTP failures become Error
//     instances with a readable message that DataState renders cleanly
export async function apiFetch<T>(
	path: string,
	init: RequestInit = {},
	timeoutMs = DEFAULT_TIMEOUT_MS
): Promise<T> {
	const ctrl = new AbortController() as TrackedController;
	inFlight.add(ctrl);
	const id = setTimeout(() => ctrl.abort(), timeoutMs);
	try {
		const res = await fetch(`${getApiUrl()}${path}`, { ...init, signal: ctrl.signal });
		if (!res.ok) {
			throw new Error(`${path}: HTTP ${res.status}`);
		}
		return (await res.json()) as T;
	} catch (e) {
		if (e instanceof DOMException && e.name === 'AbortError') {
			if (ctrl._navAbort) {
				throw new Error(`request cancelled — navigated away`, { cause: e });
			}
			throw new Error(`request timed out (${timeoutMs / 1000}s) — server is busy`, { cause: e });
		}
		throw e;
	} finally {
		clearTimeout(id);
		inFlight.delete(ctrl);
	}
}
