import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

// Anchor all counter state to the frontend root (this file's directory), NOT
// process.cwd(): `build:store` runs from frontend/webpage/ while `build:main` runs
// from frontend/. Anchoring here makes the admin and storefront builds share ONE
// registry file, so a class common to both gets one stable name.
const FRONTEND_ROOT = path.dirname(fileURLToPath(import.meta.url));
const tmpDir = path.resolve(FRONTEND_ROOT, 'tmp');
const registryFilePath = path.join(tmpDir, 'class-counter-map.txt');
const lockDir = path.join(tmpDir, 'counter.lock');

// ---------------------------------------------------------------------------
// Deterministic, persisted key -> counter registry.
//
// A given stable key ALWAYS maps to the same minified name, regardless of call
// order, build pass, or process. This is what the storefront's two-pass prerender
// (server + client build) needs: both passes resolve identical keys to identical
// names, so prerendered HTML class names stay in sync with the bundled CSS and
// hydration doesn't break. The old blind counter assigned names by encounter order,
// which diverged between passes.
// ---------------------------------------------------------------------------
let registry = null;   // Map<string key, number>
let maxCounter = 55;   // preserve the previous starting base
let dirty = false;

// Block reservation. plugins.js is imported as SEPARATE module instances within a
// single build — svelte.config.js (cssHash, 's:' keys) gets one, vite.config.ts
// (CSS modules 'm:' + svelteClassHasher 'h:' keys) gets another. Module-level state
// is NOT shared across ESM instances, so two instances counting from the same base
// would hand DIFFERENT keys the SAME number (e.g. s:SideMenu|85 and h:close-button|85
// both -> 'ax', producing a broken `.ax.ax` selector). The only shared medium is the
// registry file, so each instance reserves a disjoint range of the number space under
// the lock and writes a '#max' high-water sentinel; concurrent instances then always
// claim ranges beyond it. Within its own block an instance allocates sequentially.
const MAX_KEY = '#max';
const BLOCK_SIZE = 128;
let blockNext = 0;   // next number to hand out from our reserved block
let blockEnd = 0;    // exclusive end of our reserved block (blockNext === blockEnd -> exhausted)

// Read every `key|counter` line into `registry`, returning the highest counter seen
// (including the '#max' sentinel). Unknown keys are merged in; known keys are kept so
// a warm registry stays stable. Caller decides locking.
function readRegistryFile() {
    let fileMax = maxCounter;
    if (fs.existsSync(registryFilePath)) {
        for (const line of fs.readFileSync(registryFilePath, 'utf8').split('\n')) {
            if (!line) continue;
            const i = line.lastIndexOf('|');
            if (i === -1) continue;
            const key = line.slice(0, i);
            const n = parseInt(line.slice(i + 1), 10);
            if (Number.isNaN(n)) continue;
            if (n > fileMax) fileMax = n;
            if (key === MAX_KEY) continue;           // sentinel, not a real class key
            if (!registry.has(key)) registry.set(key, n);
        }
    }
    return fileMax;
}

// Write the full registry + the '#max' sentinel. Caller MUST hold the lock.
function writeRegistryLocked() {
    if (!fs.existsSync(tmpDir)) fs.mkdirSync(tmpDir, { recursive: true });
    const lines = [...registry.entries()].map(([k, v]) => `${k}|${v}`);
    lines.push(`${MAX_KEY}|${maxCounter}`);
    fs.writeFileSync(registryFilePath, lines.join('\n') + '\n', 'utf8');
}

function loadRegistry() {
    if (registry) return;
    registry = new Map();
    const fileMax = readRegistryFile();
    if (fileMax > maxCounter) maxCounter = fileMax;
}

