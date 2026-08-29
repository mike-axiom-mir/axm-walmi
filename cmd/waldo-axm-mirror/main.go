package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "lens-corpus":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = lensCorpusFile(os.Args[2], os.Args[3])
	case "witness-run":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = witnessRunFile(os.Args[2], os.Args[3], os.Args[4])
	case "profile-contract":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = profileContractFile(os.Args[2], os.Args[3], os.Args[4])
	case "anchor":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = anchorFile(os.Args[2], os.Args[3])
	case "lock":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = lockFile(os.Args[2], os.Args[3], os.Args[4])
	case "contamination":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = contaminationFile(os.Args[2], os.Args[3])
	case "context":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = contextFile(os.Args[2], os.Args[3])
	case "gate-claims":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = gateClaimsFile(os.Args[2], os.Args[3], os.Args[4])
	case "seal-evaluation":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = sealEvaluationFile(os.Args[2], os.Args[3])
	case "intake-sensory":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = intakeSensoryFile(os.Args[2], os.Args[3])
	case "assess-skills":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = assessSkillsFile(os.Args[2], os.Args[3])
	case "discover":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = discoveryFile(os.Args[2], os.Args[3])
	case "situated-context":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = situatedContextFile(os.Args[2], os.Args[3], os.Args[4])
	case "forge-asset":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = forgeInnerAssetFile(os.Args[2], os.Args[3])
	case "verify-asset":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		err = verifyInnerAssetFile(os.Args[2])
	case "apply-candidate":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = applyCandidateFile(os.Args[2], os.Args[3], os.Args[4])
	case "verify-experience-trajectory":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		err = verifyExperienceTrajectoryFile(os.Args[2])
	case "census-capabilities":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = censusCapabilitiesFile(os.Args[2], os.Args[3])
	case "intake-capabilities":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = intakeCapabilitiesFile(os.Args[2], os.Args[3])
	case "plan-handoff":
		if len(os.Args) != 6 {
			usage()
			os.Exit(2)
		}
		err = planHandoffFile(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
	case "seal-translation":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = sealTranslationFile(os.Args[2], os.Args[3], os.Args[4])
	case "verify-handoff-return":
		if len(os.Args) != 6 {
			usage()
			os.Exit(2)
		}
		err = verifyHandoffReturnFile(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
	case "assess-verifier-change":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = assessVerifierChangeFile(os.Args[2], os.Args[3], os.Args[4])
	case "materialize-verifier-change":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = materializeVerifierChangeFile(os.Args[2], os.Args[3], os.Args[4])
	case "plan-verifier-repair":
		if len(os.Args) != 6 {
			usage()
			os.Exit(2)
		}
		err = planVerifierRepairFile(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
	case "seal-tool-experience":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = sealToolExperienceFile(os.Args[2], os.Args[3])
	case "start-tool-memory":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = startToolMemoryFile(os.Args[2], os.Args[3])
	case "grow-tool-memory":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = growToolMemoryFile(os.Args[2], os.Args[3], os.Args[4])
	case "recall-tool-wisdom":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = recallToolWisdomFile(os.Args[2], os.Args[3], os.Args[4])
	case "seal-gated":
		if len(os.Args) != 10 {
			usage()
			os.Exit(2)
		}
		err = sealGatedFile(os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6], os.Args[7], os.Args[8], os.Args[9])
	case "seal-situated":
		if len(os.Args) != 11 {
			usage()
			os.Exit(2)
		}
		err = sealSituatedFile(os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6], os.Args[7], os.Args[8], os.Args[9], os.Args[10])
	case "seal":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		err = sealFile(os.Args[2], os.Args[3])
	case "seal-anchored":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		err = sealAnchoredFile(os.Args[2], os.Args[3], os.Args[4])
	case "seal-witnessed":
		if len(os.Args) != 6 {
			usage()
			os.Exit(2)
		}
		err = sealWitnessedFile(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
	case "verify":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		err = verifyFile(os.Args[2])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror lens-corpus <corpus-bom.json> <lens.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror witness-run <run-bom.json> <run.json> <witness.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror profile-contract <run-bom.json> <run-witness.json> <profile.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror anchor <waldo-bom.json> <anchor.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror lock <expected-anchor.json> <observed-bom.json> <receipt.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror contamination <comparison.json> <report.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror context <context-request.json> <context-packet.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror gate-claims <context-packet.json> <claim-submission.json> <assessment.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-evaluation <protocol-draft.json> <protocol-seal.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror intake-sensory <sensory-draft.json> <sensory-receipt.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror assess-skills <skill-continuity-request.json> <skill-continuity-receipt.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror discover <discovery-request.json> <discovery-packet.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror situated-context <context-packet.json> <situated-request.json> <situated-envelope.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror forge-asset <inner-asset-recipe.json> <candidate.axmasset>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror verify-asset <candidate.axmasset>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror apply-candidate <request.json> <workspace-root> <receipt.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror verify-experience-trajectory <record.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror census-capabilities <census-request.json> <self-snapshot.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror intake-capabilities <external-snapshot.json> <external-receipt.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror plan-handoff <self-snapshot.json> <external-receipt.json> <gap-request.json> <plan.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-translation <plan.json> <translation-declaration.json> <translation-receipt.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror verify-handoff-return <plan.json> <translation-receipt.json> <return-draft.json> <return-receipt.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror assess-verifier-change <registry.json> <change-request.json> <receipt.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror materialize-verifier-change <registry.json> <ready-receipt.json> <candidate-registry.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror plan-verifier-repair <registry.json> <failed-receipt.json> <repair-request.json> <repair-plan.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-tool-experience <experience-draft.json> <sealed-experience.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror start-tool-memory <sealed-experience.json> <memory-shard.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror grow-tool-memory <memory-shard.json> <sealed-experience.json> <next-memory-shard.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror recall-tool-wisdom <memory-shard.json> <wisdom-query.json> <wisdom-view.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-gated <anchor.json> <run-witness.json> <profile.json> <context.json> <claims.json> <protocol.json> <draft.json> <sealed.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-situated <anchor.json> <run-witness.json> <profile.json> <context.json> <claims.json> <protocol.json> <situated.json> <draft.json> <sealed.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal <draft.json> <sealed.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-anchored <anchor.json> <draft.json> <sealed.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror seal-witnessed <anchor.json> <run-witness.json> <draft.json> <sealed.json>")
	fmt.Fprintln(os.Stderr, "  waldo-axm-mirror verify <sealed.json>")
}

func lensCorpusFile(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}
	lens, err := axmmirror.LensCorpusBOM(data)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, lens); err != nil {
		return err
	}
	fmt.Println(lens.State, lens.BOMSHA256)
	return nil
}

