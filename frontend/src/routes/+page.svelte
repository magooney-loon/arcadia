<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { browser } from '$app/environment';

	if (browser) {
		goto(resolve('/overview/'));
	}
</script>
