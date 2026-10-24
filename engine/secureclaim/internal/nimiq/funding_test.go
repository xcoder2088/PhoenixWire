package nimiq

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	cardAddr  = "NQ46 KLJE 5TMF 4Y1A 1255 CJHJ YG1S H0NU T604"
	senderOne = "NQ28 CQ7G QEBT 10UK R9LG 83C7 YAMY JD8Q K8S4"
	senderTwo = "NQ24 0358 HL85 UQ05 SLSS C72R KEX3 NU2K RN0N"
)

type fundingTx struct {
	From, To, Data string
	Confirmations  int64
	Executed       bool
}

func fundingVerifier(t *testing.T, txs func() []fundingTx) *MainnetVerifier {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var out []map[string]any
		for _, tx := range txs() {
			out = append(out, map[string]any{"from": tx.From, "to": tx.To, "recipientData": tx.Data, "confirmations": tx.Confirmations, "executionResult": tx.Executed})
		}
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"data": out}})
	}))
	t.Cleanup(srv.Close)
	v, err := NewMainnetVerifier(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func mark(from string) fundingTx {
	return fundingTx{From: from, To: cardAddr, Data: cashlinkFundingMarker, Confirmations: 5, Executed: true}
}

func TestFundingSenderIsOnlyAnswerWhenCertain(t *testing.T) {
	ctx := context.Background()
	var list []fundingTx
	v := fundingVerifier(t, func() []fundingTx { return list })

	list = nil
	if from, err := v.FundingSender(ctx, cardAddr); from != "" || err != nil {
		t.Fatal("no funding yet", from, err)
	}
	// one sender, even paying twice or with a parasite from itself: that address
	list = []fundingTx{mark(senderOne), mark(senderOne)}
	if from, err := v.FundingSender(ctx, cardAddr); err != nil || normalized(from) != normalized(senderOne) {
		t.Fatal("one sender", from, err)
	}
	// money without the mark does not count (a gift, a refund in): the one marked sender stays
	list = []fundingTx{{From: senderTwo, To: cardAddr, Data: "", Confirmations: 5, Executed: true}, mark(senderOne)}
	if from, err := v.FundingSender(ctx, cardAddr); err != nil || normalized(from) != normalized(senderOne) {
		t.Fatal("unmarked money counted", from, err)
	}
	// a stranger sends a small marked payment BEFORE the real funding (oldest) or AFTER it (newest): two senders, nothing is chosen
	list = []fundingTx{mark(senderOne), mark(senderTwo)}
	if _, err := v.FundingSender(ctx, cardAddr); !errors.Is(err, ErrAmbiguousFunding) {
		t.Fatal("a second sender must make the origin ambiguous", err)
	}
	list = []fundingTx{mark(senderTwo), mark(senderOne)}
	if _, err := v.FundingSender(ctx, cardAddr); !errors.Is(err, ErrAmbiguousFunding) {
		t.Fatal("the order must not decide", err)
	}
	// a window as long as the one asked for may hide older transactions
	list = nil
	for i := 0; i < fundingWindow; i++ {
		list = append(list, mark(senderOne))
	}
	if _, err := v.FundingSender(ctx, cardAddr); !errors.Is(err, ErrAmbiguousFunding) {
		t.Fatal("a full window is ambiguous", err)
	}
	// a funding that is not settled is not the moment to choose
	list = []fundingTx{{From: senderOne, To: cardAddr, Data: cashlinkFundingMarker, Confirmations: 0, Executed: true}}
	if _, err := v.FundingSender(ctx, cardAddr); !errors.Is(err, ErrAmbiguousFunding) {
		t.Fatal("an unsettled funding", err)
	}
	// the card paying itself, or an invalid sender, is never a destination
	list = []fundingTx{{From: cardAddr, To: cardAddr, Data: cashlinkFundingMarker, Confirmations: 5, Executed: true}}
	if _, err := v.FundingSender(ctx, cardAddr); !errors.Is(err, ErrAmbiguousFunding) {
		t.Fatal("the card itself", err)
	}
}
