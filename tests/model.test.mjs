import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { assets, previewAmount, resolveLanguage } from '../internal/web/static/model.js';

test('locale resolver matches PhoenixMail precedence and regional normalization', () => {
  assert.equal(resolveLanguage('es', ['fr-CA']), 'es');
  assert.equal(resolveLanguage('auto', ['pt-BR', 'ZH_hant']), 'zh');
  assert.equal(resolveLanguage('unknown', 'de-DE'), 'de');
  assert.equal(resolveLanguage('auto', []), 'en');
  assert.equal(resolveLanguage('auto', [null, undefined, 'it-IT']), 'en');
});

test('preview supports explicit asset/network pairs', () => {
  assert.deepEqual(Object.keys(assets), ['NIM', 'USDT_ETHEREUM', 'USDT0_POLYGON']);
  assert.equal(assets.USDT0_POLYGON.network, 'Polygon');
});
test('decimal preview formatting never uses floating point', () => {
  assert.equal(previewAmount('00025,12345', 'NIM'), '25.12345');
  assert.equal(previewAmount('0.000001', 'USDT_ETHEREUM'), '0.000001');
  assert.equal(previewAmount('999999999999.123456', 'USDT0_POLYGON'), '999999999999.123456');
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
