package axmmirror

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	ProvenanceContextRequestSchema = "axm.waldo-witness.provenance-context-request/v0.1"
	ProvenanceContextSchema        = "axm.waldo-witness.provenance-context/v0.1"

	ContextStateReady    = "CONTEXT_READY"
	ContextStateJoinHold = "HOLD_CONTEXT_JOIN_MISMATCH"
	ContextStateSizeHold = "HOLD_CONTEXT_LIMIT"

	EvidenceStructurallyWitnessed = "STRUCTURALLY_WITNESSED"
	EvidenceRecordedAssertion     = "RECORDED_ASSERTION"
	EvidenceDeclared              = "DECLARED"
	EvidenceUnknown               = "UNKNOWN"
	EvidenceNotProvided           = "NOT_PROVIDED"

	MaxProvenanceFactBytes = 64 * 1024
)

var supportedProvenanceFields = map[string]struct{}{
	"answering.bom_sha256":                  {},
	"answering.identity_sha256":             {},
	"answering.model_id":                    {},
	"answering.name":                        {},
	"answering.run_id":                      {},
	"answering.source_id":                   {},
	"answering.source_type":                 {},
	"answering.subject":                     {},
	"corpus.bom_sha256":                     {},
	"corpus.index.commit":                   {},
	"corpus.index.state":                    {},
	"corpus.licenses":                       {},
	"corpus.paths":                          {},
	"corpus.shard_set_sha256":               {},
	"corpus.source_set_sha256":              {},
	"corpus.totals":                         {},
	"evaluation.authorship_state":           {},
	"evaluation.compared_dimensions":        {},
	"evaluation.comparison_sha256":          {},
	"evaluation.contamination_state":        {},
	"identity.lock_state":                   {},
	"training.architecture_sha256":          {},
	"training.backend":                      {},
	"training.profile.canonical":            {},
	"training.profile.data_order":           {},
	"training.profile.declared":             {},
	"training.profile.evaluation_selection": {},
	"training.profile.schema":               {},
	"training.profile.weight_set_sha256":    {},
	"training.run_bom_sha256":               {},
	"training.run_id":                       {},
	"training.simulated":                    {},
	"training.state":                        {},
}

// ProvenanceContextRequest contains verified receipts, not artifact bodies.
// Fields is an explicit allow-list and MaxFactBytes bounds the only material
// that may be exposed to a learned clone.
type ProvenanceContextRequest struct {
	Schema          string                   `json:"schema"`
	RequestID       string                   `json:"request_id"`
	Fields          []string                 `json:"fields"`
	MaxFactBytes    int                      `json:"max_fact_bytes"`
	Anchor          OriginAnchor             `json:"anchor"`
	Corpus          CorpusEvidenceLens       `json:"corpus"`
	RunWitness      *TrainingRunWitness      `json:"run_witness,omitempty"`
	ProfileContract *TrainingProfileContract `json:"profile_contract,omitempty"`
	IdentityLock    *IdentityLockReceipt     `json:"identity_lock,omitempty"`
	Contamination   *ContaminationReport     `json:"contamination,omitempty"`
}

type ProvenanceSourceReceipt struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
	State  string `json:"state"`
	SHA256 string `json:"sha256"`
}

// ProvenanceFact keeps the value, evidence class, source receipt, and ceiling
// adjacent so prose cannot silently shed the fact's limits.
type ProvenanceFact struct {
	Path                string          `json:"path"`
	EvidenceClass       string          `json:"evidence_class"`
	SourceReceiptSHA256 string          `json:"source_receipt_sha256,omitempty"`
	Value               json.RawMessage `json:"value"`
	ClaimCeiling        string          `json:"claim_ceiling"`
}

type ProvenanceContextPacket struct {
	Schema          string                    `json:"schema"`
	State           string                    `json:"state"`
	RequestID       string                    `json:"request_id"`
	RequestSHA256   string                    `json:"request_sha256"`
	RequestedFields []string                  `json:"requested_fields"`
	MaxFactBytes    int                       `json:"max_fact_bytes"`
	FactBytes       int                       `json:"fact_bytes"`
	FactsSHA256     string                    `json:"facts_sha256"`
	Sources         []ProvenanceSourceReceipt `json:"sources"`
	Facts           []ProvenanceFact          `json:"facts,omitempty"`
	PacketSHA256    string                    `json:"packet_sha256,omitempty"`
	Holds           []string                  `json:"holds,omitempty"`
	Notices         []string                  `json:"notices"`
	Authority       Authority                 `json:"authority"`
}

