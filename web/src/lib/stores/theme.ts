import { browser } from '$app/environment';
import { writable } from 'svelte/store';

type Theme = 'light' | 'dark';

const defaultTheme: Theme = 'light';

function createThemeStore() {
	const { subscribe, set, update } = writable<Theme>(defaultTheme);

	if (browser) {
		const stored = localStorage.getItem('theme') as Theme;
		if (stored) {
			set(stored);
		} else {
			const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
			set(prefersDark ? 'dark' : 'light');
		}
	}

	return {
		subscribe,
		toggle: () => update((theme) => {
			const newTheme = theme === 'light' ? 'dark' : 'light';
			if (browser) {
				localStorage.setItem('theme', newTheme);
				document.documentElement.classList.toggle('dark', newTheme === 'dark');
			}
			return newTheme;
		}),
		set: (theme: Theme) => {
			set(theme);
			if (browser) {
				localStorage.setItem('theme', theme);
				document.documentElement.classList.toggle('dark', theme === 'dark');
			}
		}
	};
}

export const theme = createThemeStore();