func witnessRunFile(runBOMPath, runPath, outputPath string) error {
	runBOMData, err := os.ReadFile(runBOMPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", runBOMPath, err)
	}
	runData, err := os.ReadFile(runPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", runPath, err)
	}
	witness, err := axmmirror.WitnessTrainingRun(runBOMData, runData)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, witness); err != nil {
		return err
	}
	fmt.Println(witness.State, witness.RunBOMSHA256)
	if witness.State != axmmirror.RunWitnessStateReady {
		return errors.New("training run witness is HOLD; inspect the written receipt")
	}
	return nil
}

func profileContractFile(runBOMPath, witnessPath, outputPath string) error {
	runBOMData, err := os.ReadFile(runBOMPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", runBOMPath, err)
	}
	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(witnessPath, &witness); err != nil {
		return err
	}
	receipt, err := axmmirror.LensTrainingProfile(runBOMData, witness)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.CanonicalProfile)
	if receipt.State != axmmirror.ProfileStateWitnessed && receipt.State != axmmirror.ProfileStateLegacyWitnessed {
		return fmt.Errorf("training profile contract is %s; inspect the written receipt", receipt.State)
	}
	return nil
}

func anchorFile(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}
	anchor, err := axmmirror.AnchorWALDOBOM(data)
	if err != nil {
		return err
	}
	if err := anchor.Validate(); err != nil {
		return err
	}
	if err := writeJSON(outputPath, anchor); err != nil {
		return err
	}
	fmt.Println(anchor.State, anchor.AnsweringIdentitySHA256)
	if anchor.State != axmmirror.AnchorStateAnchored {
		return errors.New("origin anchor is HOLD; inspect the written receipt")
	}
	return nil
}

