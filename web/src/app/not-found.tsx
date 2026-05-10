import Link from "next/link";

export default function NotFound() {
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center text-center">
      <p className="font-mono text-sm uppercase tracking-widest text-neutral-500">
        404
      </p>
      <h1 className="mt-4 text-3xl font-semibold tracking-tight">
        That page is not in the catalog.
      </h1>
      <p className="mt-3 max-w-md text-neutral-600 dark:text-neutral-400">
        The exam, session, or route you tried to reach doesn&apos;t exist
        — or the control plane no longer knows about it.
      </p>
      <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
        <Link
          href="/"
          className="rounded-md bg-neutral-900 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-neutral-700 dark:bg-white dark:text-neutral-900 dark:hover:bg-neutral-200"
        >
          Back to the catalog
        </Link>
        <a
          href="https://github.com/eltonlaice/ilabhu"
          target="_blank"
          rel="noreferrer"
          className="rounded-md border border-neutral-300 px-4 py-2 text-sm font-medium text-neutral-700 hover:bg-neutral-50 dark:border-neutral-700 dark:text-neutral-300 dark:hover:bg-neutral-900"
        >
          Open the repo →
        </a>
      </div>
    </div>
  );
}
