# Palettes

Six presets. Switch with one attribute on `<html>`; nothing else changes.

```html
<html data-theme="light" data-palette="ember">
```

| Palette | Character | Light accent | Dark accent |
|---|---|---|---|
| `harbor` | Cool grey, deep teal accent — **the default** | `#0E6F80` | `#3FB0C4` |
| `ember` | Warm paper, persimmon accent | `#C64A26` | `#F26B3E` |
| `indigo` | Slate neutrals, indigo accent | `#3D45A8` | `#8288E8` |
| `moss` | Sage neutrals, forest green accent | `#3E6B43` | `#6FB277` |
| `plum` | Mauve neutrals, plum accent | `#8E3A63` | `#D97BA8` |
| `graphite` | Monochrome, ink accent | `#26251F` | `#EDEAE0` |

Ask the user which one on first adoption in a project. Default to `harbor` if
they don't care. Record the answer as `data-palette` on `<html>` in `index.html`.

## How a palette is defined

Eight declarations. Everything else in the system is mixed from them.

```css
:root[data-palette="harbor"] {
  --seed:      #0E6F80;   /* accent in light mode  */
  --seed-dark: #3FB0C4;   /* accent in dark mode   */
  --tint:      #5E7480;   /* hue mixed into greys  */
  --c2: #B4531F; --c3: #3E7D3A; --c4: #A8770F; --c5: #5C5FA6; --c6: #8B949B;
}
```

- `--tint` tints every neutral: `--bg`, `--surface`, `--ink`, `--muted`,
  `--line` and the shadow colour are all `color-mix`ed from it. Warm tints give
  paper; cool tints give slate.
- `--seed` / `--seed-dark` drive `--accent` and its soft/line/ink variants.
  Two seeds rather than one because a hue that reads well on white is usually
  too dark on near-black, and vice versa — `graphite` inverts completely.
- `--c2`…`--c6` are categorical chart colours. Chart series 1 is always the
  accent, so `--c2`…`--c6` must be distinguishable **from the accent and from
  each other**, not just pretty.

## Adding a palette

1. Add one block like the above to the palettes section of `ui.css`.
2. Pick `--seed` dark enough for white text (light mode) and `--seed-dark`
   light enough for dark text (dark mode).
3. Verify: run `scripts/contrast-audit.js` on a page under the new palette in
   both themes. Zero failures, or the palette isn't finished.
4. Add it to the table above and to any in-app palette chooser.

## Changing the default

The default lives in two places — keep them in sync:

- the `:root { --seed / --seed-dark / --tint / --c2…--c6 }` block at the top of
  `ui.css` (what renders before `data-palette` is set);
- the `data-palette` attribute written onto `<html>`.
