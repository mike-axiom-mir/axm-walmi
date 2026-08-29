package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

const (
	VerificationTrainingProfile = "waldo-training-profile-contract"
	VerificationContextPacket   = "waldo-provenance-context"
	VerificationSourceClaims    = "waldo-source-claim-assessment"
	VerificationProtocolSeal    = "waldo-evaluation-protocol-seal"
)

// SealGated binds a behavior record to the complete deterministic boundary
// around the learned clone. It does not require source claims to pass in order
// to preserve failure evidence, but a contradicted or unresolved claim gate
// cannot be represented as a passing behavior outcome.
func SealGated(
	draft BehaviorEvidenceDraft,
	anchor OriginAnchor,
	witness TrainingRunWitness,
	profile TrainingProfileContract,
	context ProvenanceContextPacket,
	claims SourceClaimAssessment,
	protocol EvaluationProtocolSeal,
) (SealedBehaviorEvidence, error) {
	if err := profile.Validate(); err != nil {
		return SealedBehaviorEvidence{}, fmt.Errorf("training profile contract: %w", err)
	}
	if err := context.Validate(); err != nil {
		return SealedBehaviorEvidence{}, fmt.Errorf("provenance context packet: %w", err)
	}
	if err := claims.Validate(); err != nil {
		return SealedBehaviorEvidence{}, fmt.Errorf("source-claim assessment: %w", err)
	}
	if err := protocol.Validate(); err != nil {
		return SealedBehaviorEvidence{}, fmt.Errorf("evaluation protocol seal: %w", err)
	}
	if !oneOf(profile.State, ProfileStateWitnessed, ProfileStateLegacyWitnessed) {
		return SealedBehaviorEvidence{}, errors.New("training profile contract is not witnessed")
	}
	if profile.RunID != witness.RunID || profile.RunBOMSHA256 != witness.RunBOMSHA256 || profile.RunWitnessReceiptSHA256 != witness.ReceiptSHA256 {
		return SealedBehaviorEvidence{}, errors.New("training profile contract does not bind the supplied run witness")
	}
	if context.State != ContextStateReady {
		return SealedBehaviorEvidence{}, errors.New("provenance context packet is not ready")
	}
	identityFact, ok := findProvenanceFact(context, "answering.identity_sha256")
	if !ok {
		return SealedBehaviorEvidence{}, errors.New("provenance context omits answering.identity_sha256")
	}
	var contextIdentity string
	if err := decodeFactValue(identityFact, &contextIdentity); err != nil {
		return SealedBehaviorEvidence{}, err
	}
	if contextIdentity != anchor.AnsweringIdentitySHA256 {
		return SealedBehaviorEvidence{}, errors.New("provenance context does not bind the supplied answering anchor")
	}
	anchorDigest, err := digestJSON(anchor, "gated behavior anchor")
	if err != nil {
		return SealedBehaviorEvidence{}, err
	}
	if !contextHasSourceDigest(context, anchorDigest) || !contextHasSourceDigest(context, witness.Corpus.ReceiptSHA256) || !contextHasSourceDigest(context, witness.ReceiptSHA256) || !contextHasSourceDigest(context, profile.ReceiptSHA256) {
		return SealedBehaviorEvidence{}, errors.New("provenance context does not retain the supplied anchor, corpus, run, and profile receipts")
	}
	if claims.ContextPacketSHA256 != context.PacketSHA256 || claims.OutputSHA256 != draft.OutputSHA256 {
		return SealedBehaviorEvidence{}, errors.New("source-claim assessment does not bind the supplied context and behavior output")
	}
	if protocol.State != ProtocolStateReady {
		return SealedBehaviorEvidence{}, fmt.Errorf("evaluation protocol is %s, not ready", protocol.State)
	}
	if protocol.Target.AnsweringIdentitySHA256 != anchor.AnsweringIdentitySHA256 || protocol.Target.AnchorReceiptSHA256 != anchorDigest || protocol.Target.ContextPacketSHA256 != context.PacketSHA256 {
		return SealedBehaviorEvidence{}, errors.New("evaluation protocol target does not bind the supplied anchor and context")
	}
	if draft.ExperimentID != protocol.ProtocolID {
		return SealedBehaviorEvidence{}, errors.New("behavior experiment_id does not match evaluation protocol_id")
	}
	if draft.ContextSHA256 != context.PacketSHA256 {
		return SealedBehaviorEvidence{}, errors.New("behavior context_sha256 does not match the provenance context packet")
	}
	if draft.RequestSHA256 != protocol.RequestSetSHA256 {
		return SealedBehaviorEvidence{}, errors.New("behavior request_sha256 does not match the sealed evaluation request set")
	}
	if draft.PermissionState != protocol.PermissionState {
		return SealedBehaviorEvidence{}, errors.New("behavior permission_state does not match the sealed evaluation protocol")
	}

	claimVerificationState := "pass"
	switch claims.State {
	case ClaimGateStateContradicted:
		claimVerificationState = "fail"
		if draft.OutcomeState == "pass" {
			return SealedBehaviorEvidence{}, errors.New("contradicted source claims cannot be sealed as a passing behavior outcome")
		}
	case ClaimGateStateReview:
		claimVerificationState = "hold"
		if draft.OutcomeState == "pass" {
			return SealedBehaviorEvidence{}, errors.New("unresolved source claims cannot be sealed as a passing behavior outcome")
		}
	}

	for _, binding := range []Verification{
		{Name: VerificationTrainingProfile, State: "pass", EvidenceSHA256: profile.ReceiptSHA256},
		{Name: VerificationContextPacket, State: "pass", EvidenceSHA256: context.PacketSHA256},
		{Name: VerificationSourceClaims, State: claimVerificationState, EvidenceSHA256: claims.ReceiptSHA256},
		{Name: VerificationProtocolSeal, State: "pass", EvidenceSHA256: protocol.SealSHA256},
	} {
		draft, err = bindBehaviorVerification(draft, binding)
		if err != nil {
			return SealedBehaviorEvidence{}, err
		}
	}
	sort.Slice(draft.Verification, func(i, j int) bool { return draft.Verification[i].Name < draft.Verification[j].Name })
	return SealWitnessed(draft, anchor, witness)
}

func bindBehaviorVerification(draft BehaviorEvidenceDraft, required Verification) (BehaviorEvidenceDraft, error) {
	for _, existing := range draft.Verification {
		if existing.Name != required.Name {
			continue
		}
		if existing.State != required.State || existing.EvidenceSHA256 != required.EvidenceSHA256 {
			return BehaviorEvidenceDraft{}, fmt.Errorf("behavior verification %q does not match the supplied receipt", required.Name)
		}
		return draft, nil
	}
	draft.Verification = append(draft.Verification, required)
	return draft, nil
}

func contextHasSourceDigest(context ProvenanceContextPacket, digest string) bool {
	for _, source := range context.Sources {
		if source.SHA256 == digest {
			return true
		}
	}
	return false
}

func decodeFactValue(fact ProvenanceFact, target any) error {
	if err := json.Unmarshal(fact.Value, target); err != nil {
		return fmt.Errorf("decode provenance fact %q: %w", fact.Path, err)
	}
	return nil
}
