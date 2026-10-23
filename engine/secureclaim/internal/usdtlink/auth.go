package usdtlink

// Authorization of a recipient's claim. This is what stands between "someone opened the link" and "the service co-signs".
//
// A claim is authorized by ONE attempt (Authorization) that is bound, from the first request on, to: the card, the browser
// session, the destination the user confirmed, the exact Safe transaction (hash) and the Safe nonce. It goes
//
//	(e-mail check:) prepared -> link_proved -> code_sent -> verify_intent -> email_checked -> consumed
//	(claim:)        prepared -> authorized -> signing -> signed
//
// The e-mail code comes FIRST (as in any sign-in): an e-mail check is a short attempt of its own, bound to the card and the
// browser session, with no destination. Once the right code was entered, that session may, for ten minutes and ONCE, choose
// where to receive: the claim attempt below refers to the check and consumes it when the grant is issued. The check authorizes
// nothing by itself: the grant still needs the link-key proof over the exact destination and transaction.
//
//  1. Prepare: the user confirmed his destination R. The service rebuilds the payout itself and answers with the snapshot
//     (amount, fee, nonce, transaction hash) and a random challenge. NOTHING is reserved yet: knowing a card id does not let
//     anyone block the recipient's address.
//  2. ProveLink (e-mail check): the browser signs an EIP-712 LinkAuthorization (package proof, action e-mail, no destination)
//     with the link key, without sending the key to recipient claim routes. Only a holder of the link can ask for the e-mail code. The e-mail address comes
//     from the card's record, never from the client.
//  3. VerifyEmail: the code is checked by the existing e-mail code manager under a key unique to THIS check. Success marks the
//     check as done for this session (ten minutes, once).
//  3b. Authorize: the browser signs the LinkAuthorization of the claim attempt (action claim, the destination, the exact
//     transaction hash). With a live, unused check of the same session this reserves the destination for the card, consumes the
//     check and yields a secret grant, returned only to this session, valid for this one transaction hash for five minutes. The e-mail code manager and this store persist separately: the attempt is written as
//     verify_intent BEFORE the code is checked, so a crash in between is finished from the manager's own record
//     (Verified(key)) instead of being lost or replayed.
//  4. Cosign: with the grant, the service re-reads the chain (funding, Safe nonce), re-checks the card, and signs exactly the
//     hash that was confirmed. Signatures are deterministic, so a repeated call returns the same answer.
//  5. Claim: the browser adds the link-key signature and the relayer pays (Service.claim), only for a signed attempt.
//
// A global "Verified" flag never authorizes anything: only the grant of an attempt does, and only for its own hash. The e-mail
// check is per card, per session, time-limited and consumed by the one grant it allows.

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cl "github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/proof"
	"github.com/xcoder2088/NimiqSecureClaim/internal/cashlink/safe"
	"github.com/xcoder2088/NimiqSecureClaim/internal/evm"
	"github.com/xcoder2088/NimiqSecureClaim/internal/otp"
)

var (
	ErrSession       = errors.New("session_required")
	ErrAuthExpired   = errors.New("authorization_expired")
	ErrAuthState     = errors.New("authorization_state")
	ErrLinkProof     = errors.New("invalid_link_proof")
	ErrBadCode       = errors.New("invalid_or_expired_code")
	ErrBadGrant      = errors.New("invalid_grant")
	ErrQuoteChanged  = errors.New("quote_changed")
	ErrIdempotency   = errors.New("idempotency_conflict")
	ErrRateLimited   = errors.New("rate_limited")
	ErrAuthNotSigned = errors.New("authorization_not_signed")
	ErrEmailCheck    = errors.New("email_check_required") // no live, unused e-mail check for this session
)

// Authorization states.
const (
	AuthPrepared     = "prepared"
	AuthLinkProved   = "link_proved"
	AuthCodeSent     = "code_sent"
	AuthVerifyIntent = "verify_intent"
	AuthChecked      = "email_checked" // e-mail check: the right code was entered
	AuthConsumed     = "consumed"      // e-mail check: used by one claim attempt
	AuthAuthorized   = "authorized"
	AuthSigning      = "signing"
	AuthSigned       = "signed"
)

// Limits (see docs/AUTHORIZATION.md of the design review).
const (
	ChallengeLife   = 5 * time.Minute
	GrantLife       = 5 * time.Minute
	CheckLife       = 10 * time.Minute // a done e-mail check allows one claim attempt for this long
	SessionLife     = 30 * time.Minute // absolute: an attempt dies with its session
	MaxPreparations = 12               // PROVEN attempts (a check that proved the link key, a claim) per card per 15 minutes
	MaxCodeSends    = 8                // code e-mails per card per 15 minutes, over all attempts
	prepWindow      = 15 * time.Minute
)

