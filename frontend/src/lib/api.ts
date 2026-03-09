import type {
  Allocation,
  APIEnvelope,
  APIErrorBody,
  AvailabilityResult,
  CreateAllocationInput,
  DNSRecord,
  EmailRouteAvailabilityResult,
  ManagedDomain,
  MeResponse,
  SubmitPermissionApplicationInput,
  SupervisionEntry,
  UpsertMyCatchAllEmailRouteInput,
  UpsertMyDefaultEmailRouteInput,
  UpsertDNSRecordInput,
  UserEmailRoute,
  UserPermission,
} from '../types/api';

// resolveAPIBaseURL keeps local development simple while still protecting the
// deployed Pages frontend from accidentally talking to itself and parsing the
// returned HTML as JSON.
function resolveAPIBaseURL(): string {
  const configuredBaseURL = import.meta.env.VITE_API_BASE_URL?.trim();
  if (configuredBaseURL) {
    return configuredBaseURL.replace(/\/+$/, '');
  }

  if (typeof window === 'undefined') {
    return '';
  }

  const currentURL = new URL(window.location.origin);
  const { hostname, origin, port, protocol } = currentURL;
  const isFrontendSubdomain = hostname.startsWith('app.') || hostname.startsWith('admin.');
  const isLocalHostname = hostname === 'localhost' || hostname.endsWith('.localhost');

  if (isFrontendSubdomain && !isLocalHostname) {
    const apiHostname = `api.${hostname.split('.').slice(1).join('.')}`;
    const apiOrigin = `${protocol}//${apiHostname}${port ? `:${port}` : ''}`;
    return apiOrigin.replace(/\/+$/, '');
  }

  return origin.replace(/\/+$/, '');
}

// apiBaseURL stores the backend origin used by the public frontend.
export const apiBaseURL = resolveAPIBaseURL();

// APIError is the shared frontend error shape used by public pages.
export class APIError extends Error {
  code: string;
  status: number;

  constructor(message: string, code: string, status: number) {
    super(message);
    this.name = 'APIError';
    this.code = code;
    this.status = status;
  }
}

