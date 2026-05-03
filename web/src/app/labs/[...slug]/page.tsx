import Link from "next/link";
import { notFound } from "next/navigation";
import { getLab } from "@/lib/api";
import { StartSessionForm } from "@/components/StartSessionForm";

export const dynamic = "force-dynamic";

type PageProps = {
  params: Promise<{ slug: string[] }>;
};

export default async function LabPage({ params }: PageProps) {
  const { slug } = await params;
  const id = slug.map(decodeURIComponent).join("/");

  let lab;
  try {
    lab = await getLab(id);
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
          ← All labs
        </Link>
        <div className="mt-3 flex items-center gap-2 text-xs text-neutral-500">
          <span className="rounded bg-neutral-100 px-1.5 py-0.5 font-medium uppercase tracking-wide dark:bg-neutral-800">
            {lab.exam}
          </span>
          <span className="rounded bg-neutral-100 px-1.5 py-0.5 font-medium uppercase tracking-wide dark:bg-neutral-800">
            {lab.difficulty}
          </span>
          <span>~{lab.estimated_minutes} min</span>
          <span className="ml-auto font-mono text-xs text-neutral-400">
            {lab.id}
          </span>
        </div>
        <h1 className="mt-3 text-3xl font-semibold tracking-tight">
          {lab.title}
        </h1>
        <p className="mt-2 text-neutral-600 dark:text-neutral-400">
          {lab.summary}
        </p>
        <p className="mt-4 text-sm italic text-neutral-500">
          Exam objective: {lab.exam_objective}
        </p>
      </div>

      {lab.instructions ? (
        <section>
          <h2 className="mb-3 text-lg font-medium">Overview</h2>
          <p className="whitespace-pre-wrap text-neutral-700 dark:text-neutral-300">
            {lab.instructions.trim()}
          </p>
        </section>
      ) : null}

      <section>
        <h2 className="mb-3 text-lg font-medium">
          Tasks ({lab.tasks.length})
        </h2>
        <ol className="space-y-4">
          {lab.tasks.map((task, i) => (
            <li
              key={task.id}
              className="rounded-lg border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"
            >
              <div className="flex items-baseline gap-3">
                <span className="text-xs text-neutral-500">{i + 1}.</span>
                <h3 className="font-medium">{task.title}</h3>
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
          ilabhu will assume a role in your AWS account, run the lab&apos;s
          Terraform module, and hand you back a kubeconfig. The session
          auto-destroys after {lab.infrastructure.ttl_minutes} minutes — or
          sooner if you destroy it.
        </p>
        <StartSessionForm labID={lab.id} />
      </section>
    </div>
  );
}
