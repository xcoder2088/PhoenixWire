package vault

import (
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "path/filepath"
)

// FileStore persists only sealed envelopes. It is a local development storage
// primitive, NOT a funded escrow service. Only server-side trusted code may
// ever call Put or Open; there are intentionally no HTTP routes.
//
// Root must be a dedicated private directory with no untrusted writers.
// Master key material is not written to this directory.
type FileStore struct {
 root string
 vault *Vault
}
var ErrAlreadyStored=errors.New("cashlink_secret_already_stored")
var ErrMissing=errors.New("cashlink_secret_not_found")

func NewFileStore(root string,v *Vault)(*FileStore,error){
 if v==nil||root==""||!filepath.IsAbs(root){return nil,ErrInvalid}
 if err:=os.MkdirAll(root,0700);err!=nil{return nil,fmt.Errorf("vault_directory_unavailable: %w",err)}
 stat,err:=os.Lstat(root)
 if err!=nil||!stat.IsDir()||stat.Mode()&os.ModeSymlink!=0||stat.Mode().Perm()&0077!=0 {
  return nil,ErrInvalid
 }
 return &FileStore{root:root,vault:v},nil
}
func (f *FileStore) filename(id string)(string,error){
 if f==nil||!validClaimID(id){return "",ErrInvalid}
 return filepath.Join(f.root,id+".sealed.json"),nil
}
// Put is create-once and does not overwrite a previous Cashlink secret.
// Durable storage alone is NOT a state machine and does not indicate funding.
func(f *FileStore)Put(claimID string,secret []byte)error{
 target,err:=f.filename(claimID)
 if err!=nil{return err}
 envelope,err:=f.vault.Seal(claimID,secret)
 if err!=nil{return err}
 encoded,err:=json.Marshal(envelope)
 if err!=nil{return ErrInvalid}
 // Create in the same directory, never writing plaintext to an on-disk file.
 temp,err:=os.CreateTemp(f.root,".pending-*.sealed")
 if err!=nil{return err}
 defer os.Remove(temp.Name())
 if err:=temp.Chmod(0600);err!=nil{temp.Close();return err}
 if _,err:=temp.Write(encoded);err!=nil{temp.Close();return err}
 if err:=temp.Sync();err!=nil{temp.Close();return err}
 if err:=temp.Close();err!=nil{return err}
 // Hard linking ensures atomic publication and disallows overwrites/replays.
 // It fails closed if the storage backend does not support atomic hardlinks.
 if err:=os.Link(temp.Name(),target);err!=nil{
  if errors.Is(err,os.ErrExist){return ErrAlreadyStored}
  return fmt.Errorf("vault_store_unavailable: %w",err)
 }
 // Best-effort sync of directory metadata; sealed bytes were already fsynced.
 if dir,err:=os.Open(f.root);err==nil{_ = dir.Sync();_ = dir.Close()}
 return nil
}

// Open is reserved for future audited settlement code. Never expose plaintext
// to the OTP route, a recipient webpage, logs, debug endpoints, or email.
func(f *FileStore)Open(claimID string)([]byte,error){
 name,err:=f.filename(claimID)
 if err!=nil{return nil,err}
 file,err:=os.Open(name)
 if errors.Is(err,os.ErrNotExist){return nil,ErrMissing}
 if err!=nil{return nil,ErrOpen}
 defer file.Close()
 info,err:=file.Stat()
 if err!=nil||!info.Mode().IsRegular()||info.Size()>2048||info.Size()==0{return nil,ErrOpen}
 raw,err:=io.ReadAll(io.LimitReader(file,2049))
 if err!=nil||len(raw)>2048{return nil,ErrOpen}
 var env Envelope
 if err:=json.Unmarshal(raw,&env);err!=nil{return nil,ErrOpen}
 return f.vault.Open(claimID,env)
}

// Delete erases the vault reference only; physical erasure cannot be guaranteed
// on SSDs/backups. Caller must first perform an authorized state transition.
func(f *FileStore)Delete(claimID string)error{
 name,err:=f.filename(claimID)
 if err!=nil{return err}
 err=os.Remove(name)
 if errors.Is(err,os.ErrNotExist){return ErrMissing}
 return err
}

// Has checks for a sealed ciphertext file without decrypting or releasing
// its secret. It does not assert current on-chain spendability.
func(f *FileStore)Has(claimID string)bool{
 name,err:=f.filename(claimID)
 if err!=nil{return false}
 st,err:=os.Lstat(name)
 return err==nil && st.Mode().IsRegular() && st.Mode().Perm()&0077==0 && st.Size()>0 && st.Size()<=2048
}
