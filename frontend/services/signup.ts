import { POST } from '$libs/ui-runtime.svelte';

export interface ISignUpRequestResult {
  RequestID: number
  // False when the resend cooldown blocked delivery; the earlier code is still the valid one.
  Sent: boolean
  Verified: boolean
  // SUnixTime of the last delivery. formatTime() decodes this format directly.
  SentAt: number
  // Seconds left before another email may be requested.
  RetryAfterSeconds: number
}

export interface ISignUpCompanyForm {
  CompanyName: string
  Address: string
  RUC: string
  // The administrator's login is always "admin"; only their display name is asked for, optionally.
  AdminFirstName: string
  AdminLastName: string
  AdminPassword: string
}

// silentError: "this address already owns a company" and the like are expected answers here, so
// the wizard shows them inline under its title instead of as a red toast.
export const requestSignUpCode = (email: string): Promise<ISignUpRequestResult> => {
  return POST({
    data: { Email: email },
    route: 'p-signup-request',
    apiName: 'MAIN',
    silentError: true,
    headers: { 'Content-Type': 'application/json' },
  })
}

// silentError: an expired or already-used code is an expected outcome of this step, so the wizard
// shows it inline in the form instead of as a red toast floating over the page.
export const verifySignUpCode = (requestID: number, code: string): Promise<{ RequestID: number, Email: string }> => {
  return POST({
    data: { RequestID: requestID, Code: code },
    route: 'p-signup-verify',
    apiName: 'MAIN',
    silentError: true,
    headers: { 'Content-Type': 'application/json' },
  })
}

// Returns the same payload as p-user-login, so the caller finishes authenticated and the last
// registration step can use the normal private endpoints.
export const createSignUpCompany = (
  requestID: number, code: string, form: ISignUpCompanyForm, cipherKey: string,
) => {
  return POST({
    data: { RequestID: requestID, Code: code, ...form, CipherKey: cipherKey },
    route: 'p-signup-company',
    apiName: 'MAIN',
    headers: { 'Content-Type': 'application/json' },
  })
}
