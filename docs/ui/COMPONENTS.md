# UI components

Originally the navi kit's component reference (0.4.0); ticketbot owns this copy.
The classes live in `internal/web/static/ui.css`.

Copy the markup. Compose these before writing anything new.

Conventions used below: `…` means your content; every `aria-label` shown is
required, not optional.

---

## Layout shell

```html
<div class="app" id="app">
  <aside class="sidebar">
    <div class="brand">
      <div class="brand-mark">n</div>
      <div class="brand-name">Product<sup>v2.1</sup></div>   <!-- sup is optional: a real version, nothing decorative -->
    </div>
    <nav class="nav">
      <div class="nav-group">
        <div class="nav-label eyebrow">Section</div>
        <a class="nav-item active" href="#/overview"><!--icon--><span class="label">Overview</span>
          <span class="pill num">12</span></a>
      </div>
    </nav>
    <div class="sidebar-foot">…</div>
  </aside>
  <div class="main">
    <header class="topbar">…</header>
    <main class="view">…</main>
  </div>
</div>
```

`.app.collapsed` shrinks the sidebar to icons. `.app.nav-open` slides it in on
mobile. Utilities: `.row .stack .spread .grow .wrap .gap1`–`.gap8`.

## Page header

```html
<header class="page-head row spread wrap gap4">
  <div>
    <h1 class="page-title">Customers</h1>
    <p class="page-sub">One sentence of context.</p>
  </div>
  <div class="row gap2 wrap">
    <button class="btn btn-default">Secondary</button>
    <button class="btn btn-primary">Primary</button>
  </div>
</header>
```

## Buttons

```html
<button class="btn btn-primary">Primary</button>   <!-- one per view -->
<button class="btn btn-default">Default</button>
<button class="btn btn-ghost">Ghost</button>
<button class="btn btn-danger">Delete</button>
<button class="btn btn-default btn-sm">Small</button>
<button class="btn btn-default btn-lg">Large</button>
<button class="btn btn-default btn-icon" aria-label="Settings"><!--icon--></button>
<div class="seg"><button class="on">Day</button><button>Week</button></div>
```

## Cards

```html
<article class="card">
  <div class="card-head">
    <div><h3>Title</h3><p>Subtitle</p></div>
    <span class="badge outline">meta</span>
  </div>
  <div class="card-body">…</div>
  <div class="card-foot"><span>left</span><span>right</span></div>
</article>
```

`.card-pad` for a card that is just padding. Grids: `.grid.g2` `.g3` `.g4`
`.g-2-1` `.g-1-2`, gap from `--s5`.

## Stat tile

```html
<article class="card stat">
  <div class="stat-label eyebrow">Net revenue</div>
  <div class="stat-value">$466k</div>
  <div class="stat-foot">
    <span class="delta up"><!--arrow-->+12.4%</span>
    <span class="muted" style="font-size:var(--text-xs)">vs. prior period</span>
  </div>
  <div class="stat-spark"><!-- fluid sparkline svg --></div>
</article>
```

## Badges and status

```html
<span class="badge">Neutral</span>
<span class="badge ok"><i class="dot"></i>Active</span>
<span class="badge warn"><i class="dot"></i>Past due</span>
<span class="badge bad"><i class="dot"></i>Failed</span>
<span class="badge info"><i class="dot"></i>Trial</span>
<span class="badge accent">Beta</span>
<span class="badge outline">Enterprise</span>
<span class="badge ok"><i class="dot pulse"></i>Live</span>
<span class="delta up">+12.4%</span> <span class="delta down">-3.1%</span>
```

Always pair the colour with a word. Never encode state in colour alone.

## Table

Don't write this markup by hand. `dataTable(spec)` in `tables.js` builds the
card, or `tableWrap(spec)` just the `.table-wrap` for a page that owns its
card, from a column model:

