"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import {
  startSession,
  type Provider,
  type ProviderCredentials,
} from "@/lib/api";

const PROVIDER_LABELS: Record<Provider, string> = {
  aws: "AWS",
  gcp: "GCP",
  azure: "Azure",
  digitalocean: "DigitalOcean",
  "byo-hosts": "Your Linux hosts",
};

const IMPLEMENTED: Provider[] = ["aws", "digitalocean", "gcp", "azure"];

type Props = {
  examID: string;
  providers: Provider[];
};

export function StartSessionForm({ examID, providers }: Props) {
  const router = useRouter();
  const [provider, setProvider] = useState<Provider>(
    providers.find((p) => IMPLEMENTED.includes(p)) ?? providers[0],
  );
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Provider-specific state — only the active provider's fields are read.
  const [awsRoleArn, setAwsRoleArn] = useState("");
  const [awsExternalId, setAwsExternalId] = useState("");
  const [doToken, setDoToken] = useState("");
  const [gcpServiceAccountKey, setGcpServiceAccountKey] = useState("");
  const [azureTenantId, setAzureTenantId] = useState("");
  const [azureSubscriptionId, setAzureSubscriptionId] = useState("");
  const [azureClientId, setAzureClientId] = useState("");
  const [azureClientSecret, setAzureClientSecret] = useState("");

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);

    let creds: ProviderCredentials;
    if (provider === "aws") {
      creds = {
        provider,
        aws: {
          role_arn: awsRoleArn.trim(),
          external_id: awsExternalId.trim(),
        },
      };
    } else if (provider === "digitalocean") {
      creds = {
        provider,
        digitalocean: { token: doToken.trim() },
      };
    } else if (provider === "gcp") {
      const key = gcpServiceAccountKey.trim();
      try {
        JSON.parse(key);
      } catch {
        setError(
          "Service account key must be valid JSON. Paste the full file contents you downloaded from the GCP console.",
        );
        setSubmitting(false);
        return;
      }
      creds = {
        provider,
        gcp: { service_account_key: key },
      };
    } else if (provider === "azure") {
      creds = {
        provider,
        azure: {
          tenant_id: azureTenantId.trim(),
          subscription_id: azureSubscriptionId.trim(),
          client_id: azureClientId.trim(),
          client_secret: azureClientSecret.trim(),
        },
      };
    } else {
      setError(`Provider ${provider} is not implemented yet.`);
      setSubmitting(false);
      return;
    }

    try {
      const sess = await startSession(examID, creds);
      window.sessionStorage.setItem(
        `ilabhu:creds:${sess.id}`,
        JSON.stringify(creds),
      );
      router.push(`/sessions/${encodeURIComponent(sess.id)}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setSubmitting(false);
    }
  }

  return (
    <form
      onSubmit={onSubmit}
      className="space-y-4 rounded-lg border border-neutral-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900"
    >
      <div>
        <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300">
          Where to provision
        </label>
        <div className="mt-2 flex flex-wrap gap-2">
          {providers.map((p) => {
            const enabled = IMPLEMENTED.includes(p);
            const active = p === provider;
            return (
              <button
                type="button"
                key={p}
                onClick={() => enabled && setProvider(p)}
                disabled={!enabled}
                className={`rounded-md border px-3 py-1.5 text-sm font-medium transition ${
                  active
                    ? "border-neutral-900 bg-neutral-900 text-white dark:border-white dark:bg-white dark:text-neutral-900"
                    : enabled
                      ? "border-neutral-300 bg-white text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-300 dark:hover:bg-neutral-900"
                      : "cursor-not-allowed border-neutral-200 bg-neutral-50 text-neutral-400 line-through dark:border-neutral-800 dark:bg-neutral-950 dark:text-neutral-600"
                }`}
                title={enabled ? "" : "Not implemented yet"}
              >
                {PROVIDER_LABELS[p]}
              </button>
            );
          })}
        </div>
      </div>

      {provider === "aws" ? (
        <AWSFields
          roleArn={awsRoleArn}
          setRoleArn={setAwsRoleArn}
          externalId={awsExternalId}
          setExternalId={setAwsExternalId}
        />
      ) : provider === "digitalocean" ? (
        <DOFields token={doToken} setToken={setDoToken} />
      ) : provider === "gcp" ? (
        <GCPFields
          serviceAccountKey={gcpServiceAccountKey}
          setServiceAccountKey={setGcpServiceAccountKey}
        />
      ) : provider === "azure" ? (
        <AzureFields
          tenantId={azureTenantId}
          setTenantId={setAzureTenantId}
          subscriptionId={azureSubscriptionId}
          setSubscriptionId={setAzureSubscriptionId}
          clientId={azureClientId}
          setClientId={setAzureClientId}
          clientSecret={azureClientSecret}
          setClientSecret={setAzureClientSecret}
        />
      ) : (
        <p className="text-sm text-neutral-500">
          {PROVIDER_LABELS[provider]} adapter is not implemented yet.
        </p>
      )}

      {error ? (
        <div className="rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-800 dark:border-rose-900 dark:bg-rose-950/30 dark:text-rose-200">
          {error}
        </div>
      ) : null}

      <button
        type="submit"
        disabled={submitting || !IMPLEMENTED.includes(provider)}
        className="rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-neutral-700 focus:outline-none focus:ring-2 focus:ring-neutral-500 focus:ring-offset-2 disabled:opacity-50 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
      >
        {submitting ? "Provisioning..." : "Start session"}
      </button>
    </form>
  );
}

function AWSFields({
  roleArn,
  setRoleArn,
  externalId,
  setExternalId,
}: {
  roleArn: string;
  setRoleArn: (v: string) => void;
  externalId: string;
  setExternalId: (v: string) => void;
}) {
  return (
    <div className="space-y-4">
      <div>
        <label
          htmlFor="role-arn"
          className="block text-sm font-medium text-neutral-700 dark:text-neutral-300"
        >
          AWS role ARN
        </label>
        <input
          id="role-arn"
          required
          value={roleArn}
          onChange={(e) => setRoleArn(e.target.value)}
          placeholder="arn:aws:iam::123456789012:role/ilabhu-lab-runner"
          className="mt-1 block w-full rounded-md border border-neutral-300 bg-white px-3 py-2 font-mono text-sm shadow-sm focus:border-neutral-500 focus:outline-none focus:ring-1 focus:ring-neutral-500 dark:border-neutral-700 dark:bg-neutral-950"
        />
        <p className="mt-1 text-xs text-neutral-500">
          See{" "}
          <a
            className="underline"
            href="https://github.com/eltonlaice/ilabhu/blob/main/docs/byo-cloud-setup.md"
            target="_blank"
            rel="noreferrer"
          >
            docs/byo-cloud-setup.md
          </a>{" "}
          for one-time setup.
        </p>
      </div>
      <div>
        <label
          htmlFor="external-id"
          className="block text-sm font-medium text-neutral-700 dark:text-neutral-300"
        >
          External ID
        </label>
        <input
          id="external-id"
          required
          value={externalId}
          onChange={(e) => setExternalId(e.target.value)}
          type="password"
          autoComplete="off"
          className="mt-1 block w-full rounded-md border border-neutral-300 bg-white px-3 py-2 font-mono text-sm shadow-sm focus:border-neutral-500 focus:outline-none focus:ring-1 focus:ring-neutral-500 dark:border-neutral-700 dark:bg-neutral-950"
        />
        <p className="mt-1 text-xs text-neutral-500">
          Stored in <code className="font-mono">sessionStorage</code> only —
          never sent to anywhere except the control plane on this same origin.
        </p>
      </div>
    </div>
  );
}

function AzureFields({
  tenantId,
  setTenantId,
  subscriptionId,
  setSubscriptionId,
  clientId,
  setClientId,
  clientSecret,
  setClientSecret,
}: {
  tenantId: string;
  setTenantId: (v: string) => void;
  subscriptionId: string;
  setSubscriptionId: (v: string) => void;
  clientId: string;
  setClientId: (v: string) => void;
  clientSecret: string;
  setClientSecret: (v: string) => void;
}) {
  const inputClass =
    "mt-1 block w-full rounded-md border border-neutral-300 bg-white px-3 py-2 font-mono text-sm shadow-sm focus:border-neutral-500 focus:outline-none focus:ring-1 focus:ring-neutral-500 dark:border-neutral-700 dark:bg-neutral-950";
  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <label
            htmlFor="az-tenant"
            className="block text-sm font-medium text-neutral-700 dark:text-neutral-300"
          >
            Tenant ID
          </label>
          <input
            id="az-tenant"
            required
            value={tenantId}
            onChange={(e) => setTenantId(e.target.value)}
            placeholder="11111111-1111-1111-1111-111111111111"
            className={inputClass}
          />
        </div>
        <div>
          <label
            htmlFor="az-sub"
            className="block text-sm font-medium text-neutral-700 dark:text-neutral-300"
          >
            Subscription ID
          </label>
          <input
            id="az-sub"
            required
            value={subscriptionId}
            onChange={(e) => setSubscriptionId(e.target.value)}
            placeholder="22222222-2222-2222-2222-222222222222"
            className={inputClass}
          />
        </div>
        <div>
          <label
            htmlFor="az-client"
            className="block text-sm font-medium text-neutral-700 dark:text-neutral-300"
          >
            Client ID
          </label>
          <input
            id="az-client"
            required
            value={clientId}
            onChange={(e) => setClientId(e.target.value)}
            placeholder="33333333-3333-3333-3333-333333333333"
            className={inputClass}
          />
        </div>
        <div>
          <label
            htmlFor="az-secret"
            className="block text-sm font-medium text-neutral-700 dark:text-neutral-300"
          >
            Client secret
          </label>
          <input
            id="az-secret"
            required
            type="password"
            autoComplete="off"
            value={clientSecret}
            onChange={(e) => setClientSecret(e.target.value)}
            className={inputClass}
          />
        </div>
      </div>
      <p className="text-xs text-neutral-500">
        Create a Service Principal:{" "}
        <code className="font-mono">
          az ad sp create-for-rbac --name ilabhu-runner --role Contributor
          --scopes /subscriptions/&lt;sub&gt;
        </code>
        . Stored in <code className="font-mono">sessionStorage</code> only —
        never sent anywhere except the control plane on this same origin.
      </p>
    </div>
  );
}

function GCPFields({
  serviceAccountKey,
  setServiceAccountKey,
}: {
  serviceAccountKey: string;
  setServiceAccountKey: (v: string) => void;
}) {
  return (
    <div className="space-y-4">
      <div>
        <label
          htmlFor="gcp-sa-key"
          className="block text-sm font-medium text-neutral-700 dark:text-neutral-300"
        >
          GCP Service Account key (JSON)
        </label>
        <textarea
          id="gcp-sa-key"
          required
          rows={6}
          value={serviceAccountKey}
          onChange={(e) => setServiceAccountKey(e.target.value)}
          placeholder='{"type":"service_account","project_id":"my-project",...}'
          className="mt-1 block w-full rounded-md border border-neutral-300 bg-white px-3 py-2 font-mono text-xs shadow-sm focus:border-neutral-500 focus:outline-none focus:ring-1 focus:ring-neutral-500 dark:border-neutral-700 dark:bg-neutral-950"
        />
        <p className="mt-1 text-xs text-neutral-500">
          Paste the full contents of the JSON key downloaded from{" "}
          <a
            className="underline"
            href="https://console.cloud.google.com/iam-admin/serviceaccounts"
            target="_blank"
            rel="noreferrer"
          >
            console.cloud.google.com → Service Accounts
          </a>
          . The project id is auto-extracted from <code>project_id</code>.
          Stored in <code className="font-mono">sessionStorage</code> only —
          never sent anywhere except the control plane on this same origin.
        </p>
      </div>
    </div>
  );
}

function DOFields({
  token,
  setToken,
}: {
  token: string;
  setToken: (v: string) => void;
}) {
  return (
    <div className="space-y-4">
      <div>
        <label
          htmlFor="do-token"
          className="block text-sm font-medium text-neutral-700 dark:text-neutral-300"
        >
          DigitalOcean Personal Access Token
        </label>
        <input
          id="do-token"
          required
          value={token}
          onChange={(e) => setToken(e.target.value)}
          type="password"
          autoComplete="off"
          placeholder="dop_v1_..."
          className="mt-1 block w-full rounded-md border border-neutral-300 bg-white px-3 py-2 font-mono text-sm shadow-sm focus:border-neutral-500 focus:outline-none focus:ring-1 focus:ring-neutral-500 dark:border-neutral-700 dark:bg-neutral-950"
        />
        <p className="mt-1 text-xs text-neutral-500">
          Create a token at{" "}
          <a
            className="underline"
            href="https://cloud.digitalocean.com/account/api/tokens"
            target="_blank"
            rel="noreferrer"
          >
            cloud.digitalocean.com/account/api/tokens
          </a>{" "}
          with read+write scope. Stored in{" "}
          <code className="font-mono">sessionStorage</code> only — never sent
          anywhere except the control plane on this same origin.
        </p>
      </div>
    </div>
  );
}
