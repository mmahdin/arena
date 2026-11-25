package ir.theia.keycloak.sms;

import java.util.Map;

/** Backed by Keycloak SingleUseObjectProvider; atomic across Keycloak nodes. */
public interface AtomicStore {
    boolean reserve(String key, long ttlSeconds);
    void put(String key, long ttlSeconds, Map<String, String> fields);
    Map<String, String> get(String key);
    Map<String, String> consume(String key);
}
