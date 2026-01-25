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
        let isSkipBlock = false;

        if (sub.length > 0) {
          const h = sub[0];
          // If first element is not a valid header (0, 1, 2, 3), it's definitely a skip block
          if (typeof h === 'number' && h !== 0 && h !== 1 && h !== 2 && h !== 3) {
            isSkipBlock = true;
          } else {
            // If it is 0, 1, 2, or 3, it could be a skip block OR a value of the first field.
            // Go's logic checks if the first field type is primitive.
            // In TS, we don't have types, so we use the header rule as a heuristic.
            isSkipBlock = typeof h === 'number' && h !== 0 && h !== 1 && h !== 2 && h !== 3;
          }
        }

        if (isSkipBlock) {
          for (const s of sub) {
            if (typeof s === 'number') skipIndices.add(s);
          }
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
