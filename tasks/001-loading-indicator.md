Status: [IN_PROGRESS]

# Top loading indicator between page navigations (global, reusable)

## Context

The site does full page loads (server-rendered Go/chi). There is no client-side
router, so there is no built-in hook to show the user that a navigation is in
flight. oldrare solves this with a top sweep bar driven by SvelteKit's
beforeNavigate/afterNavigate (see oldrare/src/routes/+layout.svelte,
src/lib/stores/navigation.svelte.ts, and the loader-sweep keyframes in
src/app.css). This task ports that UX to borum-api with plain JS + CSS.

The loader must be **global and easily reusable**: a single self-contained
module that any page, script, or future feature (palette, XHR, new experiments)
can show/hide with one function call, with no wiring per page.

## Requirements

1. **Self-contained loader module (static/app.js)**
   - A tiny public API, e.g. `window.BorumLoader.show()` / `.hide()` (or an
     equivalent namespaced object), exposed once. It is idempotent: repeated
     `show()` calls do not duplicate the element or restart the animation
     awkwardly; `hide()` removes it.
   - The element is created lazily on first `show()` and appended to
     `document.body`, so pages that never navigate pay nothing.
   - Degrades silently if the module is missing (every call is a no-op), so
     future code can call it without feature detection at each call site.

2. **Sweep bar look + animation**
   - A fixed, `pointer-events-none` 4px bar pinned to the top of the viewport
     (z-index above content), showing a half-width gradient pill that sweeps
     left→right infinitely, matching oldrare's `loader-sweep` keyframes
     (`translateX(-100%)` → `translateX(100%)`, ~1.1s cubic-bezier, infinite).
   - Keyframes live in `static/css/animations.css` (with the other keyframes).
   - Bar colors come from existing theme tokens (CSS variables) so light + dark
     themes both work with no extra CSS.
   - `aria-hidden="true"` and non-focusable: invisible to assistive tech.

3. **Global triggers (wired once, site-wide)**
   - Document-level `click` handler: on any same-origin `a[href]` that is a real
     navigation (not hash-only, not `target=_blank`/`download`, not
     modified-click i.e. ctrl/cmd/shift/alt/middle), `show()` immediately
     before the browser leaves the page.
   - Document-level `submit` handler: `show()` for same-origin form posts
     (login, /god create/edit/delete, experiment toggle/move/intro).
   - Do NOT trigger for the img2webp form (XHR with its own progress bar).
   - Palette-driven navigations that set `location.href` (:home, :raft, tag /
     article opens) call `BorumLoader.show()` too, so the palette never bypasses
     the indicator.

4. **Hide on arrival**
   - The bar lives in the old document, so the new page simply paints without
     it; `hide()` is called on `pageshow` (covers bfcache restore) so a
     back/forward navigation never leaves a stale bar.
   - Optional smooth fade-out: set a sessionStorage flag right before
     navigating; a tiny inline script in `base.html` and `god_base.html` reads
     it on load and plays a quick fade before clearing it. Keep this optional —
     the hard requirement is no flash of the bar on first load or bfcache
     restore.

5. **Non-intrusive**
   - No layout shift (fixed, h-1), no console errors on any page, works on
     mobile + desktop, and works identically on the public layout (`base.html`)
     and the admin layout (`god_base.html`).

## Acceptance criteria

- Clicking any internal link (nav, articles, tags, god drawer) shows the sweep
  bar immediately; it disappears when the next page paints.
- Modified clicks, middle clicks, `target=_blank`, external, and hash-only links
  do NOT show the bar.
- Form posts (login, god article create/edit/delete, experiment controls) show
  the bar.
- The palette's :home / :raft / tag / article navigations show the bar.
- No flash of the bar on first page load or on back/forward (bfcache).
- A future feature can show the indicator with a single
  `BorumLoader.show()` call — no per-page wiring required.
- `go vet` passes; no new JS console errors across home, article, tag, login,
  god, and experiment pages.

## Progress

- 2026-08-15: Task opened. Confirmed approach: sweep bar (oldrare-style).
  Emphasized global, self-contained loader module usable site-wide.
