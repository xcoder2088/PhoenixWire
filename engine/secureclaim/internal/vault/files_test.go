package vault

import (
 "bytes"
 "errors"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestSealedFileIsPrivateAtomicAndContainsNoCashlinkSecret(t *testing.T){
 v:=testVault(t)
 root:=filepath.Join(t.TempDir(),"private")
 store,err:=NewFileStore(root,v)
 if err!=nil{t.Fatal(err)}
 if info,e:=os.Stat(root);e!=nil||info.Mode().Perm()!=0700{t.Fatal("vault directory permissions",e)}
 seed:=[]byte("VERY-PRIVATE-NIMIQ-CASHLINK-SPENDING-SEED")
 if err:=store.Put(testClaim,seed);err!=nil{t.Fatal(err)}
 file:=filepath.Join(root,testClaim+".sealed.json")
 info,err:=os.Stat(file)
 if err!=nil||info.Mode().Perm()!=0600{t.Fatal("sealed file permissions",err)}
 raw,err:=os.ReadFile(file)
 if err!=nil||bytes.Contains(raw,seed)||bytes.Contains(raw,[]byte("PRIVATE")){t.Fatal("secret leaked into persisted envelope",err)}
 if !bytes.Contains(raw,[]byte(`"ciphertext"`)){t.Fatal("missing ciphertext")}
 recovered,err:=store.Open(testClaim)
 if err!=nil||!bytes.Equal(recovered,seed){t.Fatal("correct recovery failed",err)}
 clear(recovered)
 if err:=store.Put(testClaim,[]byte("REPLACEMENT"));!errors.Is(err,ErrAlreadyStored){t.Fatal("overwritten existing Cashlink secret",err)}
 unchanged,err:=store.Open(testClaim)
 if err!=nil||!bytes.Equal(unchanged,seed){t.Fatal("secret changed despite create-once guard",err)}
 clear(unchanged)
 if err:=store.Delete(testClaim);err!=nil{t.Fatal(err)}
 if _,err:=store.Open(testClaim);!errors.Is(err,ErrMissing){t.Fatal("deleted envelope still readable",err)}
}
func TestFileStoreWrongClaimKeyOrTamperedContentsFail(t *testing.T){
 v:=testVault(t)
 store,err:=NewFileStore(filepath.Join(t.TempDir(),"private"),v)
 if err!=nil{t.Fatal(err)}
 if _,err:=store.Open(testClaim);!errors.Is(err,ErrMissing){t.Fatal("missing vault secret accepted",err)}
 if err:=store.Put(testClaim,[]byte("S"));err!=nil{t.Fatal(err)}
 if _,err:=store.Open(otherClaim);!errors.Is(err,ErrMissing){t.Fatal("cross-claim retrieval accepted",err)}
 other:=testVault(t)
 wrong,err:=NewFileStore(store.root,other)
 if err!=nil{t.Fatal(err)}
 if _,err:=wrong.Open(testClaim);!errors.Is(err,ErrOpen){t.Fatal("wrong key opened secret",err)}
 file:=filepath.Join(store.root,testClaim+".sealed.json")
 if err:=os.WriteFile(file,[]byte(`{"version":1,"key_id":"key-2026-01","nonce":"000000000000000000000000","ciphertext":"00000000000000000000000000000000"}`),0600);err!=nil{t.Fatal(err)}
 if _,err:=store.Open(testClaim);!errors.Is(err,ErrOpen){t.Fatal("tampered secret decrypted",err)}
}
func TestFileStoreRejectsNonPrivateDirectoryAndPathTraversal(t *testing.T){
 v:=testVault(t)
 root:=filepath.Join(t.TempDir(),"untrusted")
 if err:=os.Mkdir(root,0755);err!=nil{t.Fatal(err)}
 if _,err:=NewFileStore(root,v);!errors.Is(err,ErrInvalid){t.Fatal("world-readable vault directory allowed",err)}
 if err:=os.Chmod(root,0700);err!=nil{t.Fatal(err)}
 store,err:=NewFileStore(root,v)
 if err!=nil{t.Fatal(err)}
 for _,id:=range []string{"../../foo","demo",strings.Repeat("g",48),""}{
  if e:=store.Put(id,[]byte("secret"));!errors.Is(e,ErrInvalid){t.Fatalf("unsafe path %q accepted: %v",id,e)}
 }
}
