package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	textunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
)

type codeset struct {
	name string
	bom  bool
}

func (c codeset) family() string {
	return strings.TrimSuffix(strings.TrimSuffix(c.name, "BE"), "LE")
}

func lookupCodeset(name string) (codeset, error) {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToUpper(name))
	switch normalized {
	case "ASCII", "USASCII":
		return codeset{name: "ASCII"}, nil
	case "UTF8", "UTF8N":
		return codeset{name: "UTF-8"}, nil
	case "UTF8BOM":
		return codeset{name: "UTF-8", bom: true}, nil
	case "SJIS", "SHIFTJIS", "MSKANJI", "CP932", "WINDOWS31J":
		return codeset{name: "Shift_JIS"}, nil
	case "EUC", "EUCJP", "UJIS":
		return codeset{name: "EUC-JP"}, nil
	case "JIS", "ISO2022JP":
		return codeset{name: "ISO-2022-JP"}, nil
	case "UTF16", "UTF16BE", "UTF16LE", "UTF16BEBOM", "UTF16LEBOM",
		"UTF32", "UTF32BE", "UTF32LE", "UTF32BEBOM", "UTF32LEBOM":
		bom := strings.HasSuffix(normalized, "BOM")
		normalized = strings.TrimSuffix(normalized, "BOM")
		if len(normalized) == 5 {
			normalized += "BE"
		}
		return codeset{name: normalized[:3] + "-" + normalized[3:], bom: bom}, nil
	default:
		return codeset{}, fmt.Errorf("unsupported codeset %q", name)
	}
}

func (c codeset) codec() encoding.Encoding {
	switch c.name {
	case "Shift_JIS":
		return japanese.ShiftJIS
	case "EUC-JP":
		return japanese.EUCJP
	case "ISO-2022-JP":
		return japanese.ISO2022JP
	case "UTF-16BE":
		return textunicode.UTF16(textunicode.BigEndian, textunicode.IgnoreBOM)
	case "UTF-16LE":
		return textunicode.UTF16(textunicode.LittleEndian, textunicode.IgnoreBOM)
	case "UTF-32BE":
		return utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM)
	case "UTF-32LE":
		return utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM)
	default:
		return nil
	}
}

func (c codeset) marker() []byte {
	switch c.name {
	case "UTF-8":
		return []byte{0xef, 0xbb, 0xbf}
	case "UTF-16BE":
		return []byte{0xfe, 0xff}
	case "UTF-16LE":
		return []byte{0xff, 0xfe}
	case "UTF-32BE":
		return []byte{0, 0, 0xfe, 0xff}
	case "UTF-32LE":
		return []byte{0xff, 0xfe, 0, 0}
	default:
		return nil
	}
}

func detectBOM(data []byte) (codeset, bool) {
	// Check UTF-32 before UTF-16, since their little-endian BOMs overlap.
	for _, name := range []string{"UTF-32BE", "UTF-32LE", "UTF-16BE", "UTF-16LE", "UTF-8"} {
		c := codeset{name: name, bom: true}
		if bytes.HasPrefix(data, c.marker()) {
			return c, true
		}
	}
	return codeset{}, false
}

func detectCodeset(data []byte) (codeset, error) {
	if c, ok := detectBOM(data); ok {
		return c, nil
	}
	for _, escape := range []string{"\x1b$@", "\x1b$B", "\x1b$(D", "\x1b(I", "\x1b(J"} {
		if bytes.Contains(data, []byte(escape)) {
			return codeset{name: "ISO-2022-JP"}, nil
		}
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return codeset{}, errors.New("cannot detect encoding: use --ic for BOM-less UTF-16/32 input")
	}
	if utf8.Valid(data) {
		for _, b := range data {
			if b >= utf8.RuneSelf {
				return codeset{name: "UTF-8"}, nil
			}
		}
		return codeset{name: "ASCII"}, nil
	}

	// Legacy Japanese encodings overlap. Prefer the valid decoding with more
	// full-width Japanese text; callers can use --ic to resolve ambiguity.
	var best codeset
	bestScore := 0
	for _, name := range []string{"EUC-JP", "Shift_JIS"} {
		candidate := codeset{name: name}
		decoded, err := decodeText(data, candidate)
		if err != nil {
			continue
		}
		score := japaneseScore(decoded)
		if best.name == "" || score > bestScore {
			best, bestScore = candidate, score
		}
	}
	if best.name == "" {
		return codeset{}, errors.New("cannot detect encoding: specify --ic")
	}
	return best, nil
}

