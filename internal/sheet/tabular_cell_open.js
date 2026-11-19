(function () {
  'use strict';
  const table = document.querySelector('.doc-content > table');
  if (!table) return;

  table.addEventListener('click', function (event) {
    const link = event.target.closest('a[data-sheet-drilldown]');
    if (!link || event.detail === 0 ||
        (event.detail === 1 && !link.hasAttribute('data-sheet-reference'))) return;
    event.preventDefault();
    event.stopPropagation();
  }, true);

  table.addEventListener('dblclick', function (event) {
    if (event.defaultPrevented || event.button !== 0) return;
    let link = event.target.closest('a[data-sheet-reference]');
    if (!link) {
      if (event.target.closest('a, button, input, select, textarea')) return;
      const cell = event.target.closest('td');
      if (!cell || cell.closest('table') !== table) return;
      link = cell.querySelector('a[data-sheet-drilldown]');
    }
    if (!link) return;
    event.preventDefault();
    link.click();
  });
})();
