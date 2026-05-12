package azure

import (
	"sort"
	"strings"
	"testing"
)

func validCreds() Credentials {
	return Credentials{
		TenantID:       "11111111-1111-1111-1111-111111111111",
		SubscriptionID: "22222222-2222-2222-2222-222222222222",
		ClientID:       "33333333-3333-3333-3333-333333333333",
		ClientSecret:   "secret-value",
	}
}

func TestValidate_AcceptsFullCreds(t *testing.T) {
	if err := validCreds().Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestValidate_RejectsMissingField(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Credentials)
		want string
	}{
		{"missing tenant", func(c *Credentials) { c.TenantID = "" }, "tenant_id"},
		{"missing subscription", func(c *Credentials) { c.SubscriptionID = "" }, "subscription_id"},
		{"missing client_id", func(c *Credentials) { c.ClientID = "" }, "client_id"},
		{"missing client_secret", func(c *Credentials) { c.ClientSecret = "" }, "client_secret"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validCreds()
			tc.mut(&c)
			err := c.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestCredentials_AsEnv(t *testing.T) {
	env := validCreds().AsEnv()
	sort.Strings(env)
	want := []string{
		"ARM_CLIENT_ID=33333333-3333-3333-3333-333333333333",
		"ARM_CLIENT_SECRET=secret-value",
		"ARM_SUBSCRIPTION_ID=22222222-2222-2222-2222-222222222222",
		"ARM_TENANT_ID=11111111-1111-1111-1111-111111111111",
	}
	sort.Strings(want)
	if len(env) != len(want) {
		t.Fatalf("env len = %d, want %d (got %v)", len(env), len(want), env)
	}
	for i := range env {
		if env[i] != want[i] {
			t.Errorf("env[%d] = %q, want %q", i, env[i], want[i])
		}
	}
}

func TestCredentials_AsEnvDoesNotLeakAcrossKeys(t *testing.T) {
	c := validCreds()
	for _, e := range c.AsEnv() {
		k := strings.SplitN(e, "=", 2)[0]
		if !strings.HasPrefix(k, "ARM_") {
			t.Errorf("unexpected env key prefix: %q", k)
		}
	}
}