// CodeService is what the e-mail code manager (internal/otp.Manager) offers. The key is unique per attempt.
type CodeService interface {
	IssueClaim(ctx context.Context, key, email string) error
	Verify(key, email, code string) error
	Verified(key string) bool
}

// Snapshot is the payout this attempt authorizes, rebuilt by the service from its own data and the chain.
type Snapshot struct {
	ChainID uint64 `json:"chain_id"`
	Safe    string `json:"safe"`
	Token   string `json:"token"`
	Relayer string `json:"relayer"`
	Amount  uint64 `json:"amount_units"`
	Fee     uint64 `json:"fee_units"`
	Nonce   uint64 `json:"safe_nonce"`
	TxHash  string `json:"safe_tx_hash"`
}

// Authorization is one claim attempt. Stored on disk; contains no secret in clear (the grant is stored hashed).
type Authorization struct {
	ID          string    `json:"id"`
	CardID      string    `json:"card_id"`
	Action      uint8     `json:"action"`
	SessionHash string    `json:"session_hash"`
	Destination string    `json:"destination"`
	Snapshot    Snapshot  `json:"snapshot"`
	Challenge   string    `json:"challenge"`
	Deadline    int64     `json:"deadline"` // Unix seconds: the proof must be signed for this deadline
	State       string    `json:"state"`
	Created     time.Time `json:"created"`
	Idem        string    `json:"idem,omitempty"` // hash of (session, card, action, destination, key): a repeated Prepare returns this attempt
	CodeKey     string    `json:"code_key"`
	CodeSends   int       `json:"code_sends,omitempty"`
	SendTimes   []int64   `json:"send_times,omitempty"` // Unix seconds of every code e-mail this attempt asked for (the real quota window)
	GrantHash   string    `json:"grant_hash,omitempty"`
	GrantUntil  time.Time `json:"grant_until,omitempty"`
	Signer      string    `json:"signer,omitempty"`
	Signature   string    `json:"signature,omitempty"` // the co-signature, returned again on a repeated call
	CheckID     string    `json:"check_id,omitempty"`  // claim attempt: the e-mail check it relies on
	CheckUntil  time.Time `json:"check_until,omitempty"`
	ConsumedBy  string    `json:"consumed_by,omitempty"` // e-mail check: the claim attempt that used it
}

