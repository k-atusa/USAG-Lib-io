// test789c : USAG Lib basic io

package src

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

// Base-N Encoder
type Encoder struct {
	chars     []rune
	revMap    map[rune]int
	Threshold int
	Escape    rune
}

func (e *Encoder) Init() {
	e.chars = make([]rune, 0, 32164)
	e.revMap = make(map[rune]int)
	e.Threshold = 32164
	e.Escape = '.'

	for i := 0; i < 11172; i++ { // 1. Korean letters
		e.chars = append(e.chars, rune(0xAC00+i))
	}
	for i := 0; i < 20992; i++ { // 2. CJK letters
		e.chars = append(e.chars, rune(0x4E00+i))
	}
	for idx, char := range e.chars { // Reverse Map
		e.revMap[char] = idx
	}
}

func (e *Encoder) Encode(data []byte, isBase64 bool) string {
	if isBase64 && len(data) == 0 {
		return ""
	}
	if isBase64 {
		return base64.StdEncoding.EncodeToString(data)
	}
	return e.encodeUnicode(data)
}

func (e *Encoder) Decode(data string) ([]byte, error) {
	data = strings.ReplaceAll(data, "\r", "")
	data = strings.ReplaceAll(data, "\n", "")
	data = strings.ReplaceAll(data, " ", "")
	if data == "" {
		return []byte{}, nil
	}

	runes := []rune(data)
	if runes[0] < 128 && runes[0] != e.Escape { // Base64 mode
		return base64.StdEncoding.DecodeString(data)
	}
	return e.decodeUnicode(runes)
}

func (e *Encoder) encodeUnicode(data []byte) string {
	var result strings.Builder
	acc := 0
	bits := 0

	for _, b := range data {
		acc = (acc << 8) | int(b)
		bits += 8
		for bits >= 15 {
			bits -= 15
			val := acc >> bits // Upper 15 bits
			if bits == 0 {     // reset acc
				acc = 0
			} else {
				acc &= (1 << bits) - 1
			}

			if val < e.Threshold { // add rune
				result.WriteRune(e.chars[val])
			} else {
				result.WriteRune(e.Escape)
				result.WriteRune(e.chars[val-e.Threshold])
			}
		}
	}

	// Pad leftover
	val := ((acc << 1) | 1) << (14 - bits)
	if val < e.Threshold {
		result.WriteRune(e.chars[val])
	} else {
		result.WriteRune(e.Escape)
		result.WriteRune(e.chars[val-e.Threshold])
	}
	return result.String()
}

func (e *Encoder) decodeUnicode(runes []rune) ([]byte, error) {
	var ba bytes.Buffer
	acc := 0
	bits := 0
	n := len(runes)
	i := 0

	for i < n {
		char := runes[i]
		i++
		val := 0

		// get rune, accumulate 15-bits
		if char == e.Escape {
			if i >= n {
				return nil, errors.New("invalid escape")
			}
			nextChar := runes[i]
			i++
			val = e.revMap[nextChar] + e.Threshold
		} else {
			val = e.revMap[char]
		}
		acc = (acc << 15) | val
		bits += 15

		for i < n && bits >= 8 {
			bits -= 8
			byteVal := byte(acc >> bits)
			if bits == 0 {
				acc = 0
			} else {
				acc &= (1 << bits) - 1
			}
			ba.WriteByte(byteVal)
		}
	}

	// Cut until last 1
	for bits > 0 && (acc&1) == 0 {
		acc >>= 1
		bits--
	}
	if bits > 0 { // cut last 1
		acc >>= 1
		bits--
	}
	for bits >= 8 {
		bits -= 8
		byteVal := byte((acc >> bits) & 0xFF)
		if bits == 0 {
			acc = 0
		} else {
			acc &= (1 << bits) - 1
		}
		ba.WriteByte(byteVal)
	}
	return ba.Bytes(), nil
}

// Zip64 Writer
type Z64Writer struct {
	file   *os.File
	buffer *bytes.Buffer
	writer io.Writer // Abstract writer

	zip   *zip.Writer
	comp  uint16
	isMem bool
}

func (z *Z64Writer) Init(output string, header []byte, compress bool) error {
	z.file = nil
	z.buffer = nil
	if output == "" { // Memory buffer
		z.buffer = new(bytes.Buffer)
		z.writer = z.buffer
		z.isMem = true
	} else { // File output
		f, err := os.Create(output)
		z.file = f
		z.writer = f
		z.isMem = false
		if err != nil {
			return err
		}
	}

	// Write header first
	if _, err := z.writer.Write(header); err != nil {
		return err
	}
	z.zip = zip.NewWriter(z.writer)
	if compress {
		z.comp = zip.Deflate
	} else {
		z.comp = zip.Store
	}
	return nil
}

func (z *Z64Writer) WriteFile(name string, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Create zip header
	header := &zip.FileHeader{
		Name:   name,
		Method: z.comp,
	}
	w, err := z.zip.CreateHeader(header)
	if err != nil {
		return err
	}

	// Copy file
	_, err = io.Copy(w, f)
	return err
}

func (z *Z64Writer) WriteBin(name string, data []byte) error {
	// Create zip header
	header := &zip.FileHeader{
		Name:   name,
		Method: z.comp,
	}
	w, err := z.zip.CreateHeader(header)
	if err != nil {
		return err
	}

	// Write data
	_, err = w.Write(data)
	return err
}

