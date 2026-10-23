import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import { JSDOM } from 'jsdom';
import { assets,previewAmount,resolveLanguage,cardPresentation,paintCard } from '../internal/web/static/model.js';

test('window opens empty, then shows the pastel card shell and correct token artwork',()=>{
 const dom=new JSDOM(fs.readFileSync(new URL('../internal/web/static/index.html',import.meta.url),'utf8'),{url:'http://localhost:8790',runScripts:'outside-only'});
 Object.assign(dom.getInternalVMContext(),{assets,previewAmount,resolveLanguage,cardPresentation,paintCard});
 const source=fs.readFileSync(new URL('../internal/web/static/app.js',import.meta.url),'utf8').replace(/^import[^\n]+\n/,'');vm.runInContext(source,dom.getInternalVMContext());
 const d=dom.window.document,$=id=>d.getElementById(id);
 assert.ok(d.querySelector('.app-window .workspace .panel'));
 assert.equal($('payment-card').hidden,true);assert.equal($('preview-empty').hidden,false);
 for(const currency of ['NIM','USDT_ETHEREUM','USDC_POLYGON','USDT_POLYGON']){
  $('currency').value=currency;$('card-form').dispatchEvent(new dom.window.Event('submit',{cancelable:true}));
  assert.equal($('preview-empty').hidden,true);assert.equal($('payment-card').hidden,false);
  assert.ok($('payment-card').querySelector('.card-content .coin img'));
  assert.match($('coin').querySelector('img').src,currency==='NIM'?/nimiq-logo\.svg/:currency==='USDC_POLYGON'?/usdc-coin\.png/:/usdt\.svg/);
  assert.equal($('payment-card').querySelector('.card-action').disabled,true);
  assert.equal($('payment-card').querySelector('.card-action').parentElement,$('payment-card'));
  assert.equal($('payment-card').querySelectorAll('footer').length,0);
  assert.equal($('payment-card').querySelector('.sample').textContent,cardPresentation(d.documentElement.lang).protected);
  assert.equal($('card-unit').textContent,assets[currency].unit);
 }
 assert.equal($('payment-card').querySelectorAll('canvas').length,0);
 assert.match(d.querySelector('footer').textContent,/Powered by Phoenix Mail/);
 dom.window.close();
});
