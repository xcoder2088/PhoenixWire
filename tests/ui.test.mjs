import test from 'node:test';
import assert from 'node:assert/strict';
import vm from 'node:vm';
import { readFileSync } from 'node:fs';
import { assets, previewAmount, resolveLanguage,cardPresentation,paintCard,assetMark } from '../internal/web/static/model.js';

// Dependency-free DOM contract harness. Real browser visual QA remains a release task.
function start({ storageUnavailable = false, languages = ['fr-CA'], language = 'fr-CA', savedLanguage } = {}) {
  const html = readFileSync(new URL('../internal/web/static/classic.html', import.meta.url), 'utf8');
  const nodes = new Map();
  const translations = [];
  for (const match of html.matchAll(/data-i18n="([^"]+)"/g)) translations.push({ dataset: { i18n: match[1] }, textContent: '' });
  for (const match of html.matchAll(/id="([^"]+)"/g)) nodes.set(match[1], { value: '', dataset: {},style:{setProperty(){}}, textContent: '', handlers: {}, append(child){this.child=child;},addEventListener(event, fn) { this.handlers[event] = fn; } });
  nodes.get('currency').value = 'NIM';
  nodes.get('amount').value = '25';
  const document = { documentElement: { dataset: {} },createElement(){return {};}, getElementById(id) { return nodes.get(id); }, querySelectorAll() { return translations; } };
  const stored = new Map();
  if (savedLanguage) stored.set('wire-language', savedLanguage);
  const localStorage = { getItem(k) { if (storageUnavailable) throw Error('unavailable'); return stored.get(k); }, setItem(k,v) { if (storageUnavailable) throw Error('unavailable'); stored.set(k,v); } };
  const source = readFileSync(new URL('../internal/web/static/app.js', import.meta.url), 'utf8').replace(/^import[^\n]+\n/, '');
  vm.runInNewContext(source, { document, localStorage, assets, previewAmount, resolveLanguage,cardPresentation,paintCard,assetMark, navigator: { languages, language } });
  const event = (id, name, value) => { const node = nodes.get(id); if (value !== undefined) node.value = value; node.handlers[name]({ target: node, preventDefault() {} }); };
  return { document, nodes, translations, stored, event };
}
test('all five languages have nonempty labels and translated preview notices', () => {
  const ui = start();
  for (const lang of ['fr','en','es','de','zh']) {
    ui.event('language', 'change', lang);
    assert.equal(ui.document.documentElement.lang, lang);
    assert.ok(ui.translations.every(node => typeof node.textContent === 'string' && node.textContent.length > 0));
  }
});
test('preview renders literal content and persists no form data', () => {
  const ui = start();
  ui.nodes.get('recipient').value = '<img src=x onerror=alert(1)>@example.com';
  ui.nodes.get('message').value = '<script>alert(1)</script>';
  ui.nodes.get('amount').value = '25,50';
  ui.event('card-form', 'submit');
  assert.equal(ui.nodes.get('card-amount').textContent, '25.50');
  assert.equal(ui.nodes.get('card-message').textContent, '<script>alert(1)</script>');
  assert.equal(ui.stored.size, 0);
  ui.event('currency', 'change', 'USDC_POLYGON');
  assert.equal(ui.nodes.get('card-unit').textContent, 'USDC');
  assert.equal(ui.nodes.get('card-network').textContent, 'Polygon');
});
test('themes and accents work even when storage is unavailable', () => {
  const ui = start({ storageUnavailable: true });
  ui.event('theme', 'click');
  assert.equal(ui.document.documentElement.dataset.theme, 'light');
  ui.event('accent', 'change', 'ocean');
  assert.equal(ui.document.documentElement.dataset.palette, 'ocean');
});
test('invalid amount preserves the last valid card', () => {
  const ui = start();
  ui.event('card-form', 'submit');
  ui.nodes.get('amount').value = '-20';
  ui.event('card-form', 'submit');
  assert.equal(ui.nodes.get('card-amount').textContent, '25');
  assert.match(ui.nodes.get('feedback').textContent, /invalide/);
});

test('first launch follows device language and regional tags', () => {
  for (const [locale, expected] of [['fr-CA','fr'],['en-US','en'],['es-MX','es'],['de-AT','de'],['zh-Hant-TW','zh']]) {
    const ui = start({ languages: [locale] });
    assert.equal(ui.document.documentElement.lang, expected);
    assert.equal(ui.nodes.get('language').value, 'auto');
    assert.equal(ui.stored.size, 0);
  }
});
test('unsupported languages use the next supported preference, otherwise English', () => {
  assert.equal(start({ languages: ['it-IT','de-DE'] }).document.documentElement.lang, 'de');
  assert.equal(start({ languages: ['it-IT','ja-JP'] }).document.documentElement.lang, 'en');
  assert.equal(start({ languages: [], language: 'es-AR' }).document.documentElement.lang, 'es');
});
test('manual choice persists, takes precedence, and can return to automatic', () => {
  const ui = start({ savedLanguage: 'fr', languages: ['de-DE'] });
  assert.equal(ui.document.documentElement.lang, 'fr');
  ui.event('language', 'change', 'es');
  assert.equal(ui.document.documentElement.lang, 'es');
  assert.equal(ui.stored.get('wire-language'), 'es');
  ui.event('language', 'change', 'auto');
  assert.equal(ui.document.documentElement.lang, 'de');
  assert.equal(ui.stored.get('wire-language'), 'auto');
  assert.equal(ui.nodes.get('language').value, 'auto');
});
test('corrupt preferences or unavailable storage do not block device language detection', () => {
  assert.equal(start({ savedLanguage: '__proto__', languages: ['zh-CN'] }).document.documentElement.lang, 'zh');
  assert.equal(start({ storageUnavailable: true, languages: ['en-GB'] }).document.documentElement.lang, 'en');
});
test('the card never names the NIM for another coin, and the heading carries no network prefix', () => {
  const ui = start({ languages: ['en'], language: 'en' });
  for (const [currency, unit] of [['NIM', 'NIM'], ['USDC_POLYGON', 'USDC'], ['USDT_ETHEREUM', 'USDT']]) {
    ui.nodes.get('currency').value = currency;
    ui.event('card-form', 'submit');
    assert.ok(ui.nodes.get('card-lock').textContent.includes(unit), currency);
    if (unit !== 'NIM') assert.ok(!ui.nodes.get('card-lock').textContent.includes('NIM'), currency);
    assert.equal(ui.nodes.get('card-network').hidden, false);
  }
  const heading = ui.translations.find(n => n.dataset.i18n === 'cardHeading');
  assert.ok(!/nimiq/i.test(heading.textContent));
});