type provenanceRequestBinding struct {
	Schema       string                    `json:"schema"`
	RequestID    string                    `json:"request_id"`
	Fields       []string                  `json:"fields"`
	MaxFactBytes int                       `json:"max_fact_bytes"`
	Sources      []ProvenanceSourceReceipt `json:"sources"`
}

// BuildProvenanceContext builds an all-or-nothing packet. It never truncates a
// value to fit the byte limit: an oversized request produces a typed HOLD with
// no facts, avoiding a partial packet that could be mistaken for complete.
func BuildProvenanceContext(request ProvenanceContextRequest) (ProvenanceContextPacket, error) {
	if request.Schema != ProvenanceContextRequestSchema {
		return ProvenanceContextPacket{}, fmt.Errorf("context request schema must be %q", ProvenanceContextRequestSchema)
	}
	if strings.TrimSpace(request.RequestID) == "" {
		return ProvenanceContextPacket{}, errors.New("context request_id is required")
	}
	if request.MaxFactBytes < 1 || request.MaxFactBytes > MaxProvenanceFactBytes {
		return ProvenanceContextPacket{}, fmt.Errorf("context max_fact_bytes must be in 1..%d", MaxProvenanceFactBytes)
	}
	fields, err := canonicalProvenanceFields(request.Fields)
	if err != nil {
		return ProvenanceContextPacket{}, err
	}
	if err := request.Anchor.Validate(); err != nil {
		return ProvenanceContextPacket{}, fmt.Errorf("context anchor: %w", err)
	}
	if err := request.Corpus.Validate(); err != nil {
		return ProvenanceContextPacket{}, fmt.Errorf("context corpus: %w", err)
	}
	if request.RunWitness != nil {
		if err := request.RunWitness.Validate(); err != nil {
			return ProvenanceContextPacket{}, fmt.Errorf("context run witness: %w", err)
		}
	}
	if request.ProfileContract != nil {
		if err := request.ProfileContract.Validate(); err != nil {
			return ProvenanceContextPacket{}, fmt.Errorf("context profile contract: %w", err)
		}
	}
	if request.IdentityLock != nil {
		if err := validateIdentityLockReceipt(*request.IdentityLock); err != nil {
			return ProvenanceContextPacket{}, fmt.Errorf("context identity lock: %w", err)
		}
	}
	if request.Contamination != nil {
		if err := validateContaminationReport(*request.Contamination); err != nil {
			return ProvenanceContextPacket{}, fmt.Errorf("context contamination report: %w", err)
		}
	}

	sources, sourceDigests, err := contextSources(request)
	if err != nil {
		return ProvenanceContextPacket{}, err
	}
	packet := ProvenanceContextPacket{
		Schema: ProvenanceContextSchema, State: ContextStateReady, RequestID: request.RequestID,
		RequestedFields: fields, MaxFactBytes: request.MaxFactBytes, Sources: sources,
		Notices: []string{
			"the packet contains bounded receipt projections only; it excludes artifact bodies, raw corpus text, prompts, secrets, hidden reasoning, and absolute paths",
			"recorded corpus membership or weight never establishes that a source caused a model output",
			"the packet grants no tool, training, promotion, CANON, permission, or world-action authority",
		},
		Authority: Authority{},
	}
	binding := provenanceRequestBinding{
		Schema: ProvenanceContextRequestSchema, RequestID: request.RequestID,
		Fields: fields, MaxFactBytes: request.MaxFactBytes, Sources: sources,
	}
	packet.RequestSHA256, err = digestJSON(binding, "provenance context request")
	if err != nil {
		return ProvenanceContextPacket{}, err
	}

	packet.Holds = contextJoinHolds(request)
	if len(packet.Holds) != 0 {
		packet.State = ContextStateJoinHold
	} else {
		packet.Facts = make([]ProvenanceFact, 0, len(fields))
		for _, field := range fields {
			fact, err := contextFactForPath(field, request, sourceDigests)
			if err != nil {
				return ProvenanceContextPacket{}, err
			}
			packet.Facts = append(packet.Facts, fact)
		}
	}
	factData, err := json.Marshal(packet.Facts)
	if err != nil {
		return ProvenanceContextPacket{}, fmt.Errorf("encode provenance facts: %w", err)
	}
	packet.FactBytes = len(factData)
	packet.FactsSHA256 = digestBytes(factData)
	if packet.State == ContextStateReady && packet.FactBytes > packet.MaxFactBytes {
		packet.State = ContextStateSizeHold
		packet.Holds = []string{fmt.Sprintf("requested provenance facts require %d bytes; limit is %d", packet.FactBytes, packet.MaxFactBytes)}
		packet.Facts = nil
		factData, err = json.Marshal(packet.Facts)
		if err != nil {
			return ProvenanceContextPacket{}, fmt.Errorf("encode held provenance facts: %w", err)
		}
		packet.FactBytes = len(factData)
		packet.FactsSHA256 = digestBytes(factData)
	}
	packet.PacketSHA256, err = provenancePacketDigest(packet)
	if err != nil {
		return ProvenanceContextPacket{}, err
	}
	if err := packet.Validate(); err != nil {
		return ProvenanceContextPacket{}, fmt.Errorf("generated provenance context packet: %w", err)
	}
	return packet, nil
}

