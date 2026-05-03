"use client";

import Link from "next/link";
import { use, useEffect, useState } from "react";
import {
  destroySession,
  getExam,
  getSession,
  type ExamDetail,
  type ProviderCredentials,
  type Session,
} from "@/lib/api";
import { TaskValidator } from "@/components/TaskValidator";

type PageProps = {
  params: Promise<{ id: string }>;
};

const POLL_INTERVAL_MS = 5_000;

function statusColor(status: Session["status"]): string {
  switch (status) {
    case "ready":
      return "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300";
    case "provisioning":
    case "destroying":
      return "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300";
    case "failed":
      return "bg-rose-100 text-rose-700 dark:bg-rose-950 dark:text-rose-300";
    case "destroyed":
    default:
      return "bg-neutral-200 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300";
  }
}

export default function SessionPage({ params }: PageProps) {
  const { id } = use(params);

  const [session, setSession] = useState<Session | null>(null);
  const [exam, setExam] = useState<ExamDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [destroying, setDestroying] = useState(false);
  // refreshTick is bumped to trigger a refetch — both by the polling timer
  // and by onDestroy after a successful tear-down.
  const [refreshTick, setRefreshTick] = useState(0);

  // Fetch the session whenever the id or refreshTick changes.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const s = await getSession(id);
        if (!cancelled) {
          setSession(s);
          setError(null);
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : String(e));
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id, refreshTick]);

  // Fetch the exam manifest once we know the session's exam id, so the page
  // can render task instructions and per-task Validate buttons.
  useEffect(() => {
    if (!session || exam) return;
    let cancelled = false;
    (async () => {
      try {
        const detail = await getExam(session.exam_id);
        if (!cancelled) setExam(detail);
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : String(e));
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [session, exam]);

  // Poll while the session is in a non-terminal state.
  useEffect(() => {
    if (!session) return;
    const terminal =
      session.status === "ready" ||
      session.status === "failed" ||
      session.status === "destroyed";
    if (terminal) return;
    const t = setInterval(() => {
      setRefreshTick((n) => n + 1);
    }, POLL_INTERVAL_MS);
    return () => clearInterval(t);
  }, [session]);

  function downloadKubeconfig() {
    if (!session?.kubeconfig_b64) return;
    const blob = new Blob([atob(session.kubeconfig_b64)], {
      type: "application/yaml",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${session.exam_id.replace(/\//g, "_")}-${session.id.slice(0, 8)}.kubeconfig`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  }

  async function onDestroy() {
    if (!session) return;
    if (
      !confirm(
        "Tear down this session? Underlying cloud resources will be deleted.",
      )
    ) {
      return;
    }
    const stored = window.sessionStorage.getItem(`ilabhu:creds:${session.id}`);
    if (!stored) {
      alert(
        "Credentials are no longer in this browser session. Destroy via curl using the same provider credentials you started with.",
      );
      return;
    }
    setDestroying(true);
    try {
      const creds = JSON.parse(stored) as ProviderCredentials;
      await destroySession(session.id, creds);
      window.sessionStorage.removeItem(`ilabhu:creds:${session.id}`);
      setRefreshTick((n) => n + 1);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setDestroying(false);
    }
  }

  if (error && !session) {
    return (
      <div className="rounded-md border border-rose-200 bg-rose-50 p-4 text-sm text-rose-800 dark:border-rose-900 dark:bg-rose-950/30 dark:text-rose-200">
        <p className="font-medium">Could not load session.</p>
        <p className="mt-1 font-mono text-xs">{error}</p>
      </div>
    );
  }

  if (!session) {
    return <p className="text-sm text-neutral-500">Loading…</p>;
  }

  return (
    <div className="space-y-8">
      <div>
        <Link
          href={`/exams/${session.exam_id.split("/").map(encodeURIComponent).join("/")}`}
          className="text-sm text-neutral-500 hover:text-neutral-900 dark:hover:text-neutral-100"
        >
          ← Exam
        </Link>
        <div className="mt-3 flex items-center gap-3">
          <span
            className={`rounded px-2 py-1 text-xs font-medium uppercase tracking-wide ${statusColor(
              session.status,
            )}`}
          >
            {session.status}
          </span>
          <span className="rounded bg-neutral-100 px-1.5 py-0.5 font-mono text-xs text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">
            {session.provider}
          </span>
          <span className="font-mono text-xs text-neutral-500">{session.id}</span>
        </div>
        <h1 className="mt-3 text-2xl font-semibold tracking-tight">
          {session.exam_id}
        </h1>
      </div>

      {error ? (
        <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
          Polling error: <span className="font-mono">{error}</span>
        </div>
      ) : null}

      {session.status === "failed" && session.error ? (
        <div className="rounded-md border border-rose-200 bg-rose-50 p-4 text-sm text-rose-800 dark:border-rose-900 dark:bg-rose-950/30 dark:text-rose-200">
          <p className="font-medium">Provisioning failed.</p>
          <pre className="mt-2 overflow-auto whitespace-pre-wrap font-mono text-xs">
            {session.error}
          </pre>
        </div>
      ) : null}

      {session.status === "provisioning" ? (
        <div className="rounded-md border border-neutral-200 bg-white p-4 text-sm text-neutral-600 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-400">
          Provisioning your environment. Polls every {POLL_INTERVAL_MS / 1000}
          s. This typically takes 2–4 minutes.
        </div>
      ) : null}

      {session.status === "ready" && session.kubeconfig_b64 ? (
        <section className="space-y-3">
          <h2 className="text-lg font-medium">Access</h2>
          <div className="rounded-md border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
            <p className="text-sm text-neutral-600 dark:text-neutral-400">
              Download the kubeconfig and start using the cluster.
            </p>
            <div className="mt-3 flex flex-wrap items-center gap-3">
              <button
                onClick={downloadKubeconfig}
                className="rounded-md bg-neutral-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-neutral-700 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
              >
                Download kubeconfig
              </button>
              <code className="rounded bg-neutral-100 px-2 py-1 font-mono text-xs dark:bg-neutral-800">
                KUBECONFIG=./downloaded.kubeconfig kubectl get nodes
              </code>
            </div>
          </div>
        </section>
      ) : null}

      {session.status === "ready" && exam && exam.tasks.length > 0 ? (
        <section>
          <h2 className="mb-3 text-lg font-medium">Tasks</h2>
          <ol className="space-y-4">
            {exam.tasks.map((task, i) => (
              <li
                key={task.id}
                className="rounded-lg border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"
              >
                <div className="flex items-baseline gap-3">
                  <span className="text-xs text-neutral-500">{i + 1}.</span>
                  <h3 className="font-medium">{task.title}</h3>
                  {task.domain ? (
                    <span className="ml-auto rounded bg-neutral-100 px-1.5 py-0.5 text-xs text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400">
                      {task.domain}
                    </span>
                  ) : null}
                </div>
                <p className="mt-2 whitespace-pre-wrap text-sm text-neutral-700 dark:text-neutral-300">
                  {task.instructions.trim()}
                </p>
                <TaskValidator sessionID={session.id} taskID={task.id} />
              </li>
            ))}
          </ol>
        </section>
      ) : null}

      {session.outputs ? (
        <section>
          <h2 className="mb-2 text-lg font-medium">Terraform outputs</h2>
          <pre className="overflow-auto rounded-md border border-neutral-200 bg-white p-3 font-mono text-xs dark:border-neutral-800 dark:bg-neutral-900">
            {JSON.stringify(session.outputs, null, 2)}
          </pre>
        </section>
      ) : null}

      {(session.status === "ready" || session.status === "failed") && (
        <section className="border-t border-neutral-200 pt-6 dark:border-neutral-800">
          <h2 className="mb-2 text-lg font-medium text-rose-700 dark:text-rose-400">
            Destroy session
          </h2>
          <p className="mb-3 text-sm text-neutral-600 dark:text-neutral-400">
            Tears down everything provisioned for this session.
          </p>
          <button
            onClick={onDestroy}
            disabled={destroying}
            className="rounded-md border border-rose-300 bg-white px-3 py-1.5 text-sm font-medium text-rose-700 hover:bg-rose-50 disabled:opacity-50 dark:border-rose-900 dark:bg-neutral-950 dark:text-rose-300 dark:hover:bg-rose-950/30"
          >
            {destroying ? "Destroying…" : "Destroy"}
          </button>
        </section>
      )}
    </div>
  );
}
