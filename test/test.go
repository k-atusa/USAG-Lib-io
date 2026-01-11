// go mod init example.com
// go mod tidy
// go run test.go

package main

import (
	"fmt"
	"log"
	"os"

	basio "github.com/k-atusa/USAG-Lib-io/basio" // go.mod 설정에 맞게 경로 수정 필요
)

func main() {
	// ==========================================
	// 1. Encoder Test
	// ==========================================
	text := []byte("안녕하세요, 카투사 프로그래밍 클럽 라이브러리 테스트입니다. Hello, world!")

	// Python: data list
	dataList := [][]byte{
		{},
		{0x00},
		{0x12, 0x34},
		{0x3f, 0xff},
		{0xff, 0xee, 0xff, 0xff, 0xff, 0xdc, 0xff, 0xff},
		{0xff, 0x00, 0x00, 0x01, 0xff, 0x00, 0x00, 0x01, 0x10},
	}

	m := &basio.Encoder{}
	m.Init()

	// Base64 Encode/Decode
	testStr := m.Encode(text, true)
	decoded, _ := m.Decode(testStr)
	fmt.Printf("%s : %s\n", testStr, string(decoded))

	// Base32k (Custom) Encode/Decode
	testStr = m.Encode(text, false)
	decoded, _ = m.Decode(testStr)
	fmt.Printf("%s : %s\n", testStr, string(decoded))

	// Loop Test
	for _, data := range dataList {
		testStr = m.Encode(data, false)
		decoded, _ = m.Decode(testStr)
		// Go에서 []byte 출력은 십진수 배열로 나오므로 Python의 b'\x..' 포맷과 유사하게 확인하려면 string으로 변환하거나 포맷팅 필요
		// 여기서는 데이터 무결성 확인을 위해 바이트 슬라이스 자체를 출력
		fmt.Printf("%s : %x\n", testStr, decoded)
	}

	// ==========================================
	// 2. Large File I/O Test (AFile/BFile)
	// ==========================================
	fileName := "test0.bin"
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		f, err := os.Create(fileName)
		if err != nil {
			log.Fatal(err)
		}
		// 1GB 더미 데이터 생성 (메모리 주의: Python 로직 그대로 구현)
		// 실제 환경에서는 루프로 나누어 쓰는 것이 좋으나 Python 코드와 동일하게 처리
		dummy := make([]byte, 1024*1024*1024)
		for i := 0; i < 5; i++ {
			f.Write(dummy)
		}
		f.Close()
	}

	// File Read Test
	// Go에서는 인터페이스(AFile)를 통해 구체 타입(BFile)을 사용
	var fileIO basio.AFile = &basio.BFile{}

	err := fileIO.Open(fileName, true)
	if err != nil {
		log.Fatal(err)
	}

	fileIO.Seek(1048576)
	readData, _ := fileIO.Read(8)

	// Python: print(f"{m.read(8)}, {m.tell()}, {m.getpath()}, {m.getsize()}")
	fmt.Printf("%x, %d, %s, %d\n", readData, fileIO.Tell(), fileIO.GetPath(), fileIO.GetSize())
	fileIO.Close()

	// ==========================================
	// 3. Memory I/O Test
	// ==========================================
	memIO := &basio.BFile{}
	// Python: m.open(b"\x00\x01", False) -> 초기값 설정
	memIO.Open([]byte{0x00, 0x01}, false)
	memIO.Write([]byte{0x02, 0x03, 0x04, 0x05})

	res, _ := memIO.Close()
	// Python: print(f"{m.tell()}, {m.getpath()}, {m.getsize()}, {m.close()}")
	// 주의: Go 구현상 Append 방식이라 Python(Overwrite)과 결과가 다를 수 있음 (길이 6 vs 4)
	fmt.Printf("%d, %s, %d, %x\n", memIO.Tell(), memIO.GetPath(), memIO.GetSize(), res)

	// ==========================================
	// 4. Zip64 Writer Test
	// ==========================================
	zw := &basio.Z64Writer{}
	// Python: m = basio.Z64Writer("test1.zip", b"Hello, world!", False)
	err = zw.Init("test1.zip", []byte("Hello, world!"), false)
	if err != nil {
		log.Fatal(err)
	}

	zw.WriteBin("binary", []byte("Hello, world!"))
	zw.WriteFile("file", "test0.bin")
	zw.Close()

	// ==========================================
	// 5. Zip64 Reader Test
	// ==========================================
	zr := &basio.Z64Reader{}
	err = zr.Init("test1.zip")
	if err != nil {
		log.Fatal(err)
	}

	// Python: print(m.files, m.read(0))
	// Go의 zip.Reader는 File 슬라이스를 가짐
	fmt.Println("Zip Files:")
	for _, f := range zr.Files {
		fmt.Printf("- %s (Size: %d)\n", f.Name, f.UncompressedSize64)
	}

	firstFileContent, _ := zr.Read(0)
	fmt.Printf("Content of first file: %s\n", string(firstFileContent))

	zr.Close()
}
