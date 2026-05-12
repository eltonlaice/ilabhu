// Package azure is the Azure BYO-cloud adapter for ilabhud.
//
// Authentication is via an Azure AD Service Principal with a client secret.
// The user supplies four pieces on session start (tenant, subscription,
// client id, client secret); ilabhud exports them as ARM_* environment
// variables for the azurerm Terraform provider and drops them when the
// subprocess exits.
//
// Workload Identity Federation will follow when deployments that can issue
// OIDC tokens become a target.
package azure

import "errors"

// Credentials is the per-session Azure bundle. The secret travels through
// StartInput, into env vars for the Terraform subprocess, and is never
// persisted.
type Credentials struct {
	TenantID       string
	SubscriptionID string
	ClientID       string
	ClientSecret   string
}

// Validate ensures every required field is present. Returns the first
// missing field as an error so the user sees a clear message instead of a
// generic Azure auth failure.
func (c Credentials) Validate() error {
	switch {
	case c.TenantID == "":
		return errors.New("azure.tenant_id is required")
	case c.SubscriptionID == "":
		return errors.New("azure.subscription_id is required")
	case c.ClientID == "":
		return errors.New("azure.client_id is required")
	case c.ClientSecret == "":
		return errors.New("azure.client_secret is required")
	}
	return nil
}

// AsEnv returns the credentials as Terraform-compatible environment
// variables. The azurerm provider reads these directly.
func (c Credentials) AsEnv() []string {
	return []string{
		"ARM_TENANT_ID=" + c.TenantID,
		"ARM_SUBSCRIPTION_ID=" + c.SubscriptionID,
		"ARM_CLIENT_ID=" + c.ClientID,
		"ARM_CLIENT_SECRET=" + c.ClientSecret,
	}
}
