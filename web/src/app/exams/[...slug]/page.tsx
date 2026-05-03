import Link from "next/link";
import { notFound } from "next/navigation";
import { getExam } from "@/lib/api";
import { StartSessionForm } from "@/components/StartSessionForm";

export const dynamic = "force-dynamic";

type PageProps = {
  params: Promise<{ slug: string[] }>;
};

export default async function ExamPage({ params }: PageProps) {
  const { slug } = await params;
  const id = slug.map(decodeURIComponent).join("/");

  let exam;
  try {
    exam = await getExam(id);
  } catch (e) {
    if (e instanceof Error && e.message.startsWith("HTTP 404")) {
      notFound();
    }
    throw e;
  }

  return (
    <div className="space-y-10">
      <div>
        <Link
          href="/"
          className="text-sm text-neutral-500 hover:text-neutral-900 dark:hover:text-neutral-100"
        >
          ← All exams
        </Link>
        <div className="mt-3 flex items-center gap-2 text-xs text-neutral-500">
          <span className="rounded bg-neutral-100 px-1.5 py-0.5 font-medium uppercase tracking-wide dark:bg-neutral-800">
            {exam.exam}
          </span>
          <span className="rounded bg-neutral-100 px-1.5 py-0.5 font-medium uppercase tracking-wide dark:bg-neutral-800">
            {exam.difficulty}
          </span>
          <span>{exam.time_limit_minutes} min · pass {exam.passing_score}%</span>
          <span className="ml-auto font-mono text-xs text-neutral-400">
            {exam.id}
          </span>
        </div>
        <h1 className="mt-3 text-3xl font-semibold tracking-tight">
          {exam.title}
        </h1>
        <p className="mt-2 text-neutral-600 dark:text-neutral-400">
          {exam.summary}
        </p>
      </div>

      {exam.domains && exam.domains.length > 0 ? (
        <section>
          <h2 className="mb-2 text-lg font-medium">Domains</h2>
          <ul className="grid gap-2 sm:grid-cols-2">
            {exam.domains.map((d) => (
              <li
                key={d.name}
                className="flex items-center justify-between rounded-md border border-neutral-200 bg-white px-3 py-2 text-sm dark:border-neutral-800 dark:bg-neutral-900"
              >
                <span>{d.name}</span>
                <span className="font-mono text-xs text-neutral-500">
                  {d.weight}%
                </span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {exam.instructions ? (
        <section>
          <h2 className="mb-3 text-lg font-medium">Overview</h2>
          <p className="whitespace-pre-wrap text-neutral-700 dark:text-neutral-300">
            {exam.instructions.trim()}
          </p>
        </section>
      ) : null}

      <section>
        <h2 className="mb-3 text-lg font-medium">
          Tasks ({exam.tasks.length})
        </h2>
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
            </li>
          ))}
        </ol>
      </section>

      <section>
        <h2 className="mb-3 text-lg font-medium">Start a session</h2>
        <p className="mb-4 text-sm text-neutral-600 dark:text-neutral-400">
          ilabhu provisions the environment, hands you back a kubeconfig, and
          auto-destroys the session after {exam.infrastructure.ttl_minutes}{" "}
          minutes (or sooner if you destroy it yourself).
        </p>
        <StartSessionForm examID={exam.id} providers={exam.providers} />
      </section>
    </div>
  );
}
