import { GraphCrudClient } from '../api/graph/crud.js';
import type { EdgesResponse, EdgeFilter } from '../api/graph/types.js';

const client = new GraphCrudClient();

export interface GraphState {
	data: EdgesResponse | null;
	loading: boolean;
	error: string | null;
}

export const graph = $state<GraphState>({ data: null, loading: false, error: null });

export async function fetchEdges(filter: EdgeFilter = {}) {
	graph.loading = true;
	graph.error = null;
	try {
		graph.data = await client.edges(filter);
	} catch (e) {
		if (!String(e).includes('cancelled')) graph.error = String(e);
	} finally {
		graph.loading = false;
	}
}
