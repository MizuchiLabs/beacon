<script lang="ts">
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { CheckCircle2, XCircle, Clock, Activity } from '@lucide/svelte';
	import type { MonitorStatus } from '$lib/api/queries';

	interface Props {
		monitor: MonitorStatus;
	}

	let { monitor }: Props = $props();

	const isUp = $derived(monitor.last_status === 'up');
	const statusColor = $derived(isUp ? 'bg-green-500' : 'bg-red-500');
	const statusText = $derived(isUp ? 'Operational' : 'Down');
	const StatusIcon = $derived(isUp ? CheckCircle2 : XCircle);

	function formatResponseTime(ms: number | null): string {
		if (ms === null) return 'N/A';
		if (ms < 1000) return `${ms}ms`;
		return `${(ms / 1000).toFixed(2)}s`;
	}

	function formatDate(date: string | null): string {
		if (!date) return 'Never';
		return new Date(date).toLocaleString();
	}
</script>

<Card class="transition-all hover:shadow-lg">
	<CardHeader class="pb-3">
		<div class="flex items-start justify-between">
			<div class="flex-1">
				<CardTitle class="text-lg">{monitor.name}</CardTitle>
				<p class="mt-1 text-sm text-muted-foreground">{monitor.url}</p>
			</div>
			<div class="flex items-center gap-2">
				<div class={`h-3 w-3 rounded-full ${statusColor} animate-pulse`}></div>
				<Badge variant={isUp ? 'default' : 'destructive'} class="gap-1">
					<StatusIcon class="h-3 w-3" />
					{statusText}
				</Badge>
			</div>
		</div>
	</CardHeader>
	<CardContent>
		<div class="grid grid-cols-2 gap-4 md:grid-cols-4">
			<div class="flex flex-col gap-1">
				<div class="flex items-center gap-1 text-muted-foreground">
					<Clock class="h-4 w-4" />
					<span class="text-xs">Response Time</span>
				</div>
				<p class="text-lg font-semibold">
					{formatResponseTime(monitor.last_response_time)}
				</p>
			</div>
			<div class="flex flex-col gap-1">
				<div class="flex items-center gap-1 text-muted-foreground">
					<Activity class="h-4 w-4" />
					<span class="text-xs">Check Interval</span>
				</div>
				<p class="text-lg font-semibold">{monitor.check_interval}s</p>
			</div>
			<div class="col-span-2 flex flex-col gap-1">
				<span class="text-xs text-muted-foreground">Last Checked</span>
				<p class="text-sm font-medium">{formatDate(monitor.last_check_at)}</p>
			</div>
		</div>
	</CardContent>
</Card>
