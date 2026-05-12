<script lang="ts">
	import { T, useTask } from '@threlte/core';
	import { Text } from '@threlte/extras';
	import * as THREE from 'three';
	import SphereGrid from './SphereGrid.svelte';

	// Flat hexagon outline — 6 sided cylinder, nearly zero height, wireframe
	// rotation.x = π/2 lays it flat facing the camera on load
	const HEX_RADIUS = 0.72;
	const HEX_SIDES = 6;

	// Subtle pulse on the hex: scale oscillates between 0.97 and 1.03
	let hexRef = $state<THREE.Mesh | undefined>();
	let t = 0;

	useTask((delta: number) => {
		if (!hexRef) return;
		t += delta * 0.8;
		const s = 1 + Math.sin(t) * 0.03;
		hexRef.scale.set(s, s, 1);
	});
</script>

<SphereGrid />

<!-- Hexagon core (flat hex disk, wireframe green) -->
<T.Mesh bind:ref={hexRef} rotation.x={Math.PI / 2}>
	<T.CylinderGeometry args={[HEX_RADIUS, HEX_RADIUS, 0.005, HEX_SIDES]} />
	<T.MeshBasicMaterial color="#7ee5a8" wireframe />
</T.Mesh>

<!-- Logotype -->
<Text
	text="ARCADIA"
	fontSize={0.52}
	color="#e6e6ee"
	letterSpacing={0.38}
	anchorX="center"
	anchorY="middle"
	position={[0, -4.2, 0]}
/>
