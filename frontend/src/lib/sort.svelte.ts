/**
 * Lightweight client-side table sorting utility.
 *
 * Usage in a Svelte 5 component:
 *
 *   import { createSort } from '$lib/sort';
 *
 *   const sort = createSort('block_number', 'desc');
 *   const sorted = $derived(sort.apply(data, {
 *     block_number: (row) => row.block_number,
 *     tx_count: (row) => row.tx_count ?? 0,
 *   }));
 */

export type SortDir = 'asc' | 'desc';

export class TableSort {
	key = $state<string>('');
	dir = $state<SortDir>('asc');

	constructor(defaultKey: string, defaultDir: SortDir = 'asc') {
		this.key = defaultKey;
		this.dir = defaultDir;
	}

	/** Toggle sort on the given column key. */
	toggle(key: string) {
		if (this.key === key) {
			this.dir = this.dir === 'asc' ? 'desc' : 'asc';
		} else {
			this.key = key;
			this.dir = 'asc';
		}
	}

	/** Returns 'asc' | 'desc' | '' for a given key (used for indicator). */
	indicator(key: string): SortDir | '' {
		return this.key === key ? this.dir : '';
	}

	/**
	 * Sort a copy of `rows` in-place using the current key & direction.
	 * `accessors` maps column keys to value-extractor functions.
	 */
	apply<T>(rows: T[], accessors: Record<string, (row: T) => unknown>): T[] {
		if (!this.key || !(this.key in accessors)) return rows;
		const copy = rows.slice();
		const fn = accessors[this.key];
		const mult = this.dir === 'asc' ? 1 : -1;
		copy.sort((a, b) => {
			const va = fn(a);
			const vb = fn(b);
			if (va == null && vb == null) return 0;
			if (va == null) return 1;
			if (vb == null) return -1;
			if (typeof va === 'number' && typeof vb === 'number') return (va - vb) * mult;
			return String(va).localeCompare(String(vb)) * mult;
		});
		return copy;
	}
}

/** Convenience factory so you don't need `new`. */
export function createSort(defaultKey: string = '', defaultDir: SortDir = 'asc') {
	return new TableSort(defaultKey, defaultDir);
}