// Reserve a fresh, disjoint block of numbers from the shared file under the lock.
function reserveBlock() {
    if (!fs.existsSync(tmpDir)) fs.mkdirSync(tmpDir, { recursive: true });
    acquireLock();
    try {
        // Re-read so we start beyond any range another instance/process reserved.
        const fileMax = readRegistryFile();
        if (fileMax > maxCounter) maxCounter = fileMax;
        blockNext = maxCounter + 1;
        blockEnd = blockNext + BLOCK_SIZE;
        maxCounter = blockEnd - 1;   // claim the whole block via the sentinel
        writeRegistryLocked();       // publish the reservation so peers skip past it
        dirty = false;               // everything known is now on disk
    } finally {
        releaseLock();
    }
}

function flushRegistry() {
    if (!registry || !dirty) return;
    acquireLock();
    try {
        // Adopt any keys another build process appended since we loaded, so sequential
        // admin/store builds extend the file instead of clobbering each other.
        const fileMax = readRegistryFile();
        if (fileMax > maxCounter) maxCounter = fileMax;
        writeRegistryLocked();
        dirty = false;
    } finally {
        releaseLock();
    }
}

// Flush once on process exit — covers the CSS-module / cssHash config callbacks,
// which have no Vite build hook of their own. svelteClassHasher also flushes in
// buildEnd so the file is durable before a sibling build process starts.
process.on('exit', () => { try { flushRegistry(); } catch { /* best effort */ } });

/**
 * getCounterForKey - Deterministic minified name for a stable key.
 * Same key -> same name, always. New keys are assigned the next sequential counter.
 *
 * @param {string} key - stable, namespaced key (see makeClassKey).
 * @returns {string} The minified identifier.
 */
export function getCounterForKey(key) {
    loadRegistry();
    if (registry.has(key)) return counterMinify(registry.get(key));
    // New key: take the next number from our reserved block, reserving a fresh one
    // (disjoint from every other instance/process) when the current block is exhausted.
    if (blockNext >= blockEnd) reserveBlock();
    const n = blockNext++;
    registry.set(key, n);
    dirty = true;
    return counterMinify(n);
}

/**
 * makeClassKey - Build a stable, repo-relative, namespaced registry key.
 * Filenames are normalized relative to the frontend root and to POSIX separators so
 * the same source file yields the same key from the admin and storefront builds.
 *
 * @param {string} ns - consumer namespace: 'm' (CSS module), 's' (svelte cssHash), 'h' (class hasher).
 * @param {string} [filename] - absolute file path (or empty when unknown).
 * @param {string} [name] - class/local name (omit for file-only keys such as cssHash).
 * @returns {string} The namespaced key.
 */
export function makeClassKey(ns, filename, name) {
    let rel = filename ? path.relative(FRONTEND_ROOT, filename) : '';
    rel = rel.split(path.sep).join('/');
    return name != null ? `${ns}:${rel}:${name}` : `${ns}:${rel}`;
}

/** Force-persist the registry (called from build hooks). */
export function flushClassRegistry() { flushRegistry(); }

/**
 * replaceClassName - Hashes a Svelte <style> local class name. Uses the global 'h:'
 * namespace so the same class name maps to one hash across every component (the
 * style hasher is intentionally file-agnostic).
 *
 * @param {string} name - The original CSS class name.
 * @returns {string} The minified identifier (or original name if < 4 characters).
 */
