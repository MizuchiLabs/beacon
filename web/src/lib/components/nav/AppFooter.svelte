<script lang="ts">
	import { Github, Moon, Sun } from '@lucide/svelte';
	import Button from '$lib/components/ui/button/button.svelte';
	import { mode, toggleMode } from 'mode-watcher';
</script>

<footer class="fixed right-0 bottom-0 left-0 z-50 flex justify-center">
	<div
		class="flex min-h-12 w-full items-center justify-between rounded-t-xl border-x border-t px-4 py-1 text-sm backdrop-blur-md sm:max-w-3xl"
	>
		<p class="flex items-center text-muted-foreground">
			&copy; {new Date().getFullYear()} MizuchiLabs
		</p>

		<div class="flex items-center pl-12">
			<Button
				variant="ghost"
				size="icon"
				href="https://github.com/mizuchilabs/beacon"
				rel="noopener noreferrer"
				target="_blank"
				class="rounded-full hover:text-primary/80"
				aria-label="GitHub"
			>
				<Github size={20} />
			</Button>

			<Button
				variant="ghost"
				size="icon"
				onclick={toggleMode}
				class="rounded-full hover:text-primary/80"
			>
				{#if mode.current === 'light'}
					<Moon size={20} />
				{:else}
					<Sun size={20} />
				{/if}
			</Button>
		</div>
	</div>
</footer>
