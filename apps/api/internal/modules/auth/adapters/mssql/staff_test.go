package mssql

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
)

func TestStaffMappingValidation(t *testing.T) {
	subject := uuid.NewString()
	valid := "http://auth.localhost:8180/realms/theia-staff"
	if err := ValidateStaffBinding(valid, subject); err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []struct{ issuer, subject string }{
		{"http://auth.localhost:8180/realms/theia-users", subject},
		{"http://auth.localhost:8180/realms/theia-staff/", subject},
		{"http://auth.localhost:8180/realms/theia-staff?next=evil", subject},
		{"http://admin:pw@auth.localhost/realms/theia-staff", subject},
		{valid, "someone@example.com"},
		{valid, ""},
		{valid, subject + "-suffix"},
	} {
		if err := ValidateStaffBinding(candidate.issuer, candidate.subject); !errors.Is(err, auth.ErrInvalid) {
			t.Fatalf("unexpected acceptance of malformed issuer/subject: %v", candidate)
		}
	}
}
