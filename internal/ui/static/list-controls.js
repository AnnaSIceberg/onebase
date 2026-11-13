/* Optional enhanced list controls. Loaded only by the configured UI role scope. */
(function () {
  var baseMakeTreeRow = window.makeTreeRow;
  if (typeof baseMakeTreeRow === 'function') window.makeTreeRow = function (row) {
    var tr = baseMakeTreeRow(row);
    if (obListConfig().hideRowActions && tr.lastElementChild) tr.removeChild(tr.lastElementChild);
    return tr;
  };
window.obInitReportBlocks = function() {
  try {
    document.querySelectorAll('details.report-block').forEach(function (el) {
      if (el.classList && el.classList.contains('ob-list-settings') && document.querySelector('main.ob-list-panels')) return;
      var key = 'rb-' + location.pathname + '-' + el.dataset.block;
      var saved = localStorage.getItem(key);
      if (saved === '1') el.open = true;
      else if (saved === '0') el.open = false;
      el.addEventListener('toggle', function () { localStorage.setItem(key, el.open ? '1' : '0'); });
    });
  } catch (e) {}
};
function obInitListPanels() {
  var root = document.querySelector('main.ob-list-panels');
  if (!root) return;
  var scope = root.getAttribute('data-ob-list-panel-scope') || 'open-access';
  root.querySelectorAll('details[data-ob-list-panel], details.ob-list-settings').forEach(function (panel) {
    if (panel._obListPanelInit) return;
    panel._obListPanelInit = true;
    var name = panel.getAttribute('data-ob-list-panel') || 'settings';
    var key = 'ob-list-panel:v1:' + encodeURIComponent(scope) + ':' + location.pathname + ':' + name;
    var saved = null;
    try { saved = localStorage.getItem(key); } catch (e) {}
    panel.open = saved === '1';
    panel.addEventListener('toggle', function () {
      try { localStorage.setItem(key, panel.open ? '1' : '0'); } catch (e) {}
    });
  });
}

  obReady(obInitListPanels);
})();
