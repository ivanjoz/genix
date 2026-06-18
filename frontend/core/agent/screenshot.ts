// Screen capture via getDisplayMedia, exposed as the `screenshot` command
// handler in commands.ts. Lives in its own file because the prompt/permission
// lifecycle and the offscreen <video> reuse logic are unrelated to the rest of
// the WS command plumbing.
//
// The first call in a session triggers the browser's "share screen" prompt;
// subsequent calls reuse the resolved MediaStream. If the user clicks
// "Stop sharing" in the browser bar, the cached stream is dropped so the next
// call re-prompts.

export interface ScreenshotResult {
  MIME: string;
  Base64: string;
  Width: number;
  Height: number;
}

interface ScreenshotState {
  stream: MediaStream | null;
  videoEl: HTMLVideoElement | null;
}

const screenshotState: ScreenshotState = { stream: null, videoEl: null };

// Translates getDisplayMedia rejection reasons into messages the backend
// agent can act on. The browser's default `NotAllowedError: Permission denied`
// is too generic — the agent needs to know whether the user actively denied
// vs. e.g. no display source exists.
const describeDisplayMediaError = (err: unknown): string => {
  const name = (err as { name?: string })?.name || "";
  switch (name) {
    case "NotAllowedError":
      return "User denied permission for screenshot";
    case "NotFoundError":
      return "No screen, window, or tab available to share";
    case "AbortError":
      return "User dismissed the screen-share dialog";
    case "NotReadableError":
      return "Screen capture failed: source is in use or unreadable";
    case "OverconstrainedError":
      return "Screen capture failed: requested constraints cannot be satisfied";
    case "TypeError":
      return "Screen capture not supported in this context (needs HTTPS or localhost)";
    default: {
      const msg = (err as { message?: string })?.message || String(err);
      return `Screen capture failed: ${msg}`;
    }
  }
};

// Captures a frame from the active screen-share. First invocation triggers
// the user's "share screen" permission prompt; subsequent calls reuse the stream.
// Throws with a normalized message on denial/abort so the caller (and the
// remote agent driving us over HTTP) sees a clear reason.
export const captureScreenshot = async (): Promise<ScreenshotResult> => {
  let stream = screenshotState.stream;
  if (!stream || !stream.active) {
    // preferCurrentTab + selfBrowserSurface "include" let the user share THIS
    // tab (Chrome hides it by default). displaySurface "browser" restricts the
    // picker to Chrome tabs only — what we want for an agent screenshotting
    // the running app. All three are non-standard so we cast through any.
    const constraints = {
      video: { displaySurface: "browser" },
      audio: false,
      preferCurrentTab: true,
      selfBrowserSurface: "include",
      surfaceSwitching: "exclude",
      systemAudio: "exclude",
    } as any;
    try {
      stream = await navigator.mediaDevices.getDisplayMedia(constraints);
    } catch (err) {
      throw new Error(describeDisplayMediaError(err));
    }
    screenshotState.stream = stream;
    // If the user clicks "Stop sharing" in the browser bar, drop the cached stream.
    for (const track of stream.getVideoTracks()) {
      track.addEventListener("ended", () => {
        if (screenshotState.stream === stream) { screenshotState.stream = null; }
      });
    }
  }
  const track = stream.getVideoTracks()[0];
  const settings = track.getSettings();

  let video = screenshotState.videoEl;
  if (!video) {
    video = document.createElement("video");
    video.muted = true;
    (video as any).playsInline = true;
    video.style.cssText = "position:fixed;left:-9999px;top:-9999px;width:1px;height:1px;opacity:0;pointer-events:none";
    document.body.appendChild(video);
    screenshotState.videoEl = video;
  }
  if (video.srcObject !== stream) {
    video.srcObject = stream;
    await video.play();
  }
  // Wait for the first frame to be ready when video metadata isn't loaded yet.
  if (!video.videoWidth) {
    await new Promise<void>((resolve) => {
      const onLoaded = () => { video!.removeEventListener("loadedmetadata", onLoaded); resolve(); };
      video!.addEventListener("loadedmetadata", onLoaded);
    });
  }

  const width = settings.width || video.videoWidth;
  const height = settings.height || video.videoHeight;
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;
  const ctx = canvas.getContext("2d");
  if (!ctx) { throw new Error("2d context unavailable"); }
  ctx.drawImage(video, 0, 0, width, height);
  const dataUrl = canvas.toDataURL("image/png");
  const base64 = dataUrl.split(",")[1] || "";
  return { MIME: "image/png", Base64: base64, Width: width, Height: height };
};

// captureDomScreenshot renders the live DOM to a PNG without prompting the
// user — no getDisplayMedia, no banner, no picker. Uses modern-screenshot,
// dynamically imported so the ~20KB lib isn't in the initial bundle and is
// only fetched the first time an agent (or page) actually asks for one.
//
// Tradeoffs vs. captureScreenshot (getDisplayMedia):
//   - No permission required, works for any user, no UI intrusion.
//   - Captures the page only, not the surrounding browser chrome / other tabs.
//   - <canvas> children (e.g. billboard.js charts) only render correctly if
//     they're not tainted by cross-origin images.
let domToPngImpl: ((node: HTMLElement, options?: any) => Promise<string>) | null = null;
const loadDomToPng = async () => {
  if (domToPngImpl) { return domToPngImpl; }
  const mod = await import("modern-screenshot");
  domToPngImpl = mod.domToPng;
  return domToPngImpl;
};

// Icon fonts that the page only references through ::before/::after pseudos.
// modern-screenshot's font auto-detect attributes pseudo content to the parent
// element's font-family, so an icon font that's *only* applied on the pseudo
// is never flagged as "used" and its @font-face rule gets stripped from the
// SVG. The probe below puts each family on a real element so the detector
// picks them up. Extend this list if a new icon font ever stops rendering.
const PSEUDO_FONT_FAMILIES: string[] = [];

const appendFontProbes = (): HTMLElement => {
  const probe = document.createElement("div");
  probe.style.cssText = "position:fixed;left:-9999px;top:-9999px;visibility:hidden;pointer-events:none";
  for (const family of PSEUDO_FONT_FAMILIES) {
    const span = document.createElement("span");
    span.style.fontFamily = family;
    span.textContent = "x";
    probe.appendChild(span);
  }
  document.body.appendChild(probe);
  return probe;
};

export const captureDomScreenshot = async (): Promise<ScreenshotResult> => {
  const domToPng = await loadDomToPng();
  const root = document.documentElement;
  const width = root.clientWidth;
  const height = root.clientHeight;
  const probe = appendFontProbes();
  try {
    const dataUrl = await domToPng(root, {
      width,
      height,
      backgroundColor: "#ffffff",
    });
    const base64 = dataUrl.split(",")[1] || "";
    return { MIME: "image/png", Base64: base64, Width: width, Height: height };
  } finally {
    probe.remove();
  }
};

// Stops the cached MediaStream tracks (which removes the browser's
// "X is sharing your screen" indicator) and clears state. Called when the WS
// bridge disconnects so a fresh session prompts again.
export const releaseScreenStream = () => {
  if (!screenshotState.stream) { return; }
  for (const track of screenshotState.stream.getTracks()) { track.stop(); }
  screenshotState.stream = null;
};
