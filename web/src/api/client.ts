export interface TokenPair {
  access_token: string
  refresh_token: string
}

export interface VerifyResponse {
  user_id: string
  display_name: string
  avatar_shape: string
  created_at: string
}

export class ApiRequestError extends Error {
  constructor(public status: number, message: string) {
    super(message)
    this.name = 'ApiRequestError'
  }
}

async function request<T>(path: string, options: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new ApiRequestError(res.status, body.error || res.statusText)
  }
  return res.json()
}

export function signup(
  digitCode: number, hue: number, saturation: number, value: number, displayName = '',
): Promise<TokenPair> {
  return request('/api/auth/signup', {
    method: 'POST',
    body: JSON.stringify({ digit_code: digitCode, hue, saturation, value, display_name: displayName }),
  })
}

export function loginPicker(
  digitCode: number, hue: number, saturation: number, value: number,
): Promise<TokenPair> {
  return request('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ digit_code: digitCode, hue, saturation, value }),
  })
}

export function loginDirect(
  digitCode: number, hue: number, saturation: number, value: number,
): Promise<TokenPair> {
  return request('/api/auth/login/direct', {
    method: 'POST',
    body: JSON.stringify({ digit_code: digitCode, hue, saturation, value }),
  })
}

export function verify(token: string): Promise<VerifyResponse> {
  return request('/api/verify', {
    method: 'GET',
    headers: { Authorization: `Bearer ${token}` },
  })
}

export function updateProfile(
  token: string, displayName: string,
): Promise<{ display_name: string }> {
  return request('/api/user/profile', {
    method: 'PATCH',
    headers: { Authorization: `Bearer ${token}` },
    body: JSON.stringify({ display_name: displayName }),
  })
}
