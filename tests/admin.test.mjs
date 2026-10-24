import test from 'node:test';
import assert from 'node:assert/strict';
import {JSDOM} from 'jsdom';
import {mountAdmin,adminText} from '../internal/web/static/admin.js';
const settle=async()=>{for(let i=0;i<6;i++)await new Promise(r=>setImmediate(r));};
async function setup({authorized=true,language='en',confirm=true,error=''}={}){
 const dom=new JSDOM('<main></main>'),d=dom.window.document,calls=[],confirmations=[];let view={open:true,registered:['sender@example.test','admin@example.test'],allowed:['allowed@example.test'],blocked:['blocked@example.test'],admins:['admin@example.test']};
 dom.window.confirm=text=>{confirmations.push(text);return confirm;};
 const request=async(path,data)=>{calls.push([path,data]);if(data&&error)throw Object.assign(Error(error),{code:error});
  if(path==='/api/admin/access'&&data)view={...view,open:data.open};
  if(path==='/api/admin/block'&&data)view={...view,allowed:view.allowed.filter(e=>e!==data.email),blocked:[...view.blocked,data.email]};
  if(path==='/api/admin/allow'&&data)view={...view,blocked:view.blocked.filter(e=>e!==data.email),allowed:[...view.allowed,data.email]};
  return JSON.parse(JSON.stringify(view));};
 const panel=mountAdmin({root:d.querySelector('main'),authorized,language:()=>language,request});await settle();return{dom,d,calls,confirmations,panel};
}
test('only an exact administrator authorization mounts a panel or fetches access data',async()=>{
 for(const authorized of [false,null,'true',1]){const app=await setup({authorized});assert.equal(app.d.getElementById('wire-admin'),null);assert.equal(app.calls.length,0);app.dom.window.close();}
 const app=await setup();assert.ok(app.d.getElementById('wire-admin'));assert.equal(app.d.getElementById('wire-admin-open').checked,true);assert.equal(app.calls[0][0],'/api/admin/access');app.panel.destroy();assert.equal(app.d.getElementById('wire-admin'),null);app.dom.window.close();
});
test('opening and closing access uses the server response and closing asks for confirmation',async()=>{
 const app=await setup(),toggle=app.d.getElementById('wire-admin-open');toggle.checked=false;toggle.dispatchEvent(new app.dom.window.Event('change'));await settle();assert.equal(app.confirmations.length,1);assert.deepEqual(app.calls.at(-1),['/api/admin/access',{open:false}]);assert.equal(app.d.getElementById('wire-admin-open').checked,false);
 app.d.getElementById('wire-admin-open').checked=true;app.d.getElementById('wire-admin-open').dispatchEvent(new app.dom.window.Event('change'));await settle();assert.equal(app.confirmations.length,1);assert.equal(app.d.getElementById('wire-admin-open').checked,true);app.dom.window.close();
 const refused=await setup({confirm:false});refused.d.getElementById('wire-admin-open').checked=false;refused.d.getElementById('wire-admin-open').dispatchEvent(new refused.dom.window.Event('change'));await settle();assert.equal(refused.calls.length,1);assert.equal(refused.d.getElementById('wire-admin-open').checked,true);refused.dom.window.close();
});
test('registered, allowed and blocked lists expose safe allow/block actions and refresh from replies',async()=>{
 const app=await setup(),d=app.d;
 const rows=[...d.querySelectorAll('[data-list=registered] li')];assert.equal(rows[1].querySelector('button').disabled,true);rows[0].querySelector('button').click();await settle();assert.deepEqual(app.calls.at(-1),['/api/admin/block',{email:'sender@example.test'}]);assert.equal(app.confirmations.length,1);
 d.querySelector('[data-list=allowed] button').click();await settle();assert.deepEqual(app.calls.at(-1),['/api/admin/block',{email:'allowed@example.test'}]);assert.ok(d.querySelector('[data-list=blocked]').textContent.includes('allowed@example.test'));
 d.querySelector('[data-list=blocked] button').click();await settle();assert.deepEqual(app.calls.at(-1),['/api/admin/allow',{email:'blocked@example.test'}]);
 const form=d.getElementById('wire-admin-allow');form.querySelector('input').value='new@example.test';form.dispatchEvent(new app.dom.window.Event('submit',{cancelable:true}));await settle();assert.deepEqual(app.calls.at(-1),['/api/admin/allow',{email:'new@example.test'}]);assert.ok(d.querySelector('[data-list=allowed]').textContent.includes('new@example.test'));app.dom.window.close();
 const refused=await setup({confirm:false});refused.d.querySelector('[data-list=registered] button').click();await settle();assert.equal(refused.calls.length,1);refused.dom.window.close();
});
test('administrator errors are plain words in all five languages and controls recover',async()=>{
 for(const language of ['fr','en','es','de','zh'])for(const error of ['admin_required','cannot_block_admin','list_full','invalid_email']){
  const app=await setup({language,error}),form=app.d.getElementById('wire-admin-allow');form.querySelector('input').value='new@example.test';form.dispatchEvent(new app.dom.window.Event('submit',{cancelable:true}));await settle();const message=app.d.getElementById('wire-admin-status').textContent;assert.equal(message,adminText(language,error));assert.doesNotMatch(message,/admin_required|cannot_block_admin|list_full|invalid_email|Mainnet|OTP|Hub/);assert.equal(app.d.querySelector('[aria-busy=true]'),null);app.dom.window.close();
 }
});