func (z *Z64Writer) Close() ([]byte, error) {
	err := z.zip.Close()
	if err != nil {
		return nil, err
	}
	if z.isMem {
		temp := z.buffer.Bytes()
		z.buffer = nil
		return temp, nil
	} else {
		return nil, z.file.Close()
	}
}

// Zip64 Reader
type Z64Reader struct {
	file      *os.File
	buffer    []byte
	zipReader *zip.Reader
	Files     []*zip.File
}

func (z *Z64Reader) Init(input interface{}) error {
	z.file = nil
	z.buffer = nil
	var size int64 = 0
	var readerAt io.ReaderAt

	switch v := input.(type) {
	case string: // File path input
		f, err := os.Open(v)
		if err != nil {
			return err
		}
		stat, err := f.Stat()
		if err != nil {
			f.Close()
			return err
		}
		z.file = f
		readerAt = f
		size = stat.Size()
	case []byte: // Data input
		z.buffer = v
		readerAt = bytes.NewReader(v)
		size = int64(len(v))
	default:
		return errors.New("input must be filepath(string) or data([]byte)")
	}

	var err error
	z.zipReader, err = zip.NewReader(readerAt, size)
	if err != nil {
		if z.file != nil {
			z.file.Close()
		}
		return err
	}
	z.Files = z.zipReader.File
	return nil
}

func (z *Z64Reader) Read(idx int) ([]byte, error) {
	if idx < 0 || idx >= len(z.zipReader.File) {
		return nil, errors.New("index out of bounds")
	}
	f := z.zipReader.File[idx]
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (z *Z64Reader) Open(idx int) (io.ReadCloser, error) {
	if idx < 0 || idx >= len(z.zipReader.File) {
		return nil, errors.New("index out of bounds")
	}
	return z.zipReader.File[idx].Open()
}

func (z *Z64Reader) Close() error {
	if z.file != nil {
		return z.file.Close()
	}
	return nil
}

// Abstract File Interface
type AFile interface {
	Open(src interface{}, isRead bool) error
	Close() ([]byte, error)
	Read(size int) ([]byte, error)
	Write(data []byte) error
	Seek(pos int) error
	Tell() int
	GetPath() string
	GetSize() int
}
type BFile struct {
	file   *os.File
	buffer []byte

	path string
	size int
	pos  int

	readMode bool
	byteMode bool
}

func (f *BFile) Open(src interface{}, isRead bool) error {
	f.file = nil
	f.buffer = nil
	f.readMode = isRead

	switch v := src.(type) {
	case string: // File path input
		f.byteMode = false
		f.path = v
		f.pos = 0
		var err error
		if isRead {
			f.file, err = os.Open(v)
			if err != nil {
				return err
			}
			info, err := f.file.Stat()
			if err != nil {
				f.file.Close()
				return err
			}
			f.size = int(info.Size())
		} else {
			f.file, err = os.Create(v)
			if err != nil {
				return err
			}
		}

	case []byte: // Data input
		f.byteMode = true
		f.path = ""
		f.buffer = v
		f.pos = len(v)
		f.size = len(v)

	default:
		return errors.New("input must be filepath(string) or data([]byte)")
	}
	return nil
}

func (f *BFile) Close() ([]byte, error) {
	if f.byteMode && f.readMode {
		f.buffer = nil
		return nil, nil
	} else if f.byteMode && !f.readMode {
		temp := f.buffer
		f.buffer = nil
		return temp, nil
	} else {
		if f.file != nil {
			return nil, f.file.Close()
		}
	}
	return nil, nil
}

func (f *BFile) Read(size int) ([]byte, error) {
	if !f.readMode {
		return nil, errors.New("cannot read file in write mode")
	}
	var data []byte
	var err error
	if size < 0 || size > f.size-f.pos {
		size = f.size - f.pos
	}

	if f.byteMode {
		data = f.buffer[f.pos : f.pos+size]
		f.pos += size
	} else {
		cut := 256 * 1048576
		data = make([]byte, size)
		for i := 0; i < size/cut; i++ {
			_, err = f.file.Read(data[i*cut : i*cut+cut])
			if err != nil {
				return data, err
			}
		}
		if size%cut != 0 {
			_, err = f.file.Read(data[(size/cut)*cut:])
		}
		f.pos += size
	}
	return data, err
}

func (f *BFile) Write(data []byte) error {
	if f.readMode {
		return errors.New("cannot write file in read mode")
	}
	var wr int
	var err error

	if f.byteMode {
		f.buffer = append(f.buffer, data...)
		f.pos += len(data)
		f.size += len(data)
	} else {
		cut := 256 * 1048576
		for i := 0; i < len(data)/cut; i++ {
			n, err := f.file.Write(data[i*cut : i*cut+cut])
			wr += n
			if err != nil {
				return err
			}
		}
		if len(data)%cut != 0 {
			n, err := f.file.Write(data[(len(data)/cut)*cut:])
			wr += n
			if err != nil {
				return err
			}
		}
		f.pos += len(data)
		f.size += len(data)
		if wr != len(data) {
			return errors.New("loss of data while writing")
		}
	}
	return err
}

func (f *BFile) Seek(offset int) error {
	if !f.readMode {
		return errors.New("cannot seek file in write mode")
	}
	if offset < 0 || offset > f.size {
		offset = f.size
	}
	f.pos = offset
	if !f.byteMode {
		_, err := f.file.Seek(int64(offset), io.SeekStart)
		return err
	}
	return nil
}

func (f *BFile) Tell() int {
	return f.pos
}

func (f *BFile) GetPath() string {
	return f.path
}

func (f *BFile) GetSize() int {
	return f.size
}
