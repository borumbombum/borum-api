# tiptap (vendored)

The article body editor runs on TipTap. The whole editor is bundled into a
single self-contained IIFE file (`tiptap.bundle.js`) that exposes a `TipTap`
global. The app serves that file directly — no build step, no CDN, no network.

## Version

Pinned `@tiptap/*` at `3.30.1` (core, starter-kit, extension-link,
extension-image, extension-placeholder). StarterKit already includes
Underline — do not add `extension-underline` separately (TipTap throws on
duplicate extensions).

Exported by the `TipTap` global: `Editor`, `StarterKit`, `Link`, `Image`,
`Placeholder`.

## Rebuilding (only needed to upgrade or change the set of extensions)

Requires node + npm. Build happens in a scratch directory, never in the repo:

    npm install --prefix /tmp/opencode/tiptap-build @tiptap/core@3.30.1 @tiptap/starter-kit@3.30.1 @tiptap/extension-link@3.30.1 @tiptap/extension-image@3.30.1 @tiptap/extension-placeholder@3.30.1
    npx esbuild static/vendor/tiptap/entry.js --bundle --minify --format=iife --global-name=TipTap --outfile=static/vendor/tiptap/tiptap.bundle.js

`entry.js` in this directory is the bundle entry; change the imports there
first if you add or remove extensions, then rebuild.

## License

TipTap is MIT licensed. See https://tiptap.dev for details.
