import fs from 'fs';
import path from 'path';

const nameMapping = new Map();
const tmpDir = path.resolve(process.cwd(), 'tmp');
const counterFilePath = path.join(tmpDir, 'counter.txt');
const lockDir = path.join(tmpDir, 'counter.lock');

/**
 * replaceClassName - Converts a class name to a minified counter-based identifier.
 * 
 * @param {string} name - The original CSS class name.
 * @returns {string} The minified identifier (or original name if < 4 characters).
 */
export function replaceClassName(name) {
    if (name.length < 4) return name;
    if (nameMapping.has(name)) return nameMapping.get(name);
    const counter = getCounter()
    nameMapping.set(name, counter);
    return counter;
}

/**
 * counterMinify - Converts a number to a short, CSS-safe string identifier.
 * Logic: a, b... Z, a0, a1... aZ, b0...
 * Always starts with a letter, contains letters and numbers.
 * 
 * @param {number} n - The counter value.
 * @returns {string} The minified string.
 */
export function counterMinify(n) {
    const chars0 = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ';
    const charsN = '0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ';

    if (n < 52) return chars0[n];

    let tempN = n - 52;
    let length = 2;
    let combinations = 52 * 62;

    while (tempN >= combinations) {
        tempN -= combinations;
        length++;
        combinations *= 62;
    }

    let res = '';
    for (let i = 0; i < length - 1; i++) {
        res = charsN[tempN % 62] + res;
        tempN = Math.floor(tempN / 62);
    }
    res = chars0[tempN] + res;

    return res;
}

function acquireLock() {
    const maxRetries = 100;
    const retryDelay = 20; // ms
    let retries = 0;

    while (retries < maxRetries) {
        try {
            fs.mkdirSync(lockDir);
            return; // Lock acquired
        } catch (e) {
            if (e.code === 'EEXIST') {
                // Check if lock is stale (older than 5 seconds)
                try {
                    const stats = fs.statSync(lockDir);
                    if (Date.now() - stats.mtimeMs > 5000) {
                        try {
                            fs.rmdirSync(lockDir);
                            continue; // Retry immediately
                        } catch (rmErr) {
                            // Ignore removal error, someone else might have removed it
                        }
                    }
                } catch (statErr) {
                    // Lock might have been removed in the meantime
                }

                // Wait and retry
                const end = Date.now() + retryDelay;
                while (Date.now() < end) { }
                retries++;
            } else {
                throw e;
            }
        }
    }
    throw new Error(`Failed to acquire lock on ${lockDir} after ${maxRetries} retries.`);
}

function releaseLock() {
    try {
        fs.rmdirSync(lockDir);
    } catch (e) {
        // Ignore if already removed, but log warnings for other errors
        if (e.code !== 'ENOENT') {
            console.warn(`[counter] Warning: Failed to release lock: ${e.message}`);
        }
    }
}

/**
 * getCounter - Atomically increments the counter and returns its minified string representation.
 * Uses file locking to ensure process safety.
 * 
 * @returns {string} The minified counter string.
 */
export function getCounter() {
    if (!fs.existsSync(tmpDir)) {
        fs.mkdirSync(tmpDir, { recursive: true });
    }

    let count = 0;

    try {
        acquireLock();

        // Read current
        if (fs.existsSync(counterFilePath)) {
            const content = fs.readFileSync(counterFilePath, 'utf8').trim();
            // Format is "lockedBy:count", splits to ['lockedBy', 'count']
            const parts = content.split(':');
            count = parseInt(parts.length > 1 ? parts[1] : parts[0], 10) || 0;
        } else {
            // Default starting value if file doesn't exist
            count = 55;
        }

        // Increment
        count++;

        // Save (preserving the '0' for lockedBy for compatibility/simplicity, 
        // essentially "0:count" means not logically locked by old mechanism)
        fs.writeFileSync(counterFilePath, `0:${count}`, 'utf8');

    } finally {
        releaseLock();
    }

    return counterMinify(count);
}

/**
 * Deprecated: Kept for compatibility but effectively no-op or simple wrapper.
 * The new getCounter handles persistence automatically.
 */
