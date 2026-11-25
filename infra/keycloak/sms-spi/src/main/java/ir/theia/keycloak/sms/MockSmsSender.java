package ir.theia.keycloak.sms;

import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.attribute.PosixFilePermissions;
import java.nio.file.StandardOpenOption;
import java.nio.charset.StandardCharsets;
import java.io.IOException;
import java.util.UUID;
import java.util.logging.Logger;

/** Test-only: outbox files are local to the Keycloak container, NOT a public API. */
public final class MockSmsSender implements SmsSender {
    private static final Logger LOG = Logger.getLogger(MockSmsSender.class.getName());
    private final Path directory;
    public MockSmsSender(Path directory) { this.directory = directory; }

    @Override public void send(String phone, String code) throws SmsDeliveryException {
        try {
            if (!Files.exists(directory)) {
                Files.createDirectory(directory, PosixFilePermissions.asFileAttribute(
                        PosixFilePermissions.fromString("rwx------")));
            }
            // Reject symlinked directories. Restrict permissions before writing secrets.
            if (Files.isSymbolicLink(directory) || !Files.isDirectory(directory)) throw new IOException("Unsafe mock outbox");
            Files.setPosixFilePermissions(directory, PosixFilePermissions.fromString("rwx------"));
            Path file = directory.resolve(UUID.randomUUID() + ".otp");
            Files.createFile(file, PosixFilePermissions.asFileAttribute(
                    PosixFilePermissions.fromString("rw-------")));
            Files.writeString(file, phone + " " + code + "\n", StandardCharsets.US_ASCII,
                    StandardOpenOption.WRITE, StandardOpenOption.TRUNCATE_EXISTING);
            LOG.info("Local mock SMS OTP: " + code);
        } catch (IOException | UnsupportedOperationException ex) {
            throw new SmsDeliveryException();
        }
    }
}
