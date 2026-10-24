import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {JSDOM} from 'jsdom';
import {legalCopy, pendingMarker, pendingFields, legalLanguage, renderLegal, mountLegal, paintLegalLinks} from '../internal/web/static/legal.js';
const read = path => readFileSync(new URL('../' + path, import.meta.url), 'utf8');
const langs = ['fr','en','es','de','zh'];
const publicPages = ['terms','privacy'];
const knownPending = {terms:['operator','contact','law','minimum-age','prohibited-uses'],privacy:['operator','contact']};
function page(name, suffix='') {return new JSDOM(read('internal/web/static/'+name+'.html'), {url:'https://wire.example/'+name+'.html'+suffix});}
function inspectPending(doc, name) {
 const fields=[...doc.querySelectorAll('[data-pending-field]')];
 assert.deepEqual(fields.map(p=>p.dataset.pendingField), knownPending[name]);
 for(const field of fields){assert.equal(field.querySelector('span').textContent,pendingMarker);assert.equal(field.id,'pending-'+field.dataset.pendingField);}
 assert.equal(doc.body.textContent.split(pendingMarker).length-1,fields.length,'no unlisted placeholder');
}
for(const name of publicPages) for(const lang of langs) test(name+' is complete and has only registered pending fields in '+lang,()=>{
 const dom=page(name),doc=dom.window.document;
 try{
  renderLegal(doc,name,lang);
  assert.equal(doc.documentElement.lang,lang);
  assert.equal(doc.querySelector('h1').textContent,legalCopy[lang].ui[name==='terms'?0:1]);
  assert.equal(doc.title,doc.querySelector('h1').textContent+' · Phoenix Wire');
  assert.ok(doc.querySelectorAll('article section').length>=6);
  assert.ok([...doc.querySelectorAll('article p')].every(p=>p.textContent.trim().length>10));
  assert.doesNotMatch(doc.body.textContent,/\b(?:Mainnet|OTP|Hub|undefined|null)\b/i);
  inspectPending(doc,name);
  assert.equal(doc.querySelectorAll('script:not([src]),style,[style]').length,0);
  assert.ok([...doc.querySelectorAll('script[src],link[rel="stylesheet"]')].every(n=>(n.getAttribute('src')||n.getAttribute('href')).startsWith('/legal.')));
  if(name==='privacy'){
   const table=doc.querySelector('table');assert.equal(table.querySelectorAll('tbody tr').length,6);
   assert.ok([...table.querySelectorAll('tbody tr')].every(r=>r.children.length===3));
   for(const value of ['15','24','160','2','14','1'])assert.ok(table.textContent.includes(value));
   for(const value of ['MailerSend','Cloudflare','Chainlink','CoinGecko','Nimiq Pay','AES-256-GCM'])assert.ok(doc.body.textContent.includes(value));
   assert.equal(doc.querySelector('#not-kept p').textContent,legalCopy[lang].absent[1]);
   assert.equal(doc.querySelector('#deletion p').textContent,legalCopy[lang].deletion[1]);
   assert.match(doc.querySelector('#engine').textContent,/NIM/);
   assert.match(doc.querySelector('#engine').textContent,/USDC\/USDT/);
   assert.match(legalCopy[lang].engine[1],{fr:/clé privée/,en:/private key/,es:/clave privada/,de:/privaten Schlüssel/,zh:/私钥/}[lang]);
  } else {
   assert.match(doc.querySelector('#cards').textContent,/24/);
   assert.match(doc.querySelector('#fees').textContent,/6\s*%/);
   assert.equal(doc.querySelector('#risk p').textContent,legalCopy[lang].risk[1]);
   assert.equal(doc.querySelector('#access p').textContent,legalCopy[lang].access[1]);
  }
 }finally{dom.window.close();}
});
test('static English fallbacks contain all information and the same unresolved fields without scripts',()=>{
 assert.deepEqual(pendingFields,knownPending);
 for(const name of publicPages){const dom=page(name);try{inspectPending(dom.window.document,name);assert.ok(dom.window.document.querySelectorAll('article section').length>=6);if(name==='privacy')assert.equal(dom.window.document.querySelectorAll('tbody tr').length,6);}finally{dom.window.close();}}
 const serialized=JSON.stringify(legalCopy);
 assert.doesNotMatch(serialized,/\b(?:Mainnet|OTP|Hub)\b/i);
 assert.doesNotMatch(serialized,/90 days|90 jours|30 days|30 jours|GDPR compliant|fully anonymous|never sees/i);
 assert.equal(Object.keys(legalCopy).length,5);
 const keys=Object.keys(legalCopy.en).sort();
 for(const lang of langs)assert.deepEqual(Object.keys(legalCopy[lang]).sort(),keys);
 const docs=read('docs/PAYMENT-WINDOW.md');
 for(const field of knownPending.terms)assert.ok(docs.includes('`'+field+'`'),'owner decision must be listed: '+field);
});
test('language selection shares the Wire preference, supports automatic detection and uses no API',()=>{
 const dom=page('terms','?lang=es#not-forwarded'),win=dom.window,doc=win.document;
 try{
  let requests=0;win.fetch=()=>{requests++;throw Error('no API allowed');};
  win.localStorage.setItem('wire-language','de');win.localStorage.setItem('wire-theme','light');win.localStorage.setItem('wire-palette','ocean');
  mountLegal(doc,win);
  assert.equal(doc.documentElement.lang,'es');assert.equal(doc.documentElement.dataset.theme,'light-paper');assert.equal(doc.documentElement.dataset.palette,'ocean');
  const picker=doc.getElementById('wire-language');picker.value='zh';picker.dispatchEvent(new win.Event('change'));
  assert.equal(doc.documentElement.lang,'zh');assert.equal(win.localStorage.getItem('wire-language'),'zh');
  picker.value='auto';picker.dispatchEvent(new win.Event('change'));
  assert.equal(doc.documentElement.lang,legalLanguage('auto',win.navigator.languages));assert.equal(win.localStorage.getItem('wire-language'),'auto');
  doc.getElementById('legal-theme').click();assert.equal(doc.documentElement.dataset.theme,'dark');assert.equal(win.localStorage.getItem('wire-theme'),'dark');
  assert.equal(requests,0);
  for(const link of doc.querySelectorAll('[data-legal-link]'))assert.match(link.getAttribute('href'),/^\/(terms|privacy)\.html\?lang=(fr|en|es|de|zh)$/);
 }finally{win.close();}
});
test('storage denial and invalid preferences do not prevent reading the pages',()=>{
 const dom=page('privacy','?lang=invalid'),win=dom.window;
 try{
  Object.defineProperty(win,'localStorage',{get(){throw Error('storage denied');}});
  assert.doesNotThrow(()=>mountLegal(win.document,win));
  const picker=win.document.getElementById('wire-language');picker.value='fr';picker.dispatchEvent(new win.Event('change'));
  assert.equal(win.document.documentElement.lang,'fr');
  assert.equal(legalLanguage('__proto__',['it-IT','de-AT']),'de');
 }finally{win.close();}
});
test('sender and recipient footers link both pages; card notes have legal links without carrying card data',()=>{
 for(const path of ['internal/web/static/index.html','internal/web/static/classic.html','engine/secureclaim/web/index.html','engine/secureclaim/web/usdt.html']){
  const dom=new JSDOM(read(path));const doc=dom.window.document;
  try{
   for(const name of publicPages)assert.ok(doc.querySelector('footer a[href="/'+name+'.html"]'),path+' footer '+name);
   assert.ok(doc.querySelector('script[src="/legal.js?v=0.4.33"]'));
   assert.ok(doc.querySelector('link[href="/legal.css?v=0.4.33"]'));
   for(const lang of langs){
    paintLegalLinks(doc,lang);
    for(const name of publicPages)assert.equal(doc.querySelector('footer [data-legal-link="'+name+'"]').textContent,legalCopy[lang].ui[name==='terms'?0:1]);
    const powered=doc.querySelector('footer [data-legal-powered]');if(powered)assert.equal(powered.textContent,legalCopy[lang].ui[6]+' ');
   }
   if(path.startsWith('internal/')){
    const card=doc.querySelector('#payment-card')||doc.querySelector('template#wire-nim-template').content.querySelector('#payment-card');
    assert.ok(card.querySelector('.lock'));assert.equal(card.querySelectorAll('[data-legal-link]').length,2);
   }
  }finally{dom.window.close();}
 }
});
test('legal links follow later language changes on existing payment pages',async()=>{
 const dom=new JSDOM('<html lang="en"><body><footer><nav><a data-legal-link="terms"></a><a data-legal-link="privacy"></a></nav></footer></body></html>',{url:'https://wire.example/?claim=example#not-forwarded'});
 try{
  mountLegal(dom.window.document,dom.window);dom.window.document.documentElement.lang='de';
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(dom.window.document.querySelector('a').textContent,'Nutzungsbedingungen');
  assert.equal(dom.window.document.querySelector('a').getAttribute('href'),'/terms.html?lang=de');
  const template=dom.window.document.createElement('template');
  template.innerHTML='<nav><a data-legal-link="terms">Terms of use</a><a data-legal-link="privacy">Privacy</a></nav>';
  dom.window.document.body.append(template);
  paintLegalLinks(dom.window.document,'de');
  assert.equal(template.content.querySelector('a').textContent,'Nutzungsbedingungen');
  const card=dom.window.document.createElement('section');card.innerHTML='<nav><a data-legal-link="privacy">Privacy</a></nav>';
  dom.window.document.body.append(card);
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(card.querySelector('a').textContent,'Datenschutz');
  assert.equal(card.querySelector('a').getAttribute('href'),'/privacy.html?lang=de');
 }finally{dom.window.close();}
});
