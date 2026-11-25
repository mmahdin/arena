package main

import "testing"

func TestIssuerAllowsOnlyLocalRealm(t *testing.T) {
	for _, in := range []struct {
		env, base, want string
		ok              bool
	}{
		{"local", "", "http://auth.localhost:8180/realms/theia-staff", true},
		{"local", "http://127.0.0.1:8180", "http://127.0.0.1:8180/realms/theia-staff", true},
		{"local", "https://auth.localhost:8180", "https://auth.localhost:8180/realms/theia-staff", true},
		{"production", "", "", false},
		{"local", "http://example.com:8180", "", false},
		{"local", "http://auth.localhost:8180/hijack", "", false},
		{"local", "http://user:pass@auth.localhost:8180", "", false},
		{"local", "http://auth.localhost:8180?x=1", "", false},
	} {
		got, err := localIssuer(in.env, in.base)
		if (err == nil) != in.ok || got != in.want {
			t.Fatalf("localIssuer(%q,%q) => %q,%v", in.env, in.base, got, err)
		}
	}
}
