package digitalocean

import (
	"sort"
	"strings"
	"testing"
)

func TestCredentials_AsEnv(t *testing.T) {
	c := Credentials{Token: "dop_v1_secret_token_value"}

	env := c.AsEnv()
	sort.Strings(env)
	want := []string{
		"DIGITALOCEAN_ACCESS_TOKEN=dop_v1_secret_token_value",
		"DIGITALOCEAN_TOKEN=dop_v1_secret_token_value",
	}
	sort.Strings(want)

	if len(env) != len(want) {
		t.Fatalf("AsEnv length = %d, want %d (got %v)", len(env), len(want), env)
	}
	for i := range env {
		if env[i] != want[i] {
			t.Errorf("env[%d] = %q, want %q", i, env[i], want[i])
		}
	}
}

func TestCredentials_AsEnvDoesNotInventFields(t *testing.T) {
	c := Credentials{Token: "abc"}
	for _, e := range c.AsEnv() {
		if !strings.Contains(e, "abc") {
			t.Errorf("env entry should carry the token; got %q", e)
		}
	}
}

func TestCredentials_EmptyTokenStillShapedCorrectly(t *testing.T) {
	// Manager validates the token before reaching AsEnv; if a zero-value
	// Credentials somehow leaks through, the env vars must still be
	// well-formed `KEY=` pairs so Terraform fails with a clear auth error
	// rather than a parse error.
	c := Credentials{}
	for _, e := range c.AsEnv() {
		if !strings.Contains(e, "=") {
			t.Errorf("env entry must contain `=`; got %q", e)
		}
	}
}
