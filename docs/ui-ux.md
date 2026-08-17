# UI/UX Patterns

This document defines reusable UI patterns for the admin interface.

## Admin Cards

Standard card class for all admin boxes, stats, and sections:

```
rounded border border-hairline !p-6
```

- Use `!p-6` (Tailwind `!important` prefix) to override global CSS resets
- Use `mb-8` between card sections
- Use `gap-6` in grids inside cards

## Section Spacing

- Between major sections: `mb-8`
- Between subsections: `mb-6`
- Inside cards: `p-6`

## Typography

- Card titles: `font-mono text-[12px] uppercase tracking-wider text-faint`
- Card values: `font-mono text-[20px] font-bold`
- Card labels: `font-mono text-[11px] text-faint`

## Examples

### Stats Card

```html
<div class="rounded border border-hairline !p-6">
    <label class="mb-2 block font-mono text-[11px] text-faint">Articles</label>
    <p class="font-mono text-[20px] font-bold">{{.ArticleCount}}</p>
</div>
```

### Content Card

```html
<div class="rounded border border-hairline !p-6">
    <h3 class="mb-3 font-mono text-[12px] uppercase tracking-wider text-faint">Title</h3>
    <p class="font-mono text-[13px]">Content here</p>
</div>
```

### Grid Layout

```html
<div class="grid grid-cols-2 gap-6 md:grid-cols-4">
    <div class="rounded border border-hairline !p-6">...</div>
    <div class="rounded border border-hairline !p-6">...</div>
</div>
```
