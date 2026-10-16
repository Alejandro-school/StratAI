package handlers

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"cs2-demo-service/models"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
)

type manifestFixtureBits struct {
	data []byte
	pos  int
}

func (b *manifestFixtureBits) put(v uint32, n int) {
	for i := 0; i < n; i++ {
		if b.pos/8 == len(b.data) {
			b.data = append(b.data, 0)
		}
		b.data[b.pos/8] |= byte((v>>i)&1) << (b.pos % 8)
		b.pos++
	}
}
func (b *manifestFixtureBits) str(s string) {
	for _, v := range []byte(s) {
		b.put(uint32(v), 8)
	}
	b.put(0, 8)
}
func manifestFrame(payload []byte, compressed bool) []byte {
	b := manifestFixtureBits{}
	if compressed {
		b.put(1, 1)
	} else {
		b.put(0, 1)
	}
	b.put(uint32(len(payload)), 24)
	for _, v := range payload {
		b.put(uint32(v), 8)
	}
	return b.data
}
func manifestFixture() []byte {
	b := manifestFixtureBits{}
	b.put(1, 16)
	b.put(1, 16)
	b.put(2, 16)
	b.str("vmdl")
	b.str("models/")
	for i := 0; i < 2; i++ {
		b.put(0, 1)
		b.str("fixture")
		b.put(0, 1)
	}
	return b.data
}
func manifestLiteralLZSS(payload []byte) []byte {
	out := append([]byte("LZSS"), 0, 0, 0, 0)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(payload)))
	for len(payload) >= 8 {
		out = append(out, 0)
		out = append(out, payload[:8]...)
		payload = payload[8:]
	}
	out = append(out, byte(1<<len(payload)))
	out = append(out, payload...)
	return append(out, 0, 0)
}

func TestResourceManifestBothFramesPreserveDuplicatesAndHashes(t *testing.T) {
	payload := manifestFixture()
	for _, compressed := range []bool{false, true} {
		encoded := payload
		if compressed {
			encoded = manifestLiteralLZSS(payload)
		}
		row := decodeEconomyManifest(manifestFrame(encoded, compressed), 7, 100)
		if row.Status != "decoded" || row.Sequence != 7 || row.Tick != 100 {
			t.Fatalf("decode: %+v", row)
		}
		d := row.Decoded
		if d == nil || len(d.Entries) != 2 || len(d.TypeNameHashes) != 1 || len(d.DirectoryNameHashes) != 1 {
			t.Fatal("table count")
		}
		digest := *opaqueEconomyField([]byte("models/fixture.vmdl")).SHA256
		if d.Entries[0].ReferenceNameSHA256 != digest || d.Entries[1].ReferenceNameSHA256 != digest || d.Entries[1].Ordinal != 1 {
			t.Fatal("lost native duplicate or name hash")
		}
		if d.DecodedSHA256 != *opaqueEconomyField(payload).SHA256 || d.OuterPaddingBits != 7 || d.InnerPaddingBits != 4 {
			t.Fatal("consumption or hash")
		}
	}
}

func TestResourceManifestLZSSOverlapAndBounds(t *testing.T) {
	// One literal zero, distance=1/count=5 (overlapping copy), end marker.
	valid := append([]byte("LZSS"), 6, 0, 0, 0, 6, 0, 0, 4, 0, 0)
	if d, e := decodeManifest(manifestFrame(valid, true)); e != nil || d.DecodedBytes != 6 {
		t.Fatal("overlap", e)
	}
	for _, change := range []func([]byte){
		func(b []byte) { b[0] = 'X' }, func(b []byte) { b[4] = 7 }, func(b []byte) { b[4] = 5 },
		func(b []byte) { b[7] = 127 }, func(b []byte) { b[11] = 0x14 }, func(b []byte) { b[11] = 15 },
	} {
		b := bytes.Clone(valid)
		change(b)
		if _, e := decodeManifest(manifestFrame(b, true)); e == nil {
			t.Fatal("corrupt LZSS accepted")
		}
	}
	if _, e := decodeManifest(manifestFrame(valid[:len(valid)-1], true)); e == nil {
		t.Fatal("missing terminator")
	}
}

func TestResourceManifestRejectsTruncationPaddingAndIndices(t *testing.T) {
	payload := manifestFixture()
	valid := manifestFrame(payload, false)
	for i := 0; i < len(valid); i++ {
		if _, e := decodeManifest(valid[:i]); e == nil {
			t.Fatalf("truncation %d accepted", i)
		}
	}
	bad := bytes.Clone(valid)
	bad[len(bad)-1] |= 128
	if _, e := decodeManifest(bad); e == nil {
		t.Fatal("outer padding")
	}
	bad = append(bytes.Clone(valid), 0)
	if _, e := decodeManifest(bad); e == nil {
		t.Fatal("trailing byte")
	}
	bad = bytes.Clone(payload)
	bad[len(bad)-1] |= 128
	if _, e := decodeManifest(manifestFrame(bad, false)); e == nil {
		t.Fatal("inner padding")
	}
	bad = bytes.Clone(payload)
	bad[19] |= 1 // first entry directory index after 6+5+8 bytes
	if _, e := decodeManifest(manifestFrame(bad, false)); e == nil {
		t.Fatal("invalid directory index")
	}
	bad = bytes.Clone(payload)
	bad[0] = 0
	bad[1] = 32
	if _, e := decodeManifest(manifestFrame(bad, false)); e == nil {
		t.Fatal("table limit")
	}
	b := manifestFixtureBits{}
	b.put(1, 16)
	b.put(0, 16)
	b.put(0, 16)
	b.str(strings.Repeat("x", 4097))
	if _, e := decodeManifest(manifestFrame(b.data, false)); e == nil {
		t.Fatal("string limit")
	}
	bad = bytes.Clone(payload)
	bad[6] = 255
	if _, e := decodeManifest(manifestFrame(bad, false)); e == nil {
		t.Fatal("invalid UTF8")
	}
}

func TestResourceManifestPresenceAndSharedSequence(t *testing.T) {
	if decodeEconomyManifest(nil, 1, 0).Status != "absent" || decodeEconomyManifest([]byte{}, 1, 0).Status != "malformed_or_unsupported" || decodeEconomyManifest(make([]byte, manifestMaxBytes+1), 1, 0).Status != "read_limit_exceeded" {
		t.Fatal("presence/limit")
	}
	tt := newEconomyRuleTracker()
	rows := []models.EconomyManifestSession{}
	tt.manifests = &rows
	tt.convars(&msg.CNETMsg_SetConVar{}, 0)
	tt.session(&msg.CSVCMsg_ServerInfo{GameSessionManifest: manifestFrame(manifestFixture(), false)}, 0)
	if rows[0].Sequence != 2 || !reflect.DeepEqual(rows[0].Source, tt.ledger.Sessions[0].Manifest) {
		t.Fatal("native provenance")
	}
}

func FuzzResourceManifestNeverLeaksPartialDecode(f *testing.F) {
	f.Add(manifestFrame(manifestFixture(), false))
	f.Add(manifestFrame(manifestLiteralLZSS(manifestFixture()), true))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		row := decodeEconomyManifest(data, 1, 0)
		if (row.Status == "decoded") != (row.Decoded != nil) {
			t.Fatal("partial output")
		}
	})
}
