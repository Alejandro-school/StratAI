package handlers

import (
	"errors"
	"math/bits"
	"unicode/utf8"

	"cs2-demo-service/models"
)

// Source: Clarity 7fb3f1d07564a12efa99194d45cfbf5762ba5910,
// Resources.addManifestData and LZSS.unpack. See the pinned sources and BSD
// notice in docs/references/clarity-manifest-codec.txt. This stricter reader
// rejects partial output, invalid indices, oversized buffers and trailing data.
const manifestMaxBytes = 1 << 20
const manifestMaxItems = 4096
const manifestMaxString = 4096

type manifestBits struct {
	data []byte
	pos  int
}

func (b *manifestBits) read(n int) (uint32, error) {
	if n < 0 || n > 32 || n > len(b.data)*8-b.pos {
		return 0, errors.New("truncated_bits")
	}
	var value uint32
	for i := 0; i < n; i++ {
		value |= uint32((b.data[b.pos/8]>>(b.pos%8))&1) << i
		b.pos++
	}
	return value, nil
}

func (b *manifestBits) str() (string, error) {
	out := make([]byte, 0)
	for len(out) <= manifestMaxString {
		v, e := b.read(8)
		if e != nil {
			return "", e
		}
		if v == 0 {
			if !utf8.Valid(out) {
				return "", errors.New("invalid_utf8")
			}
			return string(out), nil
		}
		out = append(out, byte(v))
	}
	return "", errors.New("string_limit")
}

func (b *manifestBits) padding() (int, error) {
	n := len(b.data)*8 - b.pos
	if n > 7 {
		return 0, errors.New("trailing_data")
	}
	v, e := b.read(n)
	if e != nil || v != 0 {
		return 0, errors.New("nonzero_padding")
	}
	return n, nil
}

func manifestLZSS(b *manifestBits) ([]byte, error) {
	magic, e := b.read(32)
	if e != nil || magic != 0x53535a4c {
		return nil, errors.New("lzss_magic")
	}
	size, e := b.read(32)
	if e != nil || size > manifestMaxBytes {
		return nil, errors.New("decoded_size_limit")
	}
	out := make([]byte, 0, int(size))
	for {
		cmd, e := b.read(8)
		if e != nil {
			return nil, e
		}
		for i := 0; i < 8; i++ {
			if cmd&(1<<i) == 0 {
				v, e := b.read(8)
				if e != nil || len(out) >= int(size) {
					return nil, errors.New("literal_overflow_or_truncated")
				}
				out = append(out, byte(v))
			} else {
				a, e := b.read(8)
				if e != nil {
					return nil, e
				}
				c, e := b.read(8)
				if e != nil {
					return nil, e
				}
				distance, count := int(1+(a<<4|c>>4)), int(1+(c&15))
				if count == 1 {
					if len(out) != int(size) {
						return nil, errors.New("decoded_size_mismatch")
					}
					return out, nil
				}
				if distance > len(out) || count > int(size)-len(out) {
					return nil, errors.New("invalid_backreference")
				}
				for j := 0; j < count; j++ {
					out = append(out, out[len(out)-distance])
				}
			}
		}
	}
}

func decodeManifest(data []byte) (*models.EconomyManifestDecoded, error) {
	if len(data) > manifestMaxBytes {
		return nil, errors.New("encoded_size_limit")
	}
	b := &manifestBits{data: data}
	flag, e := b.read(1)
	if e != nil {
		return nil, e
	}
	size, e := b.read(24)
	if e != nil || size > manifestMaxBytes {
		return nil, errors.New("encoded_size_limit")
	}
	// The 24-bit size describes the encoded payload. Its meaning on the
	// compressed branch is independently checked against the build-10896 fixture.
	if int(size) != (len(data)*8-25)/8 {
		return nil, errors.New("encoded_size_mismatch")
	}
	var out []byte
	compression := "none"
	if flag == 1 {
		compression = "lzss"
		out, e = manifestLZSS(b)
		if e != nil {
			return nil, e
		}
	} else {
		out = make([]byte, int(size))
		for i := range out {
			v, e := b.read(8)
			if e != nil {
				return nil, e
			}
			out[i] = byte(v)
		}
	}
	consumed := b.pos
	padding, e := b.padding()
	if e != nil {
		return nil, e
	}
	inner := &manifestBits{data: out}
	nt, e := inner.read(16)
	if e != nil {
		return nil, e
	}
	nd, e := inner.read(16)
	if e != nil {
		return nil, e
	}
	ne, e := inner.read(16)
	if e != nil {
		return nil, e
	}
	if nt > manifestMaxItems || nd > manifestMaxItems || ne > manifestMaxItems || (ne > 0 && (nt == 0 || nd == 0)) {
		return nil, errors.New("table_limit_or_empty_index")
	}
	extensions, dirs := make([]string, int(nt)), make([]string, int(nd))
	result := &models.EconomyManifestDecoded{Compression: compression, EncodedBytes: int(size), DecodedBytes: len(out), DecodedSHA256: *opaqueEconomyField(out).SHA256, OuterConsumedBits: consumed, OuterPaddingBits: padding, TypeNameHashes: []string{}, DirectoryNameHashes: []string{}, Entries: []models.EconomyManifestEntry{}}
	for i := range extensions {
		extensions[i], e = inner.str()
		if e != nil {
			return nil, e
		}
		result.TypeNameHashes = append(result.TypeNameHashes, *opaqueEconomyField([]byte(extensions[i])).SHA256)
	}
	for i := range dirs {
		dirs[i], e = inner.str()
		if e != nil {
			return nil, e
		}
		result.DirectoryNameHashes = append(result.DirectoryNameHashes, *opaqueEconomyField([]byte(dirs[i])).SHA256)
	}
	for i := 0; i < int(ne); i++ {
		di, e := inner.read(max(1, bits.Len32(nd-1)))
		if e != nil {
			return nil, e
		}
		name, e := inner.str()
		if e != nil {
			return nil, e
		}
		ti, e := inner.read(max(1, bits.Len32(nt-1)))
		if e != nil {
			return nil, e
		}
		if di >= nd || ti >= nt {
			return nil, errors.New("invalid_table_index")
		}
		digest := *opaqueEconomyField([]byte(dirs[di] + name + "." + extensions[ti])).SHA256
		result.Entries = append(result.Entries, models.EconomyManifestEntry{Ordinal: i, DirectoryIndex: int(di), TypeIndex: int(ti), ReferenceNameSHA256: digest})
	}
	result.InnerConsumedBits = inner.pos
	result.InnerPaddingBits, e = inner.padding()
	if e != nil {
		return nil, e
	}
	return result, nil
}

func decodeEconomyManifest(data []byte, sequence, tick int) models.EconomyManifestSession {
	row := models.EconomyManifestSession{Sequence: sequence, Tick: tick, SourcePath: "CSVCMsg_ServerInfo.game_session_manifest", Source: opaqueEconomyField(data), Status: "absent"}
	if data == nil {
		return row
	}
	row.Status = "malformed_or_unsupported"
	if len(data) > manifestMaxBytes {
		row.Status = "read_limit_exceeded"
		return row
	}
	decoded, err := decodeManifest(data)
	if err == nil {
		row.Status = "decoded"
		row.Decoded = decoded
	}
	return row
}

func RegisterEconomyManifestHandler(ctx *models.DemoContext) {
	ctx.EconomyManifestSessions = &[]models.EconomyManifestSession{}
	registerEconomyRuleContext(ctx, true)
}
