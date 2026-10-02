<script lang="ts">
	import Github from '#lib/assets/github.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
</script>

<Button
	href="https://github.com/mizuchilabs/beacon"
	target="_blank"
	rel="noopener noreferrer"
	variant="ghost"
	size="icon-sm"
	class="fixed right-3 bottom-2 z-50"
	aria-label="Beacon on GitHub"
	title="Powered by Beacon"
>
	<Github />
</Button>
