package handlers

import (
	"bytes"
	"cs2-demo-service/models"
	"encoding/binary"
	"encoding/json"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/msg"
	"hash/crc32"
	"strings"
	"testing"
)

func vbkvFixture(body []byte) []byte {
	out := append([]byte("VBKV"), make([]byte, 4)...)
	binary.LittleEndian.PutUint32(out[4:], crc32.ChecksumIEEE(body))
	return append(out, body...)
}

func kvInteger(key string, n int32) []byte {
	out := append([]byte{2}, []byte(key)...)
	out = append(out, 0)
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(n))
	return append(out, b...)
}

func kvText(key, value string) []byte { return []byte("\x01" + key + "\x00" + value + "\x00") }

func sessionFixture() []byte {
	body := []byte{0, 0}
	body = append(body, kvInteger("c_game_mode", 1)...)
	body = append(body, kvInteger("c_game_type", 0)...)
	body = append(body, kvInteger("numSlots", -1)...)
	body = append(body, kvText("ct_arms", "fixture-reference")...)
	body = append(body, kvText("t_arms", "")...)
	body = append(body, kvText("hostname", "private-host 192.0.2.1 participant-secret")...)
	body = append(body, 0)
	body = append(body, []byte("ct_models\x00")...)
	body = append(body, kvText("", "fixture-model")...)
	body = append(body, 11)
	body = append(body, kvInteger("mp_maxmoney", 123)...)
	return vbkvFixture(append(body, 11, 11))
}

func TestSessionVBKVFieldsPrivacyAndProvenance(t *testing.T) {
	raw := sessionFixture()
	out := decodeEconomySessionBlock(raw, "config", true)
	if out.Status != "decoded" || !out.ChecksumVerified || out.Nodes != 10 || out.Scalars != 8 || len(out.Fields) != 9 {
		t.Fatalf("bad decode: %+v", out)
	}
	if len(out.EconomicNamesSeen) != 1 || out.EconomicNamesSeen[0] != "mp_maxmoney" {
		t.Fatal("economic name scan failed")
	}
	got := map[string]models.EconomySessionField{}
	for _, f := range out.Fields {
		got[f.Path] = f
	}
	if *got["//c_game_type"].Value != 0 || *got["//numSlots"].Value != -1 || got["//default_game_type"].Present {
		t.Fatal("zero/negative/absence lost")
	}
	if got["//t_arms"].ReferenceNameSHA256 == nil || *got["//ct_arms"].ReferenceNameSHA256 != *opaqueEconomyField([]byte("fixture-reference")).SHA256 {
		t.Fatal("reference-name hash differs")
	}
	serialized, _ := json.Marshal(out)
	for _, private := range []string{"private-host", "192.0.2.1", "participant-secret", "hostname", "fixture-reference", "fixture-model"} {
		if bytes.Contains(serialized, []byte(private)) {
			t.Fatal("private source leaked")
		}
	}
	if *out.Source.SHA256 != *opaqueEconomyField(raw).SHA256 {
		t.Fatal("source hash differs")
	}
	raw[0] = 0
	if *got["//c_game_mode"].Value != 1 {
		t.Fatal("source mutation changed retained observation")
	}
}

func TestSessionVBKVRejectsMalformedAndBounds(t *testing.T) {
	valid := sessionFixture()
	cases := map[string][]byte{
		"short_header": []byte("VBKV"), "bad_crc": append([]byte{}, valid...),
		"truncated_int":    vbkvFixture([]byte{2, 'x', 0, 1}),
		"unterminated_key": vbkvFixture([]byte{1, 'x'}),
		"unknown_type":     vbkvFixture([]byte{9, 0, 11}),
		"bad_utf8":         vbkvFixture([]byte{1, 0, 255, 0, 11}),
		"trailing":         vbkvFixture([]byte{11, 0}),
		"old_end":          vbkvFixture([]byte{8}),
		"long_string":      vbkvFixture(append([]byte{1}, []byte(strings.Repeat("x", sessionMaxString+1))...)),
		"deep":             vbkvFixture(append(bytes.Repeat([]byte{0, 0}, sessionMaxDepth+2), bytes.Repeat([]byte{11}, sessionMaxDepth+3)...)),
		"nodes":            vbkvFixture(append(bytes.Repeat([]byte{1, 0, 0}, sessionMaxNodes+1), 11)),
	}
	cases["bad_crc"][4] ^= 1
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			out := decodeEconomySessionBlock(raw, "config", true)
			if out.Status != "malformed" || len(out.Fields) != 0 || out.Nodes != 0 || out.ChecksumVerified {
				t.Fatalf("partial/malformed output accepted: %+v", out)
			}
		})
	}
	for n := 4; n < len(valid); n++ {
		if decodeEconomySessionBlock(valid[:n], "config", true).Status == "decoded" {
			t.Fatalf("truncation accepted at %d", n)
		}
	}
}

