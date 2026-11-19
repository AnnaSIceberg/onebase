package sheet

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (d *Document) viewerScript(sb *strings.Builder, notifyParent bool) {
	viewGroups := d.ResolvedGroups()
	for i := range viewGroups {
		// Only rendered cells supply accessible labels. Builder-only titles
		// need not enter the HTML payload and may contain unmasked source text.
		viewGroups[i].Title = ""
	}
	groups, _ := json.Marshal(viewGroups)
	// Readonly local controls: no network, form inputs or parent dirty events.
	fmt.Fprintf(sb, `<style>@media print{tr[hidden]{display:table-row!important}.sheet-fold,.sheet-levels{display:none!important}}.sheet-fold{border:1px solid currentColor;border-radius:2px;background:transparent;cursor:pointer;margin-right:5px;color:inherit;width:1.25em;height:1.25em;padding:0;line-height:1}.sheet-levels{display:inline-flex;gap:4px}.doc-toolbar .sheet-level{padding:4px 8px}.sheet-level[aria-pressed=true]{font-weight:700;outline:2px solid currentColor;outline-offset:-2px}body:has(.sheet-frozen) .doc-toolbar{left:0}.sheet-frozen{border-collapse:separate;border-spacing:0}.sheet-frozen .sheet-fixed{position:sticky}@media print{.sheet-frozen{border-collapse:collapse}.sheet-frozen .sheet-fixed{position:static!important}}</style>
<script>(function(){'use strict';var groups=(%s)||[],rows=Array.from(document.querySelectorAll('tr[data-sheet-row]'));
var folds=[],levelButtons=[],toolbar=document.querySelector('.doc-toolbar');
function paint(){rows.forEach(function(row,i){row.hidden=groups.some(function(g){return !g.Expanded&&i>g.HeaderEnd&&i<=g.End;});});folds.forEach(function(f){f.button.textContent=f.group.Expanded?'−':'+';f.button.setAttribute('aria-expanded',String(f.group.Expanded));});}
groups.forEach(function(g){var row=rows[g.Start];if(!row||!row.cells.length||g.End<=g.HeaderEnd)return;var b=document.createElement('button');b.className='sheet-fold';b.type='button';b.setAttribute('aria-label',row.cells[0].textContent||'Группа');folds.push({button:b,group:g});b.onclick=function(){g.Expanded=!g.Expanded;levelButtons.forEach(function(button){button.setAttribute('aria-pressed','false');});paint();};row.cells[0].prepend(b);});
if(groups.length&&toolbar){var min=groups.reduce(function(n,g){return Math.min(n,g.Level);},groups[0].Level),max=groups.reduce(function(n,g){return Math.max(n,g.Level);},min);
var details=rows.some(function(row,i){return groups.some(function(g){return i>g.HeaderEnd&&i<=g.End;})&&!groups.some(function(g){return i>=g.Start&&i<=g.HeaderEnd;});});
var panel=document.createElement('span');panel.className='sheet-levels';panel.setAttribute('role','group');panel.setAttribute('aria-label','Уровни группировки');
for(var n=1;n<=max-min+1+(details?1:0);n++){(function(level){var b=document.createElement('button');b.type='button';b.className='sheet-level';b.textContent=String(level);b.setAttribute('data-sheet-level',String(level));b.setAttribute('aria-pressed','false');b.onclick=function(){groups.forEach(function(g){g.Expanded=g.Level-min+1<level;});levelButtons.forEach(function(button){button.setAttribute('aria-pressed',String(button===b));});paint();};levelButtons.push(b);panel.appendChild(b);})(n);}toolbar.appendChild(panel);}paint();
var freezeRows=%d,freezeCols=%d,table=document.querySelector('.sheet-frozen');
if(table&&(freezeRows>0||freezeCols>0)){
var columns=Array.from(table.querySelectorAll('colgroup>col')),fixed=[],scheduled=0;
rows.forEach(function(row,i){Array.from(row.cells).forEach(function(c){
 var col=Number(c.dataset.sheetCol),flow=c.querySelector('[data-sheet-flow-end]');
 var top=i<freezeRows&&i+c.rowSpan<=freezeRows;
 var left=col<freezeCols&&col+c.colSpan<=freezeCols&&(!flow||Number(flow.dataset.sheetFlowEnd)<freezeCols);
 if(!top&&!left)return;
 var fill=getComputedStyle(c).backgroundColor;
 c.style.background='linear-gradient('+fill+','+fill+'),white';
 c.classList.add('sheet-fixed');
 var spill=c.classList.contains('sheet-flow-spill');
 c.style.zIndex=String(top?(left?(spill?7:6):(spill?5:4)):(spill?3:2));
 fixed.push({cell:c,row:i,col:col,top:top,left:left});
});});
function fix(){
 scheduled=0;
 var top=toolbar?toolbar.getBoundingClientRect().height:0,left=0,tops=[],lefts=[];
 for(var r=0;r<Math.min(freezeRows,rows.length);r++){
  tops.push(top);
  top+=rows[r].getBoundingClientRect().height;
 }
 for(var c=0;c<Math.min(freezeCols,columns.length);c++){
  lefts.push(left);
  left+=columns[c].getBoundingClientRect().width;
 }
 fixed.forEach(function(f){
  if(f.top)f.cell.style.top=tops[f.row]+'px';
  if(f.left)f.cell.style.left=lefts[f.col]+'px';
 });
}
function schedule(){if(!scheduled)scheduled=requestAnimationFrame(fix);}
table.addEventListener('sheet-column-resized',schedule);
window.addEventListener('resize',schedule);
if(window.ResizeObserver){
 var observer=new ResizeObserver(schedule);
 observer.observe(table);
 if(toolbar)observer.observe(toolbar);
}
fix();
}
if(%t&&window.parent!==window)document.addEventListener('click',function(e){var a=e.target.closest('[data-sheet-drilldown]');if(!a)return;e.preventDefault();window.parent.postMessage({type:'ob-tabular-cell',url:a.href},location.origin);});
})();</script>`, groups, d.FreezeRows, d.FreezeCols, notifyParent)
}
