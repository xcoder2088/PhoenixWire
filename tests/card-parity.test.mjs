import test from 'node:test';
import assert from 'node:assert/strict';
import {cardPresentation} from '../internal/web/static/model.js';
test('protected card copy and twelve pastel families are shared in five languages',()=>{
 for(const language of ['fr','en','es','de','zh']){
  const colors=new Set();
  for(let n=0;n<12;n++){
   const p=cardPresentation(language,n.toString(16).padStart(2,'0'));
   assert.ok(p.heading&&p.action&&p.locked&&p.protected);
   assert.doesNotMatch(Object.values(p).join(' '),/OTP|Mainnet/);
   colors.add(p.background);
  }
  assert.equal(colors.size,12);
 }
 assert.equal(cardPresentation('fr').heading,'Paiement protégé');
 assert.equal(cardPresentation('en').protected,'Protected');
});
