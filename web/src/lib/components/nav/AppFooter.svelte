<script lang="ts">
	import { DotIcon, Github, Moon, Sun } from '@lucide/svelte';
	import { theme } from '$lib/stores/theme';
	import Button from '$lib/components/ui/button/button.svelte';
</script>

<footer class="flex justify-center">
	<div
		class="flex min-h-12 min-w-3xl items-center justify-between gap-1 rounded-t-xl border border-border/40 bg-background/80 px-5 py-2 shadow-lg backdrop-blur-md"
	>
		<!-- Copyright -->
		<p class="flex items-center text-sm text-muted-foreground">
			&copy; {new Date().getFullYear()} MizuchiLabs
			<DotIcon />
			All rights reserved.
		</p>

		<!-- Social Links -->
		<div class="flex items-center gap-4">
			<a
				href="https://github.com/mizuchilabs"
				target="_blank"
				rel="noopener noreferrer"
				class="text-muted-foreground transition-colors hover:text-foreground"
				aria-label="GitHub"
			>
				<Github size={20} />
			</a>

			<Button variant="ghost" size="icon" onclick={() => theme.toggle()}>
				{#if $theme === 'light'}
					<Moon size={20} />
				{:else}
					<Sun size={20} />
				{/if}
			</Button>
		</div>
	</div>
</footer>
