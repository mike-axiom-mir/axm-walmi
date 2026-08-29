package axmmirror

import (
	"encoding/json"
	"testing"
)

func TestProvenanceContextBuildsBoundedTraceableFacts(t *testing.T) {
	request := validContextRequest(t)
	packet, err := BuildProvenanceContext(request)
	if err != nil {
		t.Fatal(err)
	}
	if packet.State != ContextStateReady || len(packet.Facts) != len(request.Fields) || packet.FactBytes > packet.MaxFactBytes {
		t.Fatalf("context packet = %+v", packet)
	}
	for _, fact := range packet.Facts {
		if fact.SourceReceiptSHA256 == "" || fact.ClaimCeiling == "" {
			t.Fatalf("untraceable fact = %+v", fact)
		}
	}
	if err := packet.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestProvenanceContextCanonicalizesFieldOrder(t *testing.T) {
	left := validContextRequest(t)
	right := left
	right.Fields = append([]string(nil), left.Fields...)
	for i, j := 0, len(right.Fields)-1; i < j; i, j = i+1, j-1 {
		right.Fields[i], right.Fields[j] = right.Fields[j], right.Fields[i]
	}
	leftPacket, err := BuildProvenanceContext(left)
	if err != nil {
		t.Fatal(err)
	}
	rightPacket, err := BuildProvenanceContext(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftPacket.PacketSHA256 != rightPacket.PacketSHA256 || leftPacket.RequestSHA256 != rightPacket.RequestSHA256 {
		t.Fatalf("canonical packets differ: %s != %s", leftPacket.PacketSHA256, rightPacket.PacketSHA256)
	}
}

func TestProvenanceContextHoldsJoinMismatchWithoutFacts(t *testing.T) {
	request := validContextRequest(t)
	otherBOM, otherRun := validRunFixture(t, "pytorch", false, "complete")
	otherBOM.ID = "run-0002"
	otherBOM.ModelID = "model-0002"
	otherRun.ID = otherBOM.ID
	repinRun(t, otherBOM, &otherRun)
	otherWitness := witnessFixture(t, otherBOM, otherRun)
	request.RunWitness = &otherWitness
	packet, err := BuildProvenanceContext(request)
	if err != nil {
		t.Fatal(err)
	}
	if packet.State != ContextStateJoinHold || len(packet.Holds) == 0 || len(packet.Facts) != 0 {
		t.Fatalf("context packet = %+v", packet)
	}
}

func TestProvenanceContextHoldsOversizedFactSetWithoutTruncation(t *testing.T) {
	request := validContextRequest(t)
	request.MaxFactBytes = 1
	packet, err := BuildProvenanceContext(request)
	if err != nil {
		t.Fatal(err)
	}
	if packet.State != ContextStateSizeHold || len(packet.Facts) != 0 || len(packet.Holds) == 0 {
		t.Fatalf("context packet = %+v", packet)
	}
}

func TestProvenanceContextMarksUnavailableOriginRunFactNotProvided(t *testing.T) {
	anchor := anchorModelForTest(t, func() waldoModelBOM {
		bom := validModelBOM()
		bom.CurrentRunID = ""
		bom.CurrentOriginSHA256 = repeatHex("9")
		bom.Origin = &waldoModelOrigin{
			BOM: "ORIGIN-BOM.json", SHA256: repeatHex("9"),
			Artifacts: []AnchoredArtifact{
				{Role: "weights", Path: "origin/model.safetensors", SHA256: repeatHex("4"), Bytes: 1024},
				{Role: "configuration", Path: "origin/config.json", SHA256: repeatHex("5"), Bytes: 256},
				{Role: "tokenizer", Path: "origin/tokenizer.json", SHA256: repeatHex("6"), Bytes: 512},
			},
		}
		return bom
	}())
	corpusData, err := json.Marshal(validCorpusBOM())
	if err != nil {
		t.Fatal(err)
	}
	corpus, err := LensCorpusBOM(corpusData)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := BuildProvenanceContext(ProvenanceContextRequest{
		Schema: ProvenanceContextRequestSchema, RequestID: "origin-context", MaxFactBytes: 1024,
		Fields: []string{"training.run_id"}, Anchor: anchor, Corpus: corpus,
	})
	if err != nil {
		t.Fatal(err)
	}
	if packet.State != ContextStateReady || len(packet.Facts) != 1 || packet.Facts[0].EvidenceClass != EvidenceNotProvided {
		t.Fatalf("context packet = %+v", packet)
	}
}

func TestProvenanceContextDetectsFactTamper(t *testing.T) {
	packet, err := BuildProvenanceContext(validContextRequest(t))
	if err != nil {
		t.Fatal(err)
	}
	packet.Facts[0].Value = json.RawMessage(`"tampered"`)
	if err := packet.Validate(); err == nil {
		t.Fatal("tampered context packet validated")
	}
}

func validContextRequest(t *testing.T) ProvenanceContextRequest {
	t.Helper()
	bom, run := validRunFixture(t, "pytorch", false, "complete")
	bom.ModelID = "model-0001"
	bom.Parameters.Profile = profileShuffled
	configureSingleNamedCorpus(t, &bom, &run)
	repinRun(t, bom, &run)
	bomData, err := json.MarshalIndent(bom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	runData, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	witness, err := WitnessTrainingRun(bomData, runData)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := LensTrainingProfile(bomData, witness)
	if err != nil {
		t.Fatal(err)
	}
	modelBOM := validModelBOM()
	modelBOM.ModelID = bom.ModelID
	modelBOM.ArchitectureSHA256 = bom.ArchitectureSHA256
	anchor := anchorModelForTest(t, modelBOM)
	lock, err := CompareIdentity(anchor, anchor)
	if err != nil {
		t.Fatal(err)
	}
	comparison := validComparison()
	comparison.Authorship = AuthorshipDeclaration{State: "outside-authored", EvidenceSHA256: repeatHex("9")}
	contamination, err := CheckContamination(comparison)
	if err != nil {
		t.Fatal(err)
	}
	return ProvenanceContextRequest{
		Schema: ProvenanceContextRequestSchema, RequestID: "context-0001", MaxFactBytes: MaxProvenanceFactBytes,
		Fields: []string{
			"answering.identity_sha256", "answering.model_id", "corpus.bom_sha256", "corpus.paths", "corpus.licenses",
			"training.run_id", "training.run_bom_sha256", "training.profile.canonical", "training.profile.data_order",
			"identity.lock_state", "evaluation.contamination_state", "evaluation.authorship_state",
		},
		Anchor: anchor, Corpus: witness.Corpus, RunWitness: &witness, ProfileContract: &profile,
		IdentityLock: &lock, Contamination: &contamination,
	}
}
