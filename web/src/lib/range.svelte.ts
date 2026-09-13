import { PersistedState } from 'runed';

export const timeRanges = [
	{ label: '24h', value: '86400', title: 'Last 24 hours' },
	{ label: '7d', value: '604800', title: 'Last 7 days' },
	{ label: '14d', value: '1209600', title: 'Last 14 days' },
	{ label: '30d', value: '2592000', title: 'Last 30 days' }
];

export const timeRange = new PersistedState('beacon:range', '86400');
