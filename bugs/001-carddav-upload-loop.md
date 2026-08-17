Status: [OPEN]

# CardDAV Upload Auto-Retry Loop

## Problem

After a successful VCF upload (HTTP 200), the browser automatically sends a
second POST to `/god/carddav/upload` ~1 second later — without any user
interaction.

The upload works — 303 contacts get imported and the server responds 200. But
the user never sees a clean success state. The phantom second POST triggers
a 429, which renders a red error message on screen. The UX is broken even
though the underlying functionality is not.

### Evidence (server logs)

    carddav import: 303 contacts imported
    POST .../god/carddav/upload ... - 200 41B in 3m5s     ← first completes
    POST .../god/carddav/upload ... - 429 72B in 590ms    ← cooldown blocks second

- First POST: 303 contacts, 3+ minutes, returns `{"imported":303}`
- Second POST: arrives ~1s after first completes, different TCP port (57202 → 33830)
- The 429 cooldown currently blocks it, but root cause is unknown

## What We've Tried

1. **Removed `<form>` element** — replaced with `<div>` to prevent browser form
   resubmission. No change.
2. **Button `type="button"`** — not `type="submit"`. No change.
3. **`busy` flag guard** — JS click handler checks `if (busy) return;`, set
   `true` before XHR, reset in `resetUpload()`. Should prevent re-entry. No change.
4. **Server-side cooldown** — 10-second window after successful upload returns
   429. This **blocks** the duplicate but doesn't fix the root cause. Currently
   the working band-aid.

## What We Know

- **Only trigger**: The click handler on `#upload-btn` sends POST to
  `/god/carddav/upload`. Verified in all JS files.
- **No auto-submit**: No `setTimeout`, no `setInterval`, no programmatic
  `.click()`, no form submission.
- **JS files reviewed**: `god.js` (drawer + logout), `app.js` (palette + login),
  `god-autosave.js` (article form only, guarded by `[data-autosave]`),
  `borum-loader.js` (loading bar, guarded by `form.tagName !== 'FORM'`),
  inline script in `god_carddav.html`.
- **No form on page**: `<form>` was replaced with `<div>`. No `<form>` element exists.
- **Browser not ruled out**: Could be HTTP/1.1 retry behavior for long-running
  POSTs (3+ minutes). Different TCP ports on the two requests suggest separate
  connections.

## Why It's Still Pending

We don't know what sends the second request. Every client-side code path has
been reviewed and none triggers it. Without knowing the source, we can't
fix it — only mask it with the cooldown.

## How to Diagnose

Open DevTools → Network tab → reproduce the upload → click on the second
(429) request → check the "Initiator" column/tab.

That single field will tell you definitively:

- A file + line number → real JS is firing it, somewhere we haven't looked yet
- Other / Parser → it's not page JS at all (browser network stack, extension,
  or something below the DOM)

That's the fork in the road. Everything else is a waste of time until you
know which side you're on.

While you're in there, also grab:

- **Timing tab** on the second request — does it show anything under
  "Connection" that hints it reused/derived from the first request's socket?
- **Headers on both requests** — are User-Agent, Accept, cookies identical?
  A real duplicate-fire will look identical to the original; a browser-generated
  retry may have subtly different headers (missing body-related headers,
  different Content-Length if it's a retry-without-body probe, etc.)
