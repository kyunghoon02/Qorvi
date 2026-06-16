export type WalletProviderErrorCode =
  | "missing_api_key"
  | "rate_limited"
  | "invalid_api_key"
  | "provider_unavailable"
  | "provider_timeout"
  | "provider_error"
  | "invalid_response"
  | "quota_exceeded"
  | "provider_budget_exceeded"
  | "analysis_required";

export class WalletProviderError extends Error {
  readonly code: WalletProviderErrorCode;
  readonly status: number;

  constructor({
    code,
    message,
    status,
  }: {
    code: WalletProviderErrorCode;
    message: string;
    status?: number;
  }) {
    super(message);
    this.name = "WalletProviderError";
    this.code = code;
    this.status = status ?? statusForProviderCode(code);
  }
}

export function isWalletProviderError(
  error: unknown,
): error is WalletProviderError {
  return error instanceof WalletProviderError;
}

function statusForProviderCode(code: WalletProviderErrorCode): number {
  switch (code) {
    case "missing_api_key":
    case "invalid_api_key":
      return 401;
    case "rate_limited":
    case "quota_exceeded":
    case "provider_budget_exceeded":
      return 429;
    case "analysis_required":
      return 409;
    case "provider_unavailable":
      return 503;
    case "provider_timeout":
      return 504;
    case "invalid_response":
    case "provider_error":
      return 502;
  }
}
