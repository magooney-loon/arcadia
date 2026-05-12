<script lang="ts">
	import { T } from '@threlte/core';
	import * as THREE from 'three';
	import type { Block } from '$lib/api/chain/types.js';

	interface Props {
		blocks: Block[];
	}

	let { blocks }: Props = $props();

	const MAX = 100;
	const dummy = new THREE.Object3D();
	const col = new THREE.Color();

	let meshRef = $state<THREE.InstancedMesh | undefined>();

	// Utilization 0% → cool blue (hue 0.583) — 100% → hot orange (hue 0.069)
	function heatColor(util: number): THREE.Color {
		const t = Math.max(0, Math.min((util ?? 0) / 100, 1));
		const hue = 0.583 - t * 0.514;
		return col.setHSL(hue, 0.9, 0.65);
	}

	// Scale: base 0.07, grows with tx_count up to 0.17
	function nodeScale(txCount: number): number {
		return 0.07 + Math.min((txCount ?? 0) / 80, 1) * 0.1;
	}

	$effect(() => {
		if (!meshRef || blocks.length === 0) return;

		const count = Math.min(blocks.length, MAX);
		meshRef.count = count;

		for (let i = 0; i < count; i++) {
			const b = blocks[i];
			// Most recent (i=0) at z=+3 (front of sphere), oldest at z=-3 (back)
			const z = 3 - (i / Math.max(count - 1, 1)) * 6;
			dummy.position.set(0, 0, z);
			dummy.scale.setScalar(nodeScale(b.tx_count));
			dummy.updateMatrix();
			meshRef.setMatrixAt(i, dummy.matrix);
			meshRef.setColorAt(i, heatColor(b.utilization_pct ?? 0));
		}

		meshRef.instanceMatrix.needsUpdate = true;
		if (meshRef.instanceColor) meshRef.instanceColor.needsUpdate = true;
	});
</script>

<!-- Block nodes — InstancedMesh, one sphere per block, colored by utilization -->
<T.InstancedMesh bind:ref={meshRef} args={[undefined, undefined, MAX]}>
	<T.SphereGeometry args={[1, 8, 6]} />
	<T.MeshBasicMaterial />
</T.InstancedMesh>
