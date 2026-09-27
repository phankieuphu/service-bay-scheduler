import { ApiError } from '../api/http'

// describeAuthError turns an identity-service failure into a message for the
// sign-in/sign-up forms.
export function describeAuthError(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.status) {
      case 401:
        return 'Incorrect email or password.'
      case 403:
        return err.message.includes('role')
          ? 'That role can’t be chosen at sign-up.'
          : 'This account is inactive or locked. Contact an administrator.'
      case 409:
        return 'An account with this email already exists. Try signing in instead.'
    }
    return err.message
  }
  if (err instanceof TypeError) {
    return 'Can’t reach the identity service. Check that it’s running and try again.'
  }
  return err instanceof Error ? err.message : 'Something went wrong.'
}
