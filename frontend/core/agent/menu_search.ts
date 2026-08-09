// Quick menu lookup for the agent search bar. While the user types 1-3 words,
// the chat panel shows matching menu entries so navigating never requires a
// full agent turn.
//
// Matching runs on normalizeStringN output on both sides, so accents and casing
// are irrelevant ("gestion" finds "Gestión"). That normalizer collapses spaces
// into "_", which is why the search text is split on "_" rather than " ".

import { normalizeStringN, wordInclude } from '@genix/ui/utilities';
import { security } from '$libs/ui-runtime.svelte';
import { Core } from '$core/store.svelte';
import type { IMenuRecord } from '$core/types/modules';

export interface MenuSearchOption {
  name: string; // bilingual "English|Spanish", resolved with tr() at render time
  route: string;
  icon: string;
  nameNorm: string; // normalized haystack, cached here to keep matching allocation-free
}

const MAX_RESULTS = 6;
const MAX_WORDS = 3;
const FALLBACK_ICON = 'icon-[fa--bolt]';

// The index is rebuilt only when the menu array identity changes (module switch
// or login), so the per-option normalization happens once, not per keystroke.
let indexedMenus: IMenuRecord[] | undefined;
let indexedOptions: MenuSearchOption[] = [];

const buildIndex = (menus: IMenuRecord[]): MenuSearchOption[] => {
  const options: MenuSearchOption[] = [];
  for (const menu of menus) {
    for (const option of menu.options || []) {
      const route = String(option.route || '').trim();
      // Same access filter as the side menu: never surface an unreachable page.
      if (!route || !security.canAccessRoute(route)) { continue; }
      options.push({
        name: option.name,
        route,
        icon: option.icon || FALLBACK_ICON,
        // Both language halves are indexed, so "sales" finds "Punto de Venta"
        // too. The pipe becomes a space first; normalizeStringN would otherwise
        // drop it and weld the two halves into one bogus word.
        nameNorm: normalizeStringN(String(option.name || '').replace('|', ' ')),
      });
    }
  }
  return options;
};

// searchWords returns the normalized words used for matching, or [] when the
// text is not a short keyword query (empty, or longer than MAX_WORDS words —
// at that point the user is writing a sentence for the agent, not searching).
export const searchWords = (text: string): string[] => {
  const words = normalizeStringN(text).split('_').filter((word) => word.length > 1);
  return words.length > 0 && words.length <= MAX_WORDS ? words : [];
};

export const searchMenuOptions = (text: string): MenuSearchOption[] => {
  const words = searchWords(text);
  if (words.length === 0) { return []; }

  const menus = Core.module?.menus || [];
  if (menus !== indexedMenus) {
    indexedMenus = menus;
    indexedOptions = buildIndex(menus);
  }

  const results: MenuSearchOption[] = [];
  for (const option of indexedOptions) {
    if (!wordInclude(option.nameNorm, words)) { continue; }
    results.push(option);
    if (results.length === MAX_RESULTS) { break; }
  }
  return results;
};

// Words for highlString(), which matches lowercase words against the rendered
// (accented) name. Unaccented input still matches the row, it just renders
// without the highlight.
export const highlightWords = (text: string): string[] =>
  text.toLowerCase().split(' ').filter((word) => word.length > 1);