func lockFile(expectedPath, observedPath, outputPath string) error {
	var expected axmmirror.OriginAnchor
	if err := readStrictJSON(expectedPath, &expected); err != nil {
		return err
	}
	data, err := os.ReadFile(observedPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", observedPath, err)
	}
	observed, err := axmmirror.AnchorWALDOBOM(data)
	if err != nil {
		return err
	}
	receipt, err := axmmirror.CompareIdentity(expected, observed)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.Observed.AnsweringIdentitySHA256)
	if receipt.State != axmmirror.IdentityStateLocked {
		return fmt.Errorf("release identity lock is %s; inspect the written receipt", receipt.State)
	}
	return nil
}

func contaminationFile(inputPath, outputPath string) error {
	var comparison axmmirror.EvaluationComparison
	if err := readStrictJSON(inputPath, &comparison); err != nil {
		return err
	}
	report, err := axmmirror.CheckContamination(comparison)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, report); err != nil {
		return err
	}
	fmt.Println(report.State)
	if report.State != axmmirror.ContaminationStateClear {
		return fmt.Errorf("evaluation contamination guard is %s; inspect the written report", report.State)
	}
	return nil
}

func contextFile(inputPath, outputPath string) error {
	var request axmmirror.ProvenanceContextRequest
	if err := readStrictJSON(inputPath, &request); err != nil {
		return err
	}
	packet, err := axmmirror.BuildProvenanceContext(request)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, packet); err != nil {
		return err
	}
	fmt.Println(packet.State, packet.PacketSHA256)
	if packet.State != axmmirror.ContextStateReady {
		return fmt.Errorf("provenance context is %s; inspect the written packet", packet.State)
	}
	return nil
}

func gateClaimsFile(contextPath, submissionPath, outputPath string) error {
	var packet axmmirror.ProvenanceContextPacket
	if err := readStrictJSON(contextPath, &packet); err != nil {
		return err
	}
	var submission axmmirror.SourceClaimSubmission
	if err := readStrictJSON(submissionPath, &submission); err != nil {
		return err
	}
	assessment, err := axmmirror.AssessSourceClaims(submission, packet)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, assessment); err != nil {
		return err
	}
	fmt.Println(assessment.State, assessment.ReceiptSHA256)
	if assessment.State != axmmirror.ClaimGateStateConfirmed {
		return fmt.Errorf("source-claim gate is %s; inspect the written assessment", assessment.State)
	}
	return nil
}

func sealEvaluationFile(inputPath, outputPath string) error {
	var draft axmmirror.EvaluationProtocolDraft
	if err := readStrictJSON(inputPath, &draft); err != nil {
		return err
	}
	seal, err := axmmirror.SealEvaluationProtocol(draft)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, seal); err != nil {
		return err
	}
	fmt.Println(seal.State, seal.SealSHA256)
	if seal.State != axmmirror.ProtocolStateReady {
		return fmt.Errorf("evaluation protocol is %s; inspect the written seal", seal.State)
	}
	return nil
}

func intakeSensoryFile(inputPath, outputPath string) error {
	var draft axmmirror.SensoryEvidenceDraft
	if err := readStrictJSON(inputPath, &draft); err != nil {
		return err
	}
	receipt, err := axmmirror.IntakeSensoryEvidence(draft)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.ReceiptSHA256)
	if receipt.State != axmmirror.SensoryStateReady {
		return fmt.Errorf("sensory evidence intake is %s; inspect the written receipt", receipt.State)
	}
	return nil
}

func assessSkillsFile(inputPath, outputPath string) error {
	var request axmmirror.SkillContinuityRequest
	if err := readStrictJSON(inputPath, &request); err != nil {
		return err
	}
	receipt, err := axmmirror.AssessSkillContinuity(request)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.ReceiptSHA256)
	if receipt.State != axmmirror.SkillContinuityReady {
		return fmt.Errorf("skill continuity is %s; inspect the written receipt", receipt.State)
	}
	return nil
}

