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

const IMPLEMENTED: Provider[] = ["aws"];

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
