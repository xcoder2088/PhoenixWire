import test from 'node:test';
import assert from 'node:assert/strict';
import vm from 'node:vm';
import { readFileSync } from 'node:fs';
import { assets, previewAmount } from '../internal/web/static/model.js';

// Dependency-free DOM contract harness. Real browser visual QA remains a release task.
function start({ storageUnavailable = false } = {}) {
  const html = readFileSync(new URL('../internal/web/static/index.html', import.meta.url), 'utf8');
  const nodes = new Map();
  const translations = [];
  for (const match of html.matchAll(/data-i18n="([^"]+)"/g)) translations.push({ dataset: { i18n: match[1] }, textContent: '' });
  for (const match of html.matchAll(/id="([^"]+)"/g)) nodes.set(match[1], { value: '', dataset: {}, textContent: '', handlers: {}, addEventListener(event, fn) { this.handlers[event] = fn; } });
  nodes.get('currency').value = 'NIM';
  nodes.get('amount').value = '25';
  const document = { documentElement: { dataset: {} }, getElementById(id) { return nodes.get(id); }, querySelectorAll() { return translations; } };
  const stored = new Map();
  const localStorage = { getItem(k) { if (storageUnavailable) throw Error('unavailable'); return stored.get(k); }, setItem(k,v) { if (storageUnavailable) throw Error('unavailable'); stored.set(k,v); } };
  const source = readFileSync(new URL('../internal/web/static/app.js', import.meta.url), 'utf8').replace(/^import[^\n]+\n/, '');
  vm.runInNewContext(source, { document, localStorage, assets, previewAmount });
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
  ui.event('currency', 'change', 'USDT0_POLYGON');
  assert.equal(ui.nodes.get('card-unit').textContent, 'USDT0');
  assert.equal(ui.nodes.get('card-network').textContent, 'Polygon');
});
test('themes and accents work even when storage is unavailable', () => {
  const ui = start({ storageUnavailable: true });
  ui.event('theme', 'click');
  assert.equal(ui.document.documentElement.dataset.theme, 'light');
  ui.event('accent', 'change', 'teal');
  assert.equal(ui.document.documentElement.dataset.accent, 'teal');
});
test('invalid amount preserves the last valid card', () => {
  const ui = start();
  ui.event('card-form', 'submit');
  ui.nodes.get('amount').value = '-20';
  ui.event('card-form', 'submit');
  assert.equal(ui.nodes.get('card-amount').textContent, '25');
  assert.match(ui.nodes.get('feedback').textContent, /invalide/);
});
