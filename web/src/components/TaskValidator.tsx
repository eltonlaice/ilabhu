"use client";

import { useState } from "react";
import {
  validateTask,
  type ValidateResponse,
  type ValidationResult,
} from "@/lib/api";

type Props = {
  sessionID: string;
  taskID: string;
};

type State =
  | { kind: "idle" }
  | { kind: "running" }
  | { kind: "done"; response: ValidateResponse }
  | { kind: "error"; error: string };

export function TaskValidator({ sessionID, taskID }: Props) {
  const [state, setState] = useState<State>({ kind: "idle" });

  async function run() {
    setState({ kind: "running" });
    try {
      const response = await validateTask(sessionID, taskID);
      setState({ kind: "done", response });
    } catch (e) {
      setState({
        kind: "error",
        error: e instanceof Error ? e.message : String(e),
      });
    }
  }

  return (
    <div className="mt-3">
      <button
        onClick={run}
        disabled={state.kind === "running"}
        className="rounded-md border border-neutral-300 bg-white px-3 py-1.5 text-sm font-medium text-neutral-900 hover:bg-neutral-50 disabled:opacity-50 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100 dark:hover:bg-neutral-900"
      >
        {state.kind === "running" ? "Validating…" : "Validate"}
      </button>

      {state.kind === "error" ? (
        <div className="mt-3 rounded-md border border-rose-200 bg-rose-50 p-3 text-sm text-rose-800 dark:border-rose-900 dark:bg-rose-950/30 dark:text-rose-200">
          <span className="font-mono text-xs">{state.error}</span>
        </div>
      ) : null}

      {state.kind === "done" ? (
        <ValidationSummary response={state.response} />
      ) : null}
    </div>
  );
}

function ValidationSummary({ response }: { response: ValidateResponse }) {
  return (
    <div
      className={`mt-3 rounded-md border p-3 text-sm ${
        response.all_passed
          ? "border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/30 dark:text-emerald-200"
          : "border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200"
      }`}
    >
      <p className="font-medium">
        {response.all_passed ? "All validations passed" : "Some validations failed"}
      </p>
      <ol className="mt-2 space-y-1">
        {response.results.map((r) => (
          <ResultRow key={r.index} result={r} />
        ))}
      </ol>
    </div>
  );
}

function ResultRow({ result }: { result: ValidationResult }) {
  return (
    <li className="flex items-start gap-2 font-mono text-xs">
      <span aria-hidden>{result.passed ? "✓" : "✗"}</span>
      <div>
        <span className="font-medium">
          [{result.index}] {result.kind}
        </span>
        {result.message ? (
          <p className="mt-0.5 whitespace-pre-wrap opacity-80">
            {result.message}
          </p>
        ) : null}
      </div>
    </li>
  );
}
