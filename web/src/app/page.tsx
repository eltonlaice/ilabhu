import Link from "next/link";
import { listLabs, type Lab } from "@/lib/api";

export const dynamic = "force-dynamic";

function difficultyColor(d: Lab["difficulty"]): string {
  switch (d) {
    case "easy":
      return "bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300";
    case "medium":
      return "bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300";
    case "hard":
      return "bg-rose-100 text-rose-700 dark:bg-rose-950 dark:text-rose-300";
    default:
      return "bg-neutral-100 text-neutral-700 dark:bg-neutral-800 dark:text-neutral-300";
  }
}

export default async function HomePage() {
  let labs: Lab[] = [];
  let error: string | null = null;
  try {
    labs = await listLabs();
  } catch (e) {
    error = e instanceof Error ? e.message : String(e);
  }

  return (
    <div>
      <section className="mb-10">
        <h1 className="text-3xl font-semibold tracking-tight">
          Hands-on certification labs you run in your own cloud.
        </h1>
        <p className="mt-3 max-w-2xl text-neutral-600 dark:text-neutral-400">
          Pick a lab. We provision it in your own AWS account, drop you into a
          terminal, and grade your work against the exam objective.
        </p>
      </section>

      {error ? (
        <div className="rounded-md border border-rose-200 bg-rose-50 p-4 text-sm text-rose-800 dark:border-rose-900 dark:bg-rose-950/30 dark:text-rose-200">
          <p className="font-medium">Could not reach the control plane.</p>
          <p className="mt-1 font-mono text-xs">{error}</p>
          <p className="mt-2">
            Start <code className="font-mono">ilabhud</code> on{" "}
            <code className="font-mono">:8080</code> or set{" "}
            <code className="font-mono">ILABHU_API_BASE</code>.
          </p>
        </div>
      ) : labs.length === 0 ? (
        <div className="rounded-md border border-neutral-200 bg-white p-6 text-sm text-neutral-600 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-400">
          No labs found. Add one under{" "}
          <code className="font-mono">labs/&lt;exam&gt;/&lt;id&gt;/lab.yaml</code>.
        </div>
      ) : (
        <ul className="grid gap-4 sm:grid-cols-2">
          {labs.map((lab) => (
            <li key={lab.id}>
              <Link
                href={`/labs/${encodeURIComponent(lab.id)}`}
                className="block h-full rounded-lg border border-neutral-200 bg-white p-5 transition hover:border-neutral-400 hover:shadow-sm dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-neutral-600"
              >
                <div className="flex items-center gap-2 text-xs text-neutral-500">
                  <span className="rounded bg-neutral-100 px-1.5 py-0.5 font-medium uppercase tracking-wide dark:bg-neutral-800">
                    {lab.exam}
                  </span>
                  <span
                    className={`rounded px-1.5 py-0.5 font-medium uppercase tracking-wide ${difficultyColor(
                      lab.difficulty,
                    )}`}
                  >
                    {lab.difficulty}
                  </span>
                  <span className="ml-auto">~{lab.estimated_minutes} min</span>
                </div>
                <h2 className="mt-3 text-lg font-medium">{lab.title}</h2>
                <p className="mt-1 text-sm text-neutral-600 dark:text-neutral-400">
                  {lab.summary}
                </p>
                <p className="mt-3 font-mono text-xs text-neutral-400">
                  {lab.id}
                </p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
