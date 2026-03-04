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
	},
	{
		type: 'if',
		label: 'If (Condition)',
		description:
			'Evaluates a condition and branches execution. True follows "success" edges, false follows "failure" edges.',
		category: 'logic',
		params: [
			{
				key: 'field',
				label: 'Field',
				type: 'string',
				required: true,
				placeholder: 'status'
			},
			{
				key: 'operator',
				label: 'Operator',
				type: 'select',
				required: true,
				default: '==',
				options: [
					{ value: '==', label: 'Equals (==)' },
					{ value: '!=', label: 'Not Equals (!=)' },
					{ value: '>', label: 'Greater Than (>)' },
					{ value: '<', label: 'Less Than (<)' },
					{ value: '>=', label: 'Greater or Equal (>=)' },
					{ value: '<=', label: 'Less or Equal (<=)' },
					{ value: 'contains', label: 'Contains' },
					{ value: 'not_contains', label: 'Not Contains' },
					{ value: 'exists', label: 'Exists' },
					{ value: 'not_exists', label: 'Not Exists' },
					{ value: 'is_empty', label: 'Is Empty' },
					{ value: 'not_empty', label: 'Not Empty' }
				]
			},
			{
				key: 'value',
				label: 'Value',
				type: 'string',
				placeholder: 'Compare against this value'
			}
		]
	},
	{
		type: 'loop',
		label: 'Loop',
		description: 'Iterates over an array field in the input, executing downstream nodes for each item.',
		category: 'logic',
		params: [
			{
				key: 'array_field',
				label: 'Array Field',
				type: 'string',
				required: true,
				placeholder: 'items'
			},
			{
				key: 'mode',
				label: 'Mode',
				type: 'select',
				default: 'collect',
				options: [
					{ value: 'collect', label: 'Collect (array of results)' },
					{ value: 'flatten', label: 'Flatten (merge all results)' }
				]
			}
		]
	},
	{
		type: 'set',
		label: 'Set (Variables)',
		description: 'Sets, merges, appends, or deletes variables in the workflow data context.',
		category: 'logic',
		params: [
			{
				key: 'values',
				label: 'Values',
				type: 'json',
				required: true,
				placeholder: '{"key": "value"}'
			},
			{
				key: 'mode',
				label: 'Mode',
				type: 'select',
				default: 'merge',
				options: [
					{ value: 'merge', label: 'Merge (shallow merge)' },
					{ value: 'replace', label: 'Replace (overwrite all)' },
					{ value: 'append', label: 'Append (to arrays)' },
					{ value: 'delete', label: 'Delete (remove keys)' }
				]
			}
		]
	},
	{
		type: 'merge',
		label: 'Merge',
		description: 'Merges data from multiple upstream branches into a single output.',
		category: 'logic',
		params: [
			{
				key: 'mode',
				label: 'Mode',
				type: 'select',
				default: 'combine',
				options: [
					{ value: 'combine', label: 'Combine (merge all keys)' },
					{ value: 'append', label: 'Append (array of inputs)' },
					{ value: 'pick', label: 'Pick (select specific keys)' },
					{ value: 'wait', label: 'Wait (pass-through first)' }
				]
			},
			{
				key: 'keys',
				label: 'Keys (for pick mode)',
				type: 'json',
				placeholder: '["key1", "key2"]'
			}
		]
	},
	{
		type: 'timer',
		label: 'Timer / Sleep',
		description: 'Pauses execution for a specified duration. Supports context cancellation. Max 1 hour.',
		category: 'flow',
		params: [
			{
				key: 'duration',
				label: 'Duration',
				type: 'string',
				required: true,
				placeholder: '5s, 500ms, 2m30s'
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
	ai: { bg: 'bg-violet-500/5', border: 'border-violet-500/30', text: 'text-violet-400' },
	logic: { bg: 'bg-amber-500/5', border: 'border-amber-500/30', text: 'text-amber-400' },
	flow: { bg: 'bg-emerald-500/5', border: 'border-emerald-500/30', text: 'text-emerald-400' }
} as const;
