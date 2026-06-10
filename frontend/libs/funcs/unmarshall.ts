/**
 * Unmarshalls a compact JSON array serialized by the serialize Go library.
 * The input should be an array [keys, content].
 *
 * @param encoded The serialized data [keys, content]
 * @returns The unmarshalled object or array
 */
export const unmarshall = (encoded: any): any => {
  if (!Array.isArray(encoded) || encoded.length !== 2) {
    return encoded;
  }

  const [keysDef, content] = encoded;
  if (!Array.isArray(keysDef)) {
    return encoded;
  }

  // Map to store type definitions: typeId -> { orderIdx -> fieldName }
  const keysMap: Record<number, { fields: Record<number, string>; maxIndex: number }> = {};
  for (const k of keysDef) {
    if (!Array.isArray(k)) continue;
    const typeId = k[0];
    const fields: Record<number, string> = {};
    let maxIndex = -1;
    for (let i = 1; i < k.length; i += 2) {
      const idx = k[i];
      const name = k[i + 1];
      fields[idx] = name;
      if (idx > maxIndex) maxIndex = idx;
    }
    keysMap[typeId] = { fields, maxIndex };
  }

  let lastTypeId: number | null = null;

  /**
   * Decodes a value recursively based on headers.
   */
  const decode = (val: any): any => {
    if (!Array.isArray(val) || val.length === 0) {
      return val;
    }

    const header = val[0];

    // Header 1: New object type
    // Format: [1, [typeId, ...skipIndices], ...values]
    if (header === 1) {
      if (val.length < 2) return val;
      const refBlock = val[1];
      if (!Array.isArray(refBlock)) return val;
      const typeId = refBlock[0];
      const skipIndices = new Set<number>();
      for (let i = 1; i < refBlock.length; i++) {
        skipIndices.add(refBlock[i]);
      }
      lastTypeId = typeId;
      return populate(typeId, val.slice(2), skipIndices);
    }

    // Header 0: Same type as previous object
    // Format: [0, [skipIndices]?, ...values]
    if (header === 0) {
      if (lastTypeId === null) return val;

      let skipIndices = new Set<number>();
      let valueStartIdx = 1;

      if (Array.isArray(val[1])) {
        const sub = val[1];
        // Disambiguate a skip block from a value that is itself an encoded array.
        //
        // The Go encoder (serialize/marshal.go) emits omitted fields as a skip
        // block: a non-empty array of the skipped field-order indices, e.g. `[2]`
        // or `[2, 6]`. The Go decoder tells this apart from a real value using the
        // first field's *type* — info we don't have here.
        //
        // We rely on a structural invariant instead: a skip block is always a
        // non-empty array of NON-NEGATIVE INTEGERS, whereas a value that happens to
        // be an array is an encoded struct (`[1, [refBlock], …]` → contains a nested
        // array), slice (`[2, …items]`), or map (`[3, "key", …]` → contains string
        // keys). All of those carry a non-integer element right after the header, so
        // "every element is a non-negative integer" cleanly identifies a skip block.
        // (The old heuristic rejected leading 2/3 as headers, so common skip blocks
        // like `[2]` (skip `children`) were misread as values and shifted every field.)
        const isSkipBlock =
          sub.length > 0 &&
          sub.every((s: any) => typeof s === 'number' && Number.isInteger(s) && s >= 0);

        if (isSkipBlock) {
          for (const s of sub) skipIndices.add(s);
          valueStartIdx = 2;
        }
      }

      return populate(lastTypeId, val.slice(valueStartIdx), skipIndices);
    }

    // Header 2: Array of values
    // Format: [2, ...items]
    if (header === 2) {
      const result = [];
      for (let i = 1; i < val.length; i++) {
        result.push(decode(val[i]));
      }
      return result;
    }

    // Header 3: Map of key-value pairs
    // Format: [3, key1, val1, key2, val2, ...]
    if (header === 3) {
      const result: Record<string, any> = {};
      for (let i = 1; i < val.length; i += 2) {
        if (i + 1 < val.length) {
          const key = String(val[i]);
          result[key] = decode(val[i + 1]);
        }
      }
      return result;
    }

    // Fallback for plain arrays (header "Other")
    return val.map(decode);
  };

  /**
   * Populates an object of a given type with values and skip indices.
   */
  const populate = (typeId: number, values: any[], skipIndices: Set<number>) => {
    const typeDef = keysMap[typeId];
    if (!typeDef) return values;

    const { fields, maxIndex } = typeDef;
    const obj: Record<string, any> = {};
    let valIdx = 0;
    for (let i = 0; i <= maxIndex; i++) {
      if (skipIndices.has(i)) {
        continue;
      }
      if (valIdx >= values.length) {
        break;
      }
      const fieldName = fields[i];
      if (fieldName) {
        obj[fieldName] = decode(values[valIdx]);
      }
      valIdx++;
    }
    return obj;
  };

  return decode(content);
};
