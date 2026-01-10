# test789a : USAG Lib basic io

import io
import os
import base64
import zipfile

class Encoder: # Base-N encoder
    def __init__(self):
        self._CHARS, self._REV_MAP = [], {} # pre calculated conversion table
        for i in range(11172): # korean letters (U+AC00-U+D7A3): 11172
            self._CHARS.append(chr(0xAC00 + i))
        for i in range(20992): # CJK letters (U+4E00-U+9FFF): 20992
            self._CHARS.append(chr(0x4E00 + i))
        for idx, char in enumerate(self._CHARS): # reversed table
            self._REV_MAP[char] = idx
        self._THRESHOLD, self._ESCAPE_CHAR = 32164, "." # escape control

    def encode(self, data: bytes, isBase64: bool) -> str:
        if isBase64 and len(data) == 0:
            return ""
        if isBase64:
            return base64.b64encode(data).decode('ascii')
        return self._encode_unicode(data)

    def decode(self, data: str) -> bytes:
        data = data.replace("\r", "").replace("\n", "").replace(" ", "")
        if data == "":
            return b""
        if ord(data[0]) < 128 and data[0] != self._ESCAPE_CHAR: # Base64
            return base64.b64decode(data)
        else: # Base32k
            return self._decode_unicode(data)

    def _encode_unicode(self, data: bytes) -> str:
        result, acc, bits = [], 0, 0

        # split bytes into 15-bits
        for byte in data:
            acc = (acc << 8) | byte
            bits += 8
            while bits >= 15:
                bits -= 15
                val = acc >> bits # get upper 15-bits
                acc = 0 if bits == 0 else acc & ((1 << bits) - 1) # reset acc
                if val < self._THRESHOLD:
                    result.append(self._CHARS[val]) # encode with letter
                else:
                    offset = val - self._THRESHOLD
                    result.append(self._ESCAPE_CHAR + self._CHARS[offset]) # encode with escape

        # pad leftover bits
        val = ((acc << 1) | 1) << (14 - bits)
        if val < self._THRESHOLD:
            result.append(self._CHARS[val])
        else:
            offset = val - self._THRESHOLD
            result.append(self._ESCAPE_CHAR + self._CHARS[offset])
        return "".join(result)

    def _decode_unicode(self, data: str) -> bytes:
        ba, acc, bits = bytearray(), 0, 0
        i, n = 0, len(data)

        while i < n:
            char = data[i]
            i += 1
            val = 0
            if char == self._ESCAPE_CHAR: # escape control
                if i >= n:
                    raise Exception("invalid escape")
                next_char = data[i]
                i += 1
                val = self._REV_MAP.get(next_char, 0) + self._THRESHOLD
            else: # plain unicode
                val = self._REV_MAP.get(char, 0)
            
            acc = (acc << 15) | val # accumulate with 15-bits
            bits += 15
            while i < n and bits >= 8:
                bits -= 8
                byte_val = acc >> bits # get upper 8-bits
                acc = 0 if bits == 0 else acc & ((1 << bits) - 1) # reset acc
                ba.append(byte_val)

        # cut until last 1 is found
        while bits > 0 and (acc & 1) == 0:
            acc >>= 1
            bits -= 1
        if bits > 0: # cut last 1
            acc >>= 1
            bits -= 1
        while bits >= 8:
            bits -= 8
            byte_val = acc >> bits
            acc = 0 if bits == 0 else acc & ((1 << bits) - 1)
            ba.append(byte_val)
        return bytes(ba)
    
class Z64Writer:
    def __init__(self, output: str, header: bytes, compress: bool):
        self.output = io.BytesIO() if output == "" else open(output, "wb")
        self.output.write(header) # write header first
        self.zip = zipfile.ZipFile(self.output, "a", zipfile.ZIP_DEFLATED if compress else zipfile.ZIP_STORED, allowZip64=True) # create zip writer

    def writefile(self, name:str, path: str):
        self.zip.write(path, name)

    def writebin(self, name: str, data: bytes):
        self.zip.writestr(name, data)

    def close(self):
        self.zip.close()
        if type(self.output) == io.BytesIO:
            temp = self.output.getvalue()
            self.output.close()
            self.output = temp
        else:
            self.output.close()

class Z64Reader:
    def __init__(self, input):
        self.input = io.BytesIO(input) if type(input) == bytes else open(input, "rb")
        self.zip = zipfile.ZipFile(self.input, "r", allowZip64=True) # create zip reader
        self.files = self.zip.infolist() # list of files in zip

    def read(self, idx: int) -> bytes:
        return self.zip.read(self.files[idx])
    
    def open(self, idx: int) -> io.IOBase:
        return self.zip.open(self.files[idx], "r")

    def close(self):
        self.zip.close()
        self.input.close()

class AFile: # abstract file
    def __init__(self):
        self.handle, self.readmode, self.bytemode = None, False, False
        self.path, self.size, self.pos = "", 0, 0

    def open(self, src, isRead: bool):
        if type(src) == str:
            self.handle, self.bytemode = open(src, "rb" if isRead else "wb"), False
            self.path = src
            self.size = os.path.getsize(src)
        else:
            self.handle, self.bytemode = src if isRead else io.BytesIO(src), True
            self.path = ""
            self.size = len(src) if isRead else 0
        self.readmode = isRead
        self.pos = 0

    def close(self):
        if self.readmode and self.bytemode:
            self.handle = None
        elif self.bytemode:
            temp = self.handle.getvalue()
            self.handle.close()
            self.handle = temp
        elif self.handle != None:
            self.handle.close()
            self.handle = None

    def read(self, size: int) -> bytes: # -1 to read all
        if self.readmode:
            if self.bytemode:
                if size < 0:
                    temp = self.handle[self.pos:]
                    self.pos = self.size
                else:
                    temp = self.handle[self.pos:self.pos + size]
                    self.pos += size
                if self.pos > self.size:
                    self.pos = self.size
                return temp
            else:
                if size < 0:
                    temp = self.handle.read()
                    self.pos = self.size
                else:
                    temp = self.handle.read(size)
                    self.pos += len(temp)
                if self.pos > self.size:
                    self.pos = self.size
                return temp
        else:
            raise Exception("cannot read file in write mode")
        
    def write(self, data: bytes): # do not read/write more than 1GiB at a time
        if not self.readmode:
            self.handle.write(data)
            self.size += len(data)
            self.pos += len(data)
        else:
            raise Exception("cannot write file in read mode")
        
    def seek(self, offset: int):
        if self.readmode:
            self.pos = self.size if offset < 0 or offset > self.size else offset
        else:
            raise Exception("cannot seek file in write mode")
        
    def tell(self) -> int:
        return self.pos
    
    def getpath(self) -> str:
        return self.path
    
    def getsize(self) -> int:
        return self.size