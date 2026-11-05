<script lang="ts">
	let { size = 28 }: { size?: number } = $props();
</script>

<svg
	width={size}
	height={size}
	viewBox="0 0 32 32"
	aria-hidden="true"
	class="shrink-0"
>
	<rect x="1" y="1" width="30" height="30" rx="9" class="fill-foreground" />
	<path
		d="M8.4 11.2l2.4 2.4 4.2-4.2M8.4 20.2l2.4 2.4 4.2-4.2"
		fill="none"
		class="stroke-background"
		stroke-width="2.4"
		stroke-linecap="round"
		stroke-linejoin="round"
	/>
	<path
		d="M18 12.2h5.6M18 21.2h5.6"
		fill="none"
		class="stroke-background"
		stroke-width="2.4"
		stroke-linecap="round"
	/>
</svg>
