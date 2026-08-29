package axmmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	CapabilityChoiceV017Schema        = "axm.capability-choice-contract/v0.17"
	CapabilityChoiceV017SourceReceipt = "sha256:7e8188eb87772ae6ae47ab67bd1dbf158008513ae43c63da680cfb388ca5f81d"
	CapabilityChoiceV017SourceHead    = "16900d91e60cde5c74bcfe0d51a209d508683742"
	CapabilityChoiceV017Need          = "material-evidence-view-completion"
	CapabilityChoiceV017Authority     = "READ_ONLY_PUBLIC_ARTIFACT_ACCESS"
	CapabilityChoiceV017Output        = "material-evidence-view-verified"
)

type CapabilityChoiceV017Link struct {
	ID        string   `json:"id"`
	Kind      string   `json:"kind"`
	Consumes  string   `json:"consumes"`
	Provides  string   `json:"provides"`
	Authority []string `json:"authority"`
}

type CapabilityChoiceV017Chain struct {
	ID    string                     `json:"id"`
	Links []CapabilityChoiceV017Link `json:"links"`
}

type CapabilityChoiceV017Contract struct {
	Schema                        string                      `json:"schema"`
	Status                        string                      `json:"status"`
	SourceReceipt                 string                      `json:"sourceReceipt"`
	SourceHead                    string                      `json:"sourceHead"`
	Need                          string                      `json:"need"`
	AuthorityCeiling              []string                    `json:"authorityCeiling"`
	Candidates                    []CapabilityChoiceV017Chain `json:"candidates"`
	AdvisoryPriority              map[string]*int             `json:"advisoryPriority"`
	SelectionAuthority            string                      `json:"selectionAuthority"`
	Decision                      string                      `json:"decision"`
	Reason                        string                      `json:"reason"`
	SingleSurvivorDecision        string                      `json:"singleSurvivorDecision"`
	SingleSurvivorReason          string                      `json:"singleSurvivorReason"`
	CycleEdges                    [][]string                  `json:"cycleEdges"`
	CycleDecision                 string                      `json:"cycleDecision"`
	CycleReason                   string                      `json:"cycleReason"`
	RefusedAuthority              []string                    `json:"refusedAuthority"`
	RefusalDecision               string                      `json:"refusalDecision"`
	RefusalReason                 string                      `json:"refusalReason"`
	SelectionPerformed            bool                        `json:"selectionPerformed"`
	BuildStarted                  bool                        `json:"buildStarted"`
	Promoted                      bool                        `json:"promoted"`
	CanonChanged                  bool                        `json:"canonChanged"`
	WaldoOwnsSelection            bool                        `json:"waldoOwnsSelection"`
	LocalCapabilityFabricExecuted bool                        `json:"localCapabilityFabricExecuted"`
	ReceiptDigest                 string                      `json:"receiptDigest"`
}

type CapabilityChoiceV017Witness struct {
	Schema             string   `json:"schema"`
	SourceReceipt      string   `json:"sourceReceipt"`
	SourceHead         string   `json:"sourceHead"`
	CandidateIDs       []string `json:"candidateIds"`
	CandidateSetDigest string   `json:"candidateSetDigest"`
	Decision           string   `json:"decision"`
	Reason             string   `json:"reason"`
	SelectionAuthority string   `json:"selectionAuthority"`
	WitnessDigest      string   `json:"witnessDigest,omitempty"`
}

func VerifyCapabilityChoiceV017(data []byte) (CapabilityChoiceV017Contract, error) {
	var c CapabilityChoiceV017Contract
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, fmt.Errorf("decode v0.17 choice contract: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("v0.17 choice contract has trailing JSON")
		}
		return CapabilityChoiceV017Contract{}, err
	}
	if err := c.Validate(); err != nil {
		return CapabilityChoiceV017Contract{}, err
	}
	expected, err := capChoiceV017ExternalDigest(c)
	if err != nil {
		return CapabilityChoiceV017Contract{}, err
	}
	if c.ReceiptDigest != expected {
		return CapabilityChoiceV017Contract{}, fmt.Errorf("v0.17 receipt mismatch: expected %s got %s", expected, c.ReceiptDigest)
	}
	return c, nil
}

