/**
 * Helper function to highlight search terms in a string
 * Returns an array of strings and highlighted objects
 */
export function highlString(phrase: string, words: string[]) {
  if (typeof phrase !== 'string') {
    console.error('highlString: phrase is not a string');
    console.log(phrase);
    return ['!'];
  }
  
  const arr: any[] = [phrase];
  if (!words || words.length === 0) return arr;

  for (let word of words) {
    if (word.length < 2) continue;
    for (let i = 0; i < arr.length; i++) {
      const str = arr[i];
      if (typeof str !== 'string') continue;
      const idx = str.toLowerCase().indexOf(word);
      if (idx !== -1) {
        const ini = str.slice(0, idx);
        const middle = str.slice(idx, idx + word.length);
        const fin = str.slice(idx + word.length);
        arr.splice(i, 1, ini, { type: 'em', text: middle }, fin);
        continue;
      }
    }
  }
  return arr.filter((x) => x);
}

/**
 * Creates a highlighted span element (for JSX/TSX)
 */
export function makeHighlString(content: string, search: string) {
  return highlString(content, search.split(' '));
}

