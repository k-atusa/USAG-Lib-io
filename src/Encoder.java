// test789d : USAG Lib basic io

import java.io.ByteArrayOutputStream;
import java.util.*;

public class Encoder {
    private final List<Character> _CHARS;
    private final Map<Character, Integer> _REV_MAP;
    private final int _THRESHOLD = 32164;
    private final char _ESCAPE_CHAR = '.';

    public Encoder() {
        _CHARS = new ArrayList<>(32164);
        _REV_MAP = new HashMap<>();

        // 1. Korean letters
        for (int i = 0; i < 11172; i++) {
            _CHARS.add((char) (0xAC00 + i));
        }
        // 2. CJK letters
        for (int i = 0; i < 20992; i++) {
            _CHARS.add((char) (0x4E00 + i));
        }
        // Reverse Map
        for (int i = 0; i < _CHARS.size(); i++) {
            _REV_MAP.put(_CHARS.get(i), i);
        }
    }

    public String encode(byte[] data, boolean isBase64) {
        if (isBase64 && data.length == 0) return "";
        if (isBase64) {
            return Base64.getEncoder().encodeToString(data);
        }
        return _encodeUnicode(data);
    }

    public byte[] decode(String data) {
        data = data.replace("\r", "").replace("\n", "").replace(" ", "");
        if (data.isEmpty()) return new byte[0];

        if (data.charAt(0) < 128 && data.charAt(0) != _ESCAPE_CHAR) {
            return Base64.getDecoder().decode(data);
        } else {
            return _decodeUnicode(data);
        }
    }

    private String _encodeUnicode(byte[] data) {
        StringBuilder result = new StringBuilder();
        int acc = 0;
        int bits = 0;

        for (byte b : data) {
            acc = (acc << 8) | (b & 0xFF);
            bits += 8;
            while (bits >= 15) {
                bits -= 15;
                int val = (acc >> bits) & 0x7FFF;
                acc = (bits == 0) ? 0 : acc & ((1 << bits) - 1);

                if (val < _THRESHOLD) {
                    result.append(_CHARS.get(val));
                } else {
                    int offset = val - _THRESHOLD;
                    result.append(_ESCAPE_CHAR).append(_CHARS.get(offset));
                }
            }
        }

        // Pad leftover
        int val = ((acc << 1) | 1) << (14 - bits);
        if (val < _THRESHOLD) {
            result.append(_CHARS.get(val));
        } else {
            int offset = val - _THRESHOLD;
            result.append(_ESCAPE_CHAR).append(_CHARS.get(offset));
        }
        return result.toString();
    }

    private byte[] _decodeUnicode(String data) {
        ByteArrayOutputStream ba = new ByteArrayOutputStream();
        int acc = 0;
        int bits = 0;
        int i = 0;
        int n = data.length();

        while (i < n) {
            char c = data.charAt(i);
            i++;
            int val = 0;

            if (c == _ESCAPE_CHAR) {
                if (i >= n) throw new RuntimeException("invalid escape");
                char nextChar = data.charAt(i);
                i++;
                val = _REV_MAP.getOrDefault(nextChar, 0) + _THRESHOLD;
            } else {
                val = _REV_MAP.getOrDefault(c, 0);
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