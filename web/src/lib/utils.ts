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

export function prettyJson(data: unknown): string {
	const parsed = tryParseJson(data);
	if (parsed === null || parsed === undefined) return '{}';
	return JSON.stringify(parsed, null, 2);
}

export function truncate(str: string, len: number): string {
	if (str.length <= len) return str;
	return str.slice(0, len) + '...';
}