```js
dataTable({
  id: 'workflows',                                   // stable: keys the saved layout
  columns: [
    { key: 'board', label: 'Board', sort: w => w.board_name, cell: w => esc(w.board_name) },
    { key: 'steps', label: 'Steps', align: 'r', cls: 'num', sort: w => w.nodes.length, cell: w => w.nodes.length },
    { key: 'updated', label: 'Updated', firstDir: 'desc', sort: w => tblTime(w.updated_on), cell: … },
  ],
  rows: list,
  tr: w => `class="clickable" onclick="openWorkflow(${w.id})"`,
  menu: w => [                                       // the row's kebab; no Actions column
    { label: 'Open', icon: 'edit', run: () => openWorkflow(w.id) },
    { label: 'Delete', icon: 'trash', danger: true, edit: true, run: () => deleteWorkflow(w.id) },  // edit: hidden from viewers
  ],
  sort: { key: 'board', dir: 'asc' },                // optional default
  onSort: sort => refetch(),                         // server-sorted: set sort: true on the columns
  toolbar, empty, foot,
})
```

What it renders:

```html
<div class="table-wrap" data-table="workflows">
  <table class="tbl is-sized" style="width:max(100%, calc(560px + var(--tbl-menu-w)))">  <!-- is-sized + style only with a saved layout -->
    <colgroup><col data-col="board" style="width:240px">…<col data-col="steps"><col class="col-menu"></colgroup>
    <thead><tr>
      <th data-col="board" aria-sort="ascending">
        <button type="button" class="th-sort" data-sort="board">Board<span class="sort" aria-hidden="true">↑</span></button>
        <span class="col-resize" aria-hidden="true"></span>
      </th>
      <th data-col="steps">…</th>
      <th class="th-menu"><button type="button" class="icon-btn" data-table-menu aria-haspopup="menu" aria-label="Column options">…</button></th>
    </tr></thead>
    <tbody>
      <tr class="clickable" onclick="…">
        <td><div class="cell-primary">Name</div><div class="cell-sub">sub</div></td>
        <td class="r num">4</td>
        <td class="td-menu"><button type="button" class="icon-btn" data-row-menu="0" aria-haspopup="menu" aria-label="Row actions">…</button></td>
      </tr>
    </tbody>
  </table>
</div>
```

- A sortable header is a `.th-sort` button: Enter or Space sorts, a second
  press reverses. `aria-sort` sits on the sorted `th` only. The arrow shows on
  the sorted column and on others while hovered or focused.
- `.col-resize` is the column edge: drag to resize, double-click to fit the
  widest cell. Dragging a header moves the column. Both are mouse only; the
  table stays fully usable without them.
- Headers are always left-aligned, even over right-aligned numbers or centred
  badges, so a label never travels with the divider while its column resizes.
- The trailing `.th-menu` column holds the header menu (also on right-click in
  the header): **Reset columns** clears the table's saved sort, widths and
  order.
- A row's actions live in a kebab in that same trailing column (`menu`), not in
  an Actions column. The kebab is caught before the row's own `onclick`, a row
  whose `menu` returns no items has none, and items marked `edit: true` are
  dropped for viewers.
- The first resize snapshots every column's width, and from then on the table
  is `.is-sized`: fixed layout, widths from the `<col>`s, the last column
  takes what is left so the table still fills the card, and a cell narrower
  than its content clips with an ellipsis.
- The layout is saved per `id` in `localStorage` (`tablePrefs:<id>`) and baked
  into the markup on every render, so a poll's `refreshContent` keeps it. A
  poll that lands mid-drag waits until the drag ends.
- The header follows the page scroll under the topbar: `.table-wrap` scrolls
  sideways, so `position: sticky` would stick inside it; `tables.js` sets
  `--thead-y` on the wrap instead. Below 860px the header scrolls away.

Toolbar above (`.toolbar`), `.bulkbar` when rows are selected, `.card-foot`
below with the count and `.pagination`. Pagination buttons need
`aria-label="Page 3"` and `aria-current="page"`.

## Forms

```html
<div class="field">
  <label>Workspace name</label>
  <input class="input" aria-label="Workspace name">
  <span class="hint">Helper text.</span>
  <span class="err">Error text.</span>      <!-- with .input.invalid -->
</div>

<div class="input-group"><!--icon--><input class="input" placeholder="Search…"></div>
<div class="input-affix"><span>https://</span><input class="input" aria-label="Domain"></div>
<select class="select" aria-label="Region">…</select>
<textarea class="textarea"></textarea>

<label class="check"><input type="checkbox"><span class="box"><!--check icon--></span>
  <span>Label</span></label>
<label class="check radio"><input type="radio" name="g"><span class="box"></span><span>Label</span></label>
<label class="switch"><input type="checkbox"><span class="track"><span class="thumb"></span></span>
  <span>Label</span></label>
```

