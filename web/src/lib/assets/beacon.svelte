<script lang="ts">
	let { ...restProps } = $props();
</script>

<!-- The dot takes the text color, so a status class on the svg lights it. -->
<svg xmlns="http://www.w3.org/2000/svg" viewBox="4 4 56 56" fill="#8a5cf0" {...restProps}>
	<rect x="4" y="46" width="8" height="12" rx="4" fill-opacity="0.45" />
	<rect x="16" y="34" width="8" height="24" rx="4" fill-opacity="0.72" />
	<rect x="28" y="24" width="8" height="34" rx="4" />
	<rect x="40" y="34" width="8" height="24" rx="4" fill-opacity="0.72" />
	<rect x="52" y="46" width="8" height="12" rx="4" fill-opacity="0.45" />
	<circle cx="32" cy="12" r="6" fill="currentColor" />
</svg>