func (c CapabilityChoiceV017Contract) Validate() error {
	if c.Schema != CapabilityChoiceV017Schema || c.Status != "CONTRACT_PROBE_ONLY" {
		return errors.New("v0.17 schema/status drifted")
	}
	if c.SourceReceipt != CapabilityChoiceV017SourceReceipt || c.SourceHead != CapabilityChoiceV017SourceHead || c.Need != CapabilityChoiceV017Need {
		return errors.New("v0.17 source binding drifted")
	}
	if !capChoiceV017SliceExact(c.AuthorityCeiling, []string{CapabilityChoiceV017Authority}) {
		return errors.New("v0.17 authority ceiling drifted")
	}
	if len(c.Candidates) != 2 {
		return errors.New("v0.17 requires exactly two preserved candidates")
	}
	seen := map[string]bool{}
	for _, chain := range c.Candidates {
		if seen[chain.ID] {
			return errors.New("v0.17 duplicate candidate")
		}
		seen[chain.ID] = true
		if err := capChoiceV017ValidateChain(chain); err != nil {
			return err
		}
	}
	if !seen["direct-read-chain"] || !seen["projected-read-chain"] {
		return errors.New("v0.17 candidate identities drifted")
	}
	if len(c.AdvisoryPriority) != 2 || c.AdvisoryPriority["direct-read-chain"] != nil || c.AdvisoryPriority["projected-read-chain"] != nil {
		return errors.New("v0.17 advisory facts became priority")
	}
	decision, reason := capChoiceV017EvaluateSet(c.Candidates, c.SelectionAuthority)
	if c.SelectionAuthority != "NONE" || c.Decision != decision || c.Reason != reason || decision != "HOLD" || reason != "MULTIPLE_VALID_COMPOSITIONS" {
		return errors.New("v0.17 multiple-candidate decision drifted")
	}
	survivor := []CapabilityChoiceV017Chain{capChoiceV017Find(c.Candidates, "direct-read-chain")}
	sd, sr := capChoiceV017EvaluateSet(survivor, c.SelectionAuthority)
	if c.SingleSurvivorDecision != sd || c.SingleSurvivorReason != sr || sd != "HOLD" || sr != "SELECTION_AUTHORITY_ABSENT" {
		return errors.New("v0.17 single-survivor authority drifted")
	}
	if !capChoiceV017HasCycle(c.CycleEdges) || c.CycleDecision != "HOLD" || c.CycleReason != "CYCLE_DETECTED" {
		return errors.New("v0.17 cycle evidence drifted")
	}
	for _, required := range []string{"WRITE_WORKSPACE", "INSTALL_CAPABILITY", "PROMOTE", "CANON"} {
		if !capChoiceV017Contains(c.RefusedAuthority, required) {
			return fmt.Errorf("v0.17 refusal missing %s", required)
		}
	}
	if c.RefusalDecision != "REFUSED" || c.RefusalReason != "CHILD_AUTHORITY_EXCEEDS_ANCESTOR_GAP" {
		return errors.New("v0.17 authority refusal drifted")
	}
	if c.SelectionPerformed || c.BuildStarted || c.Promoted || c.CanonChanged || c.WaldoOwnsSelection || c.LocalCapabilityFabricExecuted {
		return errors.New("v0.17 truth/authority boundary drifted")
	}
	if !capChoiceV017ValidSHA(c.ReceiptDigest) {
		return errors.New("v0.17 receipt must be sha256:<64 hex>")
	}
	return nil
}

func capChoiceV017ValidateChain(c CapabilityChoiceV017Chain) error {
	expected := map[string][]CapabilityChoiceV017Link{
		"direct-read-chain": {
			{ID: "request-skill", Kind: "SKILL", Consumes: CapabilityChoiceV017Need, Provides: "bounded-request"},
			{ID: "reader-hand", Kind: "HAND", Consumes: "bounded-request", Provides: "material-view", Authority: []string{CapabilityChoiceV017Authority}},
			{ID: "verify-organ", Kind: "ORGAN", Consumes: "material-view", Provides: CapabilityChoiceV017Output},
		},
		"projected-read-chain": {
			{ID: "request-skill", Kind: "SKILL", Consumes: CapabilityChoiceV017Need, Provides: "bounded-request"},
			{ID: "projection-organ", Kind: "ORGAN", Consumes: "bounded-request", Provides: "projected-request"},
			{ID: "projection-reader-hand", Kind: "HAND", Consumes: "projected-request", Provides: "material-view", Authority: []string{CapabilityChoiceV017Authority}},
			{ID: "verify-organ", Kind: "ORGAN", Consumes: "material-view", Provides: CapabilityChoiceV017Output},
		},
	}
	want, ok := expected[c.ID]
	if !ok || len(c.Links) != len(want) {
		return fmt.Errorf("v0.17 chain %s shape drifted", c.ID)
	}
	for i := range want {
		got := c.Links[i]
		exp := want[i]
		if got.ID != exp.ID || got.Kind != exp.Kind || got.Consumes != exp.Consumes || got.Provides != exp.Provides || !capChoiceV017SliceExact(got.Authority, exp.Authority) {
			return fmt.Errorf("v0.17 chain %s link %d drifted", c.ID, i)
		}
		for _, a := range got.Authority {
			if a != CapabilityChoiceV017Authority {
				return fmt.Errorf("v0.17 child %s exceeds authority", got.ID)
			}
		}
	}
	if outcome, reason := capChoiceV017EvaluateChain(c); outcome != "PROPOSAL_COMPLETE" || reason != "" {
		return fmt.Errorf("v0.17 chain %s does not close", c.ID)
	}
	return nil
}

