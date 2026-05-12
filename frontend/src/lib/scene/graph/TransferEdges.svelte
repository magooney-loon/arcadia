<script lang="ts">
	import { T } from '@threlte/core';
	import * as THREE from 'three';
	import { getNodePositions, getOrderedEdges } from '$lib/scene-state/layout.svelte';

	let linesRef = $state<THREE.LineSegments | undefined>();

	$effect(() => {
		if (!linesRef) return;

		const edges = getOrderedEdges();
		const positions = getNodePositions();
		if (edges.length === 0) return;

		// Find max USDC for normalising brightness
		let maxUsdc = 0;
		for (const edge of edges) {
			if (edge.total_usdc > maxUsdc) maxUsdc = edge.total_usdc;
		}
		if (maxUsdc === 0) maxUsdc = 1;

		const linePositions = new Float32Array(edges.length * 6);
		const lineColors = new Float32Array(edges.length * 6);
		const edgeColor = new THREE.Color();

		let validCount = 0;

		for (let i = 0; i < edges.length; i++) {
			const edge = edges[i];
			const from = positions[edge.from];
			const to = positions[edge.to];
			if (!from || !to) continue;

			const idx = validCount * 6;
			linePositions[idx] = from.x;
			linePositions[idx + 1] = from.y;
			linePositions[idx + 2] = from.z;
			linePositions[idx + 3] = to.x;
			linePositions[idx + 4] = to.y;
			linePositions[idx + 5] = to.z;

			// Brightness proportional to USDC volume: dim edges → bright edges
			const brightness = 0.15 + (edge.total_usdc / maxUsdc) * 0.35;
			// Base hue #9aa0b4 ≈ HSL(224°, 15%, 66%) — vary lightness
			edgeColor.setHSL(0.62, 0.15, brightness);

			lineColors[idx] = edgeColor.r;
			lineColors[idx + 1] = edgeColor.g;
			lineColors[idx + 2] = edgeColor.b;
			lineColors[idx + 3] = edgeColor.r;
			lineColors[idx + 4] = edgeColor.g;
			lineColors[idx + 5] = edgeColor.b;

			validCount++;
		}

		const geom = linesRef.geometry;
		geom.setAttribute('position', new THREE.BufferAttribute(linePositions, 3));
		geom.setAttribute('color', new THREE.BufferAttribute(lineColors, 3));
		geom.setDrawRange(0, validCount * 2);
		geom.computeBoundingSphere();
	});
</script>

<T.LineSegments bind:ref={linesRef}>
	<T.BufferGeometry />
	<T.LineBasicMaterial vertexColors transparent opacity={0.35} />
</T.LineSegments>
