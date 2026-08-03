# Nesio UI Global Style Guidelines

This document is mandatory for all future frontend changes.

## Scope

- Applies to all screens, components, popups, and form controls.
- Priority target device: iPhone 17 Pro class width (393 logical px), while remaining responsive for nearby mobile widths.

## 1. Color Rules

- Use only Nesio design tokens via Tailwind aliases:
- `bg-nesio-bg`, `bg-nesio-card`
- `text-nesio-ink`, `text-nesio-muted`, `text-nesio-accent`
- `border-nesio-border`
- `bg-nesio-accent`, `bg-nesio-accentSoft`, `bg-nesio-accentLight`, `bg-nesio-icon-bg`
- Forbidden in page-level TSX:
- hardcoded hex colors like `#xxxxxx`
- framework palette colors like `text-red-500`, `bg-blue-50`, `text-slate-500` unless explicitly mapped to tokenized utility classes.

## 2. Shared Component Classes

Always prefer shared classes from `src/styles/index.css`:

- Cards:
- `ui-card` (card + border)
- `ui-card-plain` (card without border emphasis)

- Buttons:
- `ui-btn` (base)
- `ui-btn-primary`
- `ui-btn-secondary`
- `ui-btn-ghost`
- `ui-icon-btn`

- Inputs:
- `ui-input`

- Chips:
- `ui-chip`
- `ui-chip-active`

Do not reimplement button/input/card/chip visuals with one-off long class strings if a shared class exists.

## 3. Typography Scale

Use type classes instead of arbitrary `text-[xx]` values:

- `type-display`
- `type-h1`
- `type-h2`
- `type-title`
- `type-body`
- `type-caption`

Forbidden:
- `text-[40px]`, `text-[52px]`, `text-[64px]`, or any large one-off size without explicit design review.

## 4. Radius and Spacing Rules

- Use standard radii from design scale:
- `rounded-xl`, `rounded-2xl`, `rounded-3xl`, `rounded-full`
- Avoid one-off custom radii like `rounded-[34px]`, `rounded-[36px]` in normal UI.

- Controls must maintain readable touch sizes:
- default control height from shared classes (`ui-btn`, `ui-input`, `ui-chip`)

## 5. Mobile Layout Safety (No Overflow)

- Root container must use `app-frame` to keep a stable mobile viewport.
- Scrollable route content must use `app-content-safe` to respect top safe area.
- Bottom navigation and fixed bottom actions must use `tabbar-safe` to respect home indicator area.
- No UI element may overflow horizontally.
- Use `min-w-0`, `truncate`, `break-word`, and wrapping where needed.
- Buttons must not clip text.
- Inputs/textareas must not collapse content.
- Long text should wrap or truncate with safe affordance.

## 6. iPhone 17 Pro Focus

- Layout and spacing should be visually balanced around 393 px width.
- Keep key action buttons within thumb reach and fully visible.
- Avoid giant typography that forces clipping on 393 px width.
- Validate compact states where dynamic data can become long.

## 7. Engineering Compliance Checklist

Before merge, verify:

1. No hardcoded color hex in edited TSX files.
2. No new `text-[xx]` or `rounded-[xx]` one-off values unless reviewed.
3. Forms use `ui-input`; actions use `ui-btn-*`.
4. Cards use `ui-card` or `ui-card-plain`.
5. No horizontal overflow on mobile viewport.
6. Build passes (`npm run build`).
7. UI style guard passes (`npm run lint:ui`).

## 9. CI / Local Guardrail

- Every frontend PR must run `npm run check:ui`.
- `check:ui` includes both style guard and build.
- Violations (hex colors, `text-[...]`, `rounded-[...]`, non-token palette classes) must be fixed before merge.

## 8. Exceptions

- Exceptions require explicit design-note comments and should be rare.
- If an exception is introduced, update this document with rationale.