`.switch.sm` for dense rows. **`.track` must stay `display:flex`** — the thumb
is its child and an inline parent collapses it to 0×0.

Settings rows: `.form-row` — description left, control right.

## Tabs

```html
<div class="tabs" data-tabs="name">
  <button class="on" data-tab="a">First</button>
  <button data-tab="b">Second</button>
</div>
<div data-panel="a">…</div>
<div data-panel="b" hidden>…</div>
```

## Feedback

```html
<div class="callout warn"><!--icon--><div class="body"><b>Title</b>Detail.</div></div>
<div class="banner bad"><!--icon--><div><b>Title</b> Detail.</div>
  <button class="btn btn-default btn-sm">Action</button></div>
```

`.callout` is scoped inside a card; `.banner` is full-width and page-level.
`.banner-sticky` pins under the topbar. Variants: `info` `warn` `bad` `ok`.

## Overlays

`.modal` + `.scrim.on`, `.drawer.on`, `.cmdk.on`, `.toasts > .toast[.ok|.bad]`,
`.menu` (popover), `[data-tip]` (tooltip). Tooltips render above by default;
add `data-tip-pos="bottom"` near the top of the viewport (automatic inside
`.topbar`), and `data-tip-align="left|right"` near a side edge.

## Panels docked over a canvas

For a map, a flow editor or any pane where the content should keep the full
width and the controls float on top of it. The parent needs
`position: relative`; set the width you want on `.dock`.

```html
<div class="canvas" style="position:relative">
  …content…
  <aside class="dock dock-l" style="width:190px">
    <article class="card">
      <div class="card-head"><div><h3>Steps</h3></div>
        <button class="icon-btn" aria-label="Hide the step list"><!--arrow left--></button></div>
      <div class="card-body">…</div>
    </article>
  </aside>

  <div class="dock-bar bl">
    <button class="icon-btn" aria-label="Zoom out"><!--minus--></button>
    <span class="num">75%</span>
    <button class="icon-btn" aria-label="Zoom in"><!--plus--></button>
  </div>
  <div class="dock-bar br"><span>drag to pan</span></div>
</div>
```

`.dock-l` / `.dock-r` pick the side; `.dock-bar` takes a corner —
`.tl .tr .bl .br`. A dock scrolls inside its own `.card-body`, so the card head
and foot stay put. Show a dock only when it has something to say: an inspector
with nothing selected is a column of nothing.

## A pointer drag in progress

Put `.is-dragging` on the shell for the length of the gesture, and take it off
on pointer-up. It stops the drag turning into a text selection that runs through
everything the pointer crosses, and gives every element under the pointer one
cursor — the drag's own. Docked chrome stops taking pointer events, so a drag
passes over a panel instead of ending on it.

```html
<div class="app is-dragging">…</div>           <!-- grabbing -->
<div class="app is-dragging drag-link">…</div> <!-- crosshair: wiring something up -->
<div class="app is-dragging drag-copy">…</div> <!-- copy: dragging a new item in -->
<div class="app is-dragging drag-resize">…</div> <!-- col-resize: a table column's edge -->
```

A table column being moved gets `.is-col-source` on its header, a `.tbl-ghost`
label follows the pointer and a `.tbl-drop` line marks where it will land;
both hang off `<body>`.

`assets/reorder.js` does not need this — it captures the pointer on a handle.
Canvas-style drags, which have no single capture target, do.

## Empty and loading

```html
<div class="empty">
  <div class="empty-art"><!--icon--></div>
  <div class="stack gap2"><h3>No invoices yet</h3><p>What would appear here.</p></div>
  <button class="btn btn-primary btn-sm">Create invoice</button>
</div>

<div class="skeleton" style="height:9px;width:60%"></div>
```

## Keyboard hints

```html
<span class="keys"><span class="kbd">⌘</span><span class="sep">+</span><span class="kbd">K</span></span>
```

Generate the label per platform — `⌘`/`⇧`/`⌥` on Apple, `Ctrl`/`Shift`/`Alt`
elsewhere — and give the control an `aria-label` that spells it out
("press Command K"). Never ship a glued `⌘K` glyph as the only hint.

