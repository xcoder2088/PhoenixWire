import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {mountAdmin,adminText} from '../internal/web/static/admin.js';
const settle=async()=>{for(let i=0;i<6;i++)await new Promise(r=>setImmediate(r));};
async function setup({authorized=true,language='en',confirm=true,error='',open=true,lockedOpen,initial={},holdPost=null,reply=null}={}){
 const dom=new JSDOM('<main></main>'),d=dom.window.document,calls=[],confirmations=[];
 let view={open,...(lockedOpen===undefined?{}:{locked_open:lockedOpen}),registered_count:37,registered:[],allowed:['allowed@example.test'],blocked:['blocked@example.test'],admins:['admin@example.test'],...initial};
 dom.window.confirm=text=>{confirmations.push(text);return typeof confirm==='function'?confirm(text):confirm;};
 const request=async(path,data)=>{
  calls.push([path,data]);if(data&&error)throw Object.assign(Error(error),{code:error});
  if(data&&holdPost)await holdPost;
  if(path==='/api/admin/access'&&data)view={...view,open:data.open};
  if(path==='/api/admin/block'&&data)view={...view,allowed:view.allowed.filter(e=>e!==data.email),blocked:[...new Set([...view.blocked,data.email])]};
  if(path==='/api/admin/allow'&&data)view={...view,blocked:view.blocked.filter(e=>e!==data.email),allowed:[...new Set([...view.allowed,data.email])]};
  if(path==='/api/admin/revoke'&&data)view={...view,allowed:view.allowed.filter(e=>e!==data.email)};
  if(path==='/api/admin/remove'&&data)view={...view,allowed:view.allowed.filter(e=>e!==data.email),blocked:view.blocked.filter(e=>e!==data.email)};
  if(data&&reply)view=reply(view,path,data);
  return JSON.parse(JSON.stringify(view));
 };
 const panel=mountAdmin({root:d.querySelector('main'),authorized,language:()=>language,request});await settle();
 return{dom,d,calls,confirmations,panel,setLanguage(value){language=value;panel.render();}};
}
test('only an exact administrator authorization mounts a panel or fetches access data',async()=>{
 for(const authorized of [false,null,'true',1]){const app=await setup({authorized});assert.equal(app.d.getElementById('wire-admin'),null);assert.equal(app.calls.length,0);app.dom.window.close();}
 const app=await setup();assert.ok(app.d.getElementById('wire-admin'));assert.equal(app.d.getElementById('wire-admin-open').checked,false);assert.equal(app.calls[0][0],'/api/admin/access');app.panel.destroy();assert.equal(app.d.getElementById('wire-admin'),null);app.dom.window.close();
});
test('fixed invitation-only switch, inverted payload and opening confirmation stay consistent in five languages',async()=>{
 const labels={fr:'Accès sur invitation seulement',en:'Invitation-only access',es:'Acceso solo por invitación',de:'Zugang nur auf Einladung',zh:'仅限受邀用户访问'};
 for(const language of Object.keys(labels)){
  const app=await setup({language}),toggle=()=>app.d.getElementById('wire-admin-open'),state=()=>app.d.getElementById('wire-admin-access-state');
  assert.equal(toggle().checked,false);assert.equal(toggle().getAttribute('aria-checked'),'false');assert.equal(toggle().parentElement.textContent,labels[language]);assert.equal(state().textContent,adminText(language,'open'));assert.equal(toggle().getAttribute('aria-describedby'),state().id);
  toggle().checked=true;toggle().dispatchEvent(new app.dom.window.Event('change'));await settle();
  assert.equal(app.confirmations.length,0,'closing does not ask');assert.deepEqual(app.calls.at(-1),['/api/admin/access',{open:false}]);assert.equal(toggle().checked,true);assert.equal(toggle().getAttribute('aria-checked'),'true');assert.equal(toggle().parentElement.textContent,labels[language]);assert.equal(state().textContent,adminText(language,'closed'));
  toggle().checked=false;toggle().dispatchEvent(new app.dom.window.Event('change'));await settle();assert.deepEqual(app.confirmations,[adminText(language,'confirmOpen')]);assert.deepEqual(app.calls.at(-1),['/api/admin/access',{open:true}]);assert.equal(toggle().checked,false);assert.equal(state().textContent,adminText(language,'open'));app.dom.window.close();
  const refused=await setup({language,confirm:false,open:false});const before=refused.d.getElementById('wire-admin-access-state').textContent;refused.d.getElementById('wire-admin-open').checked=false;refused.d.getElementById('wire-admin-open').dispatchEvent(new refused.dom.window.Event('change'));await settle();assert.equal(refused.calls.length,1);assert.deepEqual(refused.confirmations,[adminText(language,'confirmOpen')]);assert.equal(refused.d.getElementById('wire-admin-open').checked,true);assert.equal(refused.d.getElementById('wire-admin-access-state').textContent,before);refused.dom.window.close();
 }
 assert.equal(adminText('fr','confirmOpen'),'Ouvrir Phoenix Wire à tout le monde ?');
 assert.equal(adminText('fr','closed'),'Seuls les expéditeurs inscrits ou autorisés peuvent envoyer. Les cartes déjà envoyées restent ouvertes.');
 assert.equal(adminText('fr','open'),"Tout le monde peut s'inscrire et envoyer.");
});
test('locked-open server configuration disables only the access switch and explains it in five languages',async()=>{
 const translations={fr:"La configuration du serveur garde l'accès ouvert : cette case ne prend effet qu'une fois cette configuration retirée.",en:'The server configuration keeps access open: this checkbox only takes effect once that configuration is removed.',es:'La configuración del servidor mantiene el acceso abierto: esta casilla solo tendrá efecto cuando se retire esa configuración.',de:'Die Serverkonfiguration hält den Zugang offen: Dieses Kontrollkästchen wirkt erst, wenn diese Konfiguration entfernt wurde.',zh:'服务器配置保持访问开放：只有移除该配置后，此复选框的设置才会生效。'};
 for(const [language,message]of Object.entries(translations)){
  const app=await setup({language,lockedOpen:true}),toggle=app.d.getElementById('wire-admin-open'),note=app.d.getElementById('wire-admin-locked-open');
  assert.equal(toggle.disabled,true);assert.equal(toggle.checked,false);assert.equal(note.textContent,message);assert.doesNotMatch(note.textContent,/Mainnet|OTP|Hub/);assert.ok(toggle.getAttribute('aria-describedby').split(' ').includes(note.id));
  toggle.checked=true;toggle.dispatchEvent(new app.dom.window.Event('change'));await settle();assert.equal(app.calls.length,1);assert.equal(app.confirmations.length,0);assert.equal(app.d.getElementById('wire-admin-open').checked,false);
  const form=app.d.getElementById('wire-admin-allow');assert.equal(form.querySelector('button').disabled,false);form.querySelector('input').value='another@example.test';form.dispatchEvent(new app.dom.window.Event('submit',{cancelable:true}));await settle();assert.equal(app.d.getElementById('wire-admin-open').disabled,true);assert.equal(app.d.getElementById('wire-admin-locked-open').textContent,message);app.dom.window.close();
 }
 const unlocked=await setup({lockedOpen:false});assert.equal(unlocked.d.getElementById('wire-admin-open').disabled,false);assert.equal(unlocked.d.getElementById('wire-admin-locked-open'),null);unlocked.dom.window.close();
});
test('invited and blocked rows expose their distinct routes, confirmations and server replies',async()=>{
 const app=await setup(),d=app.d;
 assert.deepEqual([...d.querySelector('[data-list=allowed] li').querySelectorAll('button')].map(b=>b.dataset.action),['revoke','block']);
 assert.deepEqual([...d.querySelector('[data-list=blocked] li').querySelectorAll('button')].map(b=>b.dataset.action),['remove','allow']);
 d.querySelector('[data-list=allowed] [data-action=revoke]').click();await settle();
 assert.deepEqual(app.calls.at(-1),['/api/admin/revoke',{email:'allowed@example.test'}]);assert.equal(app.confirmations.length,0);
 assert.equal(d.querySelectorAll('[data-list=allowed] li').length,0);assert.equal(d.querySelectorAll('[data-list=blocked] li').length,1);
 d.querySelector('[data-list=blocked] [data-action=allow]').click();await settle();
 assert.deepEqual(app.calls.at(-1),['/api/admin/allow',{email:'blocked@example.test'}]);assert.equal(app.confirmations.length,0);
 d.querySelector('[data-list=allowed] [data-action=block]').click();await settle();
 assert.deepEqual(app.calls.at(-1),['/api/admin/block',{email:'blocked@example.test'}]);assert.equal(app.confirmations.length,1);
 assert.equal(app.confirmations.at(-1),adminText('en','confirmBlock').replace('{email}','blocked@example.test'));
 d.querySelector('[data-list=blocked] [data-action=remove]').click();await settle();
 assert.deepEqual(app.calls.at(-1),['/api/admin/remove',{email:'blocked@example.test'}]);assert.equal(app.confirmations.length,2);
 assert.equal(app.confirmations.at(-1),adminText('en','confirmRemove').replace('{email}','blocked@example.test'));
 assert.equal(d.querySelectorAll('[data-list=blocked] li').length,0);assert.equal(d.querySelectorAll('[data-list=allowed] li').length,0);app.dom.window.close();
});
test('only the registered count is shown, including when the deprecated list is absent, empty or populated',async()=>{
 for(const registered of [undefined,[],['never-list-this@example.test']]){
  const app=await setup({initial:{registered,registered_count:37}});
  assert.equal(app.d.querySelector('[data-list=registered]'),null);assert.ok(app.d.getElementById('wire-admin-counts').textContent.includes('37 registered'));
  assert.ok(app.d.getElementById('wire-admin-counts').getAttribute('aria-label').includes('37 registered senders'));
  assert.ok(!app.d.body.textContent.includes('never-list-this@example.test'));app.dom.window.close();
 }
 const app=await setup({initial:{registered:undefined,registered_count:0}});assert.ok(app.d.getElementById('wire-admin-counts').textContent.includes('0 registered'));app.dom.window.close();
 const missing=await setup({initial:{registered:['never-list-this@example.test'],registered_count:undefined}});assert.ok(missing.d.getElementById('wire-admin-counts').textContent.includes('— registered'));assert.ok(!missing.d.body.textContent.includes('never-list-this@example.test'));missing.dom.window.close();
});
test('invite and manual block forms work for an address not listed, without confirmation for an invitation',async()=>{
 const app=await setup(),d=app.d;
 for(const [kind,email] of [['allow','new@example.test'],['block','unlisted@example.test']]){
  const form=d.getElementById('wire-admin-'+kind);form.querySelector('input').value=email;form.dispatchEvent(new app.dom.window.Event('submit',{cancelable:true}));await settle();
  assert.deepEqual(app.calls.at(-1),['/api/admin/'+kind,{email}]);
  assert.equal(d.getElementById('wire-admin-'+kind).querySelector('input').value,'');
 }
 assert.equal(app.confirmations.length,1);assert.ok(d.querySelector('[data-list=blocked]').textContent.includes('unlisted@example.test'));
 assert.ok(d.getElementById('wire-admin-counts').textContent.includes('2 invited · 2 blocked · 37 registered'));app.dom.window.close();
});
test('refusing Block or Remove makes no request, including a manual block',async()=>{
 const app=await setup({confirm:false}),d=app.d;
 d.querySelector('[data-list=allowed] [data-action=block]').click();
 d.querySelector('[data-list=blocked] [data-action=remove]').click();
 const form=d.getElementById('wire-admin-block');form.querySelector('input').value='unlisted@example.test';form.dispatchEvent(new app.dom.window.Event('submit',{cancelable:true}));await settle();
 assert.equal(app.calls.length,1);assert.equal(app.confirmations.length,3);assert.equal(form.querySelector('input').value,'unlisted@example.test');app.dom.window.close();
});
test('search filters both lists locally without moving focus, and keeps total counts and the query after actions',async()=>{
 const app=await setup({initial:{allowed:['Alice@example.test','other@example.test'],blocked:['ALICE@blocked.test','blocked@example.test']}}),d=app.d;
 const search=d.getElementById('wire-admin-search');search.focus();search.value='  aLiCe  ';search.dispatchEvent(new app.dom.window.Event('input'));
 assert.equal(d.activeElement,search);assert.equal(app.calls.length,1);
 assert.deepEqual([...d.querySelectorAll('[data-list=allowed] li .wire-admin-address')].map(n=>n.textContent),['Alice@example.test']);
 assert.deepEqual([...d.querySelectorAll('[data-list=blocked] li .wire-admin-address')].map(n=>n.textContent),['ALICE@blocked.test']);
 assert.equal(d.getElementById('wire-admin-counts').textContent,'2 invited · 2 blocked · 37 registered');
 d.querySelector('[data-list=allowed] [data-action=revoke]').click();await settle();
 assert.equal(d.getElementById('wire-admin-search').value,'  aLiCe  ');assert.equal(d.querySelectorAll('[data-list=allowed] li').length,0);
 assert.equal(d.getElementById('wire-admin-counts').textContent,'1 invited · 2 blocked · 37 registered');
 app.setLanguage('fr');assert.equal(d.getElementById('wire-admin-search').value,'  aLiCe  ');assert.equal(d.getElementById('wire-admin-counts').textContent,'1 invité · 2 bloqués · 37 inscrits');
 const current=d.getElementById('wire-admin-search');current.value='nomatch';current.dispatchEvent(new app.dom.window.Event('input'));assert.equal(d.querySelectorAll('.wire-admin-empty:not([hidden])').length,2);assert.equal(app.calls.length,2);
 current.value='';current.dispatchEvent(new app.dom.window.Event('input'));assert.equal(d.querySelectorAll('[data-list=blocked] li').length,2);app.dom.window.close();
});
test('all new actions and their explanations are translated in the five languages',async()=>{
 for(const language of ['fr','en','es','de','zh']){
  const app=await setup({language}),d=app.d;
  for(const [kind,actions] of [['allowed',['revoke','block']],['blocked',['remove','allow']]])for(const action of actions)assert.equal(d.querySelector('[data-list='+kind+'] [data-action='+action+']').textContent,adminText(language,action));
  assert.equal(d.getElementById('wire-admin-action-help').textContent,adminText(language,'actionHelp'));
  assert.ok(d.body.textContent.includes(adminText(language,'refundHelp')));assert.doesNotMatch(d.body.textContent,/Mainnet|OTP|Hub|undefined/);
  for(const kind of ['allow','block'])assert.ok(d.getElementById('wire-admin-'+kind).textContent.includes(adminText(language,kind==='allow'?'email':'blockEmail')));
  d.querySelector('[data-list=blocked] [data-action=remove]').click();await settle();assert.equal(app.confirmations[0],adminText(language,'confirmRemove').replace('{email}','blocked@example.test'));app.dom.window.close();
 }
});
test('hundreds of addresses stay in bounded, keyboard-scrollable lists and render only inert text',async()=>{
 const unsafe='<img src=x onerror=alert(1)>@example.test';
 const app=await setup({initial:{allowed:[unsafe,...Array.from({length:300},(_,i)=>'guest'+i+'@example.test')],blocked:Array.from({length:300},(_,i)=>'blocked'+i+'@example.test')}}),d=app.d;
 const style=d.createElement('style');style.textContent=(await import('node:fs')).readFileSync(new URL('../internal/web/static/wire-pay.css',import.meta.url),'utf8');d.head.append(style);
 assert.equal(d.querySelectorAll('[data-list=allowed] li').length,301);assert.equal(d.querySelectorAll('[data-list=blocked] li').length,300);
 assert.equal(d.querySelector('[data-list=allowed] li img'),null);assert.equal(d.querySelector('[data-list=allowed] li .wire-admin-address').textContent,unsafe);
 for(const list of d.querySelectorAll('.wire-admin-list ul')){assert.equal(list.tabIndex,0);const css=app.dom.window.getComputedStyle(list);assert.equal(css.overflowY,'auto');assert.notEqual(css.maxHeight,'none');assert.ok(parseFloat(css.maxHeight)>0);}
 app.dom.window.close();
});
test('mutations use the returned counts and lists, protect admin block buttons and prevent overlapping posts',async()=>{
 let release;const holdPost=new Promise(resolve=>{release=resolve;});
 const app=await setup({holdPost,initial:{allowed:['admin@example.test','allowed@example.test']},reply:view=>({...view,registered_count:55})}),d=app.d;
 const rows=d.querySelectorAll('[data-list=allowed] li');assert.equal(rows[0].querySelector('[data-action=block]').disabled,true);assert.equal(rows[0].querySelector('[data-action=revoke]').disabled,false);
 rows[1].querySelector('[data-action=revoke]').click();await settle();assert.equal(d.getElementById('wire-admin').getAttribute('aria-busy'),'true');
 const form=d.getElementById('wire-admin-allow');form.querySelector('input').value='other@example.test';form.dispatchEvent(new app.dom.window.Event('submit',{cancelable:true}));assert.equal(app.calls.length,2);
 release();await settle();assert.equal(d.getElementById('wire-admin').getAttribute('aria-busy'),'false');assert.ok(d.getElementById('wire-admin-counts').textContent.includes('55 registered'));assert.equal(d.querySelectorAll('[data-list=allowed] li').length,1);app.dom.window.close();
});
test('administrator errors are plain words in all five languages and controls recover',async()=>{
 for(const language of ['fr','en','es','de','zh'])for(const error of ['admin_required','cannot_block_admin','list_full','invalid_email']){
  const app=await setup({language,error}),form=app.d.getElementById('wire-admin-allow');form.querySelector('input').value='new@example.test';form.dispatchEvent(new app.dom.window.Event('submit',{cancelable:true}));await settle();const message=app.d.getElementById('wire-admin-status').textContent;assert.equal(message,adminText(language,error));assert.doesNotMatch(message,/admin_required|cannot_block_admin|list_full|invalid_email|Mainnet|OTP|Hub/);assert.equal(app.d.querySelector('[aria-busy=true]'),null);app.dom.window.close();
 }
});

