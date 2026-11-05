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
  assert.equal(doc.querySelectorAll('.howto-tip').length, 1, 'only the USDC/USDT key tip remains');
  for (const img of images) {
   assert.match(img.getAttribute('src'), new RegExp('^/howto/[a-z-]+-' + lang + '\\.webp$'));
   assert.ok(fs.existsSync(new URL('../internal/web/static' + img.getAttribute('src'), import.meta.url)), img.getAttribute('src'));
   assert.ok(img.alt);
  }
  assert.ok(doc.querySelector('a[href="/terms.html?lang=' + lang + '"]'));
  const button = doc.querySelector('#howto-video .howto-video-btn');
  assert.ok(button, 'video button');
  assert.equal(doc.querySelector('#howto-video video'), null, 'the video is only loaded after a click');
  button.click();
  const video = doc.querySelector('#howto-video video');
  assert.equal(video.getAttribute('src'), '/howto/phoenix-wire-howto-' + lang + '.mp4');
  assert.ok(fs.statSync(new URL('../internal/web/static' + video.getAttribute('src'), import.meta.url)).size > 1e6);
  assert.equal(doc.querySelector('.howto-video-dl').getAttribute('href'), video.getAttribute('src'));
 } finally { dom.window.close(); }
});

test('the sender page links the how-to page from its footer', () => {
 const doc = new JSDOM(read('index.html')).window.document;
 assert.equal(doc.querySelector('footer a[data-howto-link]').getAttribute('href'), '/howto.html');
});
