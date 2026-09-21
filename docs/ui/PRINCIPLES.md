# UI principles

Originally the navi kit's principles (0.4.0); ticketbot owns this copy.

Requirements, not suggestions. A change that breaks one of these is not done.

## 1. Readability outranks everything

If a choice trades legibility for elegance, legibility wins.

- **Every piece of text clears 4.5:1** against its actual background. 3:1 only
  for text ≥24px, or ≥18.66px bold. Verify with `scripts/contrast-audit.js`;
  do not eyeball it.
- **Never dim text with `opacity`.** Opacity multiplies against whatever is
  behind it and silently destroys contrast. Use a lighter *colour token*
  (`--muted`, `--faint`) instead. Opacity is for whole disabled controls and
  decorative graphics only.
- **Nothing smaller than 11px**, and 11px only for uppercase eyebrow labels
  with letter-spacing. Body text is 14px. Secondary text is 12–13px.
- **Spell out what a symbol means.** A keyboard hint renders as `⌘` `K` on
  Apple and `Ctrl` `K` everywhere else, in separate keycaps, never as a glued
  `⌘K` in 10px mono. Icon-only controls carry `aria-label`. Status is a word
  plus a colour, never colour alone.
- **Numbers are monospace and tabular** (`class="num"`), so columns align and
  digits don't jitter as they update.

## 2. Tokens only

Never write a literal colour, shadow, radius or spacing value in component CSS.

```css
/* wrong */  color: #6F6B64;  padding: 13px;  border: 1px solid #E4E1DB;
/* right */  color: var(--muted);  padding: var(--s3);  border: 1px solid var(--line);
```

Spacing comes from the 4px scale: `--s1` 4 · `--s2` 8 · `--s3` 12 · `--s4` 16 ·
`--s5` 20 · `--s6` 24 · `--s8` 32 · `--s10` 40 · `--s12` 48.

Colour roles, in the order you should reach for them:

| Token | Use for |
|---|---|
| `--ink` | primary text |
| `--ink-2` | secondary text, body copy in cards |
| `--muted` | labels, captions, table headers, timestamps |
| `--faint` | disabled text, placeholders, axis labels |
| `--line` / `--line-soft` | borders, dividers |
| `--bg` / `--bg-sunken` | page and recessed backgrounds |
| `--surface` / `--surface-2` | cards and their subdued areas |
| `--accent` | one primary action per view, active nav, series 1 |
| `--accent-on-soft` | accent-coloured **text** on `--accent-soft` backgrounds |
| `--ok` `--warn` `--bad` `--info` | status, with their `-soft` backgrounds |

`--accent` is for fills. When accent-coloured text sits on `--accent-soft`, use
`--accent-on-soft` — the plain accent fails contrast there.

## 3. One accent, used sparingly

One primary button per view. Everything else is `btn-default` or `btn-ghost`.
An interface where three things compete for the eye has no hierarchy. The
accent also means "current" — active nav, selected rows, series 1 in a chart.

## 4. Hairlines, not heavy shadows

Structure comes from 1px borders and background steps. Shadows only lift things
that genuinely float: menus, modals, drawers, toasts. A card at rest gets
`--shadow-sm` at most.

## 5. Dense, but not cramped

This is an operator's tool; people read a lot of rows. Controls are 34px
(30px in dense contexts, 28px for `btn-sm`). Table cells use `--s3`/`--s4`.
Cards are `--s5`. Page gutters are `--s6`. Don't reduce these to fit more in —
reduce what you're showing instead.

## 6. Motion is feedback, not decoration

180ms on `--ease`, and only to explain a change: a panel sliding in, a row
highlighting, a toggle moving. Page-load stagger is capped at ~5 elements.
Everything collapses to near-zero under `prefers-reduced-motion`, which the kit
already handles — don't add animation that bypasses it.

## 7. Every state is designed

A component isn't done until it has: default, hover, focus-visible, active,
disabled, loading, empty, and error. Empty states say what would appear here
and offer the action that creates it. Errors say what failed and what to do.

## 8. Reachable and operable

- Hit targets ≥24×24px. In dense rows use 24px-wide stacked controls, not 13px.
- `:focus-visible` rings are never removed — the kit styles them globally.
- Icon-only buttons need `aria-label`; inputs need a label or `aria-label`;
  sortable headers need `aria-sort`; the current page needs `aria-current`.
- Anything pinned to the top of the viewport flips its tooltip below
  (`data-tip-pos="bottom"`, automatic inside `.topbar`).

## 9. Responsive by default

The layout works at 380px. The sidebar becomes a drawer below 860px, tables
scroll inside `.table-wrap`, and grids collapse. Never solve a narrow viewport
by hiding information the reader needs.

## 10. Both themes, all palettes

Every change is checked in light **and** dark. If you introduce a colour, it
must be a token so it follows the palette. Test at least `harbor` and `graphite`
— graphite inverts the accent between themes and catches assumptions.