---

# Console components

## Ordered rule cards

An ordered chain, not a stack of forms. Cards sit in a `.rule-list`, a connector
is drawn in the gap between them, and a card is **collapsed unless it carries
`.is-open`** — the head alone says what the rule does, via `.rule-sum`.

```html
<div class="rule-list">
  <article class="rule-card is-open">
    <div class="rule-head">
      <button class="rule-grip" aria-label="Drag to reorder rule 1"><!--grip dots--></button>
      <div class="order-btns">
        <button aria-label="Move rule up"><!--chevron up--></button>
        <button aria-label="Move rule down"><!--chevron down--></button>
      </div>
      <span class="rule-index">1</span>
      <input class="rule-name" value="Rule name" aria-label="Rule name">
      <span class="badge outline">stops chain</span>
      <span class="rule-sum cell-sub">New and updated · 2 conditions · 1 action</span>
      <label class="switch"><input type="checkbox"><span class="track"><span class="thumb"></span></span></label>
      <button class="btn btn-ghost btn-sm">Delete</button>
      <button class="rule-toggle icon-btn" aria-expanded="true" aria-label="Collapse rule 1"><!--chevron--></button>
    </div>
    <div class="rule-body">…</div>
  </article>
  <article class="rule-card">…collapsed: no .is-open…</article>
</div>
```

| Class | What it is |
|---|---|
| `.rule-list` | The container. `position: relative` — the ghost and drop line live in it. |
| `.is-open` | On the card. Without it the body is hidden and the head is the whole card. |
| `.rule-sum` | One-line summary, shown only while the card is closed. |
| `.rule-toggle` | Expand/collapse. Set `aria-expanded`; the chevron rotates on `.is-open`. |
| `.rule-grip` | Drag handle. Keep `.order-btns` too — the grip is pointer-only. |
| `.is-disabled` | Dims the card; independent of open/closed. |

Dragging is `assets/reorder.js`, an optional dependency-free helper. It styles
the lifted card `.is-source`, floats a `.drag-ghost` (a clone of the head) under
the pointer and shows a `.drop-line` at the insertion point. It reports the move
and never touches the DOM order itself:

```js
import { reorder } from './reorder.js'

reorder(document.querySelector('.rule-list'), {
  item: '.rule-card',
  handle: '.rule-grip',
  onMove: (from, to) => { /* reorder your data, re-render */ },
})
```

`.action-row` is the nested equivalent for a rule's actions. An action whose
fields need a second line — a message textarea, a row of flags — puts that
content in a sibling `.action-detail`, not inside the field stack:

```html
<div class="action-row">
  <div class="order-btns">…</div>
  <label class="switch sm">…</label>
  <select class="select" aria-label="Action type">…</select>
  <select class="select" aria-label="Target">…</select>
  <button class="icon-btn" aria-label="Remove action"><!--trash--></button>
  <div class="action-detail">
    <textarea class="textarea mono" aria-label="Custom message"></textarea>
  </div>
</div>
```

A taller child inside the field stack makes `align-items: center` centre the
whole stack, which lifts the first field off the row's centre line.
`.action-detail` is `flex-basis: 100%`, indented by `--action-indent` to start
under the type select.

## Condition builder

```html
<div class="cond">
  <div class="cond-tabs"><button class="on">Builder</button><button>Advanced</button>
    <span class="grow"></span><span class="cell-sub">2 conditions</span></div>
  <div class="cond-rows">
    <div class="cond-row">
      <span class="cond-join-spacer"></span>
      <select class="select" aria-label="Field">…</select>
      <select class="select" aria-label="Operator">…</select>
      <input class="input" aria-label="Value">
      <button class="icon-btn" aria-label="Remove condition"><!--trash--></button>
    </div>
    <div class="cond-row">
      <select class="cond-join" aria-label="Join"><option>and</option><option>or</option></select>
      <!-- or, where the join is set once for the whole group: -->
      <span class="cond-join">and</span>
      …
      <div class="chips typeahead">
        <span class="chip">Growth<button class="chip-x" aria-label="Remove Growth"><!--x--></button></span>
        <button class="chip-add">+ value</button>
      </div>
    </div>
  </div>
  <div class="cond-foot">
    <pre class="code">…</pre>
    <button class="btn btn-default btn-sm">Test</button>
    <span class="cond-result ok">✓ matches 31 records</span>
  </div>
</div>
```

