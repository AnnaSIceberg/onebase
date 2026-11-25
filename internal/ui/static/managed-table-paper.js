(function () {
  function showPaper(host, top, height, widths, rowHeight) {
    let paper = host.querySelector(':scope > .ob-table-grid-paper');
    if (height <= 0) {
      if (paper) paper.hidden = true;
      return;
    }
    if (!paper) {
      paper = document.createElement('div');
      paper.className = 'ob-table-grid-paper';
      paper.setAttribute('aria-hidden', 'true');
      host.appendChild(paper);
    }

    const rules = [];
    let left = 0;
    for (const width of widths) {
      left += width;
      const edge = Math.round(left);
      rules.push('linear-gradient(var(--ob-table-rule),var(--ob-table-rule)) ' + (edge - 1) + 'px 0 / 1px 100% no-repeat');
    }
    rules.push('repeating-linear-gradient(to bottom,transparent 0 ' + (rowHeight - 1) +
      'px,var(--ob-table-rule) ' + (rowHeight - 1) + 'px ' + rowHeight + 'px)');
    paper.hidden = false;
    paper.style.background = rules.join(',');
    paper.style.left = '0';
    paper.style.top = top + 'px';
    paper.style.width = Math.max(host.clientWidth, Math.round(left)) + 'px';
    paper.style.height = Math.max(0, height) + 'px';
  }

  function trackTable(box) {
    const table = box.querySelector('.tp-table');
    if (!table || !table.tHead) return;
    function draw() {
      if (!box.clientWidth) return;
      const widths = Array.from(table.tHead.rows[0].cells, cell => cell.getBoundingClientRect().width)
        .filter(width => width > 0);
      const sample = table.tBodies[0] && table.tBodies[0].rows[0];
      const rowHeight = sample && !sample.classList.contains('ob-vt-empty') ? Math.round(sample.offsetHeight) : 28;
      const top = table.offsetTop + table.offsetHeight;
      showPaper(box, top, box.clientHeight - top, widths, Math.max(2, rowHeight));
    }
    const resize = new ResizeObserver(draw);
    resize.observe(box);
    resize.observe(table);
    new MutationObserver(draw).observe(table, {
      subtree: true, childList: true, attributes: true, attributeFilter: ['style', 'hidden']
    });
    draw();
  }

  function trackGrid(host) {
    const state = host._obGridState;
    const viewport = host.querySelector('.slick-viewport');
    if (!state || !viewport) return;
    const grid = state.grid;
    const headers = host.querySelector('.slick-header-columns');
    function draw() {
      if (!viewport.clientWidth) return;
      const rowHeight = grid.getOptions().rowHeight;
      const top = state.dataView.getLength() * rowHeight;
      const widths = grid.getColumns().filter(column => !column.hidden).map(column => column.width);
      showPaper(viewport, top, viewport.clientHeight - top, widths, rowHeight);
    }
    const resize = new ResizeObserver(draw);
    resize.observe(viewport);
    if (headers) new MutationObserver(draw).observe(headers, {
      subtree: true, childList: true, attributes: true, attributeFilter: ['style']
    });
    state.dataView.onRowCountChanged.subscribe(draw);
    draw();
  }

  document.addEventListener('DOMContentLoaded', function () {
    if (!window.ResizeObserver || !window.MutationObserver) return;
    document.querySelectorAll('.ob-managed-table .ob-table-rows').forEach(trackTable);
    document.querySelectorAll('.ob-managed-table [data-sg-tp]').forEach(trackGrid);
  });
})();
