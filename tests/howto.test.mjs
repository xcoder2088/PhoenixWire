import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import { JSDOM } from 'jsdom';
import { renderHowto } from '../internal/web/static/howto.js';

const read = path => fs.readFileSync(new URL('../internal/web/static/' + path, import.meta.url), 'utf8');

for (const lang of ['fr', 'en']) test('how-to page explains every step in ' + lang + ' and its images exist', () => {
 const dom = new JSDOM(read('howto.html'));
 const doc = dom.window.document;
 try {
  renderHowto(doc, lang);
  assert.equal(doc.documentElement.lang, lang);
  assert.equal(doc.querySelectorAll('#howto-content .howto-step').length, 6);
  assert.equal(doc.querySelectorAll('.howto-num').length, 5);
  const images = [...doc.querySelectorAll('#howto-content img')];
  assert.equal(images.length, 5);
  for (const img of images) {
   assert.match(img.getAttribute('src'), new RegExp('^/howto/[a-z-]+-' + lang + '\\.webp$'));
   assert.ok(fs.existsSync(new URL('../internal/web/static' + img.getAttribute('src'), import.meta.url)), img.getAttribute('src'));
   assert.ok(img.alt);
  }
  assert.ok(doc.querySelector('a[href="/terms.html?lang=' + lang + '"]'));
 } finally { dom.window.close(); }
});

test('the sender page links the how-to page from its footer', () => {
 const doc = new JSDOM(read('index.html')).window.document;
 assert.equal(doc.querySelector('footer a[data-howto-link]').getAttribute('href'), '/howto.html');
});
