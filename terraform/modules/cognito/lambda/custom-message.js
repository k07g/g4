// Cognito "Custom message" Lambda trigger.
//
// Only handles CustomMessage_ForgotPassword: it rewrites the outgoing email
// to a link (https://<frontend>/login?email=...&code=...) instead of a bare
// verification code, so a user who clicks it lands directly on the reset
// form with the email/code already filled in and only has to type a new
// password. Every other trigger source (sign-up confirmation, resend code,
// etc.) is returned unmodified so those flows keep using Cognito's default
// code-only message, matching the existing /auth/confirm UX.
//
// This is plain Node.js rather than Go (unlike the rest of this repo)
// because it has no dependency on the app's business logic and Terraform
// can zip a single JS file directly with the archive provider - no build
// step, artifact upload, or separate deploy pipeline needed.
exports.handler = async (event) => {
  if (event.triggerSource !== "CustomMessage_ForgotPassword") {
    return event;
  }

  const baseUrl = process.env.FRONTEND_BASE_URL;
  const email = encodeURIComponent(event.request.userAttributes.email);
  // codeParameter (e.g. "{####}") must appear byte-for-byte in the message:
  // Cognito finds this exact placeholder string and replaces it with the
  // real code before sending. URL-encoding it here would change the string
  // Cognito is looking for, so the substitution would silently never happen
  // and the literal placeholder would be emailed to the user.
  const link = `${baseUrl}/login?email=${email}&code=${event.request.codeParameter}`;

  // Cognito accepts HTML in emailMessage (DEVELOPER sending account only).
  event.response.emailSubject = "パスワードの再設定";
  event.response.emailMessage = [
    "<p>以下のリンクをクリックして、パスワードを再設定してください。</p>",
    `<p><a href="${link}">${link}</a></p>`,
    "<p>このメールに心当たりがない場合は、操作の必要はありません。</p>",
  ].join("");

  return event;
};
