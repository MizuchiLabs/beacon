import { plugin as shadcn } from '@shadcn/lint';
import tsParser from '@typescript-eslint/parser';
import { defineConfig } from 'eslint/config';
import svelteParser from 'svelte-eslint-parser';

export default defineConfig([
	{
		files: ['**/*.svelte'],
		languageOptions: {
			parser: svelteParser,
			parserOptions: { parser: tsParser }
		},
		plugins: { shadcn },
		rules: {
			'shadcn/no-restyle': ['error', { allow: ['layout'] }]
		}
	},
	{
		ignores: ['src/lib/components/ui/**']
	}
]);
