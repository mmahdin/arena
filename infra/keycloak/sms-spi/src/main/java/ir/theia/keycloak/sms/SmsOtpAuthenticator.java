package ir.theia.keycloak.sms;

import java.time.Clock;
import jakarta.ws.rs.core.MultivaluedMap;
import jakarta.ws.rs.core.Response;
import org.keycloak.authentication.AuthenticationFlowContext;
import org.keycloak.authentication.AuthenticationFlowError;
import org.keycloak.authentication.Authenticator;
import org.keycloak.sessions.AuthenticationSessionModel;
import org.keycloak.models.KeycloakSession;
import org.keycloak.models.RealmModel;
import org.keycloak.models.UserModel;
import org.keycloak.forms.login.LoginFormsProvider;

/** One REQUIRED browser-flow execution. Neither password nor bypass path exists. */
public final class SmsOtpAuthenticator implements Authenticator {
    private static final String PHONE = "theia.sms.phone";
    private static final String CHALLENGE = "theia.sms.challenge";
    private static final String SENT_AT = "theia.sms.sent-at-ms";
    private final SmsConfiguration config;

    SmsOtpAuthenticator(SmsConfiguration config) { this.config = config; }

    private ChallengeService service(AuthenticationFlowContext ctx) {
        return new ChallengeService(new KeycloakAtomicStore(ctx.getSession().singleUseObjects()),
                config.sender(), config.crypto(), Clock.systemUTC());
    }

    private boolean allowed(AuthenticationFlowContext ctx) {
        return "theia-users".equals(ctx.getRealm().getName())
                && ctx.getAuthenticationSession() != null
                && ctx.getAuthenticationSession().getClient() != null
                && "theia-users-bff".equals(ctx.getAuthenticationSession().getClient().getClientId());
    }

    @Override public void authenticate(AuthenticationFlowContext ctx) {
        if (!allowed(ctx)) {
            ctx.failure(AuthenticationFlowError.ACCESS_DENIED);
            return;
        }
        AuthenticationSessionModel session = ctx.getAuthenticationSession();
        if (session.getAuthNote(CHALLENGE) != null && session.getAuthNote(PHONE) != null) showCode(ctx, null);
        else showPhone(ctx, null);
    }

    private void showPhone(AuthenticationFlowContext ctx, String error) {
        showPhone(ctx, error, null);
    }

    private void showPhone(AuthenticationFlowContext ctx, String error, String enteredPhone) {
        // Retain a corrected number in the UI, but never trust or log form content.
        String candidate = enteredPhone == null ? "" : enteredPhone;
        if (candidate.length() > 20) candidate = candidate.substring(0, 20);
        ctx.challenge(ctx.form().setAttribute("smsError", error == null ? "" : error)
                .setAttribute("smsEnteredPhone", candidate)
                .createForm("theia-sms-phone.ftl"));
    }

    static int resendRemainingSeconds(String sentAt, long now) {
        if (sentAt == null) return 0;
        try {
            long elapsed = Math.max(0, now - Long.parseLong(sentAt));
            long remainingMs = Math.max(0, ChallengeService.RESEND_COOLDOWN_SECONDS * 1000L - elapsed);
            return (int) Math.min(ChallengeService.RESEND_COOLDOWN_SECONDS, (remainingMs + 999) / 1000);
        } catch (NumberFormatException ex) {
            return 0;
        }
    }

    private Response codeForm(AuthenticationFlowContext ctx, String error) {
        AuthenticationSessionModel auth = ctx.getAuthenticationSession();
        String phone = auth.getAuthNote(PHONE);
        String masked = phone != null && phone.length() >= 4 ? "••••" + phone.substring(phone.length() - 4) : "";
        LoginFormsProvider form = ctx.form();
        return form.setAttribute("smsMaskedPhone", masked)
                .setAttribute("smsResendRemaining", resendRemainingSeconds(auth.getAuthNote(SENT_AT), System.currentTimeMillis()))
                .setAttribute("smsError", error == null ? "" : error)
                .createForm("theia-sms-code.ftl");
    }

    private void showCode(AuthenticationFlowContext ctx, String error) {
        ctx.challenge(codeForm(ctx, error));
    }

    private void clear(AuthenticationSessionModel session) {
        session.removeAuthNote(CHALLENGE);
        session.removeAuthNote(PHONE);
        session.removeAuthNote(SENT_AT);
    }

    @Override public void action(AuthenticationFlowContext ctx) {
        if (!allowed(ctx)) { ctx.failure(AuthenticationFlowError.ACCESS_DENIED); return; }
        MultivaluedMap<String, String> form = ctx.getHttpRequest().getDecodedFormParameters();
        String command = form.getFirst("smsAction");
        AuthenticationSessionModel auth = ctx.getAuthenticationSession();
        if ("request".equals(command)) {
            send(ctx, form.getFirst("phone"), auth.getAuthNote(CHALLENGE));
        } else if ("resend".equals(command) && auth.getAuthNote(CHALLENGE) != null) {
            send(ctx, auth.getAuthNote(PHONE), auth.getAuthNote(CHALLENGE));
        } else if ("change".equals(command) && auth.getAuthNote(CHALLENGE) != null) {
            changePhone(ctx);
        } else if ("verify".equals(command)) {
            verify(ctx, form.getFirst("code"));
        } else {
            showPhone(ctx, "درخواست نامعتبر است. دوباره شروع کنید.");
        }
    }

