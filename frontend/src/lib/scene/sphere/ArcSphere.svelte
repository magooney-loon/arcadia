<script lang="ts">
	import { T, useTask } from '@threlte/core';
	import { Text3DGeometry } from '@threlte/extras';
	import * as THREE from 'three';
	import SphereGrid from './SphereGrid.svelte';

	const HEX_RADIUS = 0.72;
	const HEX_SIDES = 6;

	let hexRef = $state<THREE.Mesh | undefined>();
	let textMeshRef = $state<THREE.Mesh | undefined>();
	let t = 0;

	useTask((delta: number) => {
		if (!hexRef) return;
		t += delta * 0.8;
		const s = 1 + Math.sin(t) * 0.03;
		hexRef.scale.set(s, s, 1);
	});

	// Called by Text3DGeometry once the font loads and geometry is built.
	// Directly mutates the Three.js mesh position — not $state, so runes-safe.
	function centerText() {
		if (!textMeshRef?.geometry) return;
		textMeshRef.geometry.computeBoundingBox();
		const bb = textMeshRef.geometry.boundingBox;
		if (!bb) return;
		textMeshRef.position.x = -(bb.max.x + bb.min.x) / 2;
	}
</script>

<SphereGrid />

<!-- Hexagon core (flat hex disk, wireframe green) -->
<T.Mesh bind:ref={hexRef} rotation.x={Math.PI / 2}>
	<T.CylinderGeometry args={[HEX_RADIUS, HEX_RADIUS, 0.005, HEX_SIDES]} />
	<T.MeshBasicMaterial color="#7ee5a8" wireframe />
</T.Mesh>

<!-- 3D logotype — font loads async from CDN, centered once geometry is ready -->
<T.Mesh bind:ref={textMeshRef} position={[0, -4.2, 0]}>
	<Text3DGeometry
		text="ARCADIA"
		size={0.5}
		depth={0.1}
		bevelEnabled
		bevelThickness={0.025}
		bevelSize={0.018}
		bevelSegments={5}
		smooth={0.05}
		oncreate={centerText}
	/>
	<T.MeshStandardMaterial color="#e0e0ee" metalness={0.55} roughness={0.25} />
</T.Mesh>
