// Thin client for ilabhud's HTTP API.
//
// In the browser we use a relative /api/* URL — next.config.ts rewrites that
// to the ilabhud base URL configured by ILABHU_API_BASE.
//
// On the server (Server Components, route handlers) fetch() needs an absolute
// URL, so we read ILABHU_API_BASE directly and skip the rewrite.

export type Lab = {
  id: string;
  title: string;
  exam: string;
  difficulty: "easy" | "medium" | "hard";
  summary: string;
  estimated_minutes: number;
};

export type LabTask = {
  id: string;
  title: string;
  instructions: string;
};

export type LabDetail = Lab & {
  version: number;
  exam_objective: string;
  instructions: string;
  tasks: LabTask[];
  infrastructure: {
    provider: string;
    ttl_minutes: number;
  };
};

export type SessionStatus =
  | "provisioning"
  | "ready"
  | "failed"
  | "destroying"
  | "destroyed";

export type Session = {
  id: string;
  lab_id: string;
  status: SessionStatus;
  created_at: string;
  updated_at: string;
  outputs?: Record<string, unknown>;
  kubeconfig_b64?: string;
  error?: string;
};

function apiURL(path: string): string {
  if (typeof window === "undefined") {
    const base = process.env.ILABHU_API_BASE ?? "http://127.0.0.1:8080";
    return `${base}${path}`;
  }
  return `/api${path}`;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(apiURL(path), {
    ...init,
    headers: {
      "content-type": "application/json",
      ...(init?.headers ?? {}),
    },
    cache: "no-store",
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`HTTP ${res.status}: ${body || res.statusText}`);
  }
  return res.json() as Promise<T>;
}

export async function listLabs(): Promise<Lab[]> {
  return request<Lab[]>("/v1/labs");
}

export async function getLab(id: string): Promise<LabDetail> {
  // id may contain slashes (e.g. cka/pod-resource-limits); each segment must
  // be encoded individually so the full id reaches the catch-all backend
  // route intact.
  const encoded = id.split("/").map(encodeURIComponent).join("/");
  return request<LabDetail>(`/v1/labs/${encoded}`);
}

export async function getSession(id: string): Promise<Session> {
  return request<Session>(`/v1/sessions/${encodeURIComponent(id)}`);
}

export async function startSession(input: {
  lab_id: string;
  aws_role_arn: string;
  aws_external_id: string;
}): Promise<Session> {
  return request<Session>("/v1/sessions", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export type ValidationResult = {
  index: number;
  kind: string;
  passed: boolean;
  message?: string;
};

export type ValidateResponse = {
  task_id: string;
  all_passed: boolean;
  results: ValidationResult[];
};

export async function validateTask(
  sessionID: string,
  taskID: string,
): Promise<ValidateResponse> {
  return request<ValidateResponse>(
    `/v1/sessions/${encodeURIComponent(sessionID)}/tasks/${encodeURIComponent(taskID)}/validate`,
    { method: "POST" },
  );
}

export async function destroySession(
  id: string,
  creds: { aws_role_arn: string; aws_external_id: string },
): Promise<void> {
  const res = await fetch(apiURL(`/v1/sessions/${encodeURIComponent(id)}`), {
    method: "DELETE",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(creds),
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`HTTP ${res.status}: ${body || res.statusText}`);
  }
}