// request wraps fetch so every page consistently gets cookies, JSON decoding,
// and clearer messages when the deployment accidentally returns HTML instead of API data.
async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${apiBaseURL}${path}`, {
    credentials: 'include',
    ...init,
    headers: {
      Accept: 'application/json',
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...(init.headers ?? {}),
    },
  });

  const contentType = response.headers.get('content-type')?.toLowerCase() ?? '';
  const isJSONResponse = contentType.includes('application/json');

  if (!response.ok) {
    let errorBody: APIErrorBody | null = null;
    if (isJSONResponse) {
      try {
        errorBody = (await response.json()) as APIErrorBody;
      } catch {
        errorBody = null;
      }
    }

    throw new APIError(
      errorBody?.error.message ?? `Request failed with status ${response.status}`,
      errorBody?.error.code ?? 'http_error',
      response.status,
    );
  }

  if (!isJSONResponse) {
    throw new APIError(
      '后端返回了非 JSON 响应，请检查 VITE_API_BASE_URL 或反向代理配置。',
      'invalid_response_content_type',
      response.status,
    );
  }

  const envelope = (await response.json()) as APIEnvelope<T>;
  return envelope.data;
}

// getAuthLoginURL builds the public Linux Do OAuth entrypoint.
export function getAuthLoginURL(nextPath: string): string {
  return `${apiBaseURL}/v1/auth/login?next=${encodeURIComponent(nextPath)}`;
}

// getCurrentSession loads the current browser session state.
export function getCurrentSession(): Promise<MeResponse> {
  return request<MeResponse>('/v1/me');
}

// listPublicDomains returns the managed domains currently visible to the public site.
export function listPublicDomains(): Promise<ManagedDomain[]> {
  return request<ManagedDomain[]>('/v1/public/domains');
}

// listPublicSupervisionEntries returns the privacy-safe ownership list shown on the supervision page.
export function listPublicSupervisionEntries(): Promise<SupervisionEntry[]> {
  return request<SupervisionEntry[]>('/v1/public/supervision');
}

// checkAllocationAvailability checks one subdomain prefix on a managed root domain.
export function checkAllocationAvailability(rootDomain: string, prefix: string): Promise<AvailabilityResult> {
  const query = new URLSearchParams({
    root_domain: rootDomain,
    prefix,
  });
  return request<AvailabilityResult>(`/v1/public/allocations/check?${query.toString()}`);
}

// checkPublicEmailRouteAvailability checks whether one mailbox local-part is available.
export function checkPublicEmailRouteAvailability(rootDomain: string, prefix: string): Promise<EmailRouteAvailabilityResult> {
  const query = new URLSearchParams({
    root_domain: rootDomain,
    prefix,
  });
  return request<EmailRouteAvailabilityResult>(`/v1/public/email-routes/check?${query.toString()}`);
}

// createAllocation lets the current user request a new subdomain allocation.
export function createAllocation(input: CreateAllocationInput, csrfToken: string): Promise<Allocation> {
  return request<Allocation>('/v1/my/allocations', {
    method: 'POST',
    headers: {
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify(input),
  });
}

// listMyPermissions returns the current authenticated user's permission cards.
export function listMyPermissions(): Promise<UserPermission[]> {
  return request<UserPermission[]>('/v1/my/permissions');
}

// submitPermissionApplication stores one permission application for the current user.
export function submitPermissionApplication(input: SubmitPermissionApplicationInput, csrfToken: string): Promise<UserPermission> {
  return request<UserPermission>('/v1/my/permissions/applications', {
    method: 'POST',
    headers: {
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify(input),
  });
}

// listMyEmailRoutes returns the mailbox rows visible to the current user.
export function listMyEmailRoutes(): Promise<UserEmailRoute[]> {
  return request<UserEmailRoute[]>('/v1/my/email-routes');
}

// upsertDefaultEmailRoute saves the forwarding target for the implicit default mailbox.
export function upsertDefaultEmailRoute(input: UpsertMyDefaultEmailRouteInput, csrfToken: string): Promise<UserEmailRoute> {
  return request<UserEmailRoute>('/v1/my/email-routes/default', {
    method: 'PUT',
    headers: {
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify(input),
  });
}

// upsertCatchAllEmailRoute saves the forwarding target for the permission-gated catch-all mailbox.
export function upsertCatchAllEmailRoute(input: UpsertMyCatchAllEmailRouteInput, csrfToken: string): Promise<UserEmailRoute> {
  return request<UserEmailRoute>('/v1/my/email-routes/catch-all', {
    method: 'PUT',
    headers: {
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify(input),
  });
}

// listAllocationRecords returns the live DNS records currently attached to one allocation.
export function listAllocationRecords(allocationID: number): Promise<DNSRecord[]> {
  return request<DNSRecord[]>(`/v1/my/allocations/${allocationID}/records`);
}

// createDNSRecord adds one DNS record under the given allocation.
export function createDNSRecord(
  allocationID: number,
  input: UpsertDNSRecordInput,
  csrfToken: string,
): Promise<DNSRecord> {
  return request<DNSRecord>(`/v1/my/allocations/${allocationID}/records`, {
    method: 'POST',
    headers: {
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify(input),
  });
}

// updateDNSRecord updates one DNS record under the given allocation.
export function updateDNSRecord(
  allocationID: number,
  recordID: string,
  input: UpsertDNSRecordInput,
  csrfToken: string,
): Promise<DNSRecord> {
  return request<DNSRecord>(`/v1/my/allocations/${allocationID}/records/${recordID}`, {
    method: 'PATCH',
    headers: {
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify(input),
  });
}

// deleteDNSRecord deletes one DNS record under the given allocation.
export function deleteDNSRecord(allocationID: number, recordID: string, csrfToken: string): Promise<{ deleted: boolean }> {
  return request<{ deleted: boolean }>(`/v1/my/allocations/${allocationID}/records/${recordID}`, {
    method: 'DELETE',
    headers: {
      'X-CSRF-Token': csrfToken,
    },
  });
}

// logout destroys the current browser session on the backend.
export function logout(csrfToken: string): Promise<{ logged_out: boolean }> {
  return request<{ logged_out: boolean }>('/v1/auth/logout', {
    method: 'POST',
    headers: {
      'X-CSRF-Token': csrfToken,
    },
  });
}
