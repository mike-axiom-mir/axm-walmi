package axmmirror

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	PlatformFabricReceiptSchema = "axm.waldo-mirror.fabric-bridge-experiment-receipt/v0.1"
	PlatformFabricChallenge     = "ONE_CONCEPT_MANY_BODIES"
	PlatformFabricRepository    = "mike-axiom-mir/axm-collaboration-platform"
	PlatformFabricCommit        = "fd6ec98a6a98a6666a980c359730ccec57a8cbe9"

	PlatformFabricWitnessSchema = "axm.waldo-witness.platform-fabric-bridge/v0.1"
	PlatformFabricWitnessReady  = "PLATFORM_FABRIC_WITNESSED"

	PlatformFabricDeterministicSource = "DETERMINISTIC_NATIVE"
	PlatformFabricAIOptInSource       = "AI_OPT_IN_EXTERNAL_PROPOSAL"
)

var platformFabricDonorBlobs = map[string]string{
	"shared/code-capability-fabric/code-capability-fabric-v2.js":         "7b20769ccf296943cddce192d601bde3ec6f2334",
	"shared/code-capability-fabric/grounded-consent-scope-v1.js":         "a05190bb58c50b8782f2c404b5e7dfde3d7feae3",
	"shared/code-capability-fabric/slow-creation-pilot-v1.js":            "4e667b166ab5d018aad88828fae45dcc25529627",
	"shared/code-capability-fabric/declarative-blueprint-composer-v1.js": "388343fade0bfc1507c6a526a7f3471a2d0595e5",
	"shared/code-capability-fabric/blueprint-schema-compiler-v1.js":      "f0460d40d53c266065b4876ab0bd3d6dd971b00d",
	"tools/deterministic-json-core/index.js":                             "5f6720fb6e887158c7d501fdc41be422a315050f",
}

type PlatformFabricRef struct {
	ID     string `json:"id"`
	Schema string `json:"schema"`
	SHA256 string `json:"sha256"`
}

type PlatformFabricDonor struct {
	Repository string            `json:"repository"`
	Commit     string            `json:"commit"`
	Blobs      map[string]string `json:"blobs"`
}

type PlatformFabricInterfaceRef struct {
	Direction string `json:"direction"`
	Schema    string `json:"schema"`
	SHA256    string `json:"sha256"`
}

type PlatformFabricSchemaDigestRef struct {
	Schema string `json:"schema"`
	SHA256 string `json:"sha256"`
}

type PlatformFabricBodyTruth struct {
	DeterministicReplayMatched    bool `json:"deterministicReplayMatched"`
	DesiredOutcomesRemainUnproven bool `json:"desiredOutcomesRemainUnproven"`
	AcceptanceCasesExecuted       bool `json:"acceptanceCasesExecuted"`
	PermissionsGranted            bool `json:"permissionsGranted"`
	GeneratedCodeExecuted         bool `json:"generatedCodeExecuted"`
	Installed                     bool `json:"installed"`
	Promoted                      bool `json:"promoted"`
	CanonChanged                  bool `json:"canonChanged"`
}

type PlatformFabricBodyReceipt struct {
	BodyID              string                        `json:"bodyId"`
	SourceMode          string                        `json:"sourceMode"`
	SourceRef           PlatformFabricRef             `json:"sourceRef"`
	ConceptRef          PlatformFabricRef             `json:"conceptRef"`
	IntentRef           PlatformFabricRef             `json:"intentRef"`
	BlueprintRef        PlatformFabricRef             `json:"blueprintRef"`
	InterfaceSchemaRefs []PlatformFabricInterfaceRef  `json:"interfaceSchemaRefs"`
	AcceptanceMatrixRef PlatformFabricSchemaDigestRef `json:"acceptanceMatrixRef"`
	Truth               PlatformFabricBodyTruth       `json:"truth"`
}

type PlatformFabricPrivateOrganFactory struct {
	Included           bool   `json:"included"`
	Availability       string `json:"availability"`
	EffectOnThisResult string `json:"effectOnThisResult"`
}

