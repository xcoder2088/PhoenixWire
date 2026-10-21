package otp

import "testing"

func TestMailerSendValidateRequiresRealSenderConfiguration(t *testing.T) {
	if err := (MailerSend{APIKey: "test-token", From: "otp@example.test"}).Validate(); err != nil {
		t.Fatalf("valid sender config rejected: %v", err)
	}
	for _, test := range []MailerSend{
		{APIKey: "", From: "otp@example.test"},
		{APIKey: "test-token", From: "not-an-email"},
		{APIKey: "test-token", From: "SecureClaim <otp@example.test>"},
	} {
		if err := test.Validate(); err == nil {
			t.Fatalf("invalid sender config accepted: %+v", test)
		}
	}
}
