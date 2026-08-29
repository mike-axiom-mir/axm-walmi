package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Witness struct {
	Schema              string   `json:"schema"`
	WitnessID           string   `json:"witnessId"`
	Implementation      string   `json:"implementation"`
	SourceReceiptDigest string   `json:"sourceReceiptDigest"`
	EvidenceViewDigest  string   `json:"evidenceViewDigest"`
	Verdict             string   `json:"verdict"`
	Findings            []string `json:"findings"`
	Authority           string   `json:"authority"`
	ReceiptDigest       string   `json:"receiptDigest,omitempty"`
}

func canon(v any) ([]byte, error) {
	switch x := v.(type) {
	case nil:
		return []byte("null"), nil
	case bool:
		if x {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	case string:
		return json.Marshal(x)
	case json.Number:
		s := x.String()
		if strings.ContainsAny(s, ".eE") {
			return nil, errors.New("non-integer")
		}
		n, e := strconv.ParseInt(s, 10, 64)
		if e != nil {
			return nil, e
		}
		return []byte(strconv.FormatInt(n, 10)), nil
	case []any:
		var b bytes.Buffer
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			z, err := canon(e)
			if err != nil {
				return nil, err
			}
			b.Write(z)
		}
		b.WriteByte(']')
		return b.Bytes(), nil
	case map[string]any:
		ks := make([]string, 0, len(x))
		for k := range x {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		var b bytes.Buffer
		b.WriteByte('{')
		for i, k := range ks {
			if i > 0 {
				b.WriteByte(',')
			}
			q, _ := json.Marshal(k)
			b.Write(q)
			b.WriteByte(':')
			z, err := canon(x[k])
			if err != nil {
				return nil, err
			}
			b.Write(z)
		}
		b.WriteByte('}')
		return b.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported %T", v)
	}
}
func digestMap(m map[string]any, omit string) (string, error) {
	c := map[string]any{}
	for k, v := range m {
		if k != omit {
			c[k] = v
		}
	}
	b, e := canon(c)
	if e != nil {
		return "", e
	}
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:]), nil
}
func fileDigest(p string) (string, error) {
	b, e := os.ReadFile(p)
	if e != nil {
		return "", e
	}
	s := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(s[:]), nil
}
func main() {
	if len(os.Args) != 3 {
		panic("usage witness_a source.json output.json")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var r map[string]any
	if err = dec.Decode(&r); err != nil {
		panic(err)
	}
	got, _ := r["receiptDigest"].(string)
	exp, err := digestMap(r, "receiptDigest")
	if err != nil || got != exp {
		panic("source receipt digest mismatch")
	}
	if r["schema"] != "axm.waldo-mirror.v0.13-generated-peer-audit-ring/v0.1" || r["status"] != "OBSERVED_DISSENT_PRESERVED" || r["authority"] != "NONE" {
		panic("source boundary mismatch")
	}
	peer := r["peerRing"].(map[string]any)
	if peer["auditExecutions"].(json.Number).String() != "6" || peer["evidencePasses"].(json.Number).String() != "4" || peer["holds"].(json.Number).String() != "2" || peer["independentWitnessSupplied"] != false {
		panic("peer ring topology mismatch")
	}
	truth := r["truth"].(map[string]any)
	if truth["canonChanged"] != false || truth["quorumAuthority"] != false || truth["targetCodeMutated"] != false {
		panic("truth boundary mismatch")
	}
	view, err := fileDigest(os.Args[1])
	if err != nil {
		panic(err)
	}
	w := Witness{Schema: "axm.waldo-independent-witness/v0.14", WitnessID: "witness-a-go-structural", Implementation: "go-structural-v1", SourceReceiptDigest: got, EvidenceViewDigest: view, Verdict: "EVIDENCE_PASS", Findings: []string{}, Authority: "NONE"}
	raw2, _ := json.Marshal(w)
	var wm map[string]any
	dec2 := json.NewDecoder(bytes.NewReader(raw2))
	dec2.UseNumber()
	_ = dec2.Decode(&wm)
	d, err := digestMap(wm, "receiptDigest")
	if err != nil {
		panic(err)
	}
	w.ReceiptDigest = d
	out, _ := json.MarshalIndent(w, "", "  ")
	out = append(out, '\n')
	if err = os.WriteFile(os.Args[2], out, 0644); err != nil {
		panic(err)
	}
}
