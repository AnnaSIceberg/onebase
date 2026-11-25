(function(){
 'use strict';
 if(window.obApplyTabularDocuments)return;
 window.obOpenTabularQuestion=function(question,elementName,eventName,extra,fire){
  if(!extra||typeof extra._tabular_token!=='string'||(eventName!=='ОбработкаРасшифровки'&&eventName!=='Ответ')||typeof window.obOpenQuestion!=='function')return false;
  var captured=Object.assign({},extra);
  window.obOpenQuestion(question,elementName,{onAnswer:function(variant,fields){
   var next=Object.assign({},captured,fields||{_question_answer:variant});
   ['_tabular_token','_tabular_row','_tabular_col'].forEach(function(key){next[key]=captured[key];});
   return fire(elementName,'Ответ',next);
  }});
  return true;
 };
 var activeProgress=new WeakMap(), resultPaths=new WeakMap();
 function clearResultTab(frame,key){
  resultPaths.delete(frame);
  if(!frame.hasAttribute('data-ob-tabular-result-tab'))return;
  try {
   if(window.parent!==window&&typeof window.parent.obClearReportResult==='function')window.parent.obClearReportResult(window,key);
  }catch(_){}
 }
 function openResultTab(frame, key, path) {
  if(!frame.hasAttribute('data-ob-tabular-result-tab'))return false;
  try {
   if(window.parent!==window&&typeof window.parent.obOpenReportResult==='function'&&
      window.parent.obOpenReportResult(window,key,path,'Результат отчёта')) {
    resultPaths.set(frame,path);
    return true;
   }
  } catch(_) {}
  return false;
 }
 window.obBeginTabularProgress=function(elementName){
  const button=Array.from(document.querySelectorAll('[data-ob-tabular-progress], [data-ob-workbook-save]')).find(function(el){
   return el.getAttribute('data-ob-fire-click')===elementName;
  });
  if(!button)return null;
  const workbook=button.hasAttribute('data-ob-workbook-save');
  const field=(button.getAttribute('data-ob-tabular-progress')||'').toLowerCase();
  const frame=workbook?null:Array.from(document.querySelectorAll('iframe[data-ob-tabular-field]')).find(function(el){
   return el.getAttribute('data-ob-tabular-field').toLowerCase()===field;
  });
  const status=workbook?button.nextElementSibling:frame&&frame.previousElementSibling;
  if(!status)return null;
  const percent=status.querySelector('[data-ob-tabular-percent]');
  if(!percent)return null;
  const target=frame||status;
  const progress={
   update:function(value){
    if(activeProgress.get(target)===progress)percent.textContent=value+'%';
   },
   finish:function(){
    if(activeProgress.get(target)!==progress)return;
    activeProgress.delete(target);
    status.hidden=true;
   }
  };
  activeProgress.set(target,progress);
  if(frame){
   clearResultTab(frame,field);
   frame.hidden=true;
   frame.removeAttribute('src');
  }
  progress.update(0);
  status.hidden=false;
  return progress;
 };

 const xlsxType='application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';
 const workbookPickers=new WeakMap();
 function workbookError(flash,error) {
  if(error && error.name==='AbortError')return;
  flash('Не удалось выбрать файл: '+(error && error.message || error), 'err');
 }
 async function chooseWorkbook(state, commandName) {
  // The call must be reached directly from the click that provided activation.
  const handle=await window.showSaveFilePicker({
   id:state.id,
   suggestedName:commandName||state.name.value,
   excludeAcceptAllOption:true,
   types:[{description:'Excel (.xlsx)',accept:{[xlsxType]:['.xlsx']}}]
  });
  if(!commandName){
   state.name.value=handle.name;
   state.handle=handle;
   try { localStorage.setItem(workbookPreferenceKey(state), handle.name); } catch (_) {}
  }
  return handle;
 }

 function workbookPreferenceKey(state) {
  return state.field ? state.id+':'+state.suggestion : state.id;
 }

 function workbookApplyRememberedName(state) {
  if(state.field && !state.suggestion)return;
  try { state.name.value=localStorage.getItem(workbookPreferenceKey(state))||state.name.value; } catch (_) {}
 }

 function workbookPickerReady(state,flash,commandName) {
  if(!state.field || commandName)return true;
  if(state.formPending && state.formPending()){
   flash('Подождите, пока обновится имя файла.', 'info');
   return false;
  }
  if(!state.suggestion){
   flash('Сначала заполните параметры файла.', 'info');
   return false;
  }
  return true;
 }

 window.obApplyWorkbookSuggestion=function(form,values) {
  const state=form && workbookPickers.get(form);
  if(!state || !state.field || !Object.prototype.hasOwnProperty.call(values,state.field))return;
  const suggestion=String(values[state.field]||'');
  if(suggestion===state.suggestion)return;
  state.suggestion=suggestion;
  state.name.value=suggestion;
  state.handle=null;
  workbookApplyRememberedName(state);
 };

 window.obBindWorkbookPicker=function(form,flash,formPending) {
  const field=form && form.querySelector('[data-ob-workbook-picker]');
  if(!field || workbookPickers.has(form))return;
  const name=field.querySelector('[data-ob-workbook-name]');
  const choose=field.querySelector('[data-ob-workbook-choose]');
  const state={id:field.dataset.obWorkbookPicker,field:field.dataset.obWorkbookField,
               suggestion:name.value,name:name,handle:null,busy:false,formPending:formPending};
  workbookApplyRememberedName(state);
  workbookPickers.set(form,state);
  choose.addEventListener('click',async function(){
   if(state.busy)return;
   if(typeof window.showSaveFilePicker!=='function'){
    flash('Браузер предложит скачать XLSX после формирования.', 'info');
    return;
   }
   if(!workbookPickerReady(state,flash,''))return;
   state.busy=true;
   try { await chooseWorkbook(state,''); }
   catch(error) { workbookError(flash,error); }
   finally { state.busy=false; }
  });
 };

 window.obRunWorkbookSave=function(form,elementName,flash,run) {
  const state=form&&workbookPickers.get(form);
  if(!state)return null;
  const button=Array.from(document.querySelectorAll('[data-ob-workbook-save]')).find(function(el){
   return el.getAttribute('data-ob-fire-click')===elementName;
  });
  if(!button)return null;
  if(state.busy || button.disabled)return Promise.resolve();
  const commandName=button.getAttribute('data-ob-workbook-filename')||'';
  if(typeof window.showSaveFilePicker==='function' && !workbookPickerReady(state,flash,commandName))return Promise.resolve();
  state.busy=true;
  return (async function(){
   let handle=null;
   if(typeof window.showSaveFilePicker==='function'){
    try {
     handle=commandName ? await chooseWorkbook(state,commandName) : state.handle||await chooseWorkbook(state,'');
     if(handle.requestPermission && await handle.requestPermission({mode:'readwrite'})!=='granted'){
      throw new Error('Браузер не разрешил запись в выбранный файл.');
     }
    } catch(error){
     workbookError(flash,error);
     return;
    }
   }
   await run({handle:handle});
  })().finally(function(){state.busy=false;});
 };

 window.obWriteWorkbook=async function(handle,path) {
  const response=await fetch(path,{credentials:'same-origin',mode:'same-origin',redirect:'error'});
  if(!response.ok || (response.headers.get('Content-Type')||'').split(';')[0]!==xlsxType){
   throw new Error('Сервер не вернул книгу XLSX.');
  }
  const data=await response.blob();
  const writer=await handle.createWritable({mode:'exclusive'});
  try {
   await writer.write(data);
   await writer.close();
  } catch(error) {
   try { await writer.abort(); } catch (_) {}
   throw error;
  }
 };

 window.obReadTabularFormResponse=async function(response,progress){
  if((response.headers.get('Content-Type')||'').split(';')[0]!=='application/x-ndjson')return response.json();
  var reader=response.body.getReader(), decoder=new TextDecoder(), pending='';
  try{
   while(true){
    var chunk=await reader.read();
    if(chunk.done)throw new Error('Незавершённый ответ формирования');
    pending+=decoder.decode(chunk.value,{stream:true});
    var end;
    while((end=pending.indexOf('\n'))!==-1){
     var record=JSON.parse(pending.slice(0,end));
     pending=pending.slice(end+1);
     if(typeof record.ok==='boolean')return record;
     if(progress&&Number.isInteger(record.progress)&&record.progress>=0&&record.progress<=100)progress.update(record.progress);
    }
   }
  }finally{
   reader.releaseLock();
  }
 };
 window.obApplyTabularDocuments=function(results,progress){
  if(!results)return;
  document.querySelectorAll('iframe[data-ob-tabular-field]').forEach(function(frame){
   if(activeProgress.has(frame)&&activeProgress.get(frame)!==progress)return;
   var key=frame.getAttribute('data-ob-tabular-field').toLowerCase();
   if(!Object.prototype.hasOwnProperty.call(results,key))return;
   var path=results[key];
   if(!path){clearResultTab(frame,key);frame.removeAttribute('src');frame.hidden=true;return;}
   if(!/^\/ui\/tabular\/[0-9a-f-]{36}$/.test(path))return;
   if(openResultTab(frame,key,path)){frame.hidden=true;return;}
   frame.src=path;frame.hidden=false;
  });
 };
 window.obBindTabularHost=function(fire){
  function drilldown(frame,path,href){
   var url,current;try{url=new URL(href);current=new URL(path,location.origin);}catch(_){return;}
   if(url.origin!==location.origin||url.pathname!==current.pathname+'/cell'||url.hash||!/^\/ui\/tabular\/[0-9a-f-]{36}$/.test(current.pathname))return;
   var row=url.searchParams.get('row'),col=url.searchParams.get('col');if(!/^\d+$/.test(row||'')||!/^\d+$/.test(col||''))return;
   var host=frame.closest('[data-ob-el]');if(!host)return;
   fire(host.getAttribute('data-ob-el'),'ОбработкаРасшифровки',{_tabular_token:current.pathname.split('/').pop(),_tabular_row:row,_tabular_col:col});
  }
  window.obDrilldownReportResult=function(key,href){
   document.querySelectorAll('iframe[data-ob-tabular-result-tab]').forEach(function(frame){
    if(frame.getAttribute('data-ob-tabular-field').toLowerCase()===key&&resultPaths.has(frame))drilldown(frame,resultPaths.get(frame),href);
   });
  };
  window.addEventListener('message',function(event){
   if(event.origin!==location.origin||!event.data||event.data.type!=='ob-tabular-cell'||typeof event.data.url!=='string')return;
   document.querySelectorAll('iframe[data-ob-tabular-field]').forEach(function(frame){
    if(event.source===frame.contentWindow)drilldown(frame,frame.src,event.data.url);
   });
  });
 };
})();
