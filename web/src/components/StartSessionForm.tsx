"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { startSession } from "@/lib/api";

type Props = { labID: string };

export function StartSessionForm({ labID }: Props) {
  const router = useRouter();
  const [roleArn, setRoleArn] = useState("");
  const [externalId, setExternalId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      const sess = await startSession({
        lab_id: labID,
        aws_role_arn: roleArn.trim(),
        aws_external_id: externalId.trim(),
      });
      // Persist the credentials in sessionStorage so the session detail page
      // can submit them back when the user destroys the session. They never
      // leave the browser otherwise.
      window.sessionStorage.setItem(
        `ilabhu:creds:${sess.id}`,
        JSON.stringify({
          aws_role_arn: roleArn.trim(),
          aws_external_id: externalId.trim(),
        }),
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
      {error ? (
        <div className="rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-800 dark:border-rose-900 dark:bg-rose-950/30 dark:text-rose-200">
          {error}
        </div>
      ) : null}
      <button
        type="submit"
        disabled={submitting}
        className="rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-neutral-700 focus:outline-none focus:ring-2 focus:ring-neutral-500 focus:ring-offset-2 disabled:opacity-50 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
      >
        {submitting ? "Provisioning..." : "Start session"}
      </button>
    </form>
  );
}
