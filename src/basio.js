// test789b : USAG Lib basic io

/*
require jszip: npm install jszip, <script src="https://cdnjs.cloudflare.com/ajax/libs/jszip/3.10.1/jszip.min.js"></script>
*/

// env check, load lib, !!! JS version is not designed for big data !!!
const isNode = typeof process !== 'undefined' && process.versions != null && process.versions.node != null;
let fs, JSZip;
if (isNode) {
    fs = require('fs');
    JSZip = require('jszip');
} else {
    JSZip = window.JSZip;
}

class Encoder { // Base-N encoder
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
        if (isNode) {
            return Buffer.from(uint8Array).toString('base64');
        } else {
            let binary = '';
            const len = uint8Array.byteLength;
            for (let i = 0; i < len; i++) binary += String.fromCharCode(uint8Array[i]);
            return btoa(binary);
        }
    }

    _fromBase64(str) {
        if (isNode) {
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

            while (i < n && bits >= 8) {
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

class Z64Writer { // Zip64 Writer
    /**
     * @param {string} output set empty for memory, filepath(Node) or filename(Browser)
     * @param {Uint8Array} header custom header
     * @param {boolean} compress compress flag
    */
    constructor(output, header, compress) {
        this.outputStr = output;
        this.header = header || new Uint8Array(0);
        this.compress = compress;
        this.zip = new JSZip();
        this.isMemory = (output === "");
    }

    /**
     * @param {string} name file name in zip
     * @param {string|Blob|File} src file path (Node) or Blob/File object (Browser)
     */
    writefile(name, src) {
        if (isNode) {
            if (typeof src === 'string') {
                const data = fs.readFileSync(src);
                this.writebin(name, data);
            } else {
                this.writebin(name, src); // write Blob
            }
        } else {
            if (src instanceof Blob) {
                this.writebin(name, src); // write Blob or File
            } else {
                throw new Error("writefile in browser needs Blob object");
            }
        }
    }

    /**
     * @param {string} name file name in zip
     * @param {Uint8Array|string|Blob} data binary data
     */
    writebin(name, data) {
        const options = {
            compression: this.compress ? "DEFLATE" : "STORE"
        };
        this.zip.file(name, data, options);
    }

    async close() {
        // Generate Zip
        const zipData = await this.zip.generateAsync({
            type: isNode ? "nodebuffer" : "uint8array",
            compression: this.compress ? "DEFLATE" : "STORE"
        });

        // Concat Header + Zip
        const totalLength = this.header.length + zipData.length;
        const result = new Uint8Array(totalLength);
        result.set(this.header, 0);
        result.set(zipData, this.header.length);

        if (this.isMemory) {
            this.memResult = result;
        } else {
            if (isNode) {
                fs.writeFileSync(this.outputStr, result); // write file (Node)
            } else {
                this._browserDownload(result, this.outputStr || "archive.z64"); // download (Browser)
            }
        }
    }
    
    getMemResult() {
        return this.memResult;
    }

    _browserDownload(data, filename) { // browser download helper
        const blob = new Blob([data], { type: "application/octet-stream" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.style.display = "none";
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        
        // Cleanup
        setTimeout(() => {
            document.body.removeChild(a);
            window.URL.revokeObjectURL(url);
        }, 100);
    }
}

class Z64Reader { // Zip64 Reader
    /**
     * @param {string|Blob|Uint8Array} input path string (Node), Blob or Uint8Array (Browser)
     */
    constructor(input) {
        this.input = input;
        this.zip = null;
        this.files = []; // List of JSZipObjects
    }

    async init() {
        // load zip data to memory
        let dataToLoad;
        if (isNode && typeof this.input === 'string') {
            dataToLoad = fs.readFileSync(this.input);
        } else {
            dataToLoad = this.input;
        }
        this.zip = await JSZip.loadAsync(dataToLoad);
        
        this.files = []; // push file infos
        this.zip.forEach((relativePath, file) => {
            this.files.push(file);
        });
    }

    async read(idx) {
        if (!this.zip) await this.init();
        if (idx < 0 || idx >= this.files.length) throw new Error("Index out of bounds");
        return await this.files[idx].async("uint8array");
    }

    async open(idx) {
        const data = await this.read(idx);
        const memFile = new AFile();
        await memFile.open(data, true);
        return memFile;
    }

    close() {
        this.zip = null;
        this.files = [];
    }
}

class AFile { // Abstract File
    constructor() {
        this.handle = null;
        this.readmode = false;
        this.bytemode = false;
        this.path = "";
        this.size = 0;
        this.pos = 0;
        
        // for browser
        this.blob = null; 
        this.bufferChunks = [];
    }

    /**
     * @param {string|Uint8Array|Blob} src filepath or data
     * @param {boolean} isRead set read mode
     */
    async open(src, isRead) {
        this.readmode = isRead;
        this.pos = 0;

        if (typeof src === 'string') {
            if (!isNode) throw new Error("filepath access not supported in browser");
            this.path = src;
            this.bytemode = false;
            if (isRead) {
                const stats = fs.statSync(src);
                this.size = stats.size;
                this.handle = fs.openSync(src, 'r');
            } else {
                this.handle = fs.openSync(src, 'w');
                this.size = 0;
            }

        } else {
            this.path = "";
            this.bytemode = true;
            if (isRead) { // read as uint8array or blob
                if (isNode) {
                    this.handle = (src instanceof Buffer) ? src : Buffer.from(src);
                    this.size = this.handle.length;
                } else {
                    if (src instanceof Blob) {
                        this.blob = src;
                        this.size = src.size;
                    } else {
                        this.blob = new Blob([src]);
                        this.size = this.blob.size;
                    }
                }
            } else { // write as uint8array or blob
                if (src && (src instanceof Uint8Array || src instanceof Blob || src instanceof Buffer)) { // append
                    this.bufferChunks = [src];
                    this.size = (src instanceof Blob) ? src.size : src.length;
                    this.pos = this.size;
                } else { // write
                    this.size = 0;
                    this.bufferChunks = [];
                    this.pos = 0;
                }
            }
        }
    }

    async close() {
        if (this.readmode) {
            if (!this.bytemode && isNode && this.handle !== null) { // close file handle
                fs.closeSync(this.handle);
                this.handle = null;
            }
        } else {
            if (this.bytemode) { // concat all chunks to handle
                if (isNode) {
                    this.handle = Buffer.concat(this.bufferChunks);
                } else {
                    this.handle = new Blob(this.bufferChunks);
                }
            } else if (isNode && this.handle !== null) { // close file handle
                fs.closeSync(this.handle);
                this.handle = null;
            }
        }
    }

    async read(size) {
        if (!this.readmode) throw new Error("Cannot read in write mode");

        // manage read size
        let readSize = size;
        if (readSize < 0) readSize = this.size - this.pos;
        if (this.pos + readSize > this.size) readSize = this.size - this.pos;
        if (readSize <= 0) return new Uint8Array(0);
        let result;

        if (this.bytemode) {
            if (isNode) { // get array (Node)
                result = this.handle.subarray(this.pos, this.pos + readSize);
            } else { // get blob (Browser)
                const slice = this.blob.slice(this.pos, this.pos + readSize);
                const arrayBuffer = await slice.arrayBuffer();
                result = new Uint8Array(arrayBuffer);
            }
        } else { // file read (Node)
            const buffer = Buffer.alloc(readSize);
            const bytesRead = fs.readSync(this.handle, buffer, 0, readSize, this.pos);
            result = buffer.subarray(0, bytesRead);
        }

        this.pos += result.length;
        return result;
    }

    async write(data) {
        if (this.readmode) throw new Error("Cannot write in read mode");

        // manage write data
        let dataToWrite = data;
        if (typeof data === 'string') {
            dataToWrite = new TextEncoder().encode(data);
        }

        if (this.bytemode) {
            this.bufferChunks.push(dataToWrite);
        } else {
            fs.writeSync(this.handle, dataToWrite);
        }
        this.size += dataToWrite.length;
        this.pos += dataToWrite.length;
    }

    seek(offset) {
        if (!this.readmode) throw new Error("Cannot seek in write mode");
        if (offset < 0 || offset > this.size) {
            this.pos = this.size;
        } else {
            this.pos = offset;
        }
    }

    tell() { return this.pos; }
    getPath() { return this.path; }
    getSize() { return this.size; }
    getMemResult() { return this.handle; } // Returns Buffer(Node) or Blob(Browser) after close
}

// export class
if (isNode) {
    module.exports = { Encoder, AFile, Z64Writer, Z64Reader };
} else {
    window.Basio = { Encoder, AFile, Z64Writer, Z64Reader };
}