Typeahead popup: `.typeahead-pop` inside a `.typeahead` container.

`.cond` is a container query root, and so takes its width from its parent
rather than its contents — it is `width: 100%` for that reason; do not remove
it. Under 430px — a side panel, a `.dock`, a phone — each row stacks and the
join moves to a line of its own. Same markup either way; do not write a narrow
variant.

## Log stream

```html
<div class="filter-bar">
  <select class="select" aria-label="Filter by level">…</select>
  <div class="input-group"><!--icon--><input class="input" placeholder="Search…"></div>
  <div class="grow"></div>
  <span class="badge ok"><i class="dot pulse"></i>Streaming</span>
</div>
<div class="log-list">
  <div class="log-row error">
    <span class="log-time">14:05:12</span>
    <span class="log-level">ERROR</span>
    <span class="log-msg">billing: charge failed<span class="log-attr"><b>status</b>=402</span></span>
  </div>
</div>
```

Row variants: default, `.warn`, `.error`, `.debug`.

## Typed event history

```html
<div class="events">
  <div class="event bad">
    <div class="event-card">
      <div class="event-head"><span class="title">Payment failed</span>
        <span class="src">billing</span><span class="time">09:12:55</span></div>
      <div class="event-body">Retried 3 times.</div>
      <pre class="code">POST /hooks → 502</pre>
    </div>
  </div>
</div>
```

Tones: none, `ok`, `warn`, `bad`, `accent`. Lighter alternative: `.timeline`
with `.tl-item`.

## Field diff

```html
<table class="diff-table">
  <thead><tr><th>Field</th><th>Was</th><th>Now</th></tr></thead>
  <tbody><tr><td class="field">Plan</td>
    <td><span class="diff-old">Growth</span></td>
    <td><span class="diff-new">Scale</span></td></tr></tbody>
</table>
```

## Record detail

```html
<div class="back-row">
  <a class="back-link" href="#/list"><!--left arrow-->Back to list</a>
  <span class="muted">/</span><span class="cell-sub num">CUS-4821</span>
</div>
<div class="meta-grid">
  <div><div class="eyebrow">MRR</div><div class="val num">$3,593</div></div>
</div>
```

The grid's hairlines are the container showing through a 1px gap, so the last
cell spans whatever is left of its row — otherwise a count that does not fill
the row leaves a bare strip of `--line` where the missing cells would be.

## Secrets

```html
<div class="secret reveal">sk_live_…<button class="icon-btn" aria-label="Copy key">…</button></div>
<div class="secret">sk_live_••••7f2a</div>
<div class="recovery-codes"><span>4f2a-90bd</span>…</div>
<div class="qr">…</div>
```

## Auth screens

```html
<div class="auth">
  <div class="auth-card">
    <div class="auth-brand"><div class="brand-mark">M</div><div class="brand-name">Product</div></div>
    <div><h1>Sign in</h1><p>Enter your credentials.</p></div>
    <div class="field"><label>Email</label><input class="input" type="email"></div>
    <button class="btn btn-primary btn-lg">Continue</button>
    <div class="auth-foot">…</div>
  </div>
</div>
```

One-time code: `.input.otp-input`. Password rules: `.pwd-reqs > .pwd-req[.ok]`.

## Board

`.board > .board-col > .board-col-head + .board-drop > .board-card`.
`.board-card.dragging` and `.board-col.drop-target` during a drag.

## Code and quoted text

```html
<pre class="code">plain block</pre>
<code class="code inline">inline</code>
<div class="note"><div class="note-author">Author</div><div class="note-text">…</div></div>
```

Code token spans: `.tok-field` `.tok-op` `.tok-val` `.tok-join`.

## Charts

`assets/charts.js` provides `areaChart`, `barChart`, `donutChart`, `sparkline`
and `rankBars`. They emit plain SVG using `var(--chart-1…6)`, so they follow the
palette automatically. Series 1 is always the accent. Add `.meter` for a simple
inline progress bar.
