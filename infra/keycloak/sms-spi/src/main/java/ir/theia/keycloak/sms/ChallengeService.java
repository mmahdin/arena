package ir.theia.keycloak.sms;

import java.time.Clock;
import java.util.HashMap;
import java.util.Map;

/**
 * All rate-limit slots are putIfAbsent, not per-JVM counters. A resend consumes
 * a new allowance and atomically invalidates the prior challenge. Usage budget
 * is deliberately NOT refunded on transport failure (prevents abuse amplification).
 */
public final class ChallengeService {
    public static final int LIFETIME_SECONDS = 120;
    public static final int MAX_GUESSES = 5;
    public static final int RESEND_COOLDOWN_SECONDS = 60;
    public static final int PHONE_PER_HOUR = 5;
    public static final int IP_PER_HOUR = 20;
    public static final int GLOBAL_PER_HOUR = 200;
    private final AtomicStore store;
    private final SmsSender sender;
    private final OtpCrypto crypto;
    private final Clock clock;

    public ChallengeService(AtomicStore store, SmsSender sender, OtpCrypto crypto, Clock clock) {
        this.store = store;
        this.sender = sender;
        this.crypto = crypto;
        this.clock = clock;
    }

    public record Started(String id, String phone) {}
    public enum Verified { SUCCESS, INVALID, EXPIRED, EXHAUSTED }
    public static final class RateLimited extends Exception {
        public RateLimited() { super("OTP_RATE_LIMITED"); }
    }

    private long now() { return clock.millis(); }
    private String challengeKey(String id) { return "theia:sms:challenge:" + id; }
    private boolean reserveSlots(String namespace, int count, long bucket, long ttlSeconds) {
        String prefix = "theia:sms:limit:" + namespace + ":" + bucket + ":";
        for (int slot = 0; slot < count; slot++) {
            if (store.reserve(prefix + slot, ttlSeconds)) return true;
        }
        return false;
    }

    public Started send(String rawPhone, String peerIp, String previousId) throws RateLimited, SmsSender.SmsDeliveryException {
        String phone = PhoneNumber.canonical(rawPhone);
        String privatePhone = crypto.digest("phone", phone);
        String privateIp = crypto.digest("ip", peerIp == null ? "missing" : peerIp);
        long hour = now() / 3_600_000L;
        long remaining = 3601 - ((now() / 1000L) % 3600L);
        // The cooldown is checked first. Subsequent partial budget reservations
        // intentionally fail closed rather than refunding an attempted request.
        if (!store.reserve("theia:sms:cooldown:" + privatePhone, RESEND_COOLDOWN_SECONDS)
                || !reserveSlots("phone:" + privatePhone, PHONE_PER_HOUR, hour, remaining)
                || !reserveSlots("ip:" + privateIp, IP_PER_HOUR, hour, remaining)
                || !reserveSlots("global", GLOBAL_PER_HOUR, hour, remaining)) {
            throw new RateLimited();
        }
        // Invalidate the previous code even if delivery of the replacement fails.
        if (previousId != null && !previousId.isBlank()) store.consume(challengeKey(previousId));
        String code = crypto.code();
        String id = crypto.id();
        Map<String, String> fields = new HashMap<>();
        fields.put("phoneHash", privatePhone);
        fields.put("digest", crypto.digest(id, code));
        fields.put("expiresAt", Long.toString(now() + LIFETIME_SECONDS * 1000L));
        store.put(challengeKey(id), LIFETIME_SECONDS + 5, fields);
        try { sender.send(phone, code); }
        catch (SmsSender.SmsDeliveryException failure) {
            store.consume(challengeKey(id));
            throw failure;
        } catch (RuntimeException failure) {
            store.consume(challengeKey(id));
            throw new SmsSender.SmsDeliveryException();
        }
        return new Started(id, phone);
    }

    /** Abandon an in-flight OTP when the user elects to edit the phone number. */
    public void cancel(String id) {
        if (id != null && !id.isBlank()) store.consume(challengeKey(id));
    }

    public Verified verify(String id, String phone, String input) {
        if (id == null || id.isBlank() || phone == null) return Verified.INVALID;
        Map<String, String> challenge = store.get(challengeKey(id));
        if (challenge == null || !crypto.equalsDigest(crypto.digest("phone", phone), challenge.get("phoneHash"))) return Verified.INVALID;
        long expires;
        try { expires = Long.parseLong(challenge.get("expiresAt")); }
        catch (RuntimeException ex) { return Verified.INVALID; }
        if (now() >= expires) {
            store.consume(challengeKey(id));
            return Verified.EXPIRED;
        }
        String guess;
        try { guess = PhoneNumber.code(input); }
        catch (IllegalArgumentException ex) { guess = "invalid"; }
        // Reserve a distinct one-time guess slot BEFORE testing the answer.
        // This bounds concurrent guessing across all nodes, not just this JVM.
        int attempt = -1;
        for (int i = 0; i < MAX_GUESSES; i++) {
            if (store.reserve("theia:sms:guess:" + id + ":" + i, LIFETIME_SECONDS + 5)) {
                attempt = i;
                break;
            }
        }
        if (attempt < 0) { store.consume(challengeKey(id)); return Verified.EXHAUSTED; }
        boolean valid = crypto.equalsDigest(challenge.get("digest"), crypto.digest(id, guess));
        if (valid) {
            // Only one concurrent correct verifier may consume this challenge.
            return store.consume(challengeKey(id)) != null ? Verified.SUCCESS : Verified.INVALID;
        }
        if (attempt == MAX_GUESSES - 1) {
            store.consume(challengeKey(id));
            return Verified.EXHAUSTED;
        }
        return Verified.INVALID;
    }
}
