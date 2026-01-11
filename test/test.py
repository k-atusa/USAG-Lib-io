import os
import basio

text = "안녕하세요, 카투사 프로그래밍 클럽 라이브러리 테스트입니다. Hello, world!".encode('utf-8')
data = [b"", b"\x00", b"\x12\x34", b"\x3f\xff", b"\xff\xee\xff\xff\xff\xdc\xff\xff", b"\xff\x00\x00\x01\xff\x00\x00\x01\x10"]
m = basio.Encoder()
test = m.encode(text, True)
print(f"{test} : {m.decode(test).decode('utf-8')}")
test = m.encode(text, False)
print(f"{test} : {m.decode(test).decode('utf-8')}")
for i in range(len(data)):
    test = m.encode(data[i], False)
    print(f"{test} : {m.decode(test)}")

if not os.path.exists("test0.bin"):
    with open("test0.bin", "wb") as f:
        test = b"\x00" * 1024 * 1024 * 1024
        for i in range(5):
            f.write(test)
m = basio.AFile()
m.open("test0.bin", True)
m.seek(1048576)
print(f"{m.read(8)}, {m.tell()}, {m.getpath()}, {m.getsize()}")
m.close()
m = basio.AFile()
m.open(b"\x00\x01", False)
m.write(b"\x02\x03\x04\x05")
print(f"{m.tell()}, {m.getpath()}, {m.getsize()}, {m.close()}")

m = basio.Z64Writer("test1.zip", b"Hello, world!", False)
m.writebin("큰 파일", b"Hello, world!")
m.writefile("file", "test0.bin")
m.close()
m = basio.Z64Reader("test1.zip")
print(m.files, m.read(0))
m.close()