func (packet ProvenanceContextPacket) Validate() error {
	if packet.Schema != ProvenanceContextSchema || !oneOf(packet.State, ContextStateReady, ContextStateJoinHold, ContextStateSizeHold) {
		return fmt.Errorf("unsupported provenance context identity %q state %q", packet.Schema, packet.State)
	}
	if strings.TrimSpace(packet.RequestID) == "" || packet.MaxFactBytes < 1 || packet.MaxFactBytes > MaxProvenanceFactBytes {
		return errors.New("provenance context has invalid request identity or byte limit")
	}
	for _, item := range []struct{ name, value string }{
		{"provenance context.request_sha256", packet.RequestSHA256},
		{"provenance context.facts_sha256", packet.FactsSHA256},
		{"provenance context.packet_sha256", packet.PacketSHA256},
	} {
		if err := validateSHA256(item.name, item.value); err != nil {
			return err
		}
	}
	fields, err := canonicalProvenanceFields(packet.RequestedFields)
	if err != nil {
		return err
	}
	if !stringSlicesEqual(fields, packet.RequestedFields) {
		return errors.New("provenance context requested_fields are not canonical")
	}
	if len(packet.Sources) < 2 || !sort.SliceIsSorted(packet.Sources, func(i, j int) bool { return packet.Sources[i].Name < packet.Sources[j].Name }) {
		return errors.New("provenance context sources are missing or unsorted")
	}
	sourceDigests := map[string]struct{}{}
	for i, source := range packet.Sources {
		if strings.TrimSpace(source.Name) == "" || strings.TrimSpace(source.Schema) == "" || strings.TrimSpace(source.State) == "" {
			return fmt.Errorf("provenance context source %d has incomplete identity", i)
		}
		if i > 0 && packet.Sources[i-1].Name == source.Name {
			return fmt.Errorf("provenance context has duplicate source %q", source.Name)
		}
		if err := validateSHA256(fmt.Sprintf("provenance context sources[%d].sha256", i), source.SHA256); err != nil {
			return err
		}
		sourceDigests[source.SHA256] = struct{}{}
	}
	factData, err := json.Marshal(packet.Facts)
	if err != nil {
		return fmt.Errorf("encode provenance context facts: %w", err)
	}
	if len(factData) != packet.FactBytes || digestBytes(factData) != packet.FactsSHA256 {
		return errors.New("provenance context fact bytes or digest do not match")
	}
	if packet.State == ContextStateReady {
		if len(packet.Holds) != 0 || len(packet.Facts) != len(packet.RequestedFields) || packet.FactBytes > packet.MaxFactBytes {
			return errors.New("ready provenance context is incomplete or exceeds its byte limit")
		}
	} else if len(packet.Holds) == 0 || len(packet.Facts) != 0 {
		return errors.New("held provenance context must carry reasons and expose no facts")
	}
	for i, fact := range packet.Facts {
		if fact.Path != packet.RequestedFields[i] {
			return errors.New("provenance context facts do not match requested field order")
		}
		if !oneOf(fact.EvidenceClass, EvidenceStructurallyWitnessed, EvidenceRecordedAssertion, EvidenceDeclared, EvidenceUnknown, EvidenceNotProvided) {
			return fmt.Errorf("provenance fact %q has unsupported evidence class %q", fact.Path, fact.EvidenceClass)
		}
		if !json.Valid(fact.Value) || strings.TrimSpace(fact.ClaimCeiling) == "" {
			return fmt.Errorf("provenance fact %q has invalid value or missing claim ceiling", fact.Path)
		}
		if fact.EvidenceClass == EvidenceNotProvided {
			if fact.SourceReceiptSHA256 != "" || !bytes.Equal(bytes.TrimSpace(fact.Value), []byte("null")) {
				return fmt.Errorf("not-provided provenance fact %q must use null without a source", fact.Path)
			}
		} else {
			if _, ok := sourceDigests[fact.SourceReceiptSHA256]; !ok {
				return fmt.Errorf("provenance fact %q references an unknown source receipt", fact.Path)
			}
		}
	}
	binding := provenanceRequestBinding{
		Schema: ProvenanceContextRequestSchema, RequestID: packet.RequestID,
		Fields: packet.RequestedFields, MaxFactBytes: packet.MaxFactBytes, Sources: packet.Sources,
	}
	expectedRequest, err := digestJSON(binding, "provenance context request")
	if err != nil {
		return err
	}
	if expectedRequest != packet.RequestSHA256 {
		return errors.New("provenance context request digest does not match")
	}
	if len(packet.Notices) == 0 || !packet.Authority.closed() {
		return errors.New("provenance context must state its boundaries and carry closed authority")
	}
	expectedPacket, err := provenancePacketDigest(packet)
	if err != nil {
		return err
	}
	if expectedPacket != packet.PacketSHA256 {
		return fmt.Errorf("provenance context packet digest mismatch: expected %s, got %s", packet.PacketSHA256, expectedPacket)
	}
	return nil
}

