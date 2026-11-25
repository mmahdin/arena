package ir.theia.keycloak.sms;

import java.util.Map;
import org.keycloak.models.SingleUseObjectProvider;

final class KeycloakAtomicStore implements AtomicStore {
    private final SingleUseObjectProvider store;
    KeycloakAtomicStore(SingleUseObjectProvider store) { this.store = store; }
    @Override public boolean reserve(String key, long ttlSeconds) { return store.putIfAbsent(key, ttlSeconds); }
    @Override public void put(String key, long ttlSeconds, Map<String, String> fields) { store.put(key, ttlSeconds, fields); }
    @Override public Map<String, String> get(String key) { return store.get(key); }
    @Override public Map<String, String> consume(String key) { return store.remove(key); }
}
