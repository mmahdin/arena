package main

import "testing"

func TestValidateDatabaseEndpoint(t *testing.T) {
	for _, endpoint := range []string{"sqlserver://127.0.0.1:14333", "sqlserver://[::1]:1433", "sqlserver://localhost"} {
		if err := validateDatabaseEndpoint(endpoint); err != nil {
			t.Errorf("rejected %s: %v", endpoint, err)
		}
	}
	for _, endpoint := range []string{"sqlserver://stage.example.invalid", "sqlserver://192.0.2.1", "https://127.0.0.1"} {
		if err := validateDatabaseEndpoint(endpoint); err == nil {
			t.Errorf("accepted %s", endpoint)
		}
	}
}

func TestValidateLocalAndStrongGeneratedSecret(t *testing.T) {
	valid := "Aa1!a84c7d193b35d2184f91a9e36d1e3c05a84c7d193b35d2184f91a9e36d1a"
	if err := validate("local", valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ env, password string }{
		{"production", valid}, {"stage", valid}, {"local", "weak"}, {"local", valid[:63] + "'"}, {"local", valid[:63] + "A"},
	} {
		if validate(tc.env, tc.password) == nil {
			t.Errorf("accepted disallowed input")
		}
	}
}
