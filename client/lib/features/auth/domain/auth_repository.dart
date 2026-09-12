/// Signs up and signs in against the auth API. Signing up no longer issues a
/// token: it starts a pending attempt identified by a ticket, and the token is
/// issued once the mailed verification code is accepted.
abstract interface class AuthRepository {
  /// `POST /signup` → registers a pending attempt, mails a verification code
  /// to [email], and returns the ticket that identifies the attempt.
  Future<String> signUp({required String email, required String password});

  /// `POST /signup/verify-email` → finishes the attempt behind [ticket] and
  /// returns the issued token.
  Future<String> verifySignUpEmail({
    required String ticket,
    required String code,
    required String device,
  });

  /// `POST /signup/resend-verification` → mails another code for the attempt
  /// behind [ticket] and returns the replacement ticket. The old ticket stops
  /// being usable, so the caller must keep the returned one instead.
  Future<String> resendSignUpVerification({required String ticket});

  /// `POST /signin` → authenticates and returns the issued token.
  Future<String> signIn({
    required String email,
    required String password,
    required String device,
  });

  /// Reads the persisted bearer token, or `null` when signed out.
  Future<String?> readAuthToken();

  /// Persists the bearer token issued by [verifySignUpEmail] / [signIn] so it
  /// survives app launches.
  Future<void> writeAuthToken(String token);

  /// `POST /signout` → revokes [token] on the server.
  Future<void> signOut(String token);

  /// Removes the persisted bearer token.
  Future<void> clearAuthToken();
}
