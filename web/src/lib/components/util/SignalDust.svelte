<script lang="ts">
	function mulberry32(seed: number) {
		return () => {
			seed |= 0;
			seed = (seed + 0x6d2b79f5) | 0;
			let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
			t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
			return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
		};
	}

	function scatter(seed: number, count: number) {
		const rand = mulberry32(seed);
		return Array.from({ length: count }, () => ({
			x: Math.round(rand() * 14400) / 10,
			y: Math.round(rand() * 9000) / 10,
			r: Math.round((0.3 + rand() * 1.2) * 100) / 100,
			o: Math.round((0.06 + rand() * 0.12) * 100) / 100
		}));
	}

	const layerA = scatter(11, 150);
	const layerB = scatter(29, 80);
</script>

<div
	class="pointer-events-none fixed inset-0 -z-10 overflow-hidden text-foreground"
	aria-hidden="true"
>
	<svg
		class="dust dust-a absolute inset-[-50%] h-[200%] w-[200%]"
		viewBox="0 0 1440 900"
		preserveAspectRatio="xMidYMid slice"
	>
		{#each layerA as d (d.x + '-' + d.y)}
			<circle cx={d.x} cy={d.y} r={d.r} fill="currentColor" fill-opacity={d.o} />
		{/each}
	</svg>
	<svg
		class="dust dust-b absolute inset-[-50%] h-[200%] w-[200%]"
		viewBox="0 0 1440 900"
		preserveAspectRatio="xMidYMid slice"
	>
		{#each layerB as d (d.x + '-' + d.y)}
			<circle cx={d.x} cy={d.y} r={d.r} fill="currentColor" fill-opacity={d.o} />
		{/each}
	</svg>
</div>

<style>
	.dust-a {
		animation: drift-a 90s ease-in-out infinite alternate;
	}

	.dust-b {
		animation: drift-b 120s ease-in-out infinite alternate;
	}

	@keyframes drift-a {
		to {
			transform: translate3d(120px, 70px, 0);
		}
	}

	@keyframes drift-b {
		to {
			transform: translate3d(-90px, 110px, 0);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.dust-a,
		.dust-b {
			animation: none;
		}
	}
</style>
