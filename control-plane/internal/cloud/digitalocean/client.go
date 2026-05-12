// Package digitalocean is the DigitalOcean BYO-cloud adapter for ilabhud.
//
// Unlike AWS, DigitalOcean does not have an AssumeRole-equivalent flow. The
// user supplies a Personal Access Token (PAT) on every session-start request;
// ilabhud only holds it in memory long enough to invoke Terraform.
package digitalocean

// Credentials is the per-session token bundle. The token never reaches disk —
// it travels through StartInput, into env vars for the Terraform subprocess,
// and is dropped when that subprocess exits.
type Credentials struct {
	Token string
}

// AsEnv returns the credentials as Terraform-compatible environment variables.
// The DigitalOcean Terraform provider reads DIGITALOCEAN_TOKEN by default; we
// also export DIGITALOCEAN_ACCESS_TOKEN for the doctl-style alias some
// modules use.
func (c Credentials) AsEnv() []string {
	return []string{
		"DIGITALOCEAN_TOKEN=" + c.Token,
		"DIGITALOCEAN_ACCESS_TOKEN=" + c.Token,
	}
}