// Prepared is what the browser gets from Prepare.
type Prepared struct {
	AuthorizationID string    `json:"authorization_id"`
	Snapshot        Snapshot  `json:"snapshot"`
	Challenge       string    `json:"challenge"`
	Deadline        int64     `json:"deadline"`
	EmailBinding    string    `json:"email_binding"`
	SessionBinding  string    `json:"session_binding"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// SetCodes wires the e-mail code manager and the secret used to derive grants and e-mail bindings (at least 32 bytes,
// DISTINCT from the co-signer master secret; losing it only invalidates attempts in flight).
func (s *Service) SetCodes(c CodeService, authSecret []byte) error {
	if c == nil || len(authSecret) < 32 {
		return errors.New("invalid_authorization_setup")
	}
	s.codes, s.secret = c, append([]byte(nil), authSecret...)
	return nil
}

func (s *Service) authReady() error {
	if s.codes == nil || len(s.secret) < 32 || s.cosign == nil {
		return ErrUnavailable
	}
	return nil
}

func (s *Service) mac(parts ...string) []byte {
	m := hmac.New(sha256.New, s.secret)
	for _, p := range parts {
		m.Write([]byte(p))
		m.Write([]byte{0})
	}
	return m.Sum(nil)
}

func sessionHash(session string) (string, error) {
	if len(session) < 32 || len(session) > 128 {
		return "", ErrSession
	}
	sum := sha256.Sum256([]byte(session))
	return hex.EncodeToString(sum[:]), nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("no randomness: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// ---- persistence (auths.json, same atomic scheme as the cards) ----

func (s *Service) persistAuthsLocked() error {
	encoded, err := json.Marshal(struct {
		Version int              `json:"version"`
		Auths   []*Authorization `json:"auths"`
	}{1, func() []*Authorization {
		var out []*Authorization
		for _, a := range s.auths {
			out = append(out, a)
		}
		return out
	}()})
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(s.dir, ".auths-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(encoded); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(temp.Name(), filepath.Join(s.dir, "auths.json")); err != nil {
		return err
	}
	d, err := os.Open(s.dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func (s *Service) loadAuths() error {
	raw, err := os.ReadFile(filepath.Join(s.dir, "auths.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var snap struct {
		Version int              `json:"version"`
		Auths   []*Authorization `json:"auths"`
	}
	if json.Unmarshal(raw, &snap) != nil || snap.Version != 1 {
		return errors.New("usdtlink_authorizations_corrupt")
	}
	for _, a := range snap.Auths {
		s.auths[a.ID] = a
	}
	return nil
}

// saveAuth writes the attempt; the in-memory copy is replaced only if the disk write succeeded.
func (s *Service) saveAuth(a *Authorization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, had := s.auths[a.ID]
	c := *a
	s.auths[a.ID] = &c
	if err := s.persistAuthsLocked(); err != nil {
		if had {
			s.auths[a.ID] = prev
		} else {
			delete(s.auths, a.ID)
		}
		return err
	}
	return nil
}

// saveAuthPair writes two attempts in ONE durable write: either both transitions are on disk or neither (an attempt becomes
// authorized and its e-mail check consumed together).
func (s *Service) saveAuthPair(a, b *Authorization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pa, hadA := s.auths[a.ID]
	pb, hadB := s.auths[b.ID]
	ca, cb := *a, *b
	s.auths[a.ID], s.auths[b.ID] = &ca, &cb
	if err := s.persistAuthsLocked(); err != nil {
		for id, prev := range map[string]*Authorization{a.ID: pa, b.ID: pb} {
			if (id == a.ID && hadA) || (id == b.ID && hadB) {
				s.auths[id] = prev
			} else {
				delete(s.auths, id)
			}
		}
		return err
	}
	return nil
}

func (s *Service) getAuth(id string) (*Authorization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.auths[id]
	if !ok {
		return nil, ErrNotFound
	}
	c := *a
	return &c, nil
}

// ---- the payout snapshot ----

// snapshot rebuilds the payout from the card's record and the chain. It also checks that the card can be claimed now:
// funded right now, Safe nonce zero, not expired/cancelled/in flight, destination allowed.
func (s *Service) snapshot(ctx context.Context, r *Record, dest cl.Address) (Snapshot, error) {
	if s.badDestination(r, dest) {
		return Snapshot{}, evm.ErrInvalidAddress
	}
	if err := s.refresh(ctx, r); err != nil {
		return Snapshot{}, err
	}
	if err := s.usable(r, dest); err != nil {
		return Snapshot{}, err
	}
	if !r.Funded {
		return Snapshot{}, ErrNotFunded
	}
	n, err := s.nonce(ctx, r)
	if err != nil {
		return Snapshot{}, err
	}
	if n != 0 || r.Settled != 0 {
		return Snapshot{}, ErrCancelled // the card already executed a payout
	}
	c := s.chains[r.Asset]
	card, err := r.card()
	if err != nil {
		return Snapshot{}, err
	}
	token, _ := cl.ParseAddress(c.Network.Token)
	addr, err := card.Address(c.Network.ChainID)
	if err != nil {
		return Snapshot{}, err
	}
	tx := safe.Payout(token, dest, c.Relayer.Address(), new(big.Int).SetUint64(r.Units), new(big.Int).SetUint64(r.FeeUnits), n)
	return Snapshot{ChainID: c.Network.ChainID, Safe: addr.Hex(), Token: token.Hex(), Relayer: c.Relayer.Address().Hex(), Amount: r.Units, Fee: r.FeeUnits,
		Nonce: n, TxHash: "0x" + hex.EncodeToString(tx.Hash(c.Network.ChainID, addr))}, nil
}

func (s *Service) emailBinding(r *Record) []byte { return s.mac("email", r.ID, r.Email) }

func (s *Service) proofOf(a *Authorization, r *Record) (proof.LinkAuthorization, error) {
	var p proof.LinkAuthorization
	id, err1 := hex.DecodeString(r.ID)
	aid, err2 := hex.DecodeString(a.ID)
	ch, err3 := hex.DecodeString(a.Challenge)
	sh, err4 := hex.DecodeString(a.SessionHash)
	tx, err5 := hex.DecodeString(strings.TrimPrefix(a.Snapshot.TxHash, "0x"))
	var dest cl.Address // the zero address: an e-mail check signs no destination
	var err6 error
	if a.Action != proof.ActionEmail {
		dest, err6 = cl.ParseAddress(a.Destination)
	}
	if errors.Join(err1, err2, err3, err4, err5, err6) != nil || len(id) != 16 || len(aid) != 16 || len(ch) != 32 || len(sh) != 32 || len(tx) != 32 {
		return p, ErrInvalid
	}
	copy(p.CardID[:], id)
	copy(p.AuthorizationID[:], aid)
	copy(p.SessionBinding[:], sh) // the stored hash of the session secret: public to the page, useless without the secret
	copy(p.EmailBinding[:], s.emailBinding(r))
	copy(p.SafeTxHash[:], tx)
	copy(p.Challenge[:], ch)
	p.Action, p.Destination, p.Deadline = a.Action, dest, uint64(a.Deadline)
	return p, nil
}

// ---- 0. StartCheck: the e-mail check comes first ----

// claimable says whether the card can be claimed right now, whatever the destination: funded, nonce zero, not paid, cancelled,
// expired or in flight.
func (s *Service) claimable(ctx context.Context, r *Record) error {
	if err := s.refresh(ctx, r); err != nil {
		return err
	}
	if err := s.usableState(r); err != nil {
		return err
	}
	if !r.Funded {
		return ErrNotFunded
	}
	n, err := s.nonce(ctx, r)
	if err != nil {
		return err
	}
	if n != 0 || r.Settled != 0 {
		return ErrCancelled // the card already executed a payout
	}
	return nil
}

// priorAttempt finds the attempt a repeated request refers to and counts the card's recent attempts. Only attempts that PROVED
// something are stored (an e-mail check exists only after its link-key proof; a claim needs a done check), so the count is the
// card's real budget: nobody who merely knows a card's id can use it up.
func (s *Service) priorAttempt(cardID, idemHash string) (prior *Authorization, recent int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.auths {
		if a.CardID != cardID {
			continue
		}
		if idemHash != "" && a.Idem == idemHash { // same request again: the same attempt, never a new one
			c := *a
			return &c, recent
		}
		if s.now().Sub(a.Created) < prepWindow {
			recent++
		}
	}
	return nil, recent
}

// pruneAuths forgets attempts that died long ago (a session lasts 30 minutes) and writes the file again: it must not grow for ever.
// It runs from the follower (every tick) and whenever an e-mail check is first stored.
func (s *Service) pruneAuths() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, a := range s.auths {
		if s.now().Sub(a.Created) > 3*time.Hour {
			delete(s.auths, id)
			s.authsDirty = true
		}
	}
	if s.authsDirty { // a write that failed is retried at the next tick, not forgotten
		if err := s.persistAuthsLocked(); err != nil {
			log.Printf("usdtlink: pruned authorizations could not be written, will retry: %v", err)
			return
		}
		s.authsDirty = false
	}
}

var zeroTx = "0x" + strings.Repeat("0", 64)

// An e-mail check is STATELESS until its link-key proof is valid: its id is derived (a time, a nonce and a MAC bound to the card
// and the browser session) and its challenge is derived from the id, so StartCheck stores nothing. Whoever only knows a card's id
// cannot fill any budget or storage with requests: the check is written down when ProveLink has verified the proof.
func (s *Service) checkMAC(sh, cardID string, ts uint32, nonce []byte) []byte {
	var t [4]byte
	binary.BigEndian.PutUint32(t[:], ts)
	return s.mac("checkid", sh, cardID, string(t[:]), string(nonce))[:8]
}

// newCheckID builds the id of a check: ts (4) | nonce (4) | mac (8), 16 bytes in hex. The nonce comes from the client's
// idempotency key, so the same request within the same minute gives the same id.
func (s *Service) newCheckID(sh, cardID, idem string, ts uint32) string {
	nonce := s.mac("checknonce", sh, cardID, idem, strconv.FormatUint(uint64(ts), 10))[:4]
	var t [4]byte
	binary.BigEndian.PutUint32(t[:], ts)
	return hex.EncodeToString(append(append(append([]byte{}, t[:]...), nonce...), s.checkMAC(sh, cardID, ts, nonce)...))
}

// parseCheckID verifies a derived id for this session and card and returns its start time.
func (s *Service) parseCheckID(sh, cardID, id string) (uint32, bool) {
	raw, err := hex.DecodeString(id)
	if err != nil || len(raw) != 16 {
		return 0, false
	}
	ts := binary.BigEndian.Uint32(raw[:4])
	if subtle.ConstantTimeCompare(raw[8:], s.checkMAC(sh, cardID, ts, raw[4:8])) != 1 {
		return 0, false
	}
	return ts, true
}

// derivedCheck rebuilds the (unsaved) check an id stands for, or says why it cannot.
func (s *Service) derivedCheck(sh, cardID, id string) (*Authorization, error) {
	ts, ok := s.parseCheckID(sh, cardID, id)
	if !ok {
		return nil, ErrNotFound
	}
	created := time.Unix(int64(ts), 0).UTC()
	if s.now().After(created.Add(ChallengeLife)) {
		return nil, ErrAuthExpired
	}
	a := &Authorization{ID: id, CardID: cardID, Action: proof.ActionEmail, SessionHash: sh, Snapshot: Snapshot{TxHash: zeroTx},
		Challenge: hex.EncodeToString(s.mac("checkchal", id)), Deadline: created.Add(ChallengeLife).Unix(), State: AuthPrepared, Created: created}
	a.CodeKey = "usdtlink:" + cardID + ":" + a.ID
	return a, nil
}

// StartCheck starts the e-mail check of a card for this browser session: no destination yet, nothing reserved, NOTHING STORED.
func (s *Service) StartCheck(ctx context.Context, session, cardID, idem string) (Prepared, error) {
	if err := s.authReady(); err != nil {
		return Prepared{}, err
	}
	sh, err := sessionHash(session)
	if err != nil {
		return Prepared{}, err
	}
	if len(idem) < 16 || len(idem) > 128 {
		return Prepared{}, ErrInvalid
	}
	defer s.lock(cardID)()
	r, err := s.get(cardID)
	if err != nil {
		return Prepared{}, err
	}
	if err := s.claimable(ctx, r); err != nil {
		return Prepared{}, err
	}
	ts := uint32(s.now().Unix() / 60 * 60)
	a, err := s.derivedCheck(sh, cardID, s.newCheckID(sh, cardID, idem, ts))
	if err != nil {
		return Prepared{}, err
	}
	return s.preparedView(a, r), nil
}

// ---- 1. Prepare ----

// Prepare starts a claim attempt for a destination the user confirmed, after the e-mail check of this session (checkID) is
// done. session is the browser's secret session value; only its hash is stored. idem is a client-chosen key: repeating the
// same request returns the same attempt.
func (s *Service) Prepare(ctx context.Context, session, cardID, destination, idem, checkID string) (Prepared, error) {
	if err := s.authReady(); err != nil {
		return Prepared{}, err
	}
	sh, err := sessionHash(session)
	if err != nil {
		return Prepared{}, err
	}
	if len(idem) < 16 || len(idem) > 128 {
		return Prepared{}, ErrInvalid
	}
	dest, err := cl.ParseAddress(destination)
	if err != nil {
		return Prepared{}, evm.ErrInvalidAddress
	}
	defer s.lock(cardID)()
	r, err := s.get(cardID)
	if err != nil {
		return Prepared{}, err
	}
	idemHash := hex.EncodeToString(s.mac("idem", sh, cardID, "claim", dest.Hex(), idem))
	prior, proven := s.priorAttempt(cardID, idemHash)
	if prior != nil {
		return s.preparedView(prior, r), nil
	}
	if proven >= MaxPreparations {
		return Prepared{}, ErrRateLimited
	}
	snap, err := s.snapshot(ctx, r, dest)
	if err != nil {
		return Prepared{}, err
	}
	if _, err := s.liveCheck(sh, cardID, checkID); err != nil {
		return Prepared{}, err
	}
	now := s.now().UTC()
	a := &Authorization{ID: randomHex(16), CardID: cardID, Action: proof.ActionClaim, SessionHash: sh, Destination: dest.Hex(), Snapshot: snap,
		Challenge: randomHex(32), Deadline: now.Add(ChallengeLife).Unix(), State: AuthPrepared, Created: now, Idem: idemHash, CheckID: checkID}
	a.CodeKey = "usdtlink:" + cardID + ":" + a.ID // unique per attempt, below the manager's 128 characters
	if err := s.saveAuth(a); err != nil {
		return Prepared{}, err
	}
	return s.preparedView(a, r), nil
}

// liveCheck returns the e-mail check of this session and card if the right code was entered, it is not used and not too old.
func (s *Service) liveCheck(sessionHash, cardID, checkID string) (*Authorization, error) {
	c, err := s.getAuth(checkID)
	if err != nil {
		return nil, ErrEmailCheck
	}
	if c.Action != proof.ActionEmail || c.CardID != cardID || subtle.ConstantTimeCompare([]byte(c.SessionHash), []byte(sessionHash)) != 1 || c.State != AuthChecked {
		return nil, ErrEmailCheck
	}
	if s.now().After(c.CheckUntil) {
		return nil, ErrEmailCheck
	}
	return c, nil
}

func (s *Service) preparedView(a *Authorization, r *Record) Prepared {
	return Prepared{AuthorizationID: a.ID, Snapshot: a.Snapshot, Challenge: a.Challenge, Deadline: a.Deadline,
		EmailBinding: hex.EncodeToString(s.emailBinding(r)), SessionBinding: a.SessionHash, ExpiresAt: time.Unix(a.Deadline, 0).UTC()}
}

// attempt loads an attempt and checks that it belongs to this session and card and is still alive.
func (s *Service) attempt(session, authID string) (*Authorization, *Record, error) {
	sh, err := sessionHash(session)
	if err != nil {
		return nil, nil, err
	}
	a, err := s.getAuth(authID)
	if err != nil {
		return nil, nil, err
	}
	if subtle.ConstantTimeCompare([]byte(a.SessionHash), []byte(sh)) != 1 {
		return nil, nil, ErrForbidden // another session cannot touch this attempt
	}
	r, err := s.get(a.CardID)
	if err != nil {
		return nil, nil, err
	}
	if s.now().After(a.Created.Add(SessionLife)) {
		return nil, nil, ErrAuthExpired
	}
	return a, r, nil
}

// ---- 2. ProveLink + send the code ----

// ProveLink checks the browser's LinkAuthorization signature and, only then, sends the e-mail code to the card's own
// recipient address. lang is presentation only.
func (s *Service) ProveLink(ctx context.Context, session, cardID, authID string, signature []byte, lang string) error {
	if err := s.authReady(); err != nil {
		return err
	}
	sh, err := sessionHash(session)
	if err != nil {
		return err
	}
	defer s.lock(cardID)()
	r, err := s.get(cardID)
	if err != nil {
		return err
	}
	var a *Authorization
	derived := false
	if stored, err := s.getAuth(authID); err == nil {
		// a check already written down (a resend): it must belong to this session and card and still be alive
		if stored.CardID != cardID || subtle.ConstantTimeCompare([]byte(stored.SessionHash), []byte(sh)) != 1 {
			return ErrForbidden
		}
		if s.now().After(stored.Created.Add(SessionLife)) {
			return ErrAuthExpired
		}
		a = stored
	} else if a, err = s.derivedCheck(sh, cardID, authID); err != nil {
		return err
	} else {
		derived = true
	}
	if a.Action != proof.ActionEmail {
		return ErrAuthState // only an e-mail check asks for a code
	}
	if a.State != AuthPrepared && a.State != AuthLinkProved && a.State != AuthCodeSent {
		return ErrAuthState // later states never ask for another code: they are reconciled, not restarted
	}
	if s.now().Unix() > a.Deadline && a.State == AuthPrepared {
		return ErrAuthExpired
	}
	p, err := s.proofOf(a, r)
	if err != nil {
		return err
	}
	owner, err := cl.ParseAddress(r.LinkOwner)
	if err != nil {
		return ErrInvalid
	}
	card, _ := cl.ParseAddress(r.Safe)
	who, err := p.Signer(s.chains[r.Asset].Network.ChainID, card, signature)
	if err != nil || who != owner {
		return ErrLinkProof
	}
	now := s.now()
	if derived { // the proof is valid: only now does the check exist, and only now does it use the card's budget
		s.pruneAuths()
		if _, recent := s.priorAttempt(cardID, ""); recent >= MaxPreparations {
			return ErrRateLimited
		}
	}
	if s.codeSends(r.ID) >= MaxCodeSends {
		return ErrRateLimited
	}
	// The send is counted BEFORE the e-mail leaves: a crash or a failed send cannot give back a free one.
	a.SendTimes = append(recentTimes(a.SendTimes, now), now.Unix())
	a.CodeSends++
	if a.State == AuthPrepared {
		a.State = AuthLinkProved
	}
	if err := s.saveAuth(a); err != nil {
		return err
	}
	// The language is presentation only. The key is unique to this attempt; the address is the card's own.
	if lang != "" && len(lang) <= 8 {
		ctx = otp.WithEmailLanguage(ctx, lang)
	}
	pal := Palette(r.ID) // the code e-mail wears the colour of its card
	ctx = otp.WithEmailAccent(ctx, pal[0], pal[1])
	if err := s.codes.IssueClaim(ctx, a.CodeKey, r.Email); err != nil {
		return err
	}
	a.State = AuthCodeSent
	return s.saveAuth(a)
}

// recentTimes keeps the send times of the last hour (all that any window needs).
func recentTimes(ts []int64, now time.Time) []int64 {
	var out []int64
	for _, t := range ts {
		if now.Unix()-t < 3600 {
			out = append(out, t)
		}
	}
	return out
}

func (s *Service) codeSends(cardID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, a := range s.auths {
		if a.CardID != cardID {
			continue
		}
		for _, t := range a.SendTimes { // the real moment of every send, whatever the age of the attempt
			if s.now().Unix()-t < int64(prepWindow/time.Second) {
				n++
			}
		}
	}
	return n
}

// ---- 3. VerifyCode -> grant ----

// grantToken derives the grant from the secret and the attempt: a repeated call after a lost answer returns the same one.
func (s *Service) grantToken(a *Authorization) string {
	return hex.EncodeToString(s.mac("grant", a.ID, a.SessionHash, a.Snapshot.TxHash))
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// VerifyEmail checks the e-mail code of an e-mail check. Success marks the check as done for this session: it allows ONE claim
// attempt (Authorize) during CheckLife. Nothing is reserved and no grant is issued here.
func (s *Service) VerifyEmail(ctx context.Context, session, checkID, code string) (time.Time, error) {
	if err := s.authReady(); err != nil {
		return time.Time{}, err
	}
	a0, _, err := s.attempt(session, checkID)
	if err != nil {
		return time.Time{}, err
	}
	defer s.lock(a0.CardID)()
	a, r, err := s.attempt(session, checkID)
	if err != nil {
		return time.Time{}, err
	}
	if a.Action != proof.ActionEmail {
		return time.Time{}, ErrAuthState
	}
	switch a.State {
	case AuthCodeSent:
		a.State = AuthVerifyIntent // durable BEFORE the code is checked: a crash after Verify is finished from the manager
		if err := s.saveAuth(a); err != nil {
			return time.Time{}, err
		}
		if err := s.codes.Verify(a.CodeKey, r.Email, code); err != nil {
			a.State = AuthCodeSent
			_ = s.saveAuth(a)
			return time.Time{}, ErrBadCode
		}
	case AuthVerifyIntent:
		if !s.codes.Verified(a.CodeKey) { // the previous call stopped before Verify: the code must still be right
			if err := s.codes.Verify(a.CodeKey, r.Email, code); err != nil {
				return time.Time{}, ErrBadCode
			}
		}
	case AuthChecked, AuthConsumed:
		if s.now().After(a.CheckUntil) {
			return time.Time{}, ErrAuthExpired
		}
		return a.CheckUntil, nil // a repeated call after a lost answer
	default:
		return time.Time{}, ErrAuthState
	}
	a.State, a.CheckUntil = AuthChecked, s.now().UTC().Add(CheckLife)
	if err := s.saveAuth(a); err != nil {
		return time.Time{}, err
	}
	return a.CheckUntil, nil
}

// Authorize checks the link-key proof of a claim attempt (its destination and exact transaction) and, with a live e-mail check
// of this session, reserves the card's destination, consumes the check and issues the grant.
func (s *Service) Authorize(ctx context.Context, session, authID string, signature []byte) (string, time.Time, error) {
	if err := s.authReady(); err != nil {
		return "", time.Time{}, err
	}
	a0, _, err := s.attempt(session, authID)
	if err != nil {
		return "", time.Time{}, err
	}
	defer s.lock(a0.CardID)()
	a, r, err := s.attempt(session, authID)
	if err != nil {
		return "", time.Time{}, err
	}
	if a.Action != proof.ActionClaim {
		return "", time.Time{}, ErrAuthState
	}
	switch a.State {
	case AuthAuthorized, AuthSigning, AuthSigned:
		// a repeated call: the same grant while it lives. An authorized attempt whose check is still unused (written before the
		// atomic pair existed) is reconciled here: the check is consumed now.
		if c, err := s.getAuth(a.CheckID); err == nil && c.State == AuthChecked && c.ConsumedBy == "" {
			c.State, c.ConsumedBy = AuthConsumed, a.ID
			_ = s.saveAuth(c)
		}
		return s.grantToken(a), a.GrantUntil, s.grantAlive(a)
	case AuthPrepared:
	default:
		return "", time.Time{}, ErrAuthState
	}
	if s.now().Unix() > a.Deadline {
		return "", time.Time{}, ErrAuthExpired
	}
	p, err := s.proofOf(a, r)
	if err != nil {
		return "", time.Time{}, err
	}
	owner, err := cl.ParseAddress(r.LinkOwner)
	if err != nil {
		return "", time.Time{}, ErrInvalid
	}
	card, _ := cl.ParseAddress(r.Safe)
	who, err := p.Signer(s.chains[r.Asset].Network.ChainID, card, signature)
	if err != nil || who != owner {
		return "", time.Time{}, ErrLinkProof
	}
	check, err := s.liveCheck(a.SessionHash, a.CardID, a.CheckID)
	if err != nil {
		return "", time.Time{}, err
	}
	// The check is live. Re-check the card at this very moment, then reserve the destination and issue the grant together.
	dest, _ := cl.ParseAddress(a.Destination)
	snap, err := s.snapshot(ctx, r, dest)
	if err != nil {
		return "", time.Time{}, err
	}
	if snap != a.Snapshot {
		return "", time.Time{}, ErrQuoteChanged
	}
	if r.Recipient == "" {
		r.Recipient = dest.Hex()
		r.Verified = true // compatibility projection only; nothing reads it as an authorization
		if err := s.update(r); err != nil {
			return "", time.Time{}, err
		}
	}
	token := s.grantToken(a)
	a.State, a.GrantHash, a.GrantUntil = AuthAuthorized, hashToken(token), s.now().UTC().Add(GrantLife)
	// The attempt becomes authorized and its e-mail check consumed in ONE write: "once" is exact.
	check.State, check.ConsumedBy = AuthConsumed, a.ID
	if err := s.saveAuthPair(a, check); err != nil {
		return "", time.Time{}, err
	}
	return token, a.GrantUntil, nil
}

func (s *Service) grantAlive(a *Authorization) error {
	if s.now().After(a.GrantUntil) {
		return ErrAuthExpired
	}
	return nil
}

// ---- 4. Cosign ----

// CosignClaim gives the co-signature for the one transaction this attempt confirmed.
func (s *Service) CosignClaim(ctx context.Context, session, authID, grant string) (cl.Address, []byte, error) {
	if err := s.authReady(); err != nil {
		return cl.Address{}, nil, err
	}
	a0, _, err := s.attempt(session, authID)
	if err != nil {
		return cl.Address{}, nil, err
	}
	defer s.lock(a0.CardID)()
	a, r, err := s.attempt(session, authID)
	if err != nil {
		return cl.Address{}, nil, err
	}
	if a.GrantHash == "" || subtle.ConstantTimeCompare([]byte(a.GrantHash), []byte(hashToken(grant))) != 1 {
		return cl.Address{}, nil, ErrBadGrant
	}
	if err := s.grantAlive(a); err != nil {
		return cl.Address{}, nil, err
	}
	if a.State != AuthAuthorized && a.State != AuthSigning && a.State != AuthSigned {
		return cl.Address{}, nil, ErrAuthState
	}
	dest, _ := cl.ParseAddress(a.Destination)
	// A repeated call returns the stored answer, but the card is still re-checked: a spent card gives nothing.
	snap, err := s.snapshot(ctx, r, dest)
	if err != nil {
		return cl.Address{}, nil, err
	}
	if snap != a.Snapshot {
		return cl.Address{}, nil, ErrQuoteChanged
	}
	c := s.chains[r.Asset]
	card, err := r.card()
	if err != nil {
		return cl.Address{}, nil, err
	}
	token, _ := cl.ParseAddress(c.Network.Token)
	if a.State == AuthAuthorized {
		a.State = AuthSigning // durable before the signature exists
		if err := s.saveAuth(a); err != nil {
			return cl.Address{}, nil, err
		}
	}
	signed, err := s.cosign.SignPayout(c.Network.ChainID, card, token, dest, c.Relayer.Address(), new(big.Int).SetUint64(a.Snapshot.Amount), new(big.Int).SetUint64(a.Snapshot.Fee), a.Snapshot.Nonce)
	if err != nil {
		return cl.Address{}, nil, err
	}
	if a.State != AuthSigned {
		a.State, a.Signer, a.Signature = AuthSigned, signed.Owner.Hex(), hex.EncodeToString(signed.Sig.Bytes())
		if err := s.saveAuth(a); err != nil { // the answer is durable BEFORE it leaves
			return cl.Address{}, nil, err
		}
	}
	return signed.Owner, signed.Sig.Bytes(), nil
}

// ---- 5. Claim ----

// ClaimAuthorized has the relayer pay the recipient, only for an attempt that was signed for exactly this destination.
// signatures are the link key's and the co-signer's, packed by the browser.
func (s *Service) ClaimAuthorized(ctx context.Context, session, authID, grant string, signatures []byte) (tx, state string, err error) {
	if err := s.authReady(); err != nil {
		return "", "", err
	}
	a, rec, err := s.attempt(session, authID)
	if err != nil {
		return "", "", err
	}
	if a.State != AuthSigned {
		return "", "", ErrAuthNotSigned
	}
	if a.GrantHash == "" || subtle.ConstantTimeCompare([]byte(a.GrantHash), []byte(hashToken(grant))) != 1 {
		return "", "", ErrBadGrant
	}
	// The five minutes of the grant bind every NEW broadcast. With an expired grant this call can only LOOK: it never reaches the path
	// that sends. A payout transaction that was journaled and has not failed is looked up (the refresh may discover that it was
	// mined and reverted: it then says so, and the recipient must start again with a fresh grant).
	if expired := s.grantAlive(a); expired != nil {
		if rec.Op == nil || rec.Op.Kind != "claim" || rec.Op.Tx == "" || rec.Op.Failed {
			return "", "", expired
		}
		tx := rec.Op.Tx
		pub, rerr := s.Refresh(ctx, a.CardID)
		if rerr != nil {
			return tx, StateVerifying, nil
		}
		return tx, pub.State, nil
	}
	res, err := s.claim(ctx, a.CardID, a.Destination, signatures)
	if err != nil {
		return res.ExecTx, "", err
	}
	// Sent is not paid: "paid" only once the transaction is deep enough and showed the transfers (refresh). Right now it is verifying.
	pub, rerr := s.Refresh(ctx, a.CardID)
	if rerr != nil {
		return res.ExecTx, StateVerifying, nil
	}
	return res.ExecTx, pub.State, nil
}
