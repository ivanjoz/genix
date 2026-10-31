import { goto } from '$app/navigation';
import { page } from '$app/state';
import { untrack } from 'svelte';

// Back steps: views drawn inside one route (a form opened from a list, a sub-screen) that the
// browser's back button must close exactly like the app's own back button, instead of leaving the
// page. Each open step owns one shallow history entry whose page.state.backStepDepth is its depth.
// The app's back button goes through the history too (goBackStep), so both buttons close a step by
// the same path and no history entry is left behind to swallow a later back press.

interface BackStep {
  depth: number
  pathname: string
  onBack: () => void
}

let openBackSteps: BackStep[] = []
let lastSeenDepth = 0
let lastSeenPathname = ''

const historyDepth = (): number => page.state.backStepDepth || 0

/**
 * Opens a back step: the next back press, from the browser or from goBackStep(), runs onBack.
 * Returns the cleanup that drops the step when its view is destroyed without going back (the page
 * redrawn, another route): call it from onMount or an $effect and return its result.
 */
export function openBackStep(onBack: () => void): () => void {
  const backStep = untrack(() => {
    // Two steps opened in the same tick: the history entry of the first is not written yet.
    const deepestOpenDepth = openBackSteps.reduce((deepest, openStep) => Math.max(deepest, openStep.depth), 0)
    return { depth: Math.max(historyDepth(), deepestOpenDepth) + 1, pathname: page.url.pathname, onBack }
  })
  openBackSteps.push(backStep)
  // persistState: after a reload the entry still knows its depth, so its dead steps get skipped.
  void goto(location.href, { state: { backStepDepth: backStep.depth }, shallow: true, persistState: true })
  return () => { openBackSteps = openBackSteps.filter((openStep) => openStep !== backStep) }
}

/**
 * openBackStep for a view the app may also close by itself (a side layer closed after saving). Its
 * cleanup, when the view closed on its own route without a back press, goes back through the history
 * so the step's entry does not swallow the next back press; onBack then runs, so it must do nothing
 * on a view already closed. Leaving the route only drops the step. genix-ui's Layer uses it
 * (createUiRuntime openBackStep).
 */
export function openSelfClosingBackStep(onBack: () => void): () => void {
  let isClosedByBack = false
  const pathname = untrack(() => page.url.pathname)
  const dropBackStep = openBackStep(() => { isClosedByBack = true; onBack() })
  return () => {
    if (isClosedByBack) { return }
    if (untrack(() => page.url.pathname) === pathname) { goBackStep() } else { dropBackStep() }
  }
}

/** The app's own back button: closes the deepest open step through the history. */
export function goBackStep(): void {
  if (openBackSteps.length > 0) { history.back() }
}

// Runs on every history move, whoever made it.
const closeBackStepsAbove = (depth: number, pathname: string) => {
  if (pathname !== lastSeenPathname) {
    // Another route: its views are gone, their steps close without running their onBack.
    openBackSteps = openBackSteps.filter((openStep) => openStep.pathname === pathname)
    lastSeenPathname = pathname
    lastSeenDepth = depth
    return
  }

  const closedSteps = openBackSteps.filter((openStep) => openStep.depth > depth).sort((a, b) => b.depth - a.depth)
  openBackSteps = openBackSteps.filter((openStep) => openStep.depth <= depth)
  for (const closedStep of closedSteps) { closedStep.onBack() }

  // Going back onto the entry of a step whose view was destroyed without going back (or that a
  // reload forgot): it has nothing to close, so it is skipped instead of costing a back press.
  const isBackward = depth < lastSeenDepth
  lastSeenDepth = depth
  if (isBackward && depth > 0 && !openBackSteps.some((openStep) => openStep.depth === depth)) { history.back() }
}

if (typeof window !== 'undefined') {
  $effect.root(() => {
    $effect(() => {
      const depth = historyDepth()
      const pathname = page.url.pathname
      untrack(() => closeBackStepsAbove(depth, pathname))
    })
  })
}
