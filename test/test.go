// go mod init example.com
// go mod tidy
// go run test.go

package main

import (
	"fmt"
	"log"
	"os"

	basio "github.com/k-atusa/USAG-Lib-io/src"
)

func main() {
	// 1. Encoder Test
	text := []byte("안녕하세요, 카투사 프로그래밍 클럽 라이브러리 테스트입니다. Hello, world!")
	dataList := [][]byte{
		{},
		{0x00},
		{0x12, 0x34},
		{0x3f, 0xff},
		{0xff, 0xee, 0xff, 0xff, 0xff, 0xdc, 0xff, 0xff},
		{0xff, 0x00, 0x00, 0x01, 0xff, 0x00, 0x00, 0x01, 0x10},
	}
	var m basio.Encoder
	m.Init()

	// Base64 Encode/Decode
	testStr := m.Encode(text, true)
	decoded, _ := m.Decode(testStr)
	fmt.Printf("%s : %s\n", testStr, string(decoded))

	// Base32k Encode/Decode
	testStr = m.Encode(text, false)
	decoded, _ = m.Decode(testStr)
	fmt.Printf("%s : %s\n", testStr, string(decoded))

	// Loop Test
	for _, data := range dataList {
		testStr = m.Encode(data, false)
		decoded, _ = m.Decode(testStr)
		fmt.Printf("%s : %x\n", testStr, decoded)
	}

	// 2. Large File I/O Test (AFile/BFile)
	fileName := "test0.bin"
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		f, err := os.Create(fileName)
		if err != nil {
			log.Fatal(err)
		}
		dummy := make([]byte, 1024*1024*1024)
		for i := 0; i < 5; i++ {
			f.Write(dummy)
		}
		f.Close()
	}

	// File Read Test
	var fileIO basio.AFile = &basio.BFile{}
	err := fileIO.Open(fileName, true)
	if err != nil {
		log.Fatal(err)
	}

	fileIO.Seek(1048576)
	readData, _ := fileIO.Read(8)
	fmt.Printf("%x, %d, %s, %d\n", readData, fileIO.Tell(), fileIO.GetPath(), fileIO.GetSize())
	fileIO.Close()

	// 3. Memory I/O Test
	memIO := &basio.BFile{}
	memIO.Open([]byte{0x00, 0x01}, false)
	memIO.Write([]byte{0x02, 0x03, 0x04, 0x05})

	res, _ := memIO.Close()
	fmt.Printf("%d, %s, %d, %x\n", memIO.Tell(), memIO.GetPath(), memIO.GetSize(), res)

	// 4. Zip64 Writer Test
	zw := &basio.Z64Writer{}
	err = zw.Init("test1.zip", []byte("Hello, world!"), false)
	if err != nil {
		log.Fatal(err)
	}

	zw.WriteBin("binary", []byte("Hello, world!"))
	zw.WriteFile("file", "test0.bin")
	zw.Close()

	// 5. Zip64 Reader Test
	zr := &basio.Z64Reader{}
	err = zr.Init("test1.zip")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Zip Files:")
	for _, f := range zr.Files {
		fmt.Printf("- %s (Size: %d)\n", f.Name, f.UncompressedSize64)
	}

	firstFileContent, _ := zr.Read(0)
	fmt.Printf("Content of first file: %s\n", string(firstFileContent))
	zr.Close()
}
