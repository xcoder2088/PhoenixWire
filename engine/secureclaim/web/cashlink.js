/* Mirror of PhoenixCashlink 924d491 (scripts/vendor.sh). Do not edit here. */
/* Copyright (C) 2026 Xcoder — SPDX-License-Identifier: MIT (originally AGPL-3.0-only in PhoenixCashlink; licensed here under the MIT License by the copyright holder)
 * PhoenixCashlink in the browser: link key, card address, Safe transaction hash, owner signatures, payout calldata.
 * Plain JavaScript (BigInt + WebCrypto), no dependency. Every result is checked against the Go code by test vectors
 * (testdata/vectors.json). The only secret ever handled here is the link key; it never leaves the page. */
(function(root){'use strict';
const enc=new TextEncoder();
const cat=(...a)=>{const n=a.reduce((s,x)=>s+x.length,0),o=new Uint8Array(n);let i=0;for(const x of a){o.set(x,i);i+=x.length}return o};
const hex=b=>Array.from(b,x=>x.toString(16).padStart(2,'0')).join('');
const unhex=s=>{s=String(s).replace(/^0x/,'');if(s.length%2||/[^0-9a-fA-F]/.test(s))throw Error('bad_hex');const o=new Uint8Array(s.length/2);for(let i=0;i<o.length;i++)o[i]=parseInt(s.slice(2*i,2*i+2),16);return o};
const big=b=>{let v=0n;for(const x of b)v=(v<<8n)|BigInt(x);return v};
const word=v=>{let x=BigInt(v);if(x<0n)throw Error('negative');const o=new Uint8Array(32);for(let i=31;i>=0;i--){o[i]=Number(x&0xffn);x>>=8n}if(x!==0n)throw Error('overflow');return o};

// ---- Keccak-256 (original Keccak padding, as Ethereum uses) ----
const M64=(1n<<64n)-1n;
const RC=[0x0000000000000001n,0x0000000000008082n,0x800000000000808an,0x8000000080008000n,0x000000000000808bn,0x0000000080000001n,0x8000000080008081n,0x8000000000008009n,0x000000000000008an,0x0000000000000088n,0x0000000080008009n,0x000000008000000an,0x000000008000808bn,0x800000000000008bn,0x8000000000008089n,0x8000000000008003n,0x8000000000008002n,0x8000000000000080n,0x000000000000800an,0x800000008000000an,0x8000000080008081n,0x8000000000008080n,0x0000000080000001n,0x8000000080008008n];
const ROT=[1,3,6,10,15,21,28,36,45,55,2,14,27,41,56,8,25,43,62,18,39,61,20,44],PI=[10,7,11,17,18,3,5,16,8,21,24,4,15,23,19,13,12,2,20,14,22,9,6,1];
const rotl=(x,n)=>((x<<BigInt(n))|(x>>BigInt(64-n)))&M64;
function keccakF(a){const bc=new Array(5);for(let round=0;round<24;round++){
 for(let i=0;i<5;i++)bc[i]=a[i]^a[i+5]^a[i+10]^a[i+15]^a[i+20];
 for(let i=0;i<5;i++){const t=bc[(i+4)%5]^rotl(bc[(i+1)%5],1);for(let j=0;j<25;j+=5)a[j+i]^=t}
 let t=a[1];for(let i=0;i<24;i++){const j=PI[i];bc[0]=a[j];a[j]=rotl(t,ROT[i]);t=bc[0]}
 for(let j=0;j<25;j+=5){for(let i=0;i<5;i++)bc[i]=a[j+i];for(let i=0;i<5;i++)a[j+i]^=(~bc[(i+1)%5]&M64)&bc[(i+2)%5]}
 a[0]^=RC[round]}}
function keccak256(...parts){const bytes=cat(...parts),rate=136,state=new Array(25).fill(0n),padded=new Uint8Array(Math.ceil((bytes.length+1)/rate)*rate);padded.set(bytes);padded[bytes.length]^=0x01;padded[padded.length-1]^=0x80;
 for(let off=0;off<padded.length;off+=rate){for(let i=0;i<rate/8;i++){let lane=0n;for(let k=7;k>=0;k--)lane=(lane<<8n)|BigInt(padded[off+i*8+k]);state[i]^=lane}keccakF(state)}
 const out=new Uint8Array(32);for(let i=0;i<4;i++){let lane=state[i];for(let k=0;k<8;k++){out[i*8+k]=Number(lane&0xffn);lane>>=8n}}return out}

// ---- addresses (EIP-55) ----
function checksum(addr20){const lower=hex(addr20),h=hex(keccak256(enc.encode(lower)));let o='0x';for(let i=0;i<40;i++)o+=(/[a-f]/.test(lower[i])&&parseInt(h[i],16)>=8)?lower[i].toUpperCase():lower[i];return o}
const addressBytes=a=>{const b=unhex(a);if(b.length!==20)throw Error('bad_address');return b};
const addrWord=a=>cat(new Uint8Array(12),a instanceof Uint8Array?a:addressBytes(a));
const lessAddr=(a,b)=>{for(let i=0;i<20;i++)if(a[i]!==b[i])return a[i]<b[i];return false};

// ---- secp256k1 (Jacobian coordinates, BigInt). Not constant-time: fine for a one-shot signature in a user's own browser. ----
const P=0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2fn,N=0xfffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141n;
const GX=0x79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798n,GY=0x483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8n;
const mod=(a,m=P)=>{const r=a%m;return r>=0n?r:r+m};
function inv(a,m=P){let [r0,r1]=[mod(a,m),m],[s0,s1]=[1n,0n];while(r1){const q=r0/r1;[r0,r1]=[r1,r0-q*r1];[s0,s1]=[s1,s0-q*s1]}if(r0!==1n)throw Error('no_inverse');return mod(s0,m)}
function jdouble([X,Y,Z]){if(Y===0n||Z===0n)return[0n,1n,0n];const YY=mod(Y*Y),S=mod(4n*X*YY),M=mod(3n*X*X),X3=mod(M*M-2n*S);return[X3,mod(M*(S-X3)-8n*YY*YY),mod(2n*Y*Z)]}
function jadd(p,q){if(p[2]===0n)return q;if(q[2]===0n)return p;const[X1,Y1,Z1]=p,[X2,Y2,Z2]=q,Z1Z1=mod(Z1*Z1),Z2Z2=mod(Z2*Z2),U1=mod(X1*Z2Z2),U2=mod(X2*Z1Z1),S1=mod(Y1*Z2*Z2Z2),S2=mod(Y2*Z1*Z1Z1);
 if(U1===U2)return S1===S2?jdouble(p):[0n,1n,0n];const H=mod(U2-U1),R=mod(S2-S1),HH=mod(H*H),HHH=mod(H*HH),V=mod(U1*HH),X3=mod(R*R-HHH-2n*V);return[X3,mod(R*(V-X3)-S1*HHH),mod(H*Z1*Z2)]}
function mul(k,pt){let r=[0n,1n,0n],a=pt;while(k>0n){if(k&1n)r=jadd(r,a);a=jdouble(a);k>>=1n}return r}
function affine([X,Y,Z]){if(Z===0n)throw Error('infinity');const zi=inv(Z),z2=mod(zi*zi);return[mod(X*z2),mod(Y*z2*zi)]}
const G=[GX,GY,1n];
function publicKey(priv){const k=big(priv);if(k<=0n||k>=N||priv.length!==32)throw Error('bad_private_key');const[x,y]=affine(mul(k,G));return cat(word(x),word(y))}
function addressOfKey(priv){return keccak256(publicKey(priv)).slice(12)}

// ---- ECDSA with RFC 6979 nonces (HMAC-SHA256), low-S, recovery id: identical to the Go side (decred) ----
async function hmac(key,data){const k=await crypto.subtle.importKey('raw',key,{name:'HMAC',hash:'SHA-256'},false,['sign']);return new Uint8Array(await crypto.subtle.sign('HMAC',k,data))}
async function nonce6979(priv,digest){
 const h1=word(mod(big(digest),N));let V=new Uint8Array(32).fill(1),K=new Uint8Array(32);
 K=await hmac(K,cat(V,Uint8Array.of(0),priv,h1));V=await hmac(K,V);K=await hmac(K,cat(V,Uint8Array.of(1),priv,h1));V=await hmac(K,V);
 for(;;){V=await hmac(K,V);const k=big(V);if(k>=1n&&k<N)return k;K=await hmac(K,cat(V,Uint8Array.of(0)));V=await hmac(K,V)}}
async function signDigest(priv,digest){
 if(priv.length!==32||digest.length!==32)throw Error('bad_input');const d=big(priv),z=big(digest),k=await nonce6979(priv,digest);
 const[rx,ry]=affine(mul(k,G));const r=mod(rx,N);if(r===0n)throw Error('bad_nonce');let s=mod(inv(k,N)*(z+r*d),N);if(s===0n)throw Error('bad_nonce');
 let rec=(ry&1n?1:0)|(rx>=N?2:0);if(s>N>>1n){s=N-s;rec^=1}
 return{r:word(r),s:word(s),v:27+rec}}
const sigBytes=g=>cat(g.r,g.s,Uint8Array.of(g.v));

// ---- Safe 1.4.1 (canonical deployments, same on Ethereum and Polygon) ----
const FACTORY=addressBytes('0x4e1DCf7AD4e460CfD30791CCC4F9c8a4f820ec67'),SINGLETON=addressBytes('0x41675C099F32341bf84BFc5382aF534df5C7461a'),MULTISEND_CALL_ONLY=addressBytes('0x9641d764fc13c8B624c04430C7356C1C7C8102e2');
let PROXY_CREATION_CODE=null;
const setProxyCreationCode=b=>{PROXY_CREATION_CODE=b instanceof Uint8Array?b:unhex(b)};
const SEL={setup:unhex('b63e800d'),create:unhex('1688f0b9'),exec:unhex('6a761202'),multisend:unhex('8d80ff0a'),transfer:unhex('a9059cbb')};
const DOMAIN_TYPEHASH=keccak256(enc.encode('EIP712Domain(uint256 chainId,address verifyingContract)'));
const SAFETX_TYPEHASH=keccak256(enc.encode('SafeTx(address to,uint256 value,bytes data,uint8 operation,uint256 safeTxGas,uint256 baseGas,uint256 gasPrice,address gasToken,address refundReceiver,uint256 nonce)'));
const pad32=b=>b.length%32?cat(b,new Uint8Array(32-b.length%32)):b;
function sortedOwners(owners){const l=owners.map(o=>o instanceof Uint8Array?o:addressBytes(o)).sort((a,b)=>lessAddr(a,b)?-1:lessAddr(b,a)?1:0);
 if(!l.length||l.length>3)throw Error('invalid_owners');for(let i=0;i<l.length;i++){if(l[i].every(x=>x===0)||(i&&hex(l[i-1])===hex(l[i])))throw Error('invalid_owners')}return l}
function setupData(owners,threshold){const l=sortedOwners(owners);if(!(threshold>=1&&threshold<=l.length))throw Error('invalid_threshold');const z=new Uint8Array(32);
 return cat(SEL.setup,word(256),word(threshold),z,word(256+32+32*l.length),z,z,z,z,word(l.length),...l.map(addrWord),word(0))}
const saltNonce=(chainId,cardId)=>big(keccak256(enc.encode('PhoenixCashlink v1'),word(chainId),cardId));
function proxyAddress(initializer,salt){if(!PROXY_CREATION_CODE)throw Error('proxy_creation_code_missing');
 const s=keccak256(keccak256(initializer),word(salt)),h=keccak256(Uint8Array.of(0xff),FACTORY,s,keccak256(cat(PROXY_CREATION_CODE,addrWord(SINGLETON))));return h.slice(12)}
function cardAddress({id,owners,threshold},chainId){if(id.length<16||id.length>32)throw Error('invalid_card_id');return proxyAddress(setupData(owners,threshold),saltNonce(chainId,id))}
const deployData=({id,owners,threshold},chainId)=>{const init=setupData(owners,threshold);return cat(SEL.create,addrWord(SINGLETON),word(96),word(saltNonce(chainId,id)),word(init.length),pad32(init))};
const transferData=(to,amount)=>cat(SEL.transfer,addrWord(to),word(amount));
function multiSendData(calls){const packed=cat(...calls.map(c=>cat(Uint8Array.of(0),c.to instanceof Uint8Array?c.to:addressBytes(c.to),word(c.value||0n),word(c.data.length),c.data)));return cat(SEL.multisend,word(32),word(packed.length),pad32(packed))}
// The claim (or refund): the card pays `amount` to the recipient and `fee` to the relayer, in one batch.
function payout({token,recipient,relayer,amount,fee,nonce}){return{to:MULTISEND_CALL_ONLY,value:0n,operation:1,nonce:BigInt(nonce),
 data:multiSendData([{to:token,data:transferData(recipient,amount)},{to:token,data:transferData(relayer,fee)}])}}
function txHash(tx,chainId,safe){const domain=keccak256(DOMAIN_TYPEHASH,word(chainId),addrWord(safe)),z=new Uint8Array(32);
 const body=keccak256(SAFETX_TYPEHASH,addrWord(tx.to),word(tx.value||0n),keccak256(tx.data),word(tx.operation),z,z,z,z,z,word(tx.nonce));
 return keccak256(Uint8Array.of(0x19,0x01),domain,body)}
async function signTx(tx,chainId,safe,priv){const sig=await signDigest(priv,txHash(tx,chainId,safe));return{owner:addressOfKey(priv),sig}}
const packSignatures=list=>cat(...list.slice().sort((a,b)=>lessAddr(a.owner,b.owner)?-1:lessAddr(b.owner,a.owner)?1:0).map(s=>sigBytes(s.sig)));
function execData(tx,signatures){const z=new Uint8Array(32),dataOffset=10*32,sigOffset=dataOffset+32+pad32(tx.data).length;
 return cat(SEL.exec,addrWord(tx.to),word(tx.value||0n),word(dataOffset),word(tx.operation),z,z,z,z,z,word(sigOffset),word(tx.data.length),pad32(tx.data),word(signatures.length),pad32(signatures))}

// ---- proof of link possession (package proof in Go): an EIP-712 LinkAuthorization signed with the link key ----
// It is NOT a Safe transaction (other domain, other type), so it can never be replayed as a spend of the card. The server builds
// the same message itself; the browser only signs it. All byte fields are Uint8Array (cardId, authorizationId: 16 bytes;
// sessionBinding, emailBinding, safeTxHash, challenge: 32 bytes; destination: 20 bytes); action 1 claim, 2 refund, 3 cancel, 4 email code.
const PROOF_DOMAIN_TYPEHASH=keccak256(enc.encode('EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)'));
const PROOF_NAME=keccak256(enc.encode('PhoenixCashlinkAuthorization')),PROOF_VERSION=keccak256(enc.encode('1'));
const PROOF_TYPEHASH=keccak256(enc.encode('LinkAuthorization(bytes16 cardId,bytes16 authorizationId,bytes32 sessionBinding,uint8 action,bytes32 emailBinding,address destination,bytes32 safeTxHash,bytes32 challenge,uint64 deadline)'));
const right32=b=>cat(b,new Uint8Array(32-b.length));
function linkAuthorizationHash(m,chainId,safe){
 const sizes={cardId:16,authorizationId:16,sessionBinding:32,emailBinding:32,destination:20,safeTxHash:32,challenge:32};
 for(const k in sizes)if(!(m[k] instanceof Uint8Array)||m[k].length!==sizes[k])throw Error('invalid_'+k);
 if(![1,2,3,4].includes(m.action))throw Error('invalid_action');
 if((m.action===3||m.action===4)&&(!m.destination.every(x=>x===0)||!m.safeTxHash.every(x=>x===0)))throw Error('invalid_action'); // a cancel or an e-mail code request signs no destination and no transaction
 const domain=keccak256(PROOF_DOMAIN_TYPEHASH,PROOF_NAME,PROOF_VERSION,word(chainId),addrWord(safe));
 const body=keccak256(PROOF_TYPEHASH,right32(m.cardId),right32(m.authorizationId),m.sessionBinding,word(m.action),m.emailBinding,addrWord(m.destination),m.safeTxHash,m.challenge,word(BigInt(m.deadline)));
 return keccak256(Uint8Array.of(0x19,0x01),domain,body)}
async function signLinkAuthorization(m,chainId,safe,priv){return sigBytes(await signDigest(priv,linkAuthorizationHash(m,chainId,safe)))}

// ---- the link: https://claim.example/#c=<card id>.<link key>  (both base64url; the fragment is never sent to a server) ----
const b64u=b=>btoa(String.fromCharCode(...b)).replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,'');
const unb64u=s=>{if(!/^[A-Za-z0-9_-]*$/.test(s))throw Error('bad_link');s=s.replace(/-/g,'+').replace(/_/g,'/');while(s.length%4)s+='=';return Uint8Array.from(atob(s),c=>c.charCodeAt(0))};
function newCard(){const id=crypto.getRandomValues(new Uint8Array(16));let key;do{key=crypto.getRandomValues(new Uint8Array(32))}while(big(key)>=N||big(key)===0n);return{id,key}}
const linkFragment=({id,key})=>'#c='+b64u(id)+'.'+b64u(key);
function parseFragment(fragment){const m=/^#c=([A-Za-z0-9_-]+)\.([A-Za-z0-9_-]+)$/.exec(fragment);if(!m)throw Error('bad_link');const id=unb64u(m[1]),key=unb64u(m[2]);if(id.length<16||id.length>32||key.length!==32)throw Error('bad_link');return{id,key}}

root.PhoenixCashlink={keccak256,checksum,publicKey,addressOfKey,signDigest,sigBytes,setProxyCreationCode,setupData,saltNonce,proxyAddress,cardAddress,deployData,transferData,multiSendData,payout,txHash,signTx,packSignatures,execData,linkAuthorizationHash,signLinkAuthorization,newCard,linkFragment,parseFragment,hex,unhex,FACTORY,SINGLETON,MULTISEND_CALL_ONLY};
})(typeof window!=='undefined'?window:globalThis);
