interface ErrorResponse {
  message?: string
  errors?: Record<string, string[]>
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
    readonly fields: ErrorResponse['errors'] = {},
  ) {
    super(message)
  }
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api${path}`, {
    cache: 'no-store',
    signal: AbortSignal.timeout(20_000),
    ...init,
  })

  if (!response.ok) {
    const body: ErrorResponse | null = await response.json().catch(() => null)
    throw new ApiError(response.status, body?.message || 'Запрос не выполнен', body?.errors)
  }

  return response.status === 204 ? (undefined as T) : response.json()
}

export function retryMessage(error: unknown): string {
  if (error instanceof DOMException && error.name === 'TimeoutError') {
    return 'Сервер не ответил вовремя. Попробуйте ещё раз.'
  }
  if (error instanceof TypeError) return 'Проверьте интернет-соединение и попробуйте ещё раз.'
  return 'Попробуйте ещё раз через несколько секунд.'
}
