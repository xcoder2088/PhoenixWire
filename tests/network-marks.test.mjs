import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {inflateSync} from 'node:zlib';
import {assetMark} from '../internal/web/static/model.js';

// Decode RGBA PNG pixels independently to check the exported images, including alpha and badge colours.
function pixels(file){
 const b=readFileSync(file);assert.equal(b.subarray(0,8).toString('hex'),'89504e470d0a1a0a');
 const width=b.readUInt32BE(16),height=b.readUInt32BE(20);assert.equal(b[24],8);assert.equal(b[25],6);assert.equal(b[28],0);
 const blocks=[];for(let at=8;at<b.length;){const size=b.readUInt32BE(at),kind=b.toString('ascii',at+4,at+8);if(kind==='IDAT')blocks.push(b.subarray(at+8,at+8+size));at+=size+12;}
 const raw=inflateSync(Buffer.concat(blocks)),stride=width*4,out=Buffer.alloc(stride*height);
 const paeth=(a,c,d)=>{const p=a+c-d,x=Math.abs(p-a),y=Math.abs(p-c),z=Math.abs(p-d);return x<=y&&x<=z?a:y<=z?c:d;};
 for(let y=0;y<height;y++){const filter=raw[y*(stride+1)];assert.ok(filter<=4);
  for(let x=0;x<stride;x++){const i=y*stride+x,a=x>=4?out[i-4]:0,c=y?out[i-stride]:0,d=y&&x>=4?out[i-stride-4]:0,predictor=[0,a,c,Math.floor((a+c)/2),paeth(a,c,d)][filter];out[i]=(raw[y*(stride+1)+x+1]+predictor)&255;}
 }
 return{width,height,pixel:(x,y)=>[...out.subarray((y*width+x)*4,(y*width+x)*4+4)]};
}
test('three distinct 192px transparent network marks have the requested badges and white separating rim',()=>{
 const seen=new Set();for(const [asset,colour]of [['usdt_polygon',[130,71,229]],['usdt_ethereum',[98,126,234]],['usdc_polygon',[130,71,229]]]){
  const mark=assetMark(asset),file=new URL('../internal/web/static'+mark.src,import.meta.url),data=pixels(file);
  assert.equal(data.width,192);assert.equal(data.height,192);assert.equal(data.pixel(0,0)[3],0);assert.equal(data.pixel(191,191)[3],0);assert.deepEqual(data.pixel(183,158),[...colour,255]);assert.deepEqual(data.pixel(190,155).slice(0,3),[255,255,255]);assert.ok(data.pixel(190,155)[3]>230);
  const content=readFileSync(file).toString('base64');assert.ok(!seen.has(content));seen.add(content);
 }
});
test('token/network paths and alternate text are explicit in all five languages',()=>{
 const patterns={fr:'{symbol} sur {chain}',en:'{symbol} on {chain}',es:'{symbol} en {chain}',de:'{symbol} auf {chain}',zh:'{chain} 上的 {symbol}'};
 for(const [lang,pattern]of Object.entries(patterns))for(const [asset,symbol,chain]of [['usdt_polygon','USDT','Polygon'],['usdt_ethereum','USDT','Ethereum'],['usdc_polygon','USDC','Polygon']]){
  const mark=assetMark(asset,lang);assert.equal(mark.src,'/brand/'+asset.replace('_','-')+'.png');assert.equal(mark.alt,pattern.replace('{symbol}',symbol).replace('{chain}',chain));assert.deepEqual(assetMark(asset.toUpperCase(),lang),mark);
 }
 assert.notEqual(assetMark('usdt_polygon').src,assetMark('usdt_ethereum').src);for(const invalid of ['unknown','__proto__','constructor'])assert.equal(assetMark(invalid),null);
});
