export function formatDate(iso: string | null): string {
	if (!iso) return '--';
	const d = new Date(iso);
	return d.toLocaleDateString('en-US', {
		month: 'short',
		day: 'numeric',
		hour: '2-digit',
		minute: '2-digit',
		hour12: false
	});
}

export function timeAgo(iso: string | null): string {
	if (!iso) return '--';
	const now = Date.now();
	const then = new Date(iso).getTime();
	const diff = now - then;

	if (diff < 60_000) return 'just now';
	if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
	if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
	if (diff < 604_800_000) return `${Math.floor(diff / 86_400_000)}d ago`;
	return formatDate(iso);
}

export function duration(start: string | null, end: string | null): string {
	if (!start) return '--';
	const s = new Date(start).getTime();
	const e = end ? new Date(end).getTime() : Date.now();
	const ms = e - s;

	if (ms < 1000) return `${ms}ms`;
	if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
	return `${Math.floor(ms / 60_000)}m ${Math.floor((ms % 60_000) / 1000)}s`;
}

export function tryParseJson(data: unknown): unknown {
	if (typeof data === 'string') {
		try {
			return JSON.parse(data);
		} catch {
			return data;
		}
	}
	return data;
}

/**
 * Go's encoding/json serialises []byte fields as base64 strings.
 * This helper decodes that base64 layer and then JSON-parses the
 * inner payload, so the UI can render the actual JSON object.
 * Falls back gracefully if the value is already a plain object/string.
 */
export function decodeData(data: unknown): unknown {
	if (data === null || data === undefined) return null;
	if (typeof data === 'string') {
		// Try base64-decode first
		try {
			const decoded = atob(data);
			try {
				return JSON.parse(decoded);
			} catch {
				// decoded bytes aren't JSON — return the raw decoded string
				return decoded;
			}
		} catch {
			// Not valid base64 — fall through to plain JSON parse
		}
		return tryParseJson(data);
	}
	// Already an object (shouldn't normally happen from this API, but be safe)
	return data;
}

export function prettyJson(data: unknown): string {
	const parsed = tryParseJson(data);
	if (parsed === null || parsed === undefined) return '{}';
	return JSON.stringify(parsed, null, 2);
}

/**
 * Like prettyJson but decodes base64-encoded []byte fields first.
 */
export function prettyData(data: unknown): string {
	const decoded = decodeData(data);
	if (decoded === null || decoded === undefined) return '{}';
	if (typeof decoded === 'string') return decoded;
	return JSON.stringify(decoded, null, 2);
}

export function truncate(str: string, len: number): string {
	if (str.length <= len) return str;
	return str.slice(0, len) + '...';
}
