<script lang="ts">
	import { Canvas } from '@threlte/core';
	import SceneCamera from './SceneCamera.svelte';
	import SceneLighting from './SceneLighting.svelte';
	import ArcSphere from './sphere/ArcSphere.svelte';
	import ChainSpine from './chain/ChainSpine.svelte';
</script>

<Canvas>
	<SceneCamera />
	<SceneLighting />
	<ArcSphere />
	<ChainSpine />
</Canvas>
