const {test} = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

// Run the actual B38.1 mask and filter controller together. No browser/DB.
function setup() {
  const handlers = {}, rowHandlers = {};
  const fields = [{name:'when',type:'datetime',operators:['eq','ge','is_null','not_null']}, {name:'text',type:'string',operators:['eq','contains','is_null']}];
  let row;
  function control(tag='INPUT') {
    const attrs = {};
    return {tagName:tag,type:'text',value:'',className:'',options:[],disabled:false,readOnly:false,required:false,
      selectionStart:0,selectionEnd:0,error:'',
      get selectedOptions(){return this.options.filter(o=>o.value===this.value);},
      appendChild(o){this.options.push(o);},
      setCustomValidity(v){this.error=v;}, reportValidity(){this.reported=true;},
      setSelectionRange(a,b){this.selectionStart=a;this.selectionEnd=b;},
      getAttribute(k){return attrs[k]??null;}, setAttribute(k,v){attrs[k]=String(v);}, removeAttribute(k){delete attrs[k];},
      replaceWith(next){row.value=next;},
      matches(selector){return selector.includes('.'+this.className);}, closest(){return row;},
      dispatchEvent(e){fire(e.type,this);}};
  }
  const field=control('SELECT');field.value='when';field.className='rs-f-field';
  const op=control('SELECT');op.value='eq';op.className='rs-f-op';
  op.options=['eq','ge','contains','is_null','not_null'].map(value=>({value}));
  row={dataset:{},field,op,value:control(),querySelector(selector){return selector==='.rs-f-field'?this.field:selector==='.rs-f-op'?this.op:this.value;}};
  const rows={dataset:{},querySelectorAll(){return [row];},addEventListener(k,fn){rowHandlers[k]=fn;}};
  const document={getElementById(id){if(id==='report-result-filter-fields')return {textContent:JSON.stringify(fields)};if(id==='rs-filter-rows')return rows;return {textContent:'{}'};},
    createElement(tag){return control(tag.toUpperCase());},addEventListener(k,fn){(handlers[k]??=[]).push(fn);}};
  const window={};
  const context=vm.createContext({document,window,Date,WeakMap,WeakSet,Event:class {constructor(type){this.type=type;}},MutationObserver:class {observe(){}}});
  function fire(type,target,extra={}) {
    const e={target,preventDefault(){},stopPropagation(){},...extra};
    for(const fn of handlers[type]||[])fn(e);
    if(rowHandlers[type])rowHandlers[type](e);
  }
  vm.runInContext(fs.readFileSync(__dirname+'/date-input.js','utf8'),context);
  vm.runInContext(fs.readFileSync(__dirname+'/report-result-filter.js','utf8'),context);
  return {row,window,context,document,fire,focus(){fire('focus',row.value);},type(text){for(const key of text)fire('keydown',row.value,{key});},operator(value){op.value=value;fire('change',op);}};
}

test('changing a filter operator commits a focused date without clearing it',()=>{
  const c=setup();c.focus();c.type('17112026123456');
  assert.equal(c.row.value.value,'17.11.2026 12:34:56');
  c.operator('ge');
  assert.equal(c.row.value.type,'datetime-local');
  assert.equal(c.row.value.value,'2026-11-17T12:34:56');
  assert.equal(c.row.op.value,'ge');
});

test('an incomplete date keeps its control and operator instead of becoming empty',()=>{
  const c=setup();c.focus();c.type('17');const input=c.row.value;
  c.operator('ge');
  assert.equal(c.row.value,input);assert.equal(c.row.op.value,'eq');
  assert.notEqual(input.error,'');assert.equal(input.reported,true);
});

test('a NULL condition deliberately discards an unfinished date',()=>{
  const c=setup();c.focus();c.type('17');c.operator('is_null');
  assert.equal(c.row.op.value,'is_null');assert.equal(c.row.value.value,'');
  assert.equal(c.row.value.disabled,true);assert.equal(c.row.value.required,false);
});

test('collecting settings commits the mask and refuses incomplete dates',()=>{
  const c=setup();
  // Load only the real settings functions from ui.js, excluding global UI setup.
  const ui=fs.readFileSync(__dirname+'/ui.js','utf8');
  const begin=ui.indexOf('function rsNorm('), end=ui.indexOf('\n',ui.indexOf('window.rsCollect =',begin));
  assert.ok(end>begin);
  // rsCollect is assigned as one function; find its closing line.
  const stop=ui.indexOf('\n};',ui.indexOf('window.rsCollect =',begin))+3;
  vm.runInContext(ui.slice(begin,stop),c.context);
  c.document.querySelectorAll=selector=>selector==='.rs-filter-row'?[c.row]:[];
  c.document.querySelector=()=>null;
  const hidden={value:'',dataset:{}};
  const oldGet=c.document.getElementById;
  c.document.getElementById=id=>id==='rs-json'?hidden:oldGet(id);
  c.focus();c.type('17');
  assert.equal(c.window.rsCollect(),false);assert.equal(hidden.value,'');
  c.type('112026123456');
  assert.notEqual(c.window.rsCollect(),false);
  assert.equal(JSON.parse(hidden.value).filters[0].value,'2026-11-17T12:34:56');
});