export function replaceClassName(name) {
    if (name.length < 4) return name;
    return getCounterForKey('h:' + name);
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
/**
 * maskHtmlComments - Replace the CONTENTS of every HTML comment with spaces of equal
 * length, keeping the surrounding text length intact so every index in the returned
 * string still lines up with the original source.
 *
 * Why: block detection locates `<style>`/`<script>` with a plain regex. A component
 * that merely MENTIONS `<style>` inside a doc comment (e.g. "Layout lives in the scoped
 * <style> below") would otherwise have that literal matched as a real opening tag, and
 * the lazy `[\s\S]*?` would run to the next real `</style>` — swallowing all the markup
 * in between. Every markup class then looks like it lives inside <style>, so it's never
 * found as a usage; STEP 5's `clsOccs.every()` is then vacuously true and the class gets
 * renamed in the stylesheet but NOT in the markup, breaking the scoped selector. Scanning
 * a comment-masked copy (while applying replacements to the original) avoids this.
 */
function maskHtmlComments(code) {
    return code.replace(/<!--[\s\S]*?-->/g, (m) => ' '.repeat(m.length));
}

export const svelteClassHasher = () => {
    const classMap = new Map();
    const srcDir = process.cwd();

    return {
        name: 'svelte-class-hasher',
        enforce: 'pre',

        /**
         * buildStart hook:
         * Recursively scans the project directory to build a map of every hashable class 
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
                        if (['node_modules', '.svelte-kit', 'tmp', 'static', 'build', '.git', 'ecommerce/build'].includes(entry.name)) continue;
                        scanDir(fullPath);
                    } else if (entry.isFile() && entry.name.endsWith('.svelte')) {
                        // Mask HTML comments so a `<style>` mentioned in a comment isn't
                        // matched as a real block (see maskHtmlComments).
                        const content = maskHtmlComments(fs.readFileSync(fullPath, 'utf8'));
                        const styleMatch = content.match(/<style[^>]*>([\s\S]*?)<\/style>/);
                        if (styleMatch) {
                            // Strip CSS comments first: a commented-out selector like
                            // `/* .relative {} */` must NOT be treated as a real local
                            // class, or the hasher would rename the global utility.
                            const styleContent = styleMatch[1].replace(/\/\*[\s\S]*?\*\//g, '');
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

            classMap.clear();
            scanDir(srcDir);
            console.log(`[class-hasher] Global map ready: ${classMap.size} classes protected.`);
        },

        /*
         * buildEnd hook: persist the registry so a sibling build process (e.g. the
         * storefront build that runs after the admin build) starts from our names.
         */
        async buildEnd() {
            flushClassRegistry();
        },

        /**
         * transform hook:
         * Parses each .svelte file to identify local classes and their safe usages.
         */
        async transform(code, id) {
            if (!id.endsWith('.svelte')) return null;
            if (id.includes('node_modules')) return null;

            // All block/markup DETECTION runs against a comment-masked copy so a literal
            // `<style>`/`<script>` inside an HTML comment can't be mistaken for a real
            // block (see maskHtmlComments). Indices are preserved, so every replacement
            // below is still applied to the original `code` at the same offset.
            const masked = maskHtmlComments(code);

            // STEP 1: Extract classes defined in the LOCAL <style> block.
            const styleRegex = /<style[^>]*>([\s\S]*?)<\/style>/g;
            const styleMatch = [...masked.matchAll(styleRegex)];
            if (styleMatch.length === 0) return null;

            const localClasses = new Set();
            for (const m of styleMatch) {
                // Strip CSS comments so a commented-out selector (e.g. `/* .relative {} */`)
                // is never picked up as a local class and used to rename a global utility.
                const content = m[1].replace(/\/\*[\s\S]*?\*\//g, '');
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
            while ((scrMatch = scriptCheckRegex.exec(masked)) !== null) {
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
            const scriptRanges = [...masked.matchAll(scriptRegex)].map(m => ({ s: m.index, e: m.index + m[0].length }));
            const styleRanges = styleMatch.map(m => ({ s: m.index, e: m.index + m[0].length }));

            const isScriptOrStyle = (i) =>
                scriptRanges.some(r => i >= r.s && i < r.e) ||
                styleRanges.some(r => i >= r.s && i < r.e);

            // STEP 3: Identify all candidate word occurrences in the template area.
            const sorted = Array.from(localClasses).sort((a, b) => b.length - a.length);
            const classRegex = new RegExp(`(?<![a-zA-Z0-9_-])(${sorted.map(c => c.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})(?![a-zA-Z0-9_-])`, 'g');

            const markupOccurrences = [];
            let m;
            while ((m = classRegex.exec(masked)) !== null) {
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
            while ((m = tagRegex.exec(masked)) !== null) {
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
