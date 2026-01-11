// test789d1 : USAG Lib basic io Encoder

import java.util.ArrayList;
import java.util.Base64;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class Encoder {
    private final List<Character> CHARS = new ArrayList<>();
    private final Map<Character, Integer> REV_MAP = new HashMap<>();
    private final int THRESHOLD = 32164;
    private final char ESCAPE_CHAR = '.';

    public Encoder() {
        // Korean letters (U+AC00-U+D7A3): 11172
        for (int i = 0; i < 11172; i++) {
            this.CHARS.add((char) (0xAC00 + i));
        }
        // CJK letters (U+4E00-U+9FFF): 20992
        for (int i = 0; i < 20992; i++) {
            this.CHARS.add((char) (0x4E00 + i));
        }
        for (int i = 0; i < this.CHARS.size(); i++) {
            this.REV_MAP.put(this.CHARS.get(i), i);
        }
    }

    public String encode(byte[] data, boolean isBase64) {
        if (isBase64 && data.length == 0) return "";
        if (isBase64) {
            return Base64.getEncoder().encodeToString(data);
        }
        return encodeUnicode(data);
    }

    public byte[] decode(String data) {
        String cleaned = data.replace("\r", "").replace("\n", "").replace(" ", "");
        if (cleaned.isEmpty()) return new byte[0];

        if (cleaned.charAt(0) < 128 && cleaned.charAt(0) != this.ESCAPE_CHAR) {
            return Base64.getDecoder().decode(cleaned);
        } else {
            return decodeUnicode(cleaned);
        }
    }

    private String encodeUnicode(byte[] data) {
        StringBuilder result = new StringBuilder();
        int acc = 0;
        int bits = 0;

        for (byte b : data) {
            acc = (acc << 8) | (b & 0xFF); // Java byte is signed, need mask
            bits += 8;
            while (bits >= 15) {
                bits -= 15;
                int val = (acc >> bits) & 0x7FFF;
                acc = (bits == 0) ? 0 : acc & ((1 << bits) - 1);
                
                if (val < this.THRESHOLD) {
                    result.append(this.CHARS.get(val));
                } else {
                    int offset = val - this.THRESHOLD;
                    result.append(this.ESCAPE_CHAR).append(this.CHARS.get(offset));
                }
            }
        }

        // pad leftover
        int val = ((acc << 1) | 1) << (14 - bits);
        if (val < this.THRESHOLD) {
            result.append(this.CHARS.get(val));
        } else {
            int offset = val - this.THRESHOLD;
            result.append(this.ESCAPE_CHAR).append(this.CHARS.get(offset));
        }
        return result.toString();
    }

    private byte[] decodeUnicode(String data) {
        java.io.ByteArrayOutputStream ba = new java.io.ByteArrayOutputStream();
        int acc = 0;
        int bits = 0;
        int i = 0;
        int n = data.length();

        while (i < n) {
            char c = data.charAt(i);
            i++;
            int val = 0;

            if (c == this.ESCAPE_CHAR) {
                if (i >= n) throw new RuntimeException("invalid escape");
                char nextChar = data.charAt(i);
                i++;
                val = this.REV_MAP.getOrDefault(nextChar, 0) + this.THRESHOLD;
            } else {
                val = this.REV_MAP.getOrDefault(c, 0);
            }

            acc = (acc << 15) | val;
            bits += 15;

            while (bits >= 8) {
                bits -= 8;
                int byteVal = (acc >> bits) & 0xFF;
                acc = (bits == 0) ? 0 : acc & ((1 << bits) - 1);
                ba.write(byteVal);
            }
        }

        // Cut until last 1
        while (bits > 0 && (acc & 1) == 0) {
            acc >>= 1;
            bits--;
        }
        if (bits > 0) {
            acc >>= 1;
            bits--;
        }
        while (bits >= 8) {
            bits -= 8;
            int byteVal = (acc >> bits) & 0xFF;
            acc = (bits == 0) ? 0 : acc & ((1 << bits) - 1);
            ba.write(byteVal);
        }

        return ba.toByteArray();
    }
}