import Link from "next/link";
import { listExams, type Exam, type Provider } from "@/lib/api";

export const dynamic = "force-dynamic";

function difficultyColor(d: Exam["difficulty"]): string {
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

function providerLabel(p: Provider): string {
  switch (p) {
    case "aws":
      return "AWS";
    case "gcp":
      return "GCP";
    case "azure":
      return "Azure";
    case "digitalocean":
      return "DigitalOcean";
    case "byo-hosts":
      return "Your hosts";
  }
}

export default async function HomePage() {
  let exams: Exam[] = [];
  let error: string | null = null;
  try {
    exams = await listExams();
  } catch (e) {
    error = e instanceof Error ? e.message : String(e);
  }

  return (
    <div>
      <section className="mb-10">
        <h1 className="text-3xl font-semibold tracking-tight">
          Practice the full exam, in your own infrastructure.
        </h1>
        <p className="mt-3 max-w-2xl text-neutral-600 dark:text-neutral-400">
          Pick a certification. ilabhu provisions the environment, runs you
          through the objectives, and grades each task — on your AWS, GCP,
          Azure, DigitalOcean account, or on Linux servers you already own.
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
      ) : exams.length === 0 ? (
        <div className="rounded-md border border-neutral-200 bg-white p-6 text-sm text-neutral-600 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-400">
          No exams found. Add one under{" "}
          <code className="font-mono">exams/&lt;exam&gt;/&lt;id&gt;/exam.yaml</code>.
        </div>
      ) : (
        <ul className="grid gap-4 sm:grid-cols-2">
          {exams.map((exam) => (
            <li key={exam.id}>
              <Link
                href={`/exams/${exam.id.split("/").map(encodeURIComponent).join("/")}`}
                className="block h-full rounded-lg border border-neutral-200 bg-white p-5 transition hover:border-neutral-400 hover:shadow-sm dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-neutral-600"
              >
                <div className="flex items-center gap-2 text-xs text-neutral-500">
                  <span className="rounded bg-neutral-100 px-1.5 py-0.5 font-medium uppercase tracking-wide dark:bg-neutral-800">
                    {exam.exam}
                  </span>
                  <span
                    className={`rounded px-1.5 py-0.5 font-medium uppercase tracking-wide ${difficultyColor(
                      exam.difficulty,
                    )}`}
                  >
                    {exam.difficulty}
                  </span>
                  <span className="ml-auto">
                    {exam.time_limit_minutes
                      ? `${exam.time_limit_minutes} min`
                      : `~${exam.estimated_minutes} min`}
                  </span>
                </div>
                <h2 className="mt-3 text-lg font-medium">{exam.title}</h2>
                <p className="mt-1 text-sm text-neutral-600 dark:text-neutral-400">
                  {exam.summary}
                </p>
                <div className="mt-3 flex flex-wrap items-center gap-2 text-xs">
                  <span className="text-neutral-500">Pass:</span>
                  <span className="font-medium">{exam.passing_score}%</span>
                  <span className="ml-auto flex flex-wrap items-center gap-1">
                    {exam.providers.map((p) => (
                      <span
                        key={p}
                        className="rounded bg-neutral-100 px-1.5 py-0.5 font-medium text-neutral-600 dark:bg-neutral-800 dark:text-neutral-400"
                      >
                        {providerLabel(p)}
                      </span>
                    ))}
                  </span>
                </div>
                <p className="mt-3 font-mono text-xs text-neutral-400">
                  {exam.id}
                </p>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
