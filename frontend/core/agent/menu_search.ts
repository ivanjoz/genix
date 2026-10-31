// Quick menu lookup for the agent search bar. While the user types 1-3 words,
// the chat panel shows matching menu entries so navigating never requires a
// full agent turn.
//
// Matching runs on normalizeStringN output on both sides, so accents and casing
// are irrelevant ("gestion" finds "Gestión"). That normalizer collapses spaces
// into "_", which is why the search text is split on "_" rather than " ".

import { normalizeStringN, wordInclude } from '@genix/ui/utilities';
import { Modules, canOpenRoute } from '#core/modules.svelte.ts';
import type { IPage } from '#core/modules.ts';

export interface MenuSearchOption {
  name: string; // bilingual "English|Spanish", resolved with tr() at render time
  route: string;
  icon: string;
  nameNorm: string; // normalized haystack, cached here to keep matching allocation-free
}

const MAX_RESULTS = 6;
const MAX_WORDS = 3;
const FALLBACK_ICON = 'icon-[fa--bolt]';

// Searches every page of every module (db/modules.toml), not only the module
// the side menu shows. A page shared by several modules is indexed once.
// Built once per page load: the normalization happens once, not per keystroke.
let indexedPages: (IPage & { nameNorm: string })[] | undefined;

const buildIndex = () => {
  const pagesByRoute = new Map<string, IPage>();
  for (const menuPage of Modules.flatMap((module) => module.menus.flatMap((menuGroup) => menuGroup.pages))) {
    if (!menuPage.pending) { pagesByRoute.set(menuPage.route, menuPage); }
  }
  return [...pagesByRoute.values()].map((menuPage) => ({
    ...menuPage,
    // Both language halves are indexed, so "sales" finds "Punto de Venta"
    // too. The pipe becomes a space first; normalizeStringN would otherwise
    // drop it and weld the two halves into one bogus word.
    nameNorm: normalizeStringN(menuPage.name.replace('|', ' ')),
  }));
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

  indexedPages ||= buildIndex();

  const results: MenuSearchOption[] = [];
  for (const indexedPage of indexedPages) {
    // Same access filter as the side menu, checked per search: the session can change.
    if (!wordInclude(indexedPage.nameNorm, words) || !canOpenRoute(indexedPage.route)) { continue; }
    results.push({ name: indexedPage.name, route: indexedPage.route, icon: indexedPage.icon || FALLBACK_ICON, nameNorm: indexedPage.nameNorm });
    if (results.length === MAX_RESULTS) { break; }
  }
  return results;
};

// Words for highlString(), which matches lowercase words against the rendered
// (accented) name. Unaccented input still matches the row, it just renders
// without the highlight.
export const highlightWords = (text: string): string[] =>
  text.toLowerCase().split(' ').filter((word) => word.length > 1);
