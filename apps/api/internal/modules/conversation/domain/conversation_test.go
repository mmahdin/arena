package domain

import "testing"

func TestValidateMessage(t *testing.T) {
	for _, s := range []string{"", "  ", string(make([]byte, 4001))} {
		if ValidateMessage(s) == nil {
			t.Fatal("invalid message accepted")
		}
	}
	if err := ValidateMessage("سلام"); err != nil {
		t.Fatal(err)
	}
}
