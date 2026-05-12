<script lang="ts">
	import { Canvas } from '@threlte/core';
	import SceneCamera from './SceneCamera.svelte';
	import SceneLighting from './SceneLighting.svelte';
	import ChainSpine from './chain/ChainSpine.svelte';
	import WalletGraph from './graph/WalletGraph.svelte';
</script>

<Canvas>
	<SceneCamera />
	<SceneLighting />
	<WalletGraph />
	<ChainSpine />
</Canvas>