func discoveryFile(inputPath, outputPath string) error {
	var request axmmirror.DiscoveryStanceRequest
	if err := readStrictJSON(inputPath, &request); err != nil {
		return err
	}
	packet, err := axmmirror.BuildDiscoveryStance(request)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, packet); err != nil {
		return err
	}
	fmt.Println(packet.State, packet.ReceiptSHA256)
	if packet.State != axmmirror.DiscoveryStateReady {
		return fmt.Errorf("discovery stance is %s; inspect the written packet", packet.State)
	}
	return nil
}

func situatedContextFile(contextPath, requestPath, outputPath string) error {
	var context axmmirror.ProvenanceContextPacket
	if err := readStrictJSON(contextPath, &context); err != nil {
		return err
	}
	var request axmmirror.SituatedContextRequest
	if err := readStrictJSON(requestPath, &request); err != nil {
		return err
	}
	envelope, err := axmmirror.BuildSituatedContext(request, context)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, envelope); err != nil {
		return err
	}
	fmt.Println(envelope.State, envelope.EnvelopeSHA256)
	if envelope.State != axmmirror.SituatedContextReady {
		return fmt.Errorf("situated context is %s; inspect the written envelope", envelope.State)
	}
	return nil
}

func forgeInnerAssetFile(inputPath, outputPath string) error {
	var recipe axmmirror.InnerAssetRecipe
	if err := readStrictJSON(inputPath, &recipe); err != nil {
		return err
	}
	build, err := axmmirror.ForgeInnerAsset(recipe)
	if err != nil {
		return err
	}
	bundle, err := axmmirror.EncodeInnerAssetBundle(build)
	if err != nil {
		return err
	}
	if err := writeBytesNoReplace(outputPath, bundle); err != nil {
		return err
	}
	fmt.Println(build.Candidate.State, build.Candidate.CandidateSHA256)
	if build.Candidate.State != axmmirror.InnerAssetStateReady {
		return fmt.Errorf("inner asset candidate is %s; inspect the written bundle", build.Candidate.State)
	}
	return nil
}

func verifyInnerAssetFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	candidate, err := axmmirror.VerifyInnerAssetBundle(data)
	if err != nil {
		return err
	}
	fmt.Println("OK", candidate.State, candidate.CandidateSHA256)
	return nil
}

func applyCandidateFile(requestPath, rootPath, receiptPath string) error {
	if _, err := os.Lstat(receiptPath); err == nil {
		return fmt.Errorf("output %s already exists", receiptPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect output %s: %w", receiptPath, err)
	}
	var request axmmirror.CandidateWriteRequest
	if err := readStrictJSON(requestPath, &request); err != nil {
		return err
	}
	receipt, err := axmmirror.ApplyCandidateWrites(rootPath, request)
	if err != nil {
		return err
	}
	if err := writeJSON(receiptPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.ReceiptSHA256)
	return nil
}

func verifyExperienceTrajectoryFile(path string) error {
	var record axmmirror.MirrorExperienceTrajectoryRecord
	if err := readStrictJSON(path, &record); err != nil {
		return err
	}
	projection, receipt, err := axmmirror.ProjectMirrorExperienceTrajectory(record)
	if err != nil {
		return err
	}
	if len(projection) == 0 {
		return errors.New("experience trajectory projection is empty")
	}
	fmt.Println(receipt.State, receipt.RecordSHA256, receipt.ProjectionSHA256, receipt.ReceiptSHA256)
	return nil
}

func censusCapabilitiesFile(inputPath, outputPath string) error {
	var request axmmirror.SelfCapabilityCensusRequest
	if err := readStrictJSON(inputPath, &request); err != nil {
		return err
	}
	snapshot, err := axmmirror.CensusSelfCapabilities(request)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, snapshot); err != nil {
		return err
	}
	fmt.Println(snapshot.State, snapshot.SnapshotSHA256)
	return nil
}

