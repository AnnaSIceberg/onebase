(function () {
  const main = document.querySelector('main.ob-object-form');
  const form = main && main.querySelector('#main-form');
  if (!form || form.hasAttribute("data-ob-compact-labels")) return;

  let table;
  let originalHeight = '';
  function setHeight(element, value) {
    if (element.style.height === value) return;
    element.style.height = value;
    const grid = (window._obGrids || {})[element.getAttribute('data-sg-tp')];
    if (grid && grid.div === element && element.getClientRects().length) grid.grid.resizeCanvas();
  }

  function fit() {
    const visible = Array.from(form.querySelectorAll('[data-sg-tp], .ob-object-table-scroll'))
      .filter(element => element.getClientRects().length);
    const nativeFit = Array.from(form.querySelectorAll('[data-ob-fit-window="1"], [data-ob-fit-group="1"], [data-ob-fit-grid="1"]'))
      .some(element => element.getClientRects().length);
    main.classList.toggle('ob-object-form-native-fit', !!nativeFit);
    const next = nativeFit ? undefined : visible[0];
    if (next !== table) {
      if (table) setHeight(table, originalHeight);
      table = next;
      originalHeight = table ? table.style.height : '';
    }
    if (!table) return;

    // Measure the natural card before its flex growth pushes the footer down.
    main.classList.add('ob-object-form-measuring');
    try {
      const top = main.getBoundingClientRect().top;
      let bottom = top;
      for (const child of main.children) {
        if (!child.getClientRects().length) continue;
        const margin = parseFloat(getComputedStyle(child).marginBottom) || 0;
        bottom = Math.max(bottom, child.getBoundingClientRect().bottom + margin);
      }
      const padding = parseFloat(getComputedStyle(main).paddingBottom) || 0;
      const naturalHeight = bottom - top + main.scrollTop + padding;
      const height = Math.max(120, Math.floor(table.offsetHeight + main.clientHeight - naturalHeight));
      setHeight(table, height + 'px');
    } finally {
      main.classList.remove('ob-object-form-measuring');
    }
  }

  let queued = false;
  function queueFit() {
    if (queued) return;
    queued = true;
    requestAnimationFrame(() => {
      queued = false;
      fit();
    });
  }
  if (window.ResizeObserver) {
    const resize = new ResizeObserver(queueFit);
    resize.observe(main);
    for (const child of main.children) {
      if (child.getClientRects().length) resize.observe(child);
    }
  }
  if (window.MutationObserver) {
    new MutationObserver(records => {
      if (records.some(record => record.type === 'attributes'
        ? record.target.matches('[data-ob-el], .managed-tab-content')
        : !record.target.closest('[data-sg-tp]'))) queueFit();
    }).observe(main, {subtree: true, childList: true, attributes: true, attributeFilter: ['style', 'hidden']});
  }
  window.addEventListener('resize', queueFit);
  queueFit();
})();