    private void changePhone(AuthenticationFlowContext ctx) {
        AuthenticationSessionModel auth = ctx.getAuthenticationSession();
        String previousPhone = auth.getAuthNote(PHONE);
        try {
            // Invalidate the old OTP, not just the browser's view of the challenge.
            service(ctx).cancel(auth.getAuthNote(CHALLENGE));
            clear(auth);
            showPhone(ctx, null, previousPhone == null ? null : PhoneNumber.domestic(previousPhone));
        } catch (RuntimeException ex) {
            // Never display another entry form if invalidation could not complete.
            ctx.failure(AuthenticationFlowError.INTERNAL_ERROR);
        }
    }

    private void send(AuthenticationFlowContext ctx, String phone, String previous) {
        AuthenticationSessionModel auth = ctx.getAuthenticationSession();
        String canonical;
        try { canonical = PhoneNumber.canonical(phone); }
        catch (IllegalArgumentException ex) { showPhone(ctx, "شماره موبایل ایران را درست وارد کنید.", phone); return; }
        // Never allow the browser to redirect a resend toward a different phone.
        if (previous != null && auth.getAuthNote(PHONE) != null && !canonical.equals(auth.getAuthNote(PHONE))) {
            showCode(ctx, "برای ارسال مجدد از همان شماره استفاده کنید.");
            return;
        }
        try {
            ChallengeService.Started started = service(ctx).send(canonical, ctx.getConnection().getRemoteAddr(), previous);
            auth.setAuthNote(PHONE, started.phone());
            auth.setAuthNote(CHALLENGE, started.id());
            auth.setAuthNote(SENT_AT, Long.toString(System.currentTimeMillis()));
            showCode(ctx, null);
        } catch (ChallengeService.RateLimited ex) {
            if (previous != null) showCode(ctx, "برای دریافت کد جدید کمی صبر کنید.");
            else showPhone(ctx, "برای دریافت کد جدید کمی صبر کنید.", phone);
        } catch (SmsSender.SmsDeliveryException ex) {
            clear(auth);
            showPhone(ctx, "ارسال کد ممکن نیست. بعداً دوباره تلاش کنید.");
        } catch (RuntimeException ex) {
            // Fail closed on cache/provider errors; never return a valid flow on error.
            clear(auth);
            ctx.failure(AuthenticationFlowError.INTERNAL_ERROR);
        }
    }

    private void verify(AuthenticationFlowContext ctx, String input) {
        AuthenticationSessionModel auth = ctx.getAuthenticationSession();
        String challengeId = auth.getAuthNote(CHALLENGE);
        String phone = auth.getAuthNote(PHONE);
        if (challengeId == null || phone == null) { showPhone(ctx, "کد جدید درخواست کنید."); return; }
        ChallengeService.Verified result;
        try { result = service(ctx).verify(challengeId, phone, input); }
        catch (RuntimeException ex) { ctx.failure(AuthenticationFlowError.INTERNAL_ERROR); return; }
        if (result != ChallengeService.Verified.SUCCESS) {
            if (result == ChallengeService.Verified.EXHAUSTED || result == ChallengeService.Verified.EXPIRED) {
                clear(auth);
                showPhone(ctx, "کد منقضی شد یا فرصت‌های ورود تمام شد. کد جدید درخواست کنید.");
            } else {
                ctx.failureChallenge(AuthenticationFlowError.INVALID_CREDENTIALS,
                        codeForm(ctx, "کد معتبر نیست. دوباره تلاش کنید."));
            }
            return;
        }
        clear(auth);
        try {
            // Keycloak owns identity creation; Go AUTH-06 later maps (issuer,sub)
            // to the immutable Theia internal UUID. Usernames are canonical phones.
            UserModel user = ctx.getSession().users().getUserByUsername(ctx.getRealm(), phone);
            if (user == null) {
                user = ctx.getSession().users().addUser(ctx.getRealm(), phone);
                user.setEnabled(true);
                user.setSingleAttribute("phone_number", phone);
                user.setSingleAttribute("phone_number_verified", "true");
            }
            if (!user.isEnabled() || !phone.equals(user.getUsername())) {
                ctx.failure(AuthenticationFlowError.ACCESS_DENIED);
                return;
            }
            ctx.setUser(user);
            // Only the successful OTP branch may set the signed token's
            // verification evidence; the BFF denies tokens lacking this note.
            auth.setUserSessionNote("theia_sms_verified", "sms-otp-v1");
            ctx.success();
        } catch (RuntimeException ex) {
            // Concurrent first-user creation: storage uniqueness must reject
            // duplicates. Retry a new OTP transaction; never bypass verification.
            ctx.failure(AuthenticationFlowError.INTERNAL_ERROR);
        }
    }

    @Override public boolean requiresUser() { return false; }
    @Override public boolean configuredFor(KeycloakSession session, RealmModel realm, UserModel user) { return true; }
    @Override public void setRequiredActions(KeycloakSession session, RealmModel realm, UserModel user) {}
    @Override public void close() {}
}
