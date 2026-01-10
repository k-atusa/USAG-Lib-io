// test789b : USAG Lib basic io

class Encoder {
    constructor() {
        this._CHARS = [];
        this._REV_MAP = {};

        // 1. Korean letters (U+AC00-U+D7A3): 11172
        for (let i = 0; i < 11172; i++) {
            this._CHARS.push(String.fromCharCode(0xAC00 + i));
        }
        // 2. CJK letters (U+4E00-U+9FFF): 20992
        for (let i = 0; i < 20992; i++) {
            this._CHARS.push(String.fromCharCode(0x4E00 + i));
        }
        // Build reverse map
        this._CHARS.forEach((char, idx) => {
            this._REV_MAP[char] = idx;
        });

        this._THRESHOLD = 32164;
        this._ESCAPE_CHAR = ".";
    }

    encode(data, isBase64) {
        // Ensure data is Uint8Array
        if (typeof data === 'string') {
            data = new TextEncoder().encode(data);
        } else if (!(data instanceof Uint8Array)) {
            data = new Uint8Array(data);
        }

        if (isBase64 && data.length === 0) return "";
        if (isBase64) return this._toBase64(data);
        return this._encodeUnicode(data);
    }

    decode(data) {
        data = data.replace(/[\r\n ]/g, ""); // Remove whitespace
        if (data === "") return new Uint8Array(0);

        const firstCharCode = data.charCodeAt(0);
        // Base64 Check: ASCII < 128 AND not escape char
        if (firstCharCode < 128 && data[0] !== this._ESCAPE_CHAR) {
            return this._fromBase64(data);
        } else {
            return this._decodeUnicode(data);
        }
    }

    // Helper for Base64 (Env agnostic)
    _toBase64(uint8Array) {
        if (typeof Buffer !== 'undefined') {
            return Buffer.from(uint8Array).toString('base64');
        } else {
            let binary = '';
            const len = uint8Array.byteLength;
            for (let i = 0; i < len; i++) binary += String.fromCharCode(uint8Array[i]);
            return btoa(binary);
        }
    }

    _fromBase64(str) {
        if (typeof Buffer !== 'undefined') {
            return new Uint8Array(Buffer.from(str, 'base64'));
        } else {
            const binary = atob(str);
            const len = binary.length;
            const bytes = new Uint8Array(len);
            for (let i = 0; i < len; i++) bytes[i] = binary.charCodeAt(i);
            return bytes;
        }
    }

    _encodeUnicode(data) {
        let result = [];
        let acc = 0;
        let bits = 0;

        for (let i = 0; i < data.length; i++) {
            acc = (acc << 8) | data[i];
            bits += 8;
            while (bits >= 15) {
                bits -= 15;
                const val = (acc >> bits) & 0x7FFF; // get upper 15-bits
                // reset acc logic: keep only lower 'bits'
                acc = (bits === 0) ? 0 : acc & ((1 << bits) - 1);

                if (val < this._THRESHOLD) {
                    result.push(this._CHARS[val]);
                } else {
                    const offset = val - this._THRESHOLD;
                    result.push(this._ESCAPE_CHAR + this._CHARS[offset]);
                }
            }
        }

        // Pad leftover
        const val = ((acc << 1) | 1) << (14 - bits);
        if (val < this._THRESHOLD) {
            result.push(this._CHARS[val]);
        } else {
            const offset = val - this._THRESHOLD;
            result.push(this._ESCAPE_CHAR + this._CHARS[offset]);
        }
        return result.join("");
    }

    _decodeUnicode(data) {
        const ba = [];
        let acc = 0;
        let bits = 0;
        let i = 0;
        const n = data.length;

        while (i < n) {
            const char = data[i];
            i++;
            let val = 0;

            if (char === this._ESCAPE_CHAR) {
                if (i >= n) throw new Error("invalid escape");
                const nextChar = data[i];
                i++;
                val = (this._REV_MAP[nextChar] || 0) + this._THRESHOLD;
            } else {
                val = this._REV_MAP[char] || 0;
            }

            acc = (acc << 15) | val;
            bits += 15;

            while (i <= n && bits >= 8) {
                bits -= 8;
                const byteVal = (acc >> bits) & 0xFF;
                acc = (bits === 0) ? 0 : acc & ((1 << bits) - 1);
                ba.push(byteVal);
            }
        }

        // Cut until last 1 is found
        while (bits > 0 && (acc & 1) === 0) {
            acc >>= 1;
            bits -= 1;
        }
        if (bits > 0) {
            acc >>= 1;
            bits -= 1;
        }
        while (bits >= 8) {
            bits -= 8;
            const byteVal = (acc >> bits) & 0xFF;
            acc = (bits === 0) ? 0 : acc & ((1 << bits) - 1);
            ba.push(byteVal);
        }

        return new Uint8Array(ba);
    }
}