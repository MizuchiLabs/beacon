import { PersistedState } from 'runed';

export const timeRanges = [
	{ label: '24h', value: '86400', title: 'Last 24 hours' },
	{ label: '7d', value: '604800', title: 'Last 7 days' },
	{ label: '14d', value: '1209600', title: 'Last 14 days' },
	{ label: '30d', value: '2592000', title: 'Last 30 days' }
];

const DEFAULT_RANGE = '86400';
const KNOWN_RANGES: Record<string, true> = Object.fromEntries(
	timeRanges.map((range) => [range.value, true])
);

const stored = new PersistedState('beacon:range', DEFAULT_RANGE);

const isKnownRange = (value: unknown): value is string =>
	typeof value === 'string' && KNOWN_RANGES[value] === true;

// The range is the only knob the whole page has, and the API rejects anything
// outside the known windows, so a stored value that is not one of them falls
// back to the default instead of leaving every request without a window.
export const timeRange = {
	get current() {
		return isKnownRange(stored.current) ? stored.current : DEFAULT_RANGE;
	},
	set current(value: string) {
		if (isKnownRange(value)) stored.current = value;
	}
};