func contextSources(request ProvenanceContextRequest) ([]ProvenanceSourceReceipt, map[string]string, error) {
	anchorDigest, err := digestJSON(request.Anchor, "origin anchor receipt")
	if err != nil {
		return nil, nil, err
	}
	sources := []ProvenanceSourceReceipt{
		{Name: "anchor", Schema: request.Anchor.Schema, State: request.Anchor.State, SHA256: anchorDigest},
		{Name: "corpus", Schema: request.Corpus.Schema, State: request.Corpus.State, SHA256: request.Corpus.ReceiptSHA256},
	}
	if request.RunWitness != nil {
		sources = append(sources, ProvenanceSourceReceipt{Name: "run-witness", Schema: request.RunWitness.Schema, State: request.RunWitness.State, SHA256: request.RunWitness.ReceiptSHA256})
	}
	if request.ProfileContract != nil {
		sources = append(sources, ProvenanceSourceReceipt{Name: "profile-contract", Schema: request.ProfileContract.Schema, State: request.ProfileContract.State, SHA256: request.ProfileContract.ReceiptSHA256})
	}
	if request.IdentityLock != nil {
		digest, err := digestJSON(request.IdentityLock, "identity lock receipt")
		if err != nil {
			return nil, nil, err
		}
		sources = append(sources, ProvenanceSourceReceipt{Name: "identity-lock", Schema: request.IdentityLock.Schema, State: request.IdentityLock.State, SHA256: digest})
	}
	if request.Contamination != nil {
		digest, err := digestJSON(request.Contamination, "contamination report")
		if err != nil {
			return nil, nil, err
		}
		sources = append(sources, ProvenanceSourceReceipt{Name: "contamination", Schema: request.Contamination.Schema, State: request.Contamination.State, SHA256: digest})
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Name < sources[j].Name })
	digests := make(map[string]string, len(sources))
	for _, source := range sources {
		digests[source.Name] = source.SHA256
	}
	return sources, digests, nil
}

