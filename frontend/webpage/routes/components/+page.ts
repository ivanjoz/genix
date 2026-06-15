// The /components route is a dev/preview gallery, never a deployable page — opt it out
// of prerendering (the +layout otherwise marks every route prerenderable for the
// per-company build). Without this it errors as "marked prerenderable but not found"
// whenever the crawl is scoped to a single page (e.g. the --page-base build).
export const prerender = false;
