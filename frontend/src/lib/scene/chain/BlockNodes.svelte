<script lang="ts">
	import { T, useTask } from '@threlte/core';
	import * as THREE from 'three';
	import type { Block } from '$lib/api/chain/types.js';

	interface Props {
		blocks: Block[];
	}

	let { blocks }: Props = $props();

	const MAX = 100;
	const HELIX_RADIUS = 2.5;
	const HELIX_TURNS = 3;
	const X_LEFT = -10;
	const X_SPAN = 90;

	const dummy = new THREE.Object3D();
	const col = new THREE.Color();

	let meshRef = $state<THREE.InstancedMesh | undefined>();
	let linesRef = $state<THREE.LineSegments | undefined>();
	let t = 0;

	/** Heat color with age-based dimming. ageFactor 1=newest, 0=oldest */
	function heatColor(util: number, ageFactor: number): THREE.Color {
		const u = Math.max(0, Math.min((util ?? 0) / 100, 1));
		const hue = 0.583 - u * 0.514;
		const lightness = 0.18 + ageFactor * 0.52;
		return col.setHSL(hue, 0.85, lightness);
	}

	/** Scale: base 0.18, grows with tx_count up to 0.50 */
	function nodeScale(txCount: number): number {
		return 0.18 + Math.min((txCount ?? 0) / 60, 1) * 0.32;
	}

	/** Helix position — runs left to right along X axis, spirals in YZ plane */
	function helixPos(i: number, count: number): [number, number, number] {
		const frac = count > 1 ? i / (count - 1) : 0;
		const x = X_LEFT + frac * X_SPAN;
		const angle = frac * HELIX_TURNS * Math.PI * 2;
		return [x, HELIX_RADIUS * Math.sin(angle), HELIX_RADIUS * Math.cos(angle)];
	}

	$effect(() => {
		if (!meshRef || blocks.length === 0) return;

		const count = Math.min(blocks.length, MAX);
		meshRef.count = count;

		const hasLines = linesRef && count >= 2;
		const linePositions = hasLines ? new Float32Array((count - 1) * 6) : null;
		const lineColors = hasLines ? new Float32Array((count - 1) * 6) : null;

		for (let i = 0; i < count; i++) {
			const b = blocks[i];
			const ageFactor = 1 - i / Math.max(count - 1, 1);
			const [x, y, z] = helixPos(i, count);

			dummy.position.set(x, y, z);
			dummy.scale.setScalar(nodeScale(b.tx_count));
			dummy.updateMatrix();
			meshRef.setMatrixAt(i, dummy.matrix);

			const c = heatColor(b.utilization_pct ?? 0, ageFactor);
			meshRef.setColorAt(i, c);

			// Chain link segment: block i → block i+1
			if (linePositions && lineColors && i < count - 1) {
				const [nx, ny, nz] = helixPos(i + 1, count);
				const idx = i * 6;
				linePositions[idx] = x;
				linePositions[idx + 1] = y;
				linePositions[idx + 2] = z;
				linePositions[idx + 3] = nx;
				linePositions[idx + 4] = ny;
				linePositions[idx + 5] = nz;
				lineColors[idx] = c.r;
				lineColors[idx + 1] = c.g;
				lineColors[idx + 2] = c.b;
				lineColors[idx + 3] = c.r;
				lineColors[idx + 4] = c.g;
				lineColors[idx + 5] = c.b;
			}
		}

		meshRef.instanceMatrix.needsUpdate = true;
		if (meshRef.instanceColor) meshRef.instanceColor.needsUpdate = true;

		// Push chain link geometry to GPU
		if (linesRef && linePositions && lineColors) {
			const geom = linesRef.geometry;
			geom.setAttribute('position', new THREE.BufferAttribute(linePositions, 3));
			geom.setAttribute('color', new THREE.BufferAttribute(lineColors, 3));
			geom.setDrawRange(0, (count - 1) * 2);
			geom.computeBoundingSphere();
		}
	});

	// Pulse the newest block (index 0) — "heartbeat" of the chain
	useTask((delta: number) => {
		if (!meshRef || blocks.length === 0) return;
		t += delta * 2.5;

		const b = blocks[0];
		const count = Math.min(blocks.length, MAX);
		const [x, y, z] = helixPos(0, count);
		const scale = nodeScale(b.tx_count) * (1 + Math.sin(t) * 0.15);

		dummy.position.set(x, y, z);
		dummy.scale.setScalar(scale);
		dummy.updateMatrix();
		meshRef.setMatrixAt(0, dummy.matrix);
		meshRef.instanceMatrix.needsUpdate = true;
	});
</script>

<!-- Chain links connecting consecutive blocks -->
<T.LineSegments bind:ref={linesRef}>
	<T.BufferGeometry />
	<T.LineBasicMaterial vertexColors transparent opacity={0.4} />
</T.LineSegments>

<!-- Block nodes — icosahedrons, lit by scene lights, heat-colored -->
<T.InstancedMesh bind:ref={meshRef} args={[undefined, undefined, MAX]}>
	<T.IcosahedronGeometry args={[1, 1]} />
	<T.MeshStandardMaterial metalness={0.3} roughness={0.5} />
</T.InstancedMesh>
