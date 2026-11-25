package ir.theia.keycloak.sms;

import java.util.regex.Pattern;

/** Accepted identifiers are Iranian mobile numbers only, canonicalized to E.164. */
public final class PhoneNumber {
    private static final Pattern DOMESTIC = Pattern.compile("09[0-9]{9}");
    private PhoneNumber() {}

    static String digits(String input) {
        if (input == null) return "";
        StringBuilder out = new StringBuilder(input.length());
        for (int i = 0; i < input.length(); i++) {
            char ch = input.charAt(i);
            if (ch >= '\u06f0' && ch <= '\u06f9') ch = (char) ('0' + ch - '\u06f0');
            else if (ch >= '\u0660' && ch <= '\u0669') ch = (char) ('0' + ch - '\u0660');
            if (ch == ' ' || ch == '-' || ch == '(' || ch == ')' || ch == '\u200c') continue;
            out.append(ch);
        }
        return out.toString();
    }

    public static String canonical(String raw) {
        String input = digits(raw);
        if (input.startsWith("0098")) input = input.substring(4);
        else if (input.startsWith("+98")) input = input.substring(3);
        else if (input.startsWith("98")) input = input.substring(2);
        else if (input.startsWith("+")) throw new IllegalArgumentException("INVALID_PHONE");
        if (input.startsWith("9") && input.length() == 10) input = "0" + input;
        if (!DOMESTIC.matcher(input).matches()) throw new IllegalArgumentException("INVALID_PHONE");
        return "+98" + input.substring(1);
    }

    public static String domestic(String canonical) {
        if (canonical == null || !canonical.matches("\\+989[0-9]{9}")) throw new IllegalArgumentException("INVALID_PHONE");
        return "0" + canonical.substring(3);
    }

    public static String code(String raw) {
        String value = digits(raw);
        if (!value.matches("[0-9]{6}")) throw new IllegalArgumentException("INVALID_CODE");
        return value;
    }
}
