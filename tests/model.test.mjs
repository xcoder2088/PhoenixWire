import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { assets, previewAmount, resolveLanguage,renderCardPreview } from '../internal/web/static/model.js';
import {cardPalettes} from '../internal/web/static/card-mirror.js';
import {JSDOM} from 'jsdom';

test('NIM and all stablecoin previews share the same twelve pastel palettes and inert card renderer',()=>{
 const dom=new JSDOM(readFileSync(new URL('../internal/web/static/index.html',import.meta.url),'utf8'));
 try{const template=dom.window.document.getElementById('wire-nim-template').content.querySelector('#payment-card');
  for(let i=0;i<12;i++)for(const asset of Object.keys(assets)){
   const card=template.cloneNode(true);renderCardPreview(card,{language:'fr',id:i.toString(16).padStart(2,'0'),asset,amount:'2',message:'<b>inert</b>',recipient:'demo@example.com'});
   assert.equal(card.style.getPropertyValue('--card-bg'),cardPalettes[i][0]);assert.equal(card.style.getPropertyValue('--card-border'),cardPalettes[i][1]);assert.equal(card.querySelector('#card-message').textContent,'<b>inert</b>');assert.equal(card.querySelector('#card-message').childElementCount,0);
  }
 }finally{dom.window.close();}
});

test('locale resolver matches PhoenixMail precedence and regional normalization', () => {
  assert.equal(resolveLanguage('es', ['fr-CA']), 'es');
  assert.equal(resolveLanguage('auto', ['pt-BR', 'ZH_hant']), 'zh');
  assert.equal(resolveLanguage('unknown', 'de-DE'), 'de');
  assert.equal(resolveLanguage('auto', []), 'en');
  assert.equal(resolveLanguage('auto', [null, undefined, 'it-IT']), 'en');
});

test('preview supports explicit asset/network pairs', () => {
  assert.deepEqual(Object.keys(assets), ['NIM', 'USDT_ETHEREUM', 'USDC_POLYGON', 'USDT_POLYGON']);
  assert.equal(assets.USDC_POLYGON.network, 'Polygon');
});
test('decimal preview formatting never uses floating point', () => {
  assert.equal(previewAmount('00025,12345', 'NIM'), '25.12345');
  assert.equal(previewAmount('0.000001', 'USDT_ETHEREUM'), '0.000001');
  assert.equal(previewAmount('999999999999.123456', 'USDC_POLYGON'), '999999999999.123456');
});
test('invalid amounts and networks fail closed', () => {
  for (const value of ['', '0', '0.00000', '-1', '1e3', 'Infinity', '1.000001', '<script>', '1.2.3', '1000000000000']) {
    assert.throws(() => previewAmount(value, 'NIM'));
  }
  assert.throws(() => previewAmount('25', '__proto__'));
  assert.throws(() => previewAmount('25', 'unknown'));
});
test('preview contains no network submission or HTML interpolation', () => {
  const js = readFileSync(new URL('../internal/web/static/app.js', import.meta.url), 'utf8');
  assert.doesNotMatch(js, /fetch\(|XMLHttpRequest|sendBeacon|innerHTML|insertAdjacentHTML/);
  assert.match(js, /e\.preventDefault\(\)/);
  assert.doesNotMatch(js, /localStorage\.setItem\(['"](?:sender|recipient|message)/);
});

