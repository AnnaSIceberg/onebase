package sheet

// textFlowEnd returns the last free position before the next value or merged
// cell. Styled empty cells still count as free.
func (d *Document) textFlowEnd(row, col, last int, covered map[CellKey]bool) int {
	for next := col + 1; next <= last; next++ {
		if covered[CellKey{row, next}] {
			return next - 1
		}
		cell := d.GetCell(row, next)
		if cell != nil && (cell.DisplayText() != "" || cell.RichHTML != "" || cell.Picture != "" ||
			cell.ColSpan > 1 || cell.RowSpan > 1 || cell.Drilldown != nil || cell.DrilldownPayload != nil) {
			return next - 1
		}
	}
	return last
}

const cellTextFlowCSS = `
.doc-content>table{width:max-content}
.sheet-wrap-text{white-space:pre-wrap;word-break:normal;overflow-wrap:normal}
.sheet-flow-cell>.sheet-flow{contain:inline-size}
.sheet-flow{position:relative;max-width:100%}
.sheet-flow>span{display:block;width:var(--sheet-flow-width,100%);overflow:hidden;white-space:pre;word-break:normal;overflow-wrap:normal}
.sheet-flow-spill{position:relative;z-index:1}
`

// The span may overpaint empty cells without turning them into a colspan. Its
// width follows real column geometry, including after dragging a boundary.
const cellTextFlowScript = `<script>
(function () {
  const table = document.querySelector('.doc-content>table');
  if (!table) return;
  const sources = Array.from(table.querySelectorAll('[data-sheet-flow-end]'));
  const spans = sources.map(function (source) {
    const edge = source.closest('tr').querySelector('td[data-sheet-col="' + source.dataset.sheetFlowEnd + '"]');
    return edge ? {source: source, edge: edge} : null;
  }).filter(Boolean);
  function resize() {
    const widths = spans.map(function (span) {
      return Math.max(0, span.edge.getBoundingClientRect().right - span.source.getBoundingClientRect().left - 6);
    });
    spans.forEach(function (span, i) {
      span.source.style.setProperty('--sheet-flow-width', widths[i] + 'px');
    });
  }
  table.addEventListener('sheet-column-resized', resize);
  window.addEventListener('resize', resize);
  window.addEventListener('beforeprint', resize);
  resize();
})();
</script>`
