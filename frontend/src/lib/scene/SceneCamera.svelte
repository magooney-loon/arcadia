<script lang="ts">
	import { T } from '@threlte/core';
	import { OrbitControls } from '@threlte/extras';
</script>

<T.PerspectiveCamera makeDefault fov={60} position={[0, 1.5, 10]}>
	<OrbitControls
		autoRotate
		autoRotateSpeed={0.4}
		enableDamping
		dampingFactor={0.05}
		enablePan={false}
		minDistance={5}
		maxDistance={20}
	/>
</T.PerspectiveCamera>