func contextJoinHolds(request ProvenanceContextRequest) []string {
	var holds []string
	if request.Anchor.State != AnchorStateAnchored {
		holds = append(holds, "origin anchor is not ANCHORED")
	}
	if request.Corpus.State != CorpusEvidenceStateReady {
		holds = append(holds, "corpus evidence lens is not READY")
	}
	if request.Anchor.SourceType == "run" {
		if request.RunWitness == nil {
			holds = append(holds, "run-backed anchor requires a training run witness")
		} else {
			if request.RunWitness.State != RunWitnessStateReady {
				holds = append(holds, "training run witness is not READY")
			}
			if request.RunWitness.ModelID != request.Anchor.ModelID || request.RunWitness.RunID != request.Anchor.RunID || request.Anchor.ArchitectureSHA256 != "" && request.RunWitness.ArchitectureSHA256 != request.Anchor.ArchitectureSHA256 {
				holds = append(holds, "training run witness does not match the answering anchor")
			}
			if request.RunWitness.Corpus.ReceiptSHA256 != request.Corpus.ReceiptSHA256 {
				holds = append(holds, "training run witness does not bind the supplied corpus lens")
			}
		}
		if request.ProfileContract == nil {
			holds = append(holds, "run-backed anchor requires a training profile contract")
		}
	}
	if request.ProfileContract != nil {
		if request.RunWitness == nil {
			holds = append(holds, "training profile contract has no run witness")
		} else if request.ProfileContract.RunID != request.RunWitness.RunID || request.ProfileContract.RunBOMSHA256 != request.RunWitness.RunBOMSHA256 || request.ProfileContract.RunWitnessReceiptSHA256 != request.RunWitness.ReceiptSHA256 {
			holds = append(holds, "training profile contract does not match the supplied run witness")
		}
		if !oneOf(request.ProfileContract.State, ProfileStateWitnessed, ProfileStateLegacyWitnessed) {
			holds = append(holds, "training profile contract is not witnessed")
		}
	}
	if request.IdentityLock != nil {
		if request.IdentityLock.State != IdentityStateLocked {
			holds = append(holds, "release identity lock is not LOCKED")
		}
		if request.IdentityLock.Observed.ModelID != request.Anchor.ModelID || request.IdentityLock.Observed.AnsweringIdentitySHA256 != request.Anchor.AnsweringIdentitySHA256 {
			holds = append(holds, "release identity lock does not bind the supplied answering anchor")
		}
	}
	return holds
}

