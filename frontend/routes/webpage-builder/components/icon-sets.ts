/**
 * Lazy loaders for the Iconify icon sets used by the TextBlockEditor pickers.
 *
 * Each set's JSON is imported dynamically so Vite splits it into its own async
 * chunk — the ~3 MB `mdi` / `emojione` sets stay out of the eager bundle and are
 * fetched (and browser-cached) only the first time their picker is opened.
 *
 * The Iconify format is `{ prefix, icons: { name: { body, width?, height? } },
 * aliases?: { name: { parent } }, width?, height? }`. We flatten it to a plain
 * `{ name, body }[]` plus the set viewBox dimensions for the picker grid.
 */

export type IconSetId = 'mdi' | 'emojione' | 'flat-color-icons';

export interface PickerIcon {
	name: string;
	body: string;
	/** viewBox string, e.g. "0 0 24 24" — per icon, since one picker can mix sets. */
	vb: string;
}

interface IconifyJSON {
	icons: Record<string, { body: string; width?: number; height?: number }>;
	aliases?: Record<string, { parent: string }>;
	width?: number;
	height?: number;
}

const loaders: Record<IconSetId, () => Promise<{ default: IconifyJSON }>> = {
	mdi: () => import('@iconify-json/mdi/icons.json'),
	emojione: () => import('@iconify-json/emojione/icons.json'),
	'flat-color-icons': () => import('@iconify-json/flat-color-icons/icons.json'),
};

const cache = new Map<IconSetId, Promise<PickerIcon[]>>();

export function loadIconSet(set: IconSetId): Promise<PickerIcon[]> {
	let p = cache.get(set);
	if (!p) {
		p = loaders[set]().then((mod) => {
			const data = mod.default;
			const vb = `0 0 ${data.width ?? 24} ${data.height ?? 24}`;
			const icons: PickerIcon[] = [];
			for (const name in data.icons) icons.push({ name, body: data.icons[name].body, vb });
			// Resolve aliases to their parent's body so the search also finds alias names.
			for (const name in data.aliases ?? {}) {
				const parent = data.aliases![name].parent;
				const body = data.icons[parent]?.body;
				if (body) icons.push({ name, body, vb });
			}
			icons.sort((a, b) => a.name.localeCompare(b.name));
			return icons;
		});
		cache.set(set, p);
	}
	return p;
}

/** Load several sets and concatenate them IN ORDER (no cross-set re-sort), so a combined
 * picker shows e.g. all color icons first, then all material icons, continuously. */
export function loadIconSets(sets: IconSetId[]): Promise<PickerIcon[]> {
	return Promise.all(sets.map(loadIconSet)).then((lists) => lists.flat());
}