func capChoiceV017EvaluateChain(c CapabilityChoiceV017Chain) (string, string) {
	if len(c.Links) == 0 || c.Links[0].Consumes != CapabilityChoiceV017Need {
		return "HOLD", "MISSING_LINK"
	}
	for i := 1; i < len(c.Links); i++ {
		if c.Links[i].Consumes != c.Links[i-1].Provides {
			return "HOLD", "MISSING_LINK"
		}
	}
	if c.Links[len(c.Links)-1].Provides != CapabilityChoiceV017Output {
		return "HOLD", "MISSING_LINK"
	}
	return "PROPOSAL_COMPLETE", ""
}

func capChoiceV017EvaluateSet(c []CapabilityChoiceV017Chain, authority string) (string, string) {
	feasible := 0
	for _, x := range c {
		if o, r := capChoiceV017EvaluateChain(x); o == "PROPOSAL_COMPLETE" && r == "" {
			feasible++
		}
	}
	if feasible == 0 {
		return "HOLD", "NO_VALID_COMPOSITION"
	}
	if authority == "NONE" {
		if feasible > 1 {
			return "HOLD", "MULTIPLE_VALID_COMPOSITIONS"
		}
		return "HOLD", "SELECTION_AUTHORITY_ABSENT"
	}
	return "PROPOSAL_READY_FOR_EXTERNAL_SELECTION", ""
}

func capChoiceV017HasCycle(edges [][]string) bool {
	g := map[string][]string{}
	for _, e := range edges {
		if len(e) != 2 {
			return false
		}
		g[e[0]] = append(g[e[0]], e[1])
	}
	state := map[string]uint8{}
	var visit func(string) bool
	visit = func(n string) bool {
		if state[n] == 1 {
			return true
		}
		if state[n] == 2 {
			return false
		}
		state[n] = 1
		for _, m := range g[n] {
			if visit(m) {
				return true
			}
		}
		state[n] = 2
		return false
	}
	for n := range g {
		if visit(n) {
			return true
		}
	}
	return false
}

func WitnessCapabilityChoiceV017(data []byte) (CapabilityChoiceV017Witness, error) {
	c, err := VerifyCapabilityChoiceV017(data)
	if err != nil {
		return CapabilityChoiceV017Witness{}, err
	}
	ids := []string{}
	for _, x := range c.Candidates {
		ids = append(ids, x.ID)
	}
	sort.Strings(ids)
	setDigest, err := capChoiceV017CandidateSetDigest(c.Candidates)
	if err != nil {
		return CapabilityChoiceV017Witness{}, err
	}
	w := CapabilityChoiceV017Witness{Schema: "axm.waldo-witness.capability-choice-contract/v0.17", SourceReceipt: c.SourceReceipt, SourceHead: c.SourceHead, CandidateIDs: ids, CandidateSetDigest: setDigest, Decision: c.Decision, Reason: c.Reason, SelectionAuthority: c.SelectionAuthority}
	raw, err := json.Marshal(w)
	if err != nil {
		return CapabilityChoiceV017Witness{}, err
	}
	sum := sha256.Sum256(raw)
	w.WitnessDigest = hex.EncodeToString(sum[:])
	return w, nil
}

func SealCapabilityChoiceV017(c CapabilityChoiceV017Contract) ([]byte, error) {
	d, e := capChoiceV017ExternalDigest(c)
	if e != nil {
		return nil, e
	}
	c.ReceiptDigest = d
	return json.MarshalIndent(c, "", "  ")
}
func capChoiceV017ExternalDigest(c CapabilityChoiceV017Contract) (string, error) {
	c.ReceiptDigest = ""
	raw, e := json.Marshal(c)
	if e != nil {
		return "", e
	}
	s := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(s[:]), nil
}
func capChoiceV017CandidateSetDigest(c []CapabilityChoiceV017Chain) (string, error) {
	x := append([]CapabilityChoiceV017Chain(nil), c...)
	sort.Slice(x, func(i, j int) bool { return x[i].ID < x[j].ID })
	raw, e := json.Marshal(x)
	if e != nil {
		return "", e
	}
	s := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(s[:]), nil
}
func capChoiceV017Find(c []CapabilityChoiceV017Chain, id string) CapabilityChoiceV017Chain {
	for _, x := range c {
		if x.ID == id {
			return x
		}
	}
	return CapabilityChoiceV017Chain{}
}
func capChoiceV017Contains(xs []string, w string) bool {
	for _, x := range xs {
		if x == w {
			return true
		}
	}
	return false
}
func capChoiceV017SliceExact(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func capChoiceV017ValidSHA(v string) bool {
	if !strings.HasPrefix(v, "sha256:") || len(v) != 71 {
		return false
	}
	_, e := hex.DecodeString(strings.TrimPrefix(v, "sha256:"))
	return e == nil
}