func contextFactForPath(path string, request ProvenanceContextRequest, sources map[string]string) (ProvenanceFact, error) {
	identityCeiling := "selected recorded answering identity only; not fresh artifact-byte verification"
	corpusCeiling := "recorded corpus selection and identity only; not source-to-output causality"
	trainingCeiling := "resolved recorded training contract only; not model quality or causal attribution"
	evaluationCeiling := "exact declared inventory comparison only; not semantic independence or absence of memorization"
	switch path {
	case "answering.subject":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["anchor"], request.Anchor.Subject, identityCeiling)
	case "answering.model_id":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["anchor"], request.Anchor.ModelID, identityCeiling)
	case "answering.name":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["anchor"], request.Anchor.Name, identityCeiling)
	case "answering.identity_sha256":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["anchor"], request.Anchor.AnsweringIdentitySHA256, identityCeiling)
	case "answering.source_type":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["anchor"], request.Anchor.SourceType, identityCeiling)
	case "answering.source_id":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["anchor"], request.Anchor.SourceID, identityCeiling)
	case "answering.run_id":
		if request.Anchor.RunID == "" {
			return unknownProvenanceFact(path, sources["anchor"], identityCeiling), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["anchor"], request.Anchor.RunID, identityCeiling)
	case "answering.bom_sha256":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["anchor"], request.Anchor.BOMSHA256, identityCeiling)
	case "corpus.bom_sha256":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["corpus"], request.Corpus.BOMSHA256, corpusCeiling)
	case "corpus.index.state":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["corpus"], request.Corpus.Index.State, corpusCeiling)
	case "corpus.index.commit":
		if request.Corpus.Index.Commit == "" {
			return unknownProvenanceFact(path, sources["corpus"], corpusCeiling), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["corpus"], request.Corpus.Index.Commit, corpusCeiling)
	case "corpus.paths":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["corpus"], request.Corpus.Paths, corpusCeiling)
	case "corpus.source_set_sha256":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["corpus"], request.Corpus.SourceSetSHA256, corpusCeiling)
	case "corpus.shard_set_sha256":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["corpus"], request.Corpus.ShardSetSHA256, corpusCeiling)
	case "corpus.totals":
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["corpus"], request.Corpus.Totals, corpusCeiling)
	case "corpus.licenses":
		licenses := make([]string, 0, len(request.Corpus.Licenses))
		for _, license := range request.Corpus.Licenses {
			licenses = append(licenses, license.ID)
		}
		sort.Strings(licenses)
		return newProvenanceFact(path, EvidenceRecordedAssertion, sources["corpus"], licenses, "recorded license assertion only; not a legal conclusion or grant of rights")
	case "training.run_id":
		if request.RunWitness == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["run-witness"], request.RunWitness.RunID, trainingCeiling)
	case "training.run_bom_sha256":
		if request.RunWitness == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["run-witness"], request.RunWitness.RunBOMSHA256, trainingCeiling)
	case "training.state":
		if request.RunWitness == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["run-witness"], request.RunWitness.RunState, trainingCeiling)
	case "training.backend":
		if request.RunWitness == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["run-witness"], request.RunWitness.Execution.Backend, trainingCeiling)
	case "training.architecture_sha256":
		if request.RunWitness == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["run-witness"], request.RunWitness.ArchitectureSHA256, trainingCeiling)
	case "training.simulated":
		if request.RunWitness == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["run-witness"], request.RunWitness.Simulated, trainingCeiling)
	case "training.profile.declared":
		if request.ProfileContract == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["profile-contract"], request.ProfileContract.DeclaredProfile, trainingCeiling)
	case "training.profile.canonical":
		if request.ProfileContract == nil {
			return notProvidedProvenanceFact(path), nil
		}
		if request.ProfileContract.CanonicalProfile == "" {
			return unknownProvenanceFact(path, sources["profile-contract"], trainingCeiling), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["profile-contract"], request.ProfileContract.CanonicalProfile, trainingCeiling)
	case "training.profile.schema":
		if request.ProfileContract == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["profile-contract"], request.ProfileContract.ProfileSchema, trainingCeiling)
	case "training.profile.data_order":
		if request.ProfileContract == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["profile-contract"], request.ProfileContract.Data.Order, trainingCeiling)
	case "training.profile.weight_set_sha256":
		if request.ProfileContract == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["profile-contract"], request.ProfileContract.Data.WeightSetSHA256, trainingCeiling)
	case "training.profile.evaluation_selection":
		if request.ProfileContract == nil {
			return notProvidedProvenanceFact(path), nil
		}
		if request.ProfileContract.Evaluation == nil {
			return unknownProvenanceFact(path, sources["profile-contract"], trainingCeiling), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["profile-contract"], request.ProfileContract.Evaluation.Selection, trainingCeiling)
	case "identity.lock_state":
		if request.IdentityLock == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["identity-lock"], request.IdentityLock.State, identityCeiling)
	case "evaluation.contamination_state":
		if request.Contamination == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["contamination"], request.Contamination.State, evaluationCeiling)
	case "evaluation.comparison_sha256":
		if request.Contamination == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["contamination"], request.Contamination.ComparisonSHA256, evaluationCeiling)
	case "evaluation.compared_dimensions":
		if request.Contamination == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceStructurallyWitnessed, sources["contamination"], request.Contamination.ComparedDimensions, evaluationCeiling)
	case "evaluation.authorship_state":
		if request.Contamination == nil {
			return notProvidedProvenanceFact(path), nil
		}
		return newProvenanceFact(path, EvidenceDeclared, sources["contamination"], request.Contamination.Authorship.State, "authorship remains declared; it is not independently verified by the overlap comparison")
	default:
		return ProvenanceFact{}, fmt.Errorf("unsupported provenance field %q", path)
	}
}

func newProvenanceFact(path, class, source string, value any, ceiling string) (ProvenanceFact, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return ProvenanceFact{}, fmt.Errorf("encode provenance fact %q: %w", path, err)
	}
	return ProvenanceFact{Path: path, EvidenceClass: class, SourceReceiptSHA256: source, Value: data, ClaimCeiling: ceiling}, nil
}

func notProvidedProvenanceFact(path string) ProvenanceFact {
	return ProvenanceFact{Path: path, EvidenceClass: EvidenceNotProvided, Value: json.RawMessage("null"), ClaimCeiling: "no source receipt was provided for this requested fact"}
}

func unknownProvenanceFact(path, source, ceiling string) ProvenanceFact {
	return ProvenanceFact{Path: path, EvidenceClass: EvidenceUnknown, SourceReceiptSHA256: source, Value: json.RawMessage("null"), ClaimCeiling: ceiling}
}

