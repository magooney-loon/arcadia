<script lang="ts">
	import { T } from '@threlte/core';
	import { Text3DGeometry } from '@threlte/extras';
	import * as THREE from 'three';

	let textMeshRef = $state<THREE.Mesh | undefined>();

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
