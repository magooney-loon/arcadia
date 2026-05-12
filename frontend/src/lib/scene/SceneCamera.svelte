<script lang="ts">
	import { T } from '@threlte/core';
	import { OrbitControls } from '@threlte/extras';
</script>

<T.PerspectiveCamera makeDefault fov={60} position={[0, 6, -120]}>
	<OrbitControls
		enableDamping
		dampingFactor={0.05}
		enablePan
		panSpeed={0.8}
		minDistance={3}
		maxDistance={120}
	/>
</T.PerspectiveCamera>
