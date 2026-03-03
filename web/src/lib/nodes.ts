import type { NodeTypeMeta } from './types';

export const NODE_TYPES: NodeTypeMeta[] = [
	{
		type: 'echo',
		label: 'Echo',
		description: 'Passes input through unchanged. Useful for debugging.',
		category: 'core',
		params: []
	},
	{
		type: 'math',
		label: 'Math',
		description: 'Adds two numbers (a + b) and returns the result.',
		category: 'core',
		params: [
			{ key: 'a', label: 'A', type: 'number', placeholder: 'From input or set default' },
			{ key: 'b', label: 'B', type: 'number', placeholder: 'From input or set default' }
		]
	},
	{
		type: 'http',
		label: 'HTTP Request',
		description: 'Makes an outbound HTTP request. Supports {{input.field}} template substitution.',
		category: 'integration',
		params: [
			{
				key: 'url',
				label: 'URL',
				type: 'string',
				required: true,
				placeholder: 'https://api.example.com/{{input.id}}'
			},
			{
				key: 'method',
				label: 'Method',
				type: 'select',
				default: 'GET',
				options: [
					{ value: 'GET', label: 'GET' },
					{ value: 'POST', label: 'POST' },
					{ value: 'PUT', label: 'PUT' },
					{ value: 'PATCH', label: 'PATCH' },
					{ value: 'DELETE', label: 'DELETE' }
				]
			},
			{
				key: 'body',
				label: 'Request Body',
				type: 'json',
				placeholder: '{"key": "{{input.value}}"}'
			},
			{
				key: 'timeout_seconds',
				label: 'Timeout (s)',
				type: 'number',
				default: 30,
				placeholder: '30'
			}
		]
	},
	{
		type: 'llm',
		label: 'LLM (OpenAI)',
		description: 'Calls OpenAI Chat Completions API with a prompt template.',
		category: 'ai',
		params: [
			{
				key: 'prompt',
				label: 'Prompt',
				type: 'string',
				required: true,
				placeholder: 'Summarize: {{input.text}}'
			},
			{
				key: 'model',
				label: 'Model',
				type: 'select',
				default: 'gpt-4o-mini',
				options: [
					{ value: 'gpt-4o-mini', label: 'GPT-4o Mini' },
					{ value: 'gpt-4o', label: 'GPT-4o' },
					{ value: 'gpt-4.1-mini', label: 'GPT-4.1 Mini' },
					{ value: 'gpt-4.1', label: 'GPT-4.1' }
				]
			},
			{
				key: 'system_prompt',
				label: 'System Prompt',
				type: 'string',
				placeholder: 'You are a helpful assistant.'
			},
			{
				key: 'max_tokens',
				label: 'Max Tokens',
				type: 'number',
				default: 1024,
				placeholder: '1024'
			},
			{
				key: 'temperature',
				label: 'Temperature',
				type: 'number',
				default: 1.0,
				placeholder: '0.0 - 2.0'
			}
		]
	}
];

export function getNodeMeta(type: string): NodeTypeMeta | undefined {
	return NODE_TYPES.find((n) => n.type === type);
}

// Category colors for the UI
export const CATEGORY_COLORS = {
	core: { bg: 'bg-white/5', border: 'border-white/20', text: 'text-white/70' },
	integration: { bg: 'bg-cyan-500/5', border: 'border-cyan-500/30', text: 'text-cyan-400' },
	ai: { bg: 'bg-violet-500/5', border: 'border-violet-500/30', text: 'text-violet-400' }
} as const;
