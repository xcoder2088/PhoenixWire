import test from 'node:test';
import assert from 'node:assert/strict';
import {historyEntries,recoverableCard,unpaidCard,historyPending,nimHistoryLabel,senderHistoryWords,cancelUnpaidCard} from '../internal/web/static/model.js';
const now=Date.parse('2026-10-23T14:00:00Z');
const nim=(id,age=0,extra={})=>({id,nim:true,address:'NQ-test',created:new Date(now-age).toISOString(),stage:'sent',claim_state:'unclaimed',...extra});
test('mixed history keeps only three newest transactions from the last 24 hours',()=>{
 const entries=[nim('old',86400001,{stage:'ready',claim_state:undefined}),nim('fourth',4000),nim('second',2000),nim('first',1000),{id:'third',nim:false,created:now-3000,state:'paid'}];
 assert.deepEqual(historyEntries(entries,now).map(p=>p.id),['first','second','third']);
});
test('24-hour boundary retains only older confirmed unclaimed funds',()=>{
 const entries=[nim('safe',86400000),nim('unfunded',86400000,{stage:'funding_started'}),nim('claimed',86400000,{claim_state:'claimed'}),nim('claiming',86400000,{claim_state:'claiming'}),nim('unknown',86400000,{claim_state:undefined})];
 assert.deepEqual(historyEntries(entries,now).map(p=>p.id),['safe']);
});
test('cancelled cards are hidden except a late unclaimed balance, including before expiry',()=>{
 assert.deepEqual(historyEntries([nim('cancelled',0,{stage:'cancelled',claim_state:undefined}),nim('late',90000000,{stage:'cancelled'}),nim('spent',0,{stage:'cancelled',claim_state:'claimed'})],now).map(p=>p.id),['late']);
});
test('claimed and claiming never allow recovery or another payment',()=>{
 for(const claim_state of ['claimed','claiming']){
  const p=nim('terminal',0,{claim_state});assert.equal(recoverableCard(p),false);assert.equal(unpaidCard(p,now),false);
 }
 assert.equal(recoverableCard(nim('unknown',0,{claim_state:undefined})),false);
});
test('recent unknown and sent-unclaimed cards keep polling until confirmed claimed',()=>{
 assert.equal(historyPending(nim('missing',0,{claim_state:undefined}),now),true);
 assert.equal(historyPending(nim('sent'),now),true);
 assert.equal(historyPending(nim('sending',0,{claim_state:'claiming'}),now),true);
 assert.equal(historyPending(nim('done',0,{claim_state:'claimed'}),now),false);
});
test('sender labels and cancellation prompts exist in all five languages without technical names',()=>{
 for(const lang of ['fr','en','es','de','zh']){
  for(const value of Object.values(senderHistoryWords[lang]))assert.ok(value&&!/\b(Mainnet|OTP|Hub)\b/i.test(value));
  assert.equal(nimHistoryLabel(nim('done',0,{claim_state:'claimed'}),lang),senderHistoryWords[lang].received);
  assert.equal(nimHistoryLabel(nim('sending',0,{claim_state:'claiming'}),lang),senderHistoryWords[lang].claiming);
 }
});
test('cancel success sends only the empty body and returns the cancelled payment',async()=>{
 const calls=[],p=nim('ready',0,{stage:'ready',claim_state:undefined});
 const result=await cancelUnpaidCard(p,async(...args)=>{calls.push(args);return {payment:{...p,stage:'cancelled'}};});
 assert.deepEqual(calls,[['/api/payments/ready/cancel',{}]]);assert.equal(result.payment.stage,'cancelled');assert.equal(result.received,false);
});
test('already_funded resumes confirm and is positive feedback even if confirmation is temporarily unavailable',async()=>{
 for(const offline of [false,true]){
  const calls=[],p=nim('race',0,{stage:'ready',claim_state:undefined});
  const result=await cancelUnpaidCard(p,async(path,body)=>{calls.push([path,body]);if(path.endsWith('/cancel'))throw Error('already_funded');if(offline)throw Error('chain_unavailable');return {payment:{...p,stage:'sent',claim_state:'unclaimed'}};});
  assert.equal(result.received,true);assert.deepEqual(calls.map(c=>c[0]),['/api/payments/race/cancel','/api/payments/race/confirm']);assert.notEqual(result.payment.stage,'cancelled');
 }
});
test('cannot_cancel and chain_unavailable preserve the record and remain retryable',async()=>{
 for(const code of ['cannot_cancel','chain_unavailable'])await assert.rejects(cancelUnpaidCard(nim('retry'),async()=>{throw Error(code);}),new RegExp(code));
});
