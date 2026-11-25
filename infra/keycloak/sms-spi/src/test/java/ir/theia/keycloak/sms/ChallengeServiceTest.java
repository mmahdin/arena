package ir.theia.keycloak.sms;

import org.junit.jupiter.api.Test;
import java.security.SecureRandom;
import java.time.Clock;
import java.time.Instant;
import java.time.ZoneOffset;
import java.util.Base64;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicInteger;
import static org.junit.jupiter.api.Assertions.*;

class ChallengeServiceTest {
    static class MemoryStore implements AtomicStore {
        final ConcurrentHashMap<String, Map<String, String>> data = new ConcurrentHashMap<>();
        @Override public boolean reserve(String key, long ttl) { return data.putIfAbsent(key, Map.of()) == null; }
        @Override public void put(String key, long ttl, Map<String, String> fields) { data.put(key, Map.copyOf(fields)); }
        @Override public Map<String, String> get(String key) { return data.get(key); }
        @Override public Map<String, String> consume(String key) { return data.remove(key); }
    }
    static class Sender implements SmsSender {
        String lastPhone, lastCode;
        boolean fail;
        @Override public void send(String phone, String code) throws SmsDeliveryException {
            if (fail) throw new SmsDeliveryException();
            lastPhone = phone;
            lastCode = code;
        }
    }
    private final byte[] key = new byte[32];
    private final OtpCrypto crypto = crypto();
    private static OtpCrypto crypto() {
        byte[] bytes = new byte[32]; new SecureRandom().nextBytes(bytes);
        return new OtpCrypto(Base64.getEncoder().encodeToString(bytes));
    }
    private ChallengeService service(MemoryStore store, Sender sender, Instant time) {
        return new ChallengeService(store, sender, crypto, Clock.fixed(time, ZoneOffset.UTC));
    }
    private final Instant time = Instant.parse("2026-11-19T14:00:00Z");

    @Test void acceptsPersianArabicAndE164ButRejectsLandlines() {
        assertEquals("+989123456789", PhoneNumber.canonical("۰۹۱۲۳۴۵۶۷۸۹"));
        assertEquals("+989123456789", PhoneNumber.canonical("+98 912 345 6789"));
        assertEquals("+989123456789", PhoneNumber.canonical("00989123456789"));
        assertEquals("09123456789", PhoneNumber.domestic("+989123456789"));
        assertEquals("123456", PhoneNumber.code("١٢٣٤٥٦"));
        assertThrows(IllegalArgumentException.class, () -> PhoneNumber.canonical("02122334455"));
        assertThrows(IllegalArgumentException.class, () -> PhoneNumber.canonical("+447123456789"));
        assertThrows(IllegalArgumentException.class, () -> PhoneNumber.code("12x456"));
    }

    @Test void succeedsOnceWithCorrectCodeAndDoesNotStorePlaintextOtpOrPhone() throws Exception {
        MemoryStore store = new MemoryStore(); Sender sender = new Sender();
        var svc = service(store, sender, time);
        var started = svc.send("09123456789", "127.0.0.1", null);
        assertEquals("+989123456789", sender.lastPhone);
        assertTrue(sender.lastCode.matches("[0-9]{6}"));
        var serialized = store.data.toString();
        assertFalse(serialized.contains(sender.lastCode));
        assertFalse(serialized.contains(sender.lastPhone));
        assertEquals(ChallengeService.Verified.SUCCESS, svc.verify(started.id(), started.phone(), sender.lastCode));
        assertEquals(ChallengeService.Verified.INVALID, svc.verify(started.id(), started.phone(), sender.lastCode));
    }

    @Test void enforcesExpiryAndFiveGuesses() throws Exception {
        MemoryStore store = new MemoryStore(); Sender sender = new Sender();
        var start = service(store, sender, time).send("09123456789", "127.0.0.1", null);
        var fresh = service(store, sender, time);
        for (int i = 0; i < 4; i++) assertEquals(ChallengeService.Verified.INVALID, fresh.verify(start.id(), start.phone(), "999999".equals(sender.lastCode) ? "888888" : "999999"));
        assertEquals(ChallengeService.Verified.EXHAUSTED, fresh.verify(start.id(), start.phone(), "999999".equals(sender.lastCode) ? "888888" : "999999"));
        assertEquals(ChallengeService.Verified.INVALID, fresh.verify(start.id(), start.phone(), sender.lastCode));
        // Keycloak would expire this cooldown entry; our in-memory fake has no TTL.
        store.data.keySet().removeIf(k -> k.startsWith("theia:sms:cooldown:"));
        var start2 = service(store, sender, time.plusSeconds(61)).send("09123456789", "127.0.0.1", null);
        assertEquals(ChallengeService.Verified.EXPIRED,
                service(store, sender, time.plusSeconds(61 + 121)).verify(start2.id(), start2.phone(), sender.lastCode));
    }

