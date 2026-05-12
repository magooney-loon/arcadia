<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children()}

<style>
	:global(*) {
		margin: 0;
		padding: 0;
		box-sizing: border-box;
	}
	:global(html, body) {
		width: 100%;
		height: 100%;
		overflow: hidden;
		background: #0a0a0f;
	}
</style>
