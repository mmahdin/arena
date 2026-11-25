package ir.theia.keycloak.sms;

import java.io.IOException;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.regex.Pattern;

/** HTTPS-only Kavenegar verify/lookup transport. OTP validation remains in Keycloak. */
public final class KavenegarSmsSender implements SmsSender {
    private static final Pattern KEY = Pattern.compile("[A-Za-z0-9]{20,200}");
    private static final Pattern TEMPLATE = Pattern.compile("[A-Za-z0-9_-]{1,64}");
    private static final Pattern SUCCESS = Pattern.compile("\\\"status\\\"\\s*:\\s*200\\b");
    private final HttpClient client;
    private final URI endpoint;
    private final String template;

    public KavenegarSmsSender(String apiKey, String template) {
        if (apiKey == null || !KEY.matcher(apiKey).matches() || template == null || !TEMPLATE.matcher(template).matches()) {
            throw new IllegalArgumentException("SMS provider configuration invalid");
        }
        this.template = template;
        this.endpoint = URI.create("https://api.kavenegar.com/v1/" + apiKey + "/verify/lookup.json");
        this.client = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(3))
                .followRedirects(HttpClient.Redirect.NEVER).build();
    }

    private static String encode(String s) { return URLEncoder.encode(s, StandardCharsets.UTF_8); }

    @Override public void send(String phone, String code) throws SmsDeliveryException {
        String body = "receptor=" + encode(PhoneNumber.domestic(phone))
                + "&token=" + encode(code) + "&template=" + encode(template);
        HttpRequest request = HttpRequest.newBuilder(endpoint)
                .timeout(Duration.ofSeconds(6))
                .header("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
                .POST(HttpRequest.BodyPublishers.ofString(body)).build();
        try {
            HttpResponse<String> result = client.send(request, HttpResponse.BodyHandlers.ofString());
            if (result.statusCode() != 200 || !SUCCESS.matcher(result.body()).find()) throw new SmsDeliveryException();
        } catch (IOException ex) {
            // Never log or expose HttpRequest.toString(): URL contains the API key.
            throw new SmsDeliveryException();
        } catch (InterruptedException ex) {
            Thread.currentThread().interrupt();
            throw new SmsDeliveryException();
        }
    }
}
