package ir.theia.keycloak.sms;

import java.util.List;
import org.keycloak.Config;
import org.keycloak.authentication.Authenticator;
import org.keycloak.authentication.AuthenticatorFactory;
import org.keycloak.models.AuthenticationExecutionModel;
import org.keycloak.models.KeycloakSession;
import org.keycloak.models.KeycloakSessionFactory;
import org.keycloak.provider.ProviderConfigProperty;

public final class SmsOtpAuthenticatorFactory implements AuthenticatorFactory {
    public static final String ID = "theia-sms-otp";
    private SmsConfiguration configuration;

    @Override public Authenticator create(KeycloakSession session) {
        if (configuration == null) throw new IllegalStateException("SMS OTP configuration unavailable");
        return new SmsOtpAuthenticator(configuration);
    }
    @Override public String getId() { return ID; }
    @Override public String getDisplayType() { return "Theia SMS OTP (passwordless)"; }
    @Override public String getHelpText() { return "Single required SMS-only browser login for theia-users-bff"; }
    @Override public String getReferenceCategory() { return "sms"; }
    @Override public boolean isConfigurable() { return false; }
    @Override public boolean isUserSetupAllowed() { return false; }
    @Override public AuthenticationExecutionModel.Requirement[] getRequirementChoices() {
        return new AuthenticationExecutionModel.Requirement[]{AuthenticationExecutionModel.Requirement.REQUIRED};
    }
    @Override public List<ProviderConfigProperty> getConfigProperties() { return List.of(); }
    @Override public void init(Config.Scope scope) { configuration = SmsConfiguration.fromEnvironment(); }
    @Override public void postInit(KeycloakSessionFactory factory) {}
    @Override public void close() {}
}
