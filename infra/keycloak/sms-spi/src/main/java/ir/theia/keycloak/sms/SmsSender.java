package ir.theia.keycloak.sms;

public interface SmsSender {
    /** Never put phone, code, or provider credentials in an exception or log. */
    void send(String canonicalPhone, String sixDigitCode) throws SmsDeliveryException;

    final class SmsDeliveryException extends Exception {
        public SmsDeliveryException() { super("SMS_DELIVERY_UNAVAILABLE"); }
    }
}
