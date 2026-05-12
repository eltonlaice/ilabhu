// Thin client for ilabhud's HTTP API.
//
// In the browser we use a relative /api/* URL — next.config.ts rewrites that
// to the ilabhud base URL configured by ILABHU_API_BASE.
//
// On the server (Server Components, route handlers) fetch() needs an absolute
// URL, so we read ILABHU_API_BASE directly and skip the rewrite.

export type Provider = "aws" | "gcp" | "azure" | "digitalocean" | "byo-hosts";

export type Exam = {
  id: string;
  title: string;
  exam: string;
  difficulty: "easy" | "medium" | "hard";
  summary: string;
  estimated_minutes: number;
  time_limit_minutes: number;
  passing_score: number;
  providers: Provider[];
};

export type ExamTask = {
  id: string;
  title: string;
  domain?: string;
  weight?: number;
  instructions: string;
};

export type ExamDomain = {
  name: string;
  weight: number;
};

export type ExamDetail = Exam & {
  version: number;
  instructions: string;
  domains: ExamDomain[];
  tasks: ExamTask[];
  infrastructure: {
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
  exam_id: string;
  provider: Provider;
  status: SessionStatus;
  created_at: string;
  updated_at: string;
  outputs?: Record<string, unknown>;
  kubeconfig_b64?: string;
  error?: string;
};

export type BYOHost = {
  role: string;
  address: string;
  ssh_user: string;
};

export type ProviderCredentials = {
  provider: Provider;
  aws?: { role_arn: string; external_id: string };
  digitalocean?: { token: string };
  gcp?: { service_account_key: string };
  azure?: {
    tenant_id: string;
    subscription_id: string;
    client_id: string;
    client_secret: string;
  };
  byo_hosts?: { ssh_private_key: string; hosts: BYOHost[] };
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

export async function listExams(): Promise<Exam[]> {
  return request<Exam[]>("/v1/exams");
}

export async function getExam(id: string): Promise<ExamDetail> {
  // id may contain slashes (e.g. cka/warmup); each segment must be encoded
  // individually so the full id reaches the catch-all backend route intact.
  const encoded = id.split("/").map(encodeURIComponent).join("/");
  return request<ExamDetail>(`/v1/exams/${encoded}`);
}

export async function getSession(id: string): Promise<Session> {
  return request<Session>(`/v1/sessions/${encodeURIComponent(id)}`);
}

export async function startSession(
  examID: string,
  creds: ProviderCredentials,
): Promise<Session> {
  return request<Session>("/v1/sessions", {
    method: "POST",
    body: JSON.stringify({ exam_id: examID, ...creds }),
  });
}

export async function destroySession(
  id: string,
  creds: ProviderCredentials,
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
