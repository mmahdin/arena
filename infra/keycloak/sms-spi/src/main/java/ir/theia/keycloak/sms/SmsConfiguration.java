package ir.theia.keycloak.sms;

import java.nio.file.Path;
import java.util.Locale;

/** Factory-level fail-closed config. Mock can never be selected outside local. */
public final class SmsConfiguration {
    private final SmsSender sender;
    private final OtpCrypto crypto;

    private SmsConfiguration(SmsSender sender, OtpCrypto crypto) {
        this.sender = sender;
        this.crypto = crypto;
    }
    public SmsSender sender() { return sender; }
    public OtpCrypto crypto() { return crypto; }

    public static SmsConfiguration fromEnvironment() {
        String environment = System.getenv("THEIA_SMS_ENV");
        String mode = "local".equals(environment) ? "mock"
                : System.getenv().getOrDefault("THEIA_SMS_TRANSPORT", "disabled").toLowerCase(Locale.ROOT);
        OtpCrypto crypto = new OtpCrypto(System.getenv("THEIA_SMS_HMAC_KEY_BASE64"));
        SmsSender sender;
        if ("mock".equals(mode)) {
            if (!"local".equals(System.getenv("THEIA_SMS_ENV"))
                    || !"http://auth.localhost:8180".equals(System.getenv("KC_HOSTNAME"))) {
                throw new IllegalStateException("Mock SMS requires loopback-only local Keycloak");
            }
            String outbox = System.getenv("THEIA_SMS_MOCK_OUTBOX_DIR");
            if (outbox == null || !Path.of(outbox).isAbsolute()) throw new IllegalStateException("Mock outbox must be absolute");
            if (!Path.of(outbox).normalize().startsWith(Path.of("/tmp/theia-sms-outbox"))) {
                throw new IllegalStateException("Mock outbox must be under private test path");
            }
            sender = new MockSmsSender(Path.of(outbox));
        } else if ("kavenegar".equals(mode)) {
            if (!"stage".equals(environment) && !"production".equals(environment)) throw new IllegalStateException("SMS environment invalid");
            sender = new KavenegarSmsSender(System.getenv("THEIA_SMS_KAVENEGAR_API_KEY"), System.getenv("THEIA_SMS_KAVENEGAR_TEMPLATE"));
        } else {
            throw new IllegalStateException("SMS transport not configured");
        }
        return new SmsConfiguration(sender, crypto);
    }
}
