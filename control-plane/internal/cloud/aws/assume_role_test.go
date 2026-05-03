package aws

import (
	"sort"
	"strings"
	"testing"
	"time"
)

func TestCredentials_AsEnvProducesTerraformCompatibleVars(t *testing.T) {
	c := Credentials{
		AccessKeyID:     "AKIAEXAMPLE",
		SecretAccessKey: "secret-value",
		SessionToken:    "token-value",
		Expiration:      time.Now().Add(time.Hour),
	}

	env := c.AsEnv()
	sort.Strings(env)

	want := []string{
		"AWS_ACCESS_KEY_ID=AKIAEXAMPLE",
		"AWS_SECRET_ACCESS_KEY=secret-value",
		"AWS_SESSION_TOKEN=token-value",
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

func TestCredentials_AsEnvDoesNotLeakOtherFields(t *testing.T) {
	c := Credentials{
		AccessKeyID:     "AKIAEXAMPLE",
		SecretAccessKey: "secret-value",
		SessionToken:    "token-value",
		Expiration:      time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	for _, e := range c.AsEnv() {
		if strings.Contains(e, "Expiration") || strings.Contains(e, "2030") {
			t.Errorf("env entry leaks Expiration: %q", e)
		}
	}
}