export function saveCounter(userID) {
    // No-op in new atomic design, or could force a save if really needed, 
    // but getCounter handles state.
}

/**
 * Deprecated: Kept for compatibility.
 */
export function getCounterFomFile() {
    // No-op, getCounter reads from file every time.
}


/**
 * svelteClassHasher - A Vite plugin designed for Svelte 5 to automatically hash local CSS classes.
 * 
 * PURPOSE:
 * This plugin scopes CSS specifically to a component by hashing class names defined in 
 * <style> blocks and updating all valid usages in the template. This prevents style 
 * leakage between components.
 * 
 * KEY FEATURES / SAFETY LOGIC:
 * 1. LOCAL SCOPE: It only hashes classes that exist in the file's own <style> block.
 * 2. PATTERN PROTECTION: It only replaces class names where they are explicitly used 
 *    as classes (e.g., class="", class:name, or .classList.add()).
 * 3. ALL-OR-NOTHING: If a class name is used in a way the plugin cannot safely identify 
 *    (like a dynamic variable or property name), the plugin will SKIP hashing that 
 *    entire class name for that file to prevent broken styles.
 * 4. LOGIC PROTECTION: It completely ignores <script> blocks to ensure your business 
 *    logic remains untouched.
 */
export const svelteClassHasher = () => {
    const classMap = new Map();
    const srcDir = path.resolve(process.cwd(), 'src');

    return {
        name: 'svelte-class-hasher',
        enforce: 'pre',

        /**
         * buildStart hook:
         * Recursively scans the /src directory to build a map of every hashable class 
         * found in Svelte style blocks. This ensures hash consistency across files.
         */
        async buildStart() {
            // No initialization needed for counter, it relies on file.

            const scanDir = (dir) => {
                if (!fs.existsSync(dir)) return;
                const entries = fs.readdirSync(dir, { withFileTypes: true });
                for (const entry of entries) {
                    const fullPath = path.join(dir, entry.name);
                    if (entry.isDirectory()) {
                        scanDir(fullPath);
                    } else if (entry.isFile() && entry.name.endsWith('.svelte')) {
                        const content = fs.readFileSync(fullPath, 'utf8');
                        const styleMatch = content.match(/<style[^>]*>([\s\S]*?)<\/style>/);
                        if (styleMatch) {
                            const styleContent = styleMatch[1];
                            const classSelectorRegex = /\.([a-zA-Z_-][a-zA-Z0-9_-]*)/g;
                            let m;
                            while ((m = classSelectorRegex.exec(styleContent)) !== null) {
                                const className = m[1];
                                if (className.length >= 4) {
                                    classMap.set(className, replaceClassName(className));
                                }
                            }
                        }
                    }
                }
            };

            nameMapping.clear();
            classMap.clear();
            scanDir(srcDir);
            console.log(`[class-hasher] Global map ready: ${classMap.size} classes protected.`);
        },

        /*
         * buildEnd hook:
         */
        async buildEnd() {
            // No cleanup needed
        },

        /**
         * transform hook:
         * Parses each .svelte file to identify local classes and their safe usages.
         */
        async transform(code, id) {
            if (!id.endsWith('.svelte')) return null;
            if (id.includes('node_modules')) return null;

            // STEP 1: Extract classes defined in the LOCAL <style> block.
            const styleRegex = /<style[^>]*>([\s\S]*?)<\/style>/g;
            const styleMatch = [...code.matchAll(styleRegex)];
            if (styleMatch.length === 0) return null;

            const localClasses = new Set();
            for (const m of styleMatch) {
                const content = m[1];
                const classSelectorRegex = /\.([a-zA-Z_-][a-zA-Z0-9_-]*)/g;
                let sm;
                while ((sm = classSelectorRegex.exec(content)) !== null) {
                    if (sm[1].length >= 4) localClasses.add(sm[1]);
                }
            }

            if (localClasses.size === 0) return null;

            // STEP 1.5: Exclude classes found in script blocks to prevent breaking JS logic.
            // If a class name appears in the script tag, we assume it's used programmatically
            // and skip hashing it to be safe.
            const scriptCheckRegex = /<script[^>]*>([\s\S]*?)<\/script>/g;
            let scrMatch;
            while ((scrMatch = scriptCheckRegex.exec(code)) !== null) {
                const scriptContent = scrMatch[1];
                for (const cls of Array.from(localClasses)) {
                    const pattern = new RegExp(`(?<![a-zA-Z0-9_-])${cls.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?![a-zA-Z0-9_-])`);
                    if (pattern.test(scriptContent)) {
                        localClasses.delete(cls);
                    }
                }
            }

            if (localClasses.size === 0) return null;

            // STEP 2: Map forbidden ranges (script or style blocks) to prevent accidental logic edits.
            const scriptRegex = /<script[^>]*>([\s\S]*?)<\/script>/g;
            const scriptRanges = [...code.matchAll(scriptRegex)].map(m => ({ s: m.index, e: m.index + m[0].length }));
            const styleRanges = styleMatch.map(m => ({ s: m.index, e: m.index + m[0].length }));

            const isScriptOrStyle = (i) =>
                scriptRanges.some(r => i >= r.s && i < r.e) ||
                styleRanges.some(r => i >= r.s && i < r.e);

            // STEP 3: Identify all candidate word occurrences in the template area.
            const sorted = Array.from(localClasses).sort((a, b) => b.length - a.length);
            const classRegex = new RegExp(`(?<![a-zA-Z0-9_-])(${sorted.map(c => c.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})(?![a-zA-Z0-9_-])`, 'g');

            const markupOccurrences = [];
            let m;
            while ((m = classRegex.exec(code)) !== null) {
                if (!isScriptOrStyle(m.index)) {
                    markupOccurrences.push({ index: m.index, word: m[1] });
                }
            }

            // STEP 4: Identify "Safe Zones" - specific code patterns where we are sure the usage is a CSS class.
            const ignoredIndices = new Set();
            const commentRegex = /<!--[\s\S]*?-->/g;
            while ((m = commentRegex.exec(code)) !== null) {
                for (let i = m.index; i < m.index + m[0].length; i++) ignoredIndices.add(i);
            }

            const safeIndices = new Set();
            const tagRegex = /<([a-zA-Z0-9:-]+)([\s\S]*?)>/g;
            while ((m = tagRegex.exec(code)) !== null) {
                if (isScriptOrStyle(m.index)) continue;

                const tagContent = m[2];
                const tagStart = m.index + m[0].indexOf(tagContent);

                // 4.1 Standard class attributes (class="", layerClass="", etc.)
                const attrStartRegex = /\b(?:class|[a-zA-Z0-9_-]+(?:Class|Classes|Css))\s*=\s*(['"\{])/g;
                let am;
                while ((am = attrStartRegex.exec(tagContent)) !== null) {
                    const type = am[1];
                    if (type === '"' || type === "'") {
                        const endQuoteIndex = tagContent.indexOf(type, am.index + am[0].length);
                        if (endQuoteIndex !== -1) {
                            const valStart = tagStart + am.index + am[0].length;
                            const valEnd = tagStart + endQuoteIndex;
                            markupOccurrences.forEach(o => {
                                if (o.index >= valStart && o.index + o.word.length <= valEnd) safeIndices.add(o.index);
                            });
                            attrStartRegex.lastIndex = endQuoteIndex + 1;
                        }
                    } else if (type === '{') {
                        // Handle balanced braces for complex attributes like class={condition ? 'a' : 'b'}
                        let depth = 1;
                        let j = am.index + am[0].length;
                        while (j < tagContent.length && depth > 0) {
                            if (tagContent[j] === '{') depth++;
                            else if (tagContent[j] === '}') depth--;
                            j++;
                        }
                        if (depth === 0) {
                            const bracedExpr = tagContent.substring(am.index + am[0].length, j - 1);
                            const exprStart = tagStart + am.index + am[0].length;
                            const strPat = /(['"`])([\s\S]*?)\1/g;
                            let sm;
                            while ((sm = strPat.exec(bracedExpr)) !== null) {
                                const sStart = exprStart + sm.index + 1;
                                const inner = sm[2];
                                if (sm[1] === '`') {
                                    const slots = [];
                                    const slotPat = /\$\{[\s\S]*?\}/g;
                                    let slm;
                                    while ((slm = slotPat.exec(inner)) !== null) {
                                        slots.push({ s: sStart + slm.index, e: sStart + slm.index + slm[0].length });
                                    }
                                    markupOccurrences.forEach(o => {
                                        if (o.index >= sStart && o.index + o.word.length <= sStart + inner.length) {
                                            if (!slots.some(s => o.index < s.e && o.index + o.word.length > s.s)) safeIndices.add(o.index);
                                        }
                                    });
                                } else {
                                    markupOccurrences.forEach(o => {
                                        if (o.index >= sStart && o.index + o.word.length <= sStart + inner.length) safeIndices.add(o.index);
                                    });
                                }
                            }
                            attrStartRegex.lastIndex = j;
                        }
                    }
                }

                // 4.2 Svelte class: directives
                const directiveRegex = /\bclass:([a-zA-Z0-9_-]+)/g;
                let dm;
                while ((dm = directiveRegex.exec(tagContent)) !== null) {
                    const cls = dm[1];
                    const start = tagStart + dm.index + dm[0].indexOf(cls);
                    markupOccurrences.forEach(o => { if (o.index === start) safeIndices.add(o.index); });
                }

                // 4.3 JavaScript classList methods inside event handlers
                const classListRegex = /\.classList\.(?:add|remove|toggle|contains)\s*\(\s*(['"`])([\s\S]*?)\1\s*\)/g;
                let clm;
                while ((clm = classListRegex.exec(tagContent)) !== null) {
                    const val = clm[2];
                    const start = tagStart + clm.index + clm[0].indexOf(val);
                    markupOccurrences.forEach(o => {
                        if (o.index >= start && o.index + o.word.length <= start + val.length) safeIndices.add(o.index);
                    });
                }
            }

            // STEP 5: Integrity Verification.
            // Filter out any class that is used in a "non-safe" way to prevent broken selectors.
            const allowedClasses = new Set();
            for (const cls of localClasses) {
                const clsOccs = markupOccurrences.filter(o => o.word === cls && !ignoredIndices.has(o.index));
                const allSafe = clsOccs.every(o => safeIndices.has(o.index));

                if (allSafe) {
                    allowedClasses.add(cls);
                } else {
                    const firstUnsafe = clsOccs.find(o => !safeIndices.has(o.index));
                    if (firstUnsafe) {
                        const lineNumber = code.substring(0, firstUnsafe.index).split('\n').length;
                        const line = code.split('\n')[lineNumber - 1].trim();
                        console.warn(`[class-hasher] WARNING: Class "${cls}" in ${path.basename(id)}:${lineNumber} has non-standard usage. Skipping entire class for safety. Line: "${line}"`);
                    }
                }
            }

            if (allowedClasses.size === 0) return null;

            // STEP 6: Apply all safe replacements.
            const replacements = [];

            // Update style block selectors
            for (const m of styleMatch) {
                const content = m[1];
                const contentStart = m.index + m[0].indexOf(content);
                const selectorRegex = /\.([a-zA-Z_-][a-zA-Z0-9_-]*)/g;
                let sm;
                while ((sm = selectorRegex.exec(content)) !== null) {
                    if (allowedClasses.has(sm[1])) {
                        replacements.push({ i: contentStart + sm.index + 1, l: sm[1].length, r: replaceClassName(sm[1]) });
                    }
                }
            }

            // Update template markup
            for (const occ of markupOccurrences) {
                if (allowedClasses.has(occ.word)) {
                    replacements.push({ i: occ.index, l: occ.word.length, r: replaceClassName(occ.word) });
                }
            }

            if (replacements.length === 0) return null;

            replacements.sort((a, b) => b.i - a.i);
            let newCode = code;
            for (const rep of replacements) {
                newCode = newCode.substring(0, rep.i) + rep.r + newCode.substring(rep.i + rep.l);
            }

            return { code: newCode, map: null };
        }
    };
};
