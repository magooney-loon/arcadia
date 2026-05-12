<script lang="ts">
	import { Canvas } from '@threlte/core';
	import SceneCamera from './SceneCamera.svelte';
	import SceneLighting from './SceneLighting.svelte';
	import ChainSpine from './chain/ChainSpine.svelte';
	import WalletGraph from './graph/WalletGraph.svelte';
	import TokenParticles from './particles/TokenParticles.svelte';
	import ParticleSpawner from './particles/ParticleSpawner.svelte';
</script>

<Canvas>
	<SceneCamera />
	<SceneLighting />
	<ParticleSpawner />
	<WalletGraph />
	<TokenParticles />
	<ChainSpine />
</Canvas>
