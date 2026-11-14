<script lang="ts">
	import Github from '#lib/assets/github.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
</script>

<!-- In the flow of the page. A fixed one covers content on a phone. -->
<footer class="z-10 flex justify-end px-3 pb-2">
	<Button
		href="https://github.com/mizuchilabs/beacon"
		target="_blank"
		rel="noopener noreferrer"
		variant="ghost"
		size="icon-sm"
		aria-label="Beacon on GitHub"
		title="Powered by Beacon"
	>
		<Github />
	</Button>
</footer>
