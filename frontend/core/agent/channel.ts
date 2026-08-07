// Channel token: the single identifier of this tab's agent stream, carrying the
// three values that make it unique — company, user and tab.
//
//   bytes = uvarint(companyID) ‖ uvarint(userID) ‖ 6 random bytes (tab)
//   token = base64url(bytes), unpadded
//
// Typical ids fit in one byte each, so the whole thing is 11 characters. The
// varints are what keep it short: a decimal company id costs 6-7 characters on
// its own.
//
// It is an *identifier*, never a credential: the browser still proves who it is
// with its session token, and both the backend and the bridge reject a channel
// whose identity doesn't match the authenticated one.
//
// Mirrored in backend/agent/channel.go and sse_bridge/channel.go. The three
// copies must agree byte for byte.

// TAB_RANDOM_BYTES is the tab's entropy: 6 bytes = 48 bits, and exactly 8
// characters once base64url-encoded, which is the tab id's budget.
const TAB_RANDOM_BYTES = 6;

const encodeBase64Url = (bytes: Uint8Array): string => {
  let binaryText = '';
  for (const byteValue of bytes) { binaryText += String.fromCharCode(byteValue); }
  return btoa(binaryText).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
};

const decodeBase64Url = (encodedText: string): Uint8Array => {
  const standardBase64 = encodedText.replace(/-/g, '+').replace(/_/g, '/');
  const binaryText = atob(standardBase64);
  const bytes = new Uint8Array(binaryText.length);
  for (let index = 0; index < binaryText.length; index++) { bytes[index] = binaryText.charCodeAt(index); }
  return bytes;
};

const appendUvarint = (target: number[], value: number) => {
  // Same LEB128 the Go side writes: 7 bits per byte, high bit marks "continues".
  while (value >= 0x80) {
    target.push((value & 0x7f) | 0x80);
    value = Math.floor(value / 128);
  }
  target.push(value);
};

// mintTabID generates a fresh 8-character tab id (6 random bytes).
export const mintTabID = (): string =>
  encodeBase64Url(crypto.getRandomValues(new Uint8Array(TAB_RANDOM_BYTES)));

// isValidTabID guards against a stale sessionStorage value from an older build,
// which would otherwise produce a token the server can't decode.
export const isValidTabID = (tabID: string): boolean => {
  if (tabID.length !== 8) { return false; }
  try {
    return decodeBase64Url(tabID).length === TAB_RANDOM_BYTES;
  } catch {
    return false;
  }
};

// encodeChannelToken builds the token naming one tab's stream. Returns "" when
// the identity isn't known yet (no company or no session), which callers treat
// as "the agent can't connect".
export const encodeChannelToken = (companyID: number, userID: number, tabID: string): string => {
  if (!(companyID > 0) || !(userID > 0) || !isValidTabID(tabID)) { return ''; }

  const tokenBytes: number[] = [];
  appendUvarint(tokenBytes, companyID);
  appendUvarint(tokenBytes, userID);
  tokenBytes.push(...decodeBase64Url(tabID));
  return encodeBase64Url(new Uint8Array(tokenBytes));
};
