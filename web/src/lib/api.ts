// Thin client for ilabhud's HTTP API. The browser hits /api/* on the same
// origin; next.config.ts rewrites that to the ilabhud base URL configured by
// ILABHU_API_BASE (default http://127.0.0.1:8080).

export type Lab = {
  id: string;
  title: string;
  exam: string;
  difficulty: "easy" | "medium" | "hard";
  summary: string;
  estimated_minutes: number;
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

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api${path}`, {
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

export async function destroySession(
  id: string,
  creds: { aws_role_arn: string; aws_external_id: string },
): Promise<void> {
  const res = await fetch(`/api/v1/sessions/${encodeURIComponent(id)}`, {
    method: "DELETE",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(creds),
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`HTTP ${res.status}: ${body || res.statusText}`);
  }
}