func canonicalProvenanceFields(input []string) ([]string, error) {
	if len(input) == 0 {
		return nil, errors.New("context request requires at least one field")
	}
	result := append([]string(nil), input...)
	for _, field := range result {
		if _, ok := supportedProvenanceFields[field]; !ok {
			return nil, fmt.Errorf("unsupported provenance field %q", field)
		}
	}
	sort.Strings(result)
	for i := 1; i < len(result); i++ {
		if result[i] == result[i-1] {
			return nil, fmt.Errorf("duplicate provenance field %q", result[i])
		}
	}
	return result, nil
}

func validateIdentityLockReceipt(receipt IdentityLockReceipt) error {
	if receipt.Schema != IdentityLockSchema || !oneOf(receipt.State, IdentityStateLocked, IdentityStateDrift, IdentityStateHold) {
		return fmt.Errorf("unsupported identity lock identity %q state %q", receipt.Schema, receipt.State)
	}
	for name, ref := range map[string]IdentityAnchorRef{"expected": receipt.Expected, "observed": receipt.Observed} {
		if strings.TrimSpace(ref.Subject) == "" || strings.TrimSpace(ref.ModelID) == "" {
			return fmt.Errorf("identity lock %s reference is incomplete", name)
		}
		if err := validateSHA256("identity lock "+name+".bom_sha256", ref.BOMSHA256); err != nil {
			return err
		}
		if err := validateSHA256("identity lock "+name+".answering_identity_sha256", ref.AnsweringIdentitySHA256); err != nil {
			return err
		}
	}
	if receipt.State == IdentityStateLocked && len(receipt.Changes) != 0 {
		return errors.New("LOCKED identity receipt carries changes")
	}
	if receipt.State == IdentityStateDrift && len(receipt.Changes) == 0 {
		return errors.New("DRIFT identity receipt carries no changes")
	}
	if !receipt.Authority.closed() {
		return errors.New("identity lock receipt must carry closed authority")
	}
	return nil
}

func validateContaminationReport(report ContaminationReport) error {
	if report.Schema != ContaminationReportSchema || !oneOf(report.State, ContaminationStateClear, ContaminationStateHold, ContaminationStateContaminated) {
		return fmt.Errorf("unsupported contamination report identity %q state %q", report.Schema, report.State)
	}
	if err := validateSHA256("contamination report.comparison_sha256", report.ComparisonSHA256); err != nil {
		return err
	}
	seenDimensions := map[string]struct{}{}
	for _, dimension := range report.ComparedDimensions {
		if strings.TrimSpace(dimension) == "" {
			return errors.New("contamination report has an empty compared dimension")
		}
		if _, exists := seenDimensions[dimension]; exists {
			return fmt.Errorf("contamination report has duplicate dimension %q", dimension)
		}
		seenDimensions[dimension] = struct{}{}
	}
	for i, overlap := range report.Overlaps {
		if strings.TrimSpace(overlap.Dimension) == "" || len(overlap.Values) == 0 {
			return fmt.Errorf("contamination overlap %d is incomplete", i)
		}
	}
	switch report.State {
	case ContaminationStateClear:
		if len(report.ComparedDimensions) == 0 || len(report.Overlaps) != 0 || len(report.Holds) != 0 {
			return errors.New("CLEAR contamination report is incomplete or contradictory")
		}
	case ContaminationStateHold:
		if len(report.Holds) == 0 || len(report.Overlaps) != 0 {
			return errors.New("HOLD contamination report requires holds and no overlaps")
		}
	case ContaminationStateContaminated:
		if len(report.Overlaps) == 0 {
			return errors.New("CONTAMINATED report requires an overlap")
		}
	}
	if !oneOf(report.Authorship.State, "RECORDED_ASSERTION", "REFUTED_BY_DECLARATION", "UNKNOWN") || strings.TrimSpace(report.Authorship.Basis) == "" {
		return errors.New("contamination report has invalid authorship finding")
	}
	if report.Authorship.EvidenceSHA256 != "" {
		if err := validateSHA256("contamination report.authorship.evidence_sha256", report.Authorship.EvidenceSHA256); err != nil {
			return err
		}
	}
	if !report.Authority.closed() {
		return errors.New("contamination report must carry closed authority")
	}
	return nil
}

func provenancePacketDigest(packet ProvenanceContextPacket) (string, error) {
	packet.PacketSHA256 = ""
	return digestJSON(packet, "provenance context packet")
}

func digestJSON(value any, name string) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode %s: %w", name, err)
	}
	return digestBytes(data), nil
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
