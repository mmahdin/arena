package ir.theia.keycloak.sms;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.SecureRandom;
import java.util.Base64;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;

/** No plaintext OTP or identifying phone is persisted in the single-use store. */
public final class OtpCrypto {
    private final byte[] key;
    private final SecureRandom random = new SecureRandom();

    public OtpCrypto(String keyBase64) {
        try { this.key = Base64.getDecoder().decode(keyBase64); }
        catch (RuntimeException ex) { throw new IllegalArgumentException("OTP HMAC key invalid"); }
        if (key.length < 32) throw new IllegalArgumentException("OTP HMAC key too short");
    }

    public String code() { return String.format(java.util.Locale.ROOT, "%06d", random.nextInt(1_000_000)); }
    public String id() {
        byte[] bytes = new byte[24];
        random.nextBytes(bytes);
        return Base64.getUrlEncoder().withoutPadding().encodeToString(bytes);
    }
    public String digest(String purpose, String value) {
        try {
            Mac hmac = Mac.getInstance("HmacSHA256");
            hmac.init(new SecretKeySpec(key, "HmacSHA256"));
            byte[] hash = hmac.doFinal((purpose + "\u0000" + value).getBytes(StandardCharsets.UTF_8));
            return Base64.getUrlEncoder().withoutPadding().encodeToString(hash);
        } catch (java.security.GeneralSecurityException ex) { throw new IllegalStateException("OTP crypto unavailable"); }
    }
    public boolean equalsDigest(String a, String b) {
        return a != null && b != null && MessageDigest.isEqual(a.getBytes(StandardCharsets.US_ASCII), b.getBytes(StandardCharsets.US_ASCII));
    }
}
