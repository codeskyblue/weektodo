<script lang="ts">
	import { app } from '#lib/app.svelte';
	import type { TaskRow as Row } from '#lib/types';
	import TaskRow from './TaskRow.svelte';

	let { rows }: { rows: Row[] } = $props();
</script>

<div class="space-y-0.5 max-sm:space-y-0" role="list">
	{#each rows as row (row.task.id)}
		<TaskRow {row} focusRequest={app.focusRequest} />
	{/each}
</div>