type PlatformFabricTopTruth struct {
	DonorFabricModified                bool `json:"donorFabricModified"`
	LiveAIInvocationPerformedByHarness bool `json:"liveAiInvocationPerformedByHarness"`
	NetworkRequestedByHarness          bool `json:"networkRequestedByHarness"`
	GeneratedExecutableCodeProduced    bool `json:"generatedExecutableCodeProduced"`
	CandidateCodeExecuted              bool `json:"candidateCodeExecuted"`
	RuntimeBehaviorProven              bool `json:"runtimeBehaviorProven"`
	HumanQualityApproved               bool `json:"humanQualityApproved"`
	InstallationPerformed              bool `json:"installationPerformed"`
	PromotionPerformed                 bool `json:"promotionPerformed"`
	CanonChanged                       bool `json:"canonChanged"`
}

type PlatformFabricReceipt struct {
	Schema               string                            `json:"schema"`
	Status               string                            `json:"status"`
	Challenge            string                            `json:"challenge"`
	Donor                PlatformFabricDonor               `json:"donor"`
	ConceptRef           PlatformFabricRef                 `json:"conceptRef"`
	DeterministicDefault bool                              `json:"deterministicDefault"`
	AIOptInUsed          bool                              `json:"aiOptInUsed"`
	AIProposalRef        *PlatformFabricRef                `json:"aiProposalRef"`
	Bodies               []PlatformFabricBodyReceipt       `json:"bodies"`
	PrivateOrganFactory  PlatformFabricPrivateOrganFactory `json:"privateOrganFactory"`
	Truth                PlatformFabricTopTruth            `json:"truth"`
	ReceiptSHA256        string                            `json:"receiptSha256"`
}

type PlatformFabricBodyWitness struct {
	BodyID                 string `json:"body_id"`
	SourceMode             string `json:"source_mode"`
	BlueprintSHA256        string `json:"blueprint_sha256"`
	AcceptanceMatrixSHA256 string `json:"acceptance_matrix_sha256"`
}

type PlatformFabricWitness struct {
	Schema                      string                      `json:"schema"`
	State                       string                      `json:"state"`
	SourceReceiptSHA256         string                      `json:"source_receipt_sha256"`
	PlatformCommit              string                      `json:"platform_commit"`
	ConceptSHA256               string                      `json:"concept_sha256"`
	DeterministicDefault        bool                        `json:"deterministic_default"`
	AIOptInUsed                 bool                        `json:"ai_opt_in_used"`
	Bodies                      []PlatformFabricBodyWitness `json:"bodies"`
	PrivateOrganFactoryIncluded bool                        `json:"private_organ_factory_included"`
	Notices                     []string                    `json:"notices"`
	Authority                   Authority                   `json:"authority"`
	WitnessSHA256               string                      `json:"witness_sha256,omitempty"`
}

func WitnessPlatformFabricReceipt(data []byte) (PlatformFabricWitness, error) {
	receipt, err := VerifyPlatformFabricReceipt(data)
	if err != nil {
		return PlatformFabricWitness{}, err
	}
	bodies := make([]PlatformFabricBodyWitness, 0, len(receipt.Bodies))
	for _, body := range receipt.Bodies {
		bodies = append(bodies, PlatformFabricBodyWitness{
			BodyID:                 body.BodyID,
			SourceMode:             body.SourceMode,
			BlueprintSHA256:        body.BlueprintRef.SHA256,
			AcceptanceMatrixSHA256: body.AcceptanceMatrixRef.SHA256,
		})
	}
	sort.Slice(bodies, func(i, j int) bool { return bodies[i].BodyID < bodies[j].BodyID })
	witness := PlatformFabricWitness{
		Schema:                      PlatformFabricWitnessSchema,
		State:                       PlatformFabricWitnessReady,
		SourceReceiptSHA256:         receipt.ReceiptSHA256,
		PlatformCommit:              receipt.Donor.Commit,
		ConceptSHA256:               receipt.ConceptRef.SHA256,
		DeterministicDefault:        receipt.DeterministicDefault,
		AIOptInUsed:                 receipt.AIOptInUsed,
		Bodies:                      bodies,
		PrivateOrganFactoryIncluded: receipt.PrivateOrganFactory.Included,
		Notices: []string{
			"the platform Code Fabric ran in its native repository; this witness verifies a typed receipt and does not execute copied Fabric code",
			"AI opt-in identifies an external proposal source only; the deterministic Fabric still owns composition and the receipt does not prove live model execution",
			"the private Organ Factory was absent from the public donor baseline and no capability is inferred from unavailable private code",
		},
		Authority: Authority{},
	}
	witness.WitnessSHA256, err = platformFabricWitnessDigest(witness)
	if err != nil {
		return PlatformFabricWitness{}, err
	}
	if err := witness.Validate(); err != nil {
		return PlatformFabricWitness{}, fmt.Errorf("generated platform Fabric witness: %w", err)
	}
	return witness, nil
}

