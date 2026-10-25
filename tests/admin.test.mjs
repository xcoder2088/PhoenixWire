import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {mountAdmin,adminText} from '../internal/web/static/admin.js';
import {relayText,amountText} from '../internal/web/static/admin-relay.js';
const settle=async()=>{for(let i=0;i<6;i++)await new Promise(r=>setImmediate(r));};
async function setup({authorized=true,language='en',confirm=true,error='',open=true,lockedOpen,initial={},holdPost=null,reply=null,relays=null,withdrawal=null}={}){
 const dom=new JSDOM('<main></main>'),d=dom.window.document,calls=[],confirmations=[];
 let view={open,...(lockedOpen===undefined?{}:{locked_open:lockedOpen}),registered_count:37,registered:[],allowed:['allowed@example.test'],blocked:['blocked@example.test'],admins:['admin@example.test'],...initial};
 dom.window.confirm=text=>{confirmations.push(text);return typeof confirm==='function'?confirm(text):confirm;};
 const request=async(path,data)=>{
  calls.push([path,data]);if(data&&error&&!path.includes('relayers'))throw Object.assign(Error(error),{code:error});
  if(path==='/api/admin/relayers'&&!data){if(!relays)throw Object.assign(Error('usdt_unavailable'),{code:'usdt_unavailable'});return JSON.parse(JSON.stringify(relays));}
  if(path==='/api/admin/relayers/withdraw'){if(withdrawal instanceof Error)throw Object.assign(Error(withdrawal.message),{code:withdrawal.message});return withdrawal;}
  if(path==='/api/admin/maintenance'&&data)view={...view,maintenance:data.on};
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


const relayView={relayers:[{chain:'polygon',chain_id:137,address:'0x6a24200f9ce9F10a68e2F901424e6F946919e8EE',native_symbol:'POL',native_wei:'10000000000000000000',tokens:[{asset:'usdc_polygon',symbol:'USDC',units:'3250000'},{asset:'usdt_polygon',symbol:'USDT',units:'0'}],pending_cards:2,live_cards:2,reserve_wei:'500000000000000000',withdrawable_wei:'9499000000000000000'}],
 withdraw_to:['0x4444444444444444444444444444444444444444'],totals:[{asset:'usdc_polygon',payouts:12,earned:'3.000000',cost:'1.200000',net:'1.800000'}],last:[]};
test('maintenance is a switch that asks first, posts the new state and says what it does',async()=>{
 const app=await setup({confirm:false}),d=app.d,toggle=d.getElementById('wire-admin-maintenance');
 assert.equal(toggle.checked,false);assert.equal(toggle.getAttribute('role'),'switch');assert.match(d.getElementById('wire-admin-maintenance-state').textContent,/not in maintenance/);
 toggle.click();await settle();assert.equal(app.calls.filter(([p])=>p==='/api/admin/maintenance').length,0);assert.equal(app.confirmations.length,1);app.dom.window.close();
 const on=await setup(),t2=on.d.getElementById('wire-admin-maintenance');t2.click();await settle();
 assert.deepEqual(on.calls.at(-1),['/api/admin/maintenance',{on:true}]);assert.equal(on.d.getElementById('wire-admin-maintenance').checked,true);assert.match(on.d.getElementById('wire-admin-maintenance-state').textContent,/only administrators can sign in/);
 on.d.getElementById('wire-admin-maintenance').click();await settle();assert.deepEqual(on.calls.at(-1),['/api/admin/maintenance',{on:false}]);assert.equal(on.confirmations.length,1);on.dom.window.close();
 for(const language of ['fr','en','es','de','zh']){const t=await setup({language});assert.ok(t.d.getElementById('wire-admin-maintenance').parentElement.textContent.includes(adminText(language,'maintenance')));t.dom.window.close();}
});
test('the gas wallets load only when opened, show what stays, what can go and what the cards earned',async()=>{
 const app=await setup({relays:relayView}),d=app.d,box=d.getElementById('wire-admin-relays');
 assert.equal(app.calls.filter(([p])=>p==='/api/admin/relayers').length,0);
 box.open=true;box.dispatchEvent(new app.dom.window.Event('toggle'));await settle();
 assert.equal(app.calls.filter(([p])=>p==='/api/admin/relayers').length,1);
 const text=box.textContent;
 assert.match(text,/Polygon · 0x6a24…e8EE/);assert.ok(text.includes('10.0000 POL'));assert.ok(text.includes('0.5000 POL'));assert.ok(text.includes('9.4990 POL'));
 assert.ok(text.includes('Fees held USDC · Polygon'));assert.ok(text.includes('3.2500 USDC'));assert.equal(d.querySelector('.wire-admin-earned dd').textContent,'12');assert.ok(text.includes('3.000000'));assert.ok(text.includes('1.200000'));assert.ok(text.includes('1.800000'));
 assert.equal(d.getElementById('wire-admin-relay-to').value,'0x4444444444444444444444444444444444444444');assert.equal(d.querySelector('label[for=wire-admin-relay-to]').control.id,'wire-admin-relay-to');
 const buttons=[...box.querySelectorAll('button[data-what]')];
 assert.deepEqual(buttons.map(b=>[b.dataset.what,b.disabled]),[['native',false],['usdc_polygon',false],['usdt_polygon',true]]);app.dom.window.close();
});
test('withdrawing asks first, sends the chain, token and the authorized address, and reports the transaction',async()=>{
 const tx='0x'+'ab'.repeat(32);
 const refused=await setup({relays:relayView,confirm:false});refused.d.getElementById('wire-admin-relays').open=true;refused.d.getElementById('wire-admin-relays').dispatchEvent(new refused.dom.window.Event('toggle'));await settle();
 refused.d.querySelector('button[data-what=usdc_polygon]').click();await settle();assert.equal(refused.calls.filter(([p])=>p.endsWith('/withdraw')).length,0);
assert.match(refused.confirmations[0],/Withdraw 3\.2500 USDC from the Polygon wallet to 0x4444/);refused.dom.window.close();
 const app=await setup({relays:relayView,withdrawal:{tx,amount:'3250000',explorer:'https://polygonscan.com/tx/',status:'sent'}}),d=app.d;
 d.getElementById('wire-admin-relays').open=true;d.getElementById('wire-admin-relays').dispatchEvent(new app.dom.window.Event('toggle'));await settle();
 d.querySelector('button[data-what=usdc_polygon]').click();await settle();
 assert.deepEqual(app.calls.find(([p])=>p.endsWith('/withdraw')),['/api/admin/relayers/withdraw',{chain:'polygon',what:'usdc_polygon',to:'0x4444444444444444444444444444444444444444'}]);
 const link=d.querySelector('.wire-admin-relay-result a');assert.equal(link.href,'https://polygonscan.com/tx/'+tx);assert.equal(link.rel,'noopener noreferrer');
 assert.equal(app.calls.filter(([p])=>p==='/api/admin/relayers').length,2); // the balances are read again
 app.dom.window.close();
});
test('withdrawal errors are plain words, an empty list of addresses switches withdrawals off, and no wallet is no section content',async()=>{
 const none=await setup({relays:{...relayView,withdraw_to:[]}});none.d.getElementById('wire-admin-relays').open=true;none.d.getElementById('wire-admin-relays').dispatchEvent(new none.dom.window.Event('toggle'));await settle();
 assert.match(none.d.getElementById('wire-admin-relays').textContent,/Withdrawals are off/);assert.ok([...none.d.querySelectorAll('button[data-what]')].every(b=>b.disabled));none.dom.window.close();
 for(const code of ['nothing_to_withdraw','withdraw_destination_not_allowed','reserve_required']){
  const app=await setup({relays:relayView,withdrawal:new Error(code)});app.d.getElementById('wire-admin-relays').open=true;app.d.getElementById('wire-admin-relays').dispatchEvent(new app.dom.window.Event('toggle'));await settle();
  app.d.querySelector('button[data-what=native]').click();await settle();assert.equal(app.d.querySelector('.wire-admin-relay-status').textContent,relayText('en',code));app.dom.window.close();
 }
 const unavailable=await setup();unavailable.d.getElementById('wire-admin-relays').open=true;unavailable.d.getElementById('wire-admin-relays').dispatchEvent(new unavailable.dom.window.Event('toggle'));await settle();
 assert.equal(unavailable.d.querySelector('.wire-admin-relay-status').textContent,relayText('en','usdt_unavailable'));unavailable.dom.window.close();
 for(const language of ['fr','en','es','de','zh'])for(const key of ['relays','withdraw','confirmWithdraw','nothing_to_withdraw','withdraw_destination_not_allowed','reserve_required','help'])assert.ok(relayText(language,key).length>1);
});
test('amounts are cut, never rounded, and never go through a float',()=>{
 assert.equal(amountText('9499999999999999999',18,4),'9.4999');assert.equal(amountText('3250000',6,4),'3.2500');assert.equal(amountText('0',18,4),'0.0000');assert.equal(amountText('oops',18,4),'—');
});

test('tiny positive amounts never look like zero; invalid and negative balances are not shown as valid amounts',()=>{
 assert.equal(amountText('1',18,4),'<0.0001');assert.equal(amountText('1',6,4),'<0.0001');
 for(const value of ['-1','1.5','',null])assert.equal(amountText(value,18,4),'—');
 assert.equal(amountText('1',-1,4),'—');assert.equal(amountText('1',18,19),'—');
});
test('relay controls stay labeled and bounded in five languages and all Wire appearances',async()=>{
 const {readFileSync}=await import('node:fs');
 for(const language of ['fr','en','es','de','zh']){
  const app=await setup({language,relays:{...relayView,withdraw_to:[...relayView.withdraw_to,'0x'+'55'.repeat(20)]}}),d=app.d;
  Object.defineProperty(app.dom.window,'innerWidth',{value:375});
  const style=d.createElement('style');style.textContent=readFileSync(new URL('../internal/web/static/wire-pay.css',import.meta.url),'utf8');d.head.append(style);
  const box=d.getElementById('wire-admin-relays');box.open=true;box.dispatchEvent(new app.dom.window.Event('toggle'));await settle();
  assert.equal(box.firstElementChild.tagName,'SUMMARY');assert.equal(box.firstElementChild.textContent,relayText(language,'relays'));
  assert.equal(box.querySelector('.wire-admin-relay-status').getAttribute('aria-live'),'polite');
  assert.equal(d.querySelectorAll('.wire-relay-metrics dt').length,d.querySelectorAll('.wire-relay-metrics dd').length);
  const select=d.getElementById('wire-admin-relay-to');select.value='0x'+'55'.repeat(20);select.dispatchEvent(new app.dom.window.Event('change'));
  box.querySelector('button[data-what=native]').click();await settle();
  assert.equal(app.calls.find(([path])=>path.endsWith('/withdraw'))[1].to,'0x'+'55'.repeat(20));
  for(const theme of ['dark','light-paper'])for(const palette of ['phoenixfire','aurora','ocean']){
   d.documentElement.dataset.theme=theme;d.documentElement.dataset.palette=palette;
   for(const button of box.querySelectorAll('button'))assert.ok(parseFloat(app.dom.window.getComputedStyle(button).minHeight)>=40);
   assert.ok(parseFloat(app.dom.window.getComputedStyle(d.getElementById('wire-admin-maintenance')).minHeight)>=40);
  }
  assert.doesNotMatch(box.textContent,/Mainnet|OTP|Hub|undefined/);app.dom.window.close();
 }
});
test('maintenance keeps keyboard focus after its response replaces the checkbox',async()=>{
 const app=await setup(),toggle=app.d.getElementById('wire-admin-maintenance');toggle.focus();toggle.click();await settle();
 assert.equal(app.d.activeElement,app.d.getElementById('wire-admin-maintenance'));assert.equal(app.d.activeElement.checked,true);app.dom.window.close();
});
