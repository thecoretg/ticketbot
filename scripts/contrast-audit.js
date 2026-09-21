/* Readability audit
 *
 * Paste into the browser console on any ticketbot page, or run it
 * through a headless driver. It reports, for the page as rendered:
 *
 *   1. text that fails WCAG AA contrast against its real background
 *   2. tooltips that would render off-screen
 *   3. interactive elements with no accessible name
 *   4. hit targets under 24x24
 *
 * Returns a summary object and prints tables. Zero failures is the bar.
 *
 *   await uiAudit()                       // current theme + palette
 *   await uiAudit({ allPalettes: true })  // every palette x both themes
 */
(function () {
  const PALETTES = ['ember', 'harbor', 'indigo', 'moss', 'plum', 'graphite'];

  const parseColor = c => {
    if (!c) return null;
    let m = c.match(/^rgba?\(([^)]+)\)/);
    if (m) { const p = m[1].split(/[\s,\/]+/).filter(Boolean).map(Number); return { r: p[0], g: p[1], b: p[2], a: p[3] ?? 1 }; }
    m = c.match(/^color\(srgb ([^)]+)\)/);
    if (m) { const p = m[1].split(/[\s\/]+/).filter(Boolean).map(Number); return { r: p[0] * 255, g: p[1] * 255, b: p[2] * 255, a: p[3] ?? 1 }; }
    return null;
  };
  const luminance = ({ r, g, b }) => {
    const f = v => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
    return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b);
  };
  const contrast = (a, b) => {
    const l1 = luminance(a), l2 = luminance(b);
    return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
  };
  const over = (fg, bg) => ({
    r: fg.r * fg.a + bg.r * (1 - fg.a),
    g: fg.g * fg.a + bg.g * (1 - fg.a),
    b: fg.b * fg.a + bg.b * (1 - fg.a), a: 1,
  });
  const backgroundOf = el => {
    let n = el;
    while (n && n !== document.documentElement) {
      const c = parseColor(getComputedStyle(n).backgroundColor);
      if (c && c.a > 0.05) return c;
      n = n.parentElement;
    }
    return parseColor(getComputedStyle(document.body).backgroundColor) || { r: 255, g: 255, b: 255, a: 1 };
  };
  const label = el =>
    (el.getAttribute('aria-label') || el.getAttribute('title') || el.getAttribute('data-tip') ||
     el.textContent.trim() || el.getAttribute('placeholder') ||
     (el.labels && el.labels.length ? el.labels[0].textContent.trim() : '') || '').trim();

  function scanOnce() {
    const contrastFails = [], tipFails = [], nameFails = [], targetFails = [];

    document.querySelectorAll('body *').forEach(el => {
      const cs = getComputedStyle(el);
      if (cs.visibility === 'hidden' || cs.display === 'none') return;
      if (parseFloat(cs.opacity) < 0.05) return;  // fully transparent: nothing to read
      const rect = el.getBoundingClientRect();
      if (!rect.width || !rect.height) return;

      // 1. contrast — only elements that own visible text
      const own = [...el.childNodes]
        .filter(n => n.nodeType === 3 && n.textContent.trim())
        .map(n => n.textContent.trim()).join(' ');
      if (own) {
        const fg = parseColor(cs.color), bg = backgroundOf(el);
        if (fg && bg) {
          let eff = fg.a < 1 ? over(fg, bg) : fg;
          const op = parseFloat(cs.opacity);
          if (op < 1) eff = over({ ...eff, a: op }, bg);
          const px = parseFloat(cs.fontSize);
          const large = px >= 24 || (px >= 18.66 && parseInt(cs.fontWeight) >= 600);
          const need = large ? 3 : 4.5;
          const cr = contrast(eff, bg);
          if (cr < need) contrastFails.push({
            element: (el.className || el.tagName).toString().slice(0, 44),
            text: own.slice(0, 40), size: px + 'px',
            ratio: +cr.toFixed(2), needs: need,
          });
        }
      }

      // 2. tooltip geometry
      if (el.hasAttribute('data-tip')) {
        const below = !!el.closest('.topbar') || el.getAttribute('data-tip-pos') === 'bottom';
        const align = el.getAttribute('data-tip-align');
        const w = el.getAttribute('data-tip').length * 6.4 + 18, h = 23;
        const top = below ? rect.bottom + 7 : rect.top - 7 - h;
        const left = align === 'right' ? rect.right - w
                   : align === 'left' ? rect.left
                   : rect.left + rect.width / 2 - w / 2;
        const bad = [];
        if (top < 0) bad.push('off top');
        if (rect.top < innerHeight && top + h > innerHeight) bad.push('off bottom');
        if (left < 0) bad.push('off left');
        if (left + w > innerWidth) bad.push('off right');
        if (bad.length) tipFails.push({ tip: el.getAttribute('data-tip'), problem: bad.join(' + '), fix: below ? 'use data-tip-align' : 'add data-tip-pos="bottom"' });
      }

      // 3 + 4. accessible name, hit target
      if (el.matches('button, a[href], input, select, textarea, [role="button"]')) {
        if (!label(el)) nameFails.push({ element: (el.className || el.tagName).toString().slice(0, 44), html: el.outerHTML.slice(0, 80) });
        const expanded = el.classList.contains('hit-expand') || el.closest('.order-btns') || el.classList.contains('rule-grip');
        if (!expanded && el.matches('button, [role="button"]') && (rect.width < 24 || rect.height < 24))
          targetFails.push({ element: (el.className || el.tagName).toString().slice(0, 44), size: `${Math.round(rect.width)}x${Math.round(rect.height)}` });
      }
    });
    return { contrastFails, tipFails, nameFails, targetFails };
  }

  window.uiAudit = async function ({ allPalettes = false } = {}) {
    const root = document.documentElement;
    const startTheme = root.dataset.theme, startPalette = root.dataset.palette;
    const combos = allPalettes
      ? PALETTES.flatMap(p => ['light', 'dark'].map(t => [p, t]))
      : [[startPalette || 'ember', startTheme || 'light']];

    const worst = new Map(), tips = new Map(), names = new Map(), targets = new Map();
    for (const [pal, theme] of combos) {
      root.dataset.palette = pal; root.dataset.theme = theme;
      // plain timeout, not rAF: rAF never fires in a background tab
      await new Promise(r => setTimeout(r, 40));
      // let entry animations land, or a fading-in element reads as zero contrast
      await Promise.race([
        Promise.allSettled(document.getAnimations().map(a => a.finished)),
        new Promise(r => setTimeout(r, 600)),
      ]);
      const res = scanOnce();
      res.contrastFails.forEach(f => {
        const k = f.element + ' :: ' + f.text;
        if (!worst.has(k) || worst.get(k).ratio > f.ratio) worst.set(k, { ...f, where: `${pal}/${theme}` });
      });
      res.tipFails.forEach(f => tips.set(f.tip, f));
      res.nameFails.forEach(f => names.set(f.html, f));
      res.targetFails.forEach(f => targets.set(f.element + f.size, f));
    }
    root.dataset.theme = startTheme; root.dataset.palette = startPalette;

    const out = {
      contrast: [...worst.values()].sort((a, b) => a.ratio - b.ratio),
      tooltips: [...tips.values()],
      missingNames: [...names.values()],
      smallTargets: [...targets.values()],
    };
    const total = out.contrast.length + out.tooltips.length + out.missingNames.length + out.smallTargets.length;

    console.log(`%cui audit — ${combos.length} combination(s), ${total} issue(s)`,
      `font-weight:bold;color:${total ? '#c0392b' : '#2e7d5b'}`);
    if (out.contrast.length)     { console.log('%cContrast failures', 'font-weight:bold'); console.table(out.contrast); }
    if (out.tooltips.length)     { console.log('%cTooltips off-screen', 'font-weight:bold'); console.table(out.tooltips); }
    if (out.missingNames.length) { console.log('%cNo accessible name', 'font-weight:bold'); console.table(out.missingNames); }
    if (out.smallTargets.length) { console.log('%cHit targets under 24px', 'font-weight:bold'); console.table(out.smallTargets); }
    if (!total) console.log('%c✓ clean', 'color:#2e7d5b');
    return out;
  };

  console.log('uiAudit() ready — call it, or uiAudit({ allPalettes: true })');
})();
