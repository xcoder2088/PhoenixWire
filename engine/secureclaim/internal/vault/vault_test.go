package vault

import (
 "bytes"
 "crypto/rand"
 "encoding/json"
 "errors"
 "strings"
 "sync"
 "testing"
)

const testClaim = "0123456789abcdef0123456789abcdef0123456789abcdef"
const otherClaim = "fedcba9876543210fedcba9876543210fedcba9876543210"

func testVault(t *testing.T)*Vault{
 t.Helper()
 master:=make([]byte,KeyBytes)
 if _,err:=rand.Read(master);err!=nil{t.Fatal(err)}
 v,err:=New(master,"key-2026-01")
 clear(master)
 if err!=nil{t.Fatal(err)}
 t.Cleanup(v.Close)
 return v
}
func TestSealRoundTripDoesNotLeakPlaintext(t *testing.T){
 v:=testVault(t)
 secret:=[]byte("DO-NOT-LOG-OR-EMAIL-THIS-NIMIQ-CASHLINK-SEED")
 original:=append([]byte(nil),secret...)
 a,err:=v.Seal(testClaim,secret)
 if err!=nil{t.Fatal(err)}
 b,err:=v.Seal(testClaim,secret)
 if err!=nil{t.Fatal(err)}
 if a.Version!=FormatVersion||a.KeyID!="key-2026-01"||a.Nonce==b.Nonce||a.Ciphertext==b.Ciphertext {
  t.Fatal("envelope metadata or nonce uniqueness failed")
 }
 raw,err:=json.Marshal(a)
 if err!=nil{t.Fatal(err)}
 if bytes.Contains(raw,secret)||bytes.Contains(raw,[]byte("DO-NOT-LOG")) {t.Fatal("plaintext secret leaked")}
 recovered,err:=v.Open(testClaim,a)
 if err!=nil||!bytes.Equal(recovered,secret){t.Fatal("could not decrypt with correct key and claim",err)}
 if !bytes.Equal(secret,original){t.Fatal("seal changed caller's secret")}
 clear(recovered)
}
func TestWrongClaimMasterVersionAndTamperingFailClosed(t *testing.T){
 v:=testVault(t)
 a,err:=v.Seal(testClaim,[]byte("SECRET"))
 if err!=nil{t.Fatal(err)}
 if _,err:=v.Open(otherClaim,a);!errors.Is(err,ErrOpen){t.Fatal("wrong claim decrypted",err)}
 wrong:=testVault(t)
 if _,err:=wrong.Open(testClaim,a);!errors.Is(err,ErrOpen){t.Fatal("wrong master decrypted",err)}
 mutations:=[]Envelope{
  {Version:2,KeyID:a.KeyID,Nonce:a.Nonce,Ciphertext:a.Ciphertext},
  {Version:a.Version,KeyID:"wrong",Nonce:a.Nonce,Ciphertext:a.Ciphertext},
  {Version:a.Version,KeyID:a.KeyID,Nonce:strings.Repeat("0",len(a.Nonce)),Ciphertext:a.Ciphertext},
  {Version:a.Version,KeyID:a.KeyID,Nonce:a.Nonce,Ciphertext:strings.Repeat("0",len(a.Ciphertext))},
  {Version:a.Version,KeyID:a.KeyID,Nonce:"non-hex",Ciphertext:a.Ciphertext},
  {Version:a.Version,KeyID:a.KeyID,Nonce:a.Nonce,Ciphertext:"too-short"},
 }
 for i,e:=range mutations{
  if _,err:=v.Open(testClaim,e);!errors.Is(err,ErrOpen){t.Fatalf("tamper %d decrypted: %v",i,err)}
 }
}
func TestBadInputAndKeyNeverAccepted(t *testing.T){
 v:=testVault(t)
 for _,id:=range []string{"","demo","DEADBEEF",strings.Repeat("g",48),otherClaim+"0"}{
  if _,err:=v.Seal(id,[]byte("S"));!errors.Is(err,ErrInvalid){t.Fatalf("accepted id %q: %v",id,err)}
 }
 for _,secret:=range [][]byte{nil,{},make([]byte,MaxSecretBytes+1)}{
  if _,err:=v.Seal(testClaim,secret);!errors.Is(err,ErrInvalid){t.Fatalf("accepted unsafe secret size %d",len(secret))}
 }
 if _,err:=New(make([]byte,16),"key");!errors.Is(err,ErrInvalid){t.Fatal("accepted short key")}
 if _,err:=New(make([]byte,32),"../invalid");!errors.Is(err,ErrInvalid){t.Fatal("accepted invalid key identifier")}
 if _,err:=DecodeMasterHex("not-a-master-key");!errors.Is(err,ErrInvalid){t.Fatal("accepted password instead of random key")}
 raw:=make([]byte,32);if _,e:=rand.Read(raw);e!=nil{t.Fatal(e)}
 decoded,err:=DecodeMasterHex(hexEncode(raw));if err!=nil||!bytes.Equal(decoded,raw){t.Fatal("valid master encoding failed",err)}
 clear(decoded);clear(raw)
}
func hexEncode(value []byte)string{
 const chars="0123456789abcdef"
 out:=make([]byte,len(value)*2)
 for i,v:=range value{out[2*i]=chars[v>>4];out[2*i+1]=chars[v&15]}
 return string(out)
}
func TestConcurrentClaimsCannotCrossDecrypt(t *testing.T){
 v:=testVault(t)
 const count=32
 var wg sync.WaitGroup
 errs:=make(chan error,count)
 for i:=0;i<count;i++{
  wg.Add(1)
  go func(i int){
   defer wg.Done()
   id:=[]byte(testClaim);id[i%len(id)]= "abcdef0123456789"[i%16]
   // Ensure two distinct IDs regardless of starting input byte.
   id[(i+3)%len(id)]= "abcdef"[i%6]
   secret:=[]byte{byte(i),0x31,0x79,0x93}
   envelope,e:=v.Seal(string(id),secret)
   if e!=nil {errs<-e;return}
   recovered,e:=v.Open(string(id),envelope)
   if e!=nil||!bytes.Equal(recovered,secret){errs<-errors.New("parallel seal/open mismatch");return}
   clear(recovered)
  }(i)
 }
 wg.Wait();close(errs)
 for err:=range errs{t.Error(err)}
}
func TestCloseInvalidatesKey(t *testing.T){
 v:=testVault(t)
 saved,err:=v.Seal(testClaim,[]byte("old"))
 if err!=nil{t.Fatal(err)}
 v.Close()
 if _,err:=v.Open(testClaim,saved);!errors.Is(err,ErrOpen){t.Fatal("closed master decrypted earlier envelope",err)}
}
