<script lang="ts">
	import { T } from '@threlte/core';
</script>

<T.Mesh>
	<T.SphereGeometry args={[3, 24, 16]} />
	<T.MeshBasicMaterial wireframe color="#9aa0b4" transparent opacity={0.25} />
</T.Mesh>