func intakeCapabilitiesFile(inputPath, outputPath string) error {
	var snapshot axmmirror.ExternalCapabilitySnapshot
	if err := readStrictJSON(inputPath, &snapshot); err != nil {
		return err
	}
	receipt, err := axmmirror.IntakeExternalCapabilities(snapshot)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.ReceiptSHA256)
	if receipt.State != axmmirror.ExternalCapabilityLive && receipt.State != axmmirror.ExternalCapabilitySourceOnly {
		return fmt.Errorf("external capability intake is %s; inspect the written receipt", receipt.State)
	}
	return nil
}

func planHandoffFile(selfPath, externalPath, requestPath, outputPath string) error {
	var self axmmirror.SelfCapabilitySnapshot
	if err := readStrictJSON(selfPath, &self); err != nil {
		return err
	}
	var external axmmirror.ExternalCapabilityReceipt
	if err := readStrictJSON(externalPath, &external); err != nil {
		return err
	}
	var request axmmirror.CapabilityGapRequest
	if err := readStrictJSON(requestPath, &request); err != nil {
		return err
	}
	plan, err := axmmirror.PlanCapabilityHandoff(self, external, request)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, plan); err != nil {
		return err
	}
	fmt.Println(plan.State, plan.PlanSHA256)
	if plan.State != axmmirror.HandoffStateReady && plan.State != axmmirror.HandoffStateLocal {
		return fmt.Errorf("capability handoff plan is %s; inspect the written plan", plan.State)
	}
	return nil
}

func sealTranslationFile(planPath, declarationPath, outputPath string) error {
	var plan axmmirror.CapabilityHandoffPlan
	if err := readStrictJSON(planPath, &plan); err != nil {
		return err
	}
	var declaration axmmirror.CapabilityTranslationDeclaration
	if err := readStrictJSON(declarationPath, &declaration); err != nil {
		return err
	}
	receipt, err := axmmirror.SealCapabilityTranslation(plan, declaration)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.ReceiptSHA256)
	if receipt.State != axmmirror.TranslationStateNotRequired && receipt.State != axmmirror.TranslationStateNoDeclaredLoss {
		return fmt.Errorf("translation loss receipt is %s; inspect the written receipt", receipt.State)
	}
	return nil
}

func verifyHandoffReturnFile(planPath, translationPath, draftPath, outputPath string) error {
	var plan axmmirror.CapabilityHandoffPlan
	if err := readStrictJSON(planPath, &plan); err != nil {
		return err
	}
	var translation axmmirror.TranslationLossReceipt
	if err := readStrictJSON(translationPath, &translation); err != nil {
		return err
	}
	var draft axmmirror.CapabilityReturnDraft
	if err := readStrictJSON(draftPath, &draft); err != nil {
		return err
	}
	receipt, err := axmmirror.VerifyCapabilityReturn(plan, translation, draft)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.ReceiptSHA256)
	if receipt.State != axmmirror.CapabilityReturnVerified {
		return fmt.Errorf("capability return is %s; inspect the written receipt", receipt.State)
	}
	return nil
}

func assessVerifierChangeFile(registryPath, requestPath, outputPath string) error {
	var registry axmmirror.VerifierRegistry
	if err := readStrictJSON(registryPath, &registry); err != nil {
		return err
	}
	var request axmmirror.VerifierChangeRequest
	if err := readStrictJSON(requestPath, &request); err != nil {
		return err
	}
	receipt, err := axmmirror.AssessVerifierChange(registry, request)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, receipt); err != nil {
		return err
	}
	fmt.Println(receipt.State, receipt.ReceiptSHA256)
	if receipt.State != axmmirror.VerifierChangeReady {
		return fmt.Errorf("verifier change is %s; inspect the written rollback or HOLD receipt", receipt.State)
	}
	return nil
}

func materializeVerifierChangeFile(registryPath, receiptPath, outputPath string) error {
	var registry axmmirror.VerifierRegistry
	if err := readStrictJSON(registryPath, &registry); err != nil {
		return err
	}
	var receipt axmmirror.VerifierChangeReceipt
	if err := readStrictJSON(receiptPath, &receipt); err != nil {
		return err
	}
	candidate, err := axmmirror.MaterializeVerifierChange(registry, receipt)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, candidate); err != nil {
		return err
	}
	fmt.Println("VERIFIER_REGISTRY_CANDIDATE", candidate.RegistrySHA256)
	return nil
}