func japaneseScore(data []byte) int {
	score := 0
	for _, r := range string(data) {
		switch {
		case r >= 0x3041 && r <= 0x3096: // Hiragana
			score += 4
		case r >= 0x30a1 && r <= 0x30fa: // Katakana
			score += 3
		case r >= 0x4e00 && r <= 0x9fff: // CJK ideographs
			score += 2
		case r >= 0xff61 && r <= 0xff9f: // Half-width kana
			score++
		}
	}
	return score
}

func decodeText(data []byte, c codeset) ([]byte, error) {
	if marker := c.marker(); len(marker) > 0 {
		data = bytes.TrimPrefix(data, marker)
	}
	if c.name == "UTF-8" || c.name == "ASCII" {
		if !utf8.Valid(data) {
			return nil, errors.New("invalid UTF-8 input")
		}
		if c.name == "ASCII" && !isASCII(data) {
			return nil, errors.New("invalid ASCII input")
		}
		return data, nil
	}
	if strings.HasPrefix(c.name, "UTF-") {
		if err := validateUnicode(data, c); err != nil {
			return nil, err
		}
	}
	codec := c.codec()
	if codec == nil {
		return nil, fmt.Errorf("unsupported input encoding %q", c.name)
	}
	decoded, err := codec.NewDecoder().Bytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", c.name, err)
	}
	// Japanese decoders replace invalid sequences instead of returning errors.
	if !strings.HasPrefix(c.name, "UTF-") && bytes.Contains(decoded, []byte("\ufffd")) {
		return nil, fmt.Errorf("invalid %s input", c.name)
	}
	return decoded, nil
}

func validateUnicode(data []byte, c codeset) error {
	var order binary.ByteOrder = binary.BigEndian
	if strings.HasSuffix(c.name, "LE") {
		order = binary.LittleEndian
	}
	unit := 2
	if strings.HasPrefix(c.name, "UTF-32") {
		unit = 4
	}
	invalid := fmt.Errorf("invalid %s input", c.name)
	if len(data)%unit != 0 {
		return invalid
	}
	for i := 0; i < len(data); i += unit {
		if unit == 4 {
			value := order.Uint32(data[i:])
			if value > utf8.MaxRune || value >= 0xd800 && value <= 0xdfff {
				return invalid
			}
			continue
		}
		value := order.Uint16(data[i:])
		if value >= 0xd800 && value <= 0xdbff {
			i += 2
			if i == len(data) {
				return invalid
			}
			low := order.Uint16(data[i:])
			if low < 0xdc00 || low > 0xdfff {
				return invalid
			}
		} else if value >= 0xdc00 && value <= 0xdfff {
			return invalid
		}
	}
	return nil
}

func encodeText(data []byte, c codeset) ([]byte, error) {
	var output []byte
	if c.name == "UTF-8" || c.name == "ASCII" {
		if c.name == "ASCII" && !isASCII(data) {
			return nil, errors.New("output contains non-ASCII characters")
		}
		output = data
	} else {
		codec := c.codec()
		if codec == nil {
			return nil, fmt.Errorf("unsupported output encoding %q", c.name)
		}
		var err error
		output, err = codec.NewEncoder().Bytes(data)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", c.name, err)
		}
	}
	if c.bom {
		output = append(c.marker(), output...)
	}
	return output, nil
}

func isASCII(data []byte) bool {
	for _, b := range data {
		if b >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
