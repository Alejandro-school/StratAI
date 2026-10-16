package handlers

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"sort"
	"strings"
	"unicode/utf8"

	"cs2-demo-service/models"
)

// Bounded VBKV dialect, based on the pinned readers documented in the contract.
// No speculative offsets, decompressors or textual scans of opaque blocks.
const sessionMaxBytes = 65536
const sessionMaxNodes = 4096
const sessionMaxString = 4096
const sessionMaxDepth = 16

type sessionKVReader struct {
	data                []byte
	pos, nodes, scalars int
	fields              map[string][]models.EconomySessionField
	selected            map[string]string
	economic            map[string]bool
	seen                map[string]bool
}

func (r *sessionKVReader) take(n int) ([]byte, error) {
	if n < 0 || n > len(r.data)-r.pos {
		return nil, errors.New("truncated")
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b, nil
}

func (r *sessionKVReader) text() (string, error) {
	remaining := r.data[r.pos:]
	n := bytes.IndexByte(remaining[:min(len(remaining), sessionMaxString+1)], 0)
	if n < 0 {
		return "", errors.New("unterminated_or_oversized_string")
	}
	b, err := r.take(n + 1)
	if err != nil || !utf8.Valid(b[:n]) {
		return "", errors.New("invalid_utf8")
	}
	return string(b[:n]), nil
}

func (r *sessionKVReader) object(parent string, depth int) error {
	if depth > sessionMaxDepth {
		return errors.New("depth_limit")
	}
	for {
		raw, err := r.take(1)
		if err != nil {
			return err
		}
		typ := int(raw[0])
		if typ == 11 {
			return nil
		}
		r.nodes++
		if r.nodes > sessionMaxNodes {
			return errors.New("node_limit")
		}
		key, err := r.text()
		if err != nil {
			return err
		}
		path := parent + "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
		if r.economic[key] {
			r.seen[key] = true
		}
		field := models.EconomySessionField{Path: path, Occurrence: len(r.fields[path]), NodeIndex: intPointer(r.nodes), Present: true, NativeType: intPointer(typ), Status: "native_type_unsupported"}
		switch typ {
		case 0:
			if err = r.object(path, depth+1); err != nil {
				return err
			}
		case 1:
			value, e := r.text()
			if e != nil {
				return e
			}
			if r.selected[path] == "reference_name_sha256" {
				// This is a digest of the native reference name, NEVER asset content.
				field.ReferenceNameSHA256 = opaqueEconomyField([]byte(value)).SHA256
				field.Status = "observed_reference_name_hash"
			}
		case 2:
			value, e := r.take(4)
			if e != nil {
				return e
			}
			if typ == 2 && r.selected[path] == "int32" {
				n := int32(binary.LittleEndian.Uint32(value))
				field.Value = &n
				field.Status = "observed_native_integer"
			}
		default:
			return errors.New("unsupported_node_type")
		}
		if typ != 0 {
			r.scalars++
		}
		if _, wanted := r.selected[path]; wanted {
			r.fields[path] = append(r.fields[path], field)
		}
	}
}

func decodeEconomySessionBlock(data []byte, sourcePath string, interpret bool) models.EconomySessionBlock {
	out := models.EconomySessionBlock{SourcePath: sourcePath, Source: opaqueEconomyField(data), Status: "absent", Format: "unverified", Compression: "unverified", Fields: []models.EconomySessionField{}, EconomicNamesSeen: []string{}}
	if data == nil {
		return out
	}
	out.Status = "unsupported_format"
	if len(data) > sessionMaxBytes {
		out.Status = "read_limit_exceeded"
		return out
	}
	if !interpret || len(data) < 4 || string(data[:4]) != "VBKV" {
		return out
	}
	out.Format = "vbkv_alt_end_11"
	out.Compression = "none"
	out.Status = "malformed"
	if len(data) < 8 || binary.LittleEndian.Uint32(data[4:8]) != crc32.ChecksumIEEE(data[8:]) {
		return out
	}
	probe, _ := models.LoadEconomySessionPayloadProbe()
	economic, _ := models.LoadEconomyRuleProbe()
	r := sessionKVReader{data: data[8:], selected: probe.Fields, fields: map[string][]models.EconomySessionField{}, economic: map[string]bool{}, seen: map[string]bool{}}
	for _, name := range economic.Convars {
		r.economic[name] = true
	}
	if err := r.object("", 0); err != nil || r.pos != len(r.data) {
		return out
	}
	out.Status = "decoded"
	out.ChecksumVerified = true
	out.Nodes = r.nodes
	out.Scalars = r.scalars
	paths := []string{}
	for path := range probe.Fields {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		fields, ok := r.fields[path]
		if !ok {
			fields = []models.EconomySessionField{{Path: path, Status: "not_seen_in_decoded_tree"}}
		}
		out.Fields = append(out.Fields, fields...)
	}
	for name := range r.seen {
		out.EconomicNamesSeen = append(out.EconomicNamesSeen, name)
	}
	sort.Strings(out.EconomicNamesSeen)
	return out
}

func RegisterEconomySessionPayloadHandler(ctx *models.DemoContext) {
	registerEconomyRuleContext(ctx, true)
}