func planVerifierRepairFile(registryPath, failedPath, requestPath, outputPath string) error {
	var registry axmmirror.VerifierRegistry
	if err := readStrictJSON(registryPath, &registry); err != nil {
		return err
	}
	var failed axmmirror.VerifierChangeReceipt
	if err := readStrictJSON(failedPath, &failed); err != nil {
		return err
	}
	var request axmmirror.RepairBuddyRequest
	if err := readStrictJSON(requestPath, &request); err != nil {
		return err
	}
	plan, err := axmmirror.PlanVerifierRepair(registry, failed, request)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, plan); err != nil {
		return err
	}
	fmt.Println(plan.State, plan.PlanSHA256)
	if plan.State != axmmirror.RepairBuddyCandidateReady {
		return fmt.Errorf("Repair Buddy plan is %s; inspect the written plan", plan.State)
	}
	return nil
}

func sealToolExperienceFile(inputPath, outputPath string) error {
	var experience axmmirror.IdentityToolExperience
	if err := readStrictJSON(inputPath, &experience); err != nil {
		return err
	}
	sealed, err := axmmirror.SealIdentityToolExperience(experience)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println("TOOL_EXPERIENCE_SEALED", sealed.ExperienceSHA256)
	return nil
}

func startToolMemoryFile(experiencePath, outputPath string) error {
	var experience axmmirror.IdentityToolExperience
	if err := readStrictJSON(experiencePath, &experience); err != nil {
		return err
	}
	shard, err := axmmirror.StartIdentityToolMemory(experience)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, shard); err != nil {
		return err
	}
	fmt.Println(shard.State, shard.ShardSHA256)
	return nil
}

func growToolMemoryFile(currentPath, experiencePath, outputPath string) error {
	var current axmmirror.IdentityToolMemoryShard
	if err := readStrictJSON(currentPath, &current); err != nil {
		return err
	}
	var experience axmmirror.IdentityToolExperience
	if err := readStrictJSON(experiencePath, &experience); err != nil {
		return err
	}
	next, err := axmmirror.GrowIdentityToolMemory(current, experience)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, next); err != nil {
		return err
	}
	fmt.Println(next.State, next.ShardSHA256)
	return nil
}

func recallToolWisdomFile(shardPath, queryPath, outputPath string) error {
	var shard axmmirror.IdentityToolMemoryShard
	if err := readStrictJSON(shardPath, &shard); err != nil {
		return err
	}
	var query axmmirror.IdentityWisdomQuery
	if err := readStrictJSON(queryPath, &query); err != nil {
		return err
	}
	view, err := axmmirror.RecallIdentityToolWisdom(shard, query)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, view); err != nil {
		return err
	}
	fmt.Println(view.State, view.ViewSHA256)
	if view.State != axmmirror.ToolWisdomReady {
		return fmt.Errorf("identity tool wisdom is %s; inspect the written view", view.State)
	}
	return nil
}

func sealGatedFile(anchorPath, witnessPath, profilePath, contextPath, claimsPath, protocolPath, draftPath, outputPath string) error {
	var anchor axmmirror.OriginAnchor
	if err := readStrictJSON(anchorPath, &anchor); err != nil {
		return err
	}
	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(witnessPath, &witness); err != nil {
		return err
	}
	var profile axmmirror.TrainingProfileContract
	if err := readStrictJSON(profilePath, &profile); err != nil {
		return err
	}
	var context axmmirror.ProvenanceContextPacket
	if err := readStrictJSON(contextPath, &context); err != nil {
		return err
	}
	var claims axmmirror.SourceClaimAssessment
	if err := readStrictJSON(claimsPath, &claims); err != nil {
		return err
	}
	var protocol axmmirror.EvaluationProtocolSeal
	if err := readStrictJSON(protocolPath, &protocol); err != nil {
		return err
	}
	var draft axmmirror.BehaviorEvidenceDraft
	if err := readStrictJSON(draftPath, &draft); err != nil {
		return err
	}
	sealed, err := axmmirror.SealGated(draft, anchor, witness, profile, context, claims, protocol)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println(sealed.SHA256)
	return nil
}

