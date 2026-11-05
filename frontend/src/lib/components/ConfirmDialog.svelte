<script lang="ts">
	import { app } from '#lib/app.svelte';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
</script>

<AlertDialog.Root bind:open={app.confirm.open}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>{app.confirm.title}</AlertDialog.Title>
			{#if app.confirm.description}
				<AlertDialog.Description class="max-h-60 overflow-y-auto whitespace-pre-line text-left">
					{app.confirm.description}
				</AlertDialog.Description>
			{/if}
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel onclick={() => app.cancelConfirm()}>取消</AlertDialog.Cancel>
			<AlertDialog.Action
				variant={app.confirm.destructive ? 'destructive' : 'default'}
				onclick={() => app.confirmAction()}
			>
				{app.confirm.confirmLabel}
			</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