    @Test void cooldownAndResendInvalidatePrevious() throws Exception {
        MemoryStore store = new MemoryStore(); Sender sender = new Sender();
        var old = service(store, sender, time).send("09123456789", "127.0.0.1", null);
        assertThrows(ChallengeService.RateLimited.class, () -> service(store, sender, time).send(old.phone(), "127.0.0.1", old.id()));
        // MemoryStore models atomicity only. Expired cache entries would be purged by Keycloak;
        // remove the cooldown key to simulate 61 seconds of time passing.
        store.data.keySet().removeIf(k -> k.startsWith("theia:sms:cooldown:"));
        var newer = service(store, sender, time.plusSeconds(61)).send(old.phone(), "127.0.0.1", old.id());
        assertEquals(ChallengeService.Verified.INVALID, service(store, sender, time.plusSeconds(61)).verify(old.id(), old.phone(), sender.lastCode));
        assertEquals(ChallengeService.Verified.SUCCESS, service(store, sender, time.plusSeconds(61)).verify(newer.id(), newer.phone(), sender.lastCode));
    }

    @Test void changingPhoneInvalidatesTheOldChallengeWithoutResettingRateLimit() throws Exception {
        MemoryStore store = new MemoryStore(); Sender sender = new Sender();
        var svc = service(store, sender, time);
        var first = svc.send("09123456789", "127.0.0.1", null);
        String oldCode = sender.lastCode;
        svc.cancel(first.id());
        assertEquals(ChallengeService.Verified.INVALID, svc.verify(first.id(), first.phone(), oldCode));
        assertThrows(ChallengeService.RateLimited.class,
                () -> svc.send(first.phone(), "127.0.0.1", null));
        var other = svc.send("09123456788", "127.0.0.1", null);
        assertEquals(ChallengeService.Verified.SUCCESS,
                svc.verify(other.id(), other.phone(), sender.lastCode));
    }

    @Test void providerFailureConsumesBudgetAndInvalidatesChallenge() {
        MemoryStore store = new MemoryStore(); Sender sender = new Sender(); sender.fail = true;
        var svc = service(store, sender, time);
        assertThrows(SmsSender.SmsDeliveryException.class, () -> svc.send("09123456789", "127.0.0.1", null));
        assertFalse(store.data.keySet().stream().anyMatch(k -> k.startsWith("theia:sms:challenge:")));
        assertThrows(ChallengeService.RateLimited.class, () -> svc.send("09123456789", "127.0.0.1", null));
    }

    @Test void limitsPerPhoneAndPerIpAndGlobalBeforeProviderSend() throws Exception {
        MemoryStore store = new MemoryStore(); Sender sender = new Sender();
        var svc = service(store, sender, time);
        for (int i = 0; i < ChallengeService.IP_PER_HOUR; i++) {
            String phone = String.format("0912000%04d", i);
            svc.send(phone, "1.2.3.4", null);
        }
        assertThrows(ChallengeService.RateLimited.class, () -> svc.send("09129999999", "1.2.3.4", null));
    }

    @Test void phoneSendBudgetIsBoundedAcrossDifferentPeers() throws Exception {
        MemoryStore store = new MemoryStore(); Sender sender = new Sender();
        var svc = service(store, sender, time);
        for (int i = 0; i < ChallengeService.PHONE_PER_HOUR; i++) {
            // Simulate Keycloak expiring the 60-second cooldown after each send.
            store.data.keySet().removeIf(k -> k.startsWith("theia:sms:cooldown:"));
            svc.send("09123456789", "10.0.0." + (i + 1), null);
        }
        store.data.keySet().removeIf(k -> k.startsWith("theia:sms:cooldown:"));
        assertThrows(ChallengeService.RateLimited.class,
                () -> svc.send("09123456789", "10.0.0.50", null));
    }

    @Test void globalCostBudgetStopsDistinctPhoneAndIpRequests() throws Exception {
        MemoryStore store = new MemoryStore(); Sender sender = new Sender();
        var svc = service(store, sender, time);
        for (int i = 0; i < ChallengeService.GLOBAL_PER_HOUR; i++) {
            svc.send(String.format("0912%07d", i), "10.0.1." + i, null);
        }
        assertThrows(ChallengeService.RateLimited.class,
                () -> svc.send("09129999999", "10.0.2.1", null));
    }

    @Test void concurrentCorrectVerificationConsumesOnlyOnce() throws Exception {
        MemoryStore store = new MemoryStore(); Sender sender = new Sender();
        var svc = service(store, sender, time);
        var started = svc.send("09123456789", "127.0.0.1", null);
        var success = new AtomicInteger();
        try (var executor = java.util.concurrent.Executors.newFixedThreadPool(8)) {
            var tasks = new java.util.ArrayList<java.util.concurrent.Future<?>>();
            for (int i = 0; i < 8; i++) tasks.add(executor.submit(() -> {
                if (svc.verify(started.id(), started.phone(), sender.lastCode) == ChallengeService.Verified.SUCCESS) success.incrementAndGet();
            }));
            for (var task : tasks) task.get();
        }
        assertEquals(1, success.get());
    }
}