func (witness PlatformFabricWitness) Validate() error {
	if witness.Schema != PlatformFabricWitnessSchema || witness.State != PlatformFabricWitnessReady {
		return errors.New("platform Fabric witness has unsupported schema or state")
	}
	if err := validatePrefixedSHA256("platform Fabric witness source receipt", witness.SourceReceiptSHA256); err != nil {
		return err
	}
	if witness.PlatformCommit != PlatformFabricCommit {
		return errors.New("platform Fabric witness commit mismatch")
	}
	if err := validatePrefixedSHA256("platform Fabric witness concept", witness.ConceptSHA256); err != nil {
		return err
	}
	if !witness.DeterministicDefault {
		return errors.New("platform Fabric witness lost deterministic default")
	}
	expectedBodies := 3
	if witness.AIOptInUsed {
		expectedBodies = 4
	}
	if len(witness.Bodies) != expectedBodies {
		return fmt.Errorf("platform Fabric witness requires %d bodies, got %d", expectedBodies, len(witness.Bodies))
	}
	if witness.PrivateOrganFactoryIncluded {
		return errors.New("platform Fabric witness cannot include unavailable private Organ Factory")
	}
	seen := map[string]bool{}
	aiBodies := 0
	previousID := ""
	for _, body := range witness.Bodies {
		if strings.TrimSpace(body.BodyID) == "" || seen[body.BodyID] {
			return errors.New("platform Fabric witness has empty or duplicate body id")
		}
		seen[body.BodyID] = true
		if previousID != "" && body.BodyID <= previousID {
			return errors.New("platform Fabric witness bodies must remain strictly sorted by id")
		}
		previousID = body.BodyID
		if body.SourceMode != PlatformFabricDeterministicSource && body.SourceMode != PlatformFabricAIOptInSource {
			return errors.New("platform Fabric witness body source mode is unsupported")
		}
		if body.SourceMode == PlatformFabricAIOptInSource {
			aiBodies++
		}
		if err := validatePrefixedSHA256("platform Fabric witness blueprint", body.BlueprintSHA256); err != nil {
			return err
		}
		if err := validatePrefixedSHA256("platform Fabric witness acceptance matrix", body.AcceptanceMatrixSHA256); err != nil {
			return err
		}
	}
	if witness.AIOptInUsed && aiBodies != 1 {
		return errors.New("platform Fabric AI-opt-in witness requires exactly one AI-proposed body")
	}
	if !witness.AIOptInUsed && aiBodies != 0 {
		return errors.New("platform Fabric deterministic witness contains an AI-proposed body")
	}
	if len(witness.Notices) < 3 || !witness.Authority.closed() {
		return errors.New("platform Fabric witness requires boundary notices and closed authority")
	}
	if err := validateSHA256("platform Fabric witness_sha256", witness.WitnessSHA256); err != nil {
		return err
	}
	expected, err := platformFabricWitnessDigest(witness)
	if err != nil {
		return err
	}
	if expected != witness.WitnessSHA256 {
		return errors.New("platform Fabric witness digest mismatch")
	}
	return nil
}