func TestSessionRepeatedFieldsPreserveNativeOrder(t *testing.T) {
	body := []byte{0, 0}
	body = append(body, kvInteger("numSlots", 1)...)
	body = append(body, kvInteger("numSlots", 2)...)
	body = append(body, []byte("\x00ct_models\x00")...)
	body = append(body, kvText("", "first")...)
	body = append(body, kvText("", "second")...)
	out := decodeEconomySessionBlock(vbkvFixture(append(body, 11, 11, 11)), "config", true)
	if out.Status != "decoded" || len(out.Fields) != 11 || out.Nodes != 6 {
		t.Fatalf("repeated fields lost: %+v", out)
	}
	groups := map[string][]models.EconomySessionField{}
	for _, f := range out.Fields {
		groups[f.Path] = append(groups[f.Path], f)
	}
	slots := groups["//numSlots"]
	refs := groups["//ct_models/"]
	if len(slots) != 2 || *slots[0].Value != 1 || *slots[1].Value != 2 || slots[1].Occurrence != 1 || *slots[1].NodeIndex != 3 {
		t.Fatal("repeated integers collapsed")
	}
	if len(refs) != 2 || *refs[0].ReferenceNameSHA256 != *opaqueEconomyField([]byte("first")).SHA256 || *refs[1].ReferenceNameSHA256 != *opaqueEconomyField([]byte("second")).SHA256 || *refs[0].NodeIndex != 5 {
		t.Fatal("empty keys collapsed")
	}
}

func TestSessionAbsentUnknownLimitAndNativeType(t *testing.T) {
	if decodeEconomySessionBlock(nil, "config", true).Status != "absent" {
		t.Fatal("absence lost")
	}
	for _, raw := range [][]byte{{}, []byte("unknown"), {0x1, 0x33, 0x56, 0x4b}} {
		out := decodeEconomySessionBlock(raw, "config", true)
		if !out.Source.Present || out.Status != "unsupported_format" {
			t.Fatal("present vs absent lost")
		}
	}
	if decodeEconomySessionBlock(make([]byte, sessionMaxBytes+1), "config", true).Status != "read_limit_exceeded" {
		t.Fatal("size bound ignored")
	}
	// A VBKV-looking manifest is still unverified: no channel format was accredited.
	if decodeEconomySessionBlock(sessionFixture(), "manifest", false).Status != "unsupported_format" {
		t.Fatal("unverified channel decoded")
	}
	body := append([]byte{0, 0}, kvText("numSlots", "10")...)
	out := decodeEconomySessionBlock(vbkvFixture(append(body, 11, 11)), "config", true)
	if out.Status != "decoded" {
		t.Fatal("valid string rejected")
	}
	for _, f := range out.Fields {
		if f.Path == "//numSlots" && (f.Status != "native_type_unsupported" || f.Value != nil || *f.NativeType != 1) {
			t.Fatal("string coerced to number")
		}
	}
}

func TestSessionCallbackSequenceAndOptIn(t *testing.T) {
	tracker := newEconomyRuleTracker()
	tracker.session(&msg.CSVCMsg_ServerInfo{}, 0)
	if tracker.payloads != nil {
		t.Fatal("legacy path changed")
	}
	tracker.payloads = &models.EconomySessionPayloads{Sessions: []models.EconomySessionPayload{}}
	info := &msg.CSVCMsg_ServerInfo{GameSessionManifest: []byte{1, 2}, GameSessionConfig: &msg.CSVCMsg_GameSessionConfiguration{Data: sessionFixture()}}
	tracker.session(info, 0)
	tracker.convars(&msg.CNETMsg_SetConVar{}, 0)
	tracker.session(info, 2)
	first := tracker.payloads.Sessions[0]
	second := tracker.payloads.Sessions[1]
	if first.Sequence != 2 || second.Sequence != 4 || first.Tick != 0 || second.Tick != 2 {
		t.Fatal("callback provenance differs")
	}
	if first.Blocks[0].SourcePath != "CSVCMsg_ServerInfo.game_session_manifest" || first.Blocks[1].SourcePath != "CSVCMsg_ServerInfo.game_session_config.data" {
		t.Fatal("field source differs")
	}
	if *first.Blocks[1].Source.SHA256 != *tracker.ledger.Sessions[1].ConfigurationData.SHA256 {
		t.Fatal("context binding differs")
	}
}