func sealSituatedFile(anchorPath, witnessPath, profilePath, contextPath, claimsPath, protocolPath, situatedPath, draftPath, outputPath string) error {
	var anchor axmmirror.OriginAnchor
	if err := readStrictJSON(anchorPath, &anchor); err != nil {
		return err
	}
	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(witnessPath, &witness); err != nil {
		return err
	}
	var profile axmmirror.TrainingProfileContract
	if err := readStrictJSON(profilePath, &profile); err != nil {
		return err
	}
	var context axmmirror.ProvenanceContextPacket
	if err := readStrictJSON(contextPath, &context); err != nil {
		return err
	}
	var claims axmmirror.SourceClaimAssessment
	if err := readStrictJSON(claimsPath, &claims); err != nil {
		return err
	}
	var protocol axmmirror.EvaluationProtocolSeal
	if err := readStrictJSON(protocolPath, &protocol); err != nil {
		return err
	}
	var situated axmmirror.SituatedContextEnvelope
	if err := readStrictJSON(situatedPath, &situated); err != nil {
		return err
	}
	var draft axmmirror.BehaviorEvidenceDraft
	if err := readStrictJSON(draftPath, &draft); err != nil {
		return err
	}
	sealed, err := axmmirror.SealSituatedGated(draft, anchor, witness, profile, context, claims, protocol, situated)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println(sealed.SHA256)
	return nil
}

func sealFile(inputPath, outputPath string) error {
	var draft axmmirror.BehaviorEvidenceDraft
	if err := readStrictJSON(inputPath, &draft); err != nil {
		return err
	}
	sealed, err := axmmirror.Seal(draft)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println(sealed.SHA256)
	return nil
}

func sealAnchoredFile(anchorPath, inputPath, outputPath string) error {
	var anchor axmmirror.OriginAnchor
	if err := readStrictJSON(anchorPath, &anchor); err != nil {
		return err
	}
	var draft axmmirror.BehaviorEvidenceDraft
	if err := readStrictJSON(inputPath, &draft); err != nil {
		return err
	}
	sealed, err := axmmirror.SealAnchored(draft, anchor)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println(sealed.SHA256)
	return nil
}

func sealWitnessedFile(anchorPath, witnessPath, inputPath, outputPath string) error {
	var anchor axmmirror.OriginAnchor
	if err := readStrictJSON(anchorPath, &anchor); err != nil {
		return err
	}
	var witness axmmirror.TrainingRunWitness
	if err := readStrictJSON(witnessPath, &witness); err != nil {
		return err
	}
	var draft axmmirror.BehaviorEvidenceDraft
	if err := readStrictJSON(inputPath, &draft); err != nil {
		return err
	}
	sealed, err := axmmirror.SealWitnessed(draft, anchor, witness)
	if err != nil {
		return err
	}
	if err := writeJSON(outputPath, sealed); err != nil {
		return err
	}
	fmt.Println(sealed.SHA256)
	return nil
}

func verifyFile(path string) error {
	var sealed axmmirror.SealedBehaviorEvidence
	if err := readStrictJSON(path, &sealed); err != nil {
		return err
	}
	if err := sealed.Verify(); err != nil {
		return err
	}
	fmt.Println("OK", sealed.SHA256)
	return nil
}

func readStrictJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := axmmirror.DecodeStrictJSON(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	return writeBytesNoReplace(path, data)
}

func writeBytesNoReplace(path string, data []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create output directory for %s: %w", path, err)
	}
	temporary, err := os.CreateTemp(directory, ".waldo-axm-mirror-*")
	if err != nil {
		return fmt.Errorf("create temporary output for %s: %w", path, err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write temporary output for %s: %w", path, err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		return fmt.Errorf("set output mode for %s: %w", path, err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync output for %s: %w", path, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close output for %s: %w", path, err)
	}
	// Linking the synced temporary file is an atomic, no-replace commit. It
	// prevents a second run from silently overwriting an earlier receipt.
	if err := os.Link(temporaryPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("output %s already exists", path)
		}
		return fmt.Errorf("commit output %s: %w", path, err)
	}
	return nil
}
