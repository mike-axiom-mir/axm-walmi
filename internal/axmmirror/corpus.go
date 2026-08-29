package axmmirror

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	pathpkg "path"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	CorpusEvidenceLensSchemaV1 = "axm.waldo-witness.corpus-evidence-lens/v0.1"
	CorpusEvidenceLensSchema   = "axm.waldo-witness.corpus-evidence-lens/v0.2"
	CorpusEvidenceStateReady   = "READY"

	waldoTextRecordSchema        = 2
	waldoFormerTextRecordSchema  = 1
	waldoTextWriterRecipe        = "parquet-go/0.30.1/zstd-6/page-1m/rg-64m/v9-privacy-redaction"
	waldoFormerMainContentRecipe = "parquet-go/0.30.1/zstd-6/page-1m/rg-64m/v8-main-content"
	waldoFormerAssessmentRecipe  = "parquet-go/0.30.1/zstd-6/page-1m/rg-64m/v7-content-assessment"
	waldoFormerTextBOMRecipe     = "parquet-go/0.30.1/zstd-6/page-1m/rg-64m/v5-bom"
	waldoFormerTextRecipe        = "parquet-go/0.30.1/zstd-6/page-1m/rg-64m/v4"
	waldoPrivacyRedactionPolicy  = "waldo/privacy-redaction-v1"
)

// EvidenceMeasures is WALDO's exact additive shard, document, token, and byte
// accounting. It is copied into receipts rather than recomputed by a model.
type EvidenceMeasures struct {
	Shards int64 `json:"shards"`
	Docs   int64 `json:"docs"`
	Tokens int64 `json:"tokens"`
	Bytes  int64 `json:"bytes"`
}

// ModalityEvidence preserves WALDO's per-modality accounting without treating
// samples from different modalities as mutually exclusive documents.
type ModalityEvidence struct {
	Samples      int64 `json:"samples,omitempty"`
	Items        int64 `json:"items,omitempty"`
	Tokens       int64 `json:"tokens,omitempty"`
	DurationMS   int64 `json:"duration_ms,omitempty"`
	ContentBytes int64 `json:"content_bytes,omitempty"`
}

type CorpusIndexEvidence struct {
	Remote string `json:"remote,omitempty"`
	Commit string `json:"commit,omitempty"`
	Dirty  bool   `json:"dirty"`
	State  string `json:"state"`
}

type CorpusLicensePolicy struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

type SourceLicenseAssertion struct {
	Normalized  string `json:"normalized,omitempty"`
	Declaration string `json:"declaration,omitempty"`
	EvidenceURL string `json:"evidence_url,omitempty"`
	State       string `json:"state"`
}

type CorpusSourceEvidence struct {
	Name              string                 `json:"name"`
	Source            string                 `json:"source"`
	Version           string                 `json:"version,omitempty"`
	URL               string                 `json:"url"`
	SHA256            string                 `json:"sha256"`
	License           SourceLicenseAssertion `json:"license"`
	SourceFileCount   int                    `json:"source_file_count"`
	SourceFilesSHA256 string                 `json:"source_files_sha256,omitempty"`
}

type CorpusManifestEvidence struct {
	Path              string                 `json:"path"`
	SHA256            string                 `json:"sha256"`
	Name              string                 `json:"name"`
	LicenseAssertions []string               `json:"license_assertions"`
	Sources           []CorpusSourceEvidence `json:"sources"`
}

type CorpusLicenseEvidence struct {
	ID       string           `json:"id"`
	State    string           `json:"state"`
	Measures EvidenceMeasures `json:"measures"`
}

type CorpusAttestationEvidence struct {
	State         string `json:"state"`
	Embedded      int64  `json:"embedded"`
	ImplicitV4    int64  `json:"implicit_v4"`
	DeepValidated int64  `json:"deep_validated"`
	NotRecorded   int64  `json:"not_recorded"`
}

// CorpusRecordFilterEvidence binds the exact portable selection policy without
// expanding an unbounded per-corpus filter map into the public receipt.
type CorpusRecordFilterEvidence struct {
	State             string `json:"state"`
	Schema            int    `json:"schema,omitempty"`
	PolicySHA256      string `json:"policy_sha256"`
	GlobalDeclared    bool   `json:"global_declared"`
	CorpusFilterCount int    `json:"corpus_filter_count"`
}

// CorpusAssessmentEvidence keeps deterministic row classifiers separate from
// source meaning. DetectorSetSHA256 binds every field/detector pair while the
// additive counts remain directly inspectable.
type CorpusAssessmentEvidence struct {
	State                     string `json:"state"`
	AssessedShards            int64  `json:"assessed_shards"`
	LegacyShards              int64  `json:"legacy_shards"`
	DetectorSetSHA256         string `json:"detector_set_sha256"`
	EmailAddressRecords       int64  `json:"email_address_records"`
	RepetitiveContentRecords  int64  `json:"repetitive_content_records"`
	BoilerplateContentRecords int64  `json:"boilerplate_content_records"`
}

// CorpusPrivacyRedactionEvidence reports exact deterministic transformations
// recorded by current WALDO shards. It never upgrades those counts into a
// claim of anonymity, privacy-law compliance, or absence of indirect identity.
type CorpusPrivacyRedactionEvidence struct {
	State                      string `json:"state"`
	Policy                     string `json:"policy,omitempty"`
	NamesRetained              bool   `json:"names_retained"`
	RedactedShards             int64  `json:"redacted_shards"`
	UnredactedCompatibleShards int64  `json:"unredacted_compatible_shards"`
	EmailAddresses             int64  `json:"email_addresses"`
	IPAddresses                int64  `json:"ip_addresses"`
	PhoneNumbers               int64  `json:"phone_numbers"`
	MailRoutingHeaders         int64  `json:"mail_routing_headers"`
	Credentials                int64  `json:"credentials"`
}

// CorpusEvidenceLens is a bounded, deterministic projection of a documented
// OpenWALDO corpus BOM. READY means that the receipt passed the structural and
// aggregate checks implemented here; the boundary notices state what it does
// not establish.
type CorpusEvidenceLens struct {
	Schema           string                          `json:"schema"`
	State            string                          `json:"state"`
	BOMSHA256        string                          `json:"bom_sha256"`
	DocumentSHA256   string                          `json:"document_sha256"`
	ReceiptSHA256    string                          `json:"receipt_sha256,omitempty"`
	Index            CorpusIndexEvidence             `json:"index"`
	Paths            []string                        `json:"paths"`
	LicensePolicy    CorpusLicensePolicy             `json:"license_policy"`
	RecordFilter     *CorpusRecordFilterEvidence     `json:"record_filter,omitempty"`
	Manifests        []CorpusManifestEvidence        `json:"manifests"`
	SourceSetSHA256  string                          `json:"source_set_sha256"`
	ShardSetSHA256   string                          `json:"shard_set_sha256"`
	Totals           EvidenceMeasures                `json:"totals"`
	Modalities       map[string]ModalityEvidence     `json:"modalities,omitempty"`
	Licenses         []CorpusLicenseEvidence         `json:"licenses"`
	Attestation      CorpusAttestationEvidence       `json:"attestation"`
	Assessment       *CorpusAssessmentEvidence       `json:"assessment,omitempty"`
	PrivacyRedaction *CorpusPrivacyRedactionEvidence `json:"privacy_redaction,omitempty"`
	Notices          []string                        `json:"notices"`
	Authority        Authority                       `json:"authority"`
}

// The wire types below mirror the documented schema-1 JSON order. They stay
// local to this fork: the organ consumes WALDO's durable artifact instead of
// importing WALDO implementation packages as an accidental API.
type waldoCorpusBOM struct {
	Kind         string                      `json:"kind"`
	Schema       int                         `json:"schema"`
	Subject      string                      `json:"subject"`
	Index        waldoCorpusIndex            `json:"index"`
	Paths        []string                    `json:"paths"`
	Policy       CorpusLicensePolicy         `json:"license_policy,omitempty"`
	RecordFilter *waldoRecordFilterPolicy    `json:"record_filter,omitempty"`
	Manifests    []waldoCorpusManifest       `json:"manifests"`
	SubManifests []waldoCorpusSubManifest    `json:"sub_manifests,omitempty"`
	Shards       []waldoCorpusShard          `json:"shards"`
	Totals       EvidenceMeasures            `json:"totals"`
	Modalities   map[string]ModalityEvidence `json:"modalities,omitempty"`
	Licenses     map[string]EvidenceMeasures `json:"licenses"`
}

type waldoRecordFilterPolicy struct {
	Schema  int                          `json:"schema"`
	Global  *waldoRecordFilter           `json:"global,omitempty"`
	Corpora map[string]waldoRecordFilter `json:"corpora,omitempty"`
}

type waldoRecordFilter struct {
	MainContent *bool                 `json:"main_content,omitempty"`
	Exclude     *waldoExclusionFilter `json:"exclude,omitempty"`
	Licenses    *waldoValueFilter     `json:"licenses,omitempty"`
	Languages   *waldoValueFilter     `json:"languages,omitempty"`
	Sources     *waldoValueFilter     `json:"sources,omitempty"`
	Date        *waldoDateFilter      `json:"date,omitempty"`
}

type waldoExclusionFilter struct {
	RepetitiveContent  *bool    `json:"repetitive_content,omitempty"`
	BoilerplateContent *bool    `json:"boilerplate_content,omitempty"`
	Licenses           []string `json:"licenses,omitempty"`
}

type waldoValueFilter struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

type waldoDateFilter struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

type waldoCorpusIndex struct {
	Remote string `json:"remote,omitempty"`
	Commit string `json:"commit,omitempty"`
	Dirty  bool   `json:"dirty,omitempty"`
}

type waldoCorpusSubManifest struct {
	Manifest     string                      `json:"manifest"`
	ParentSHA256 string                      `json:"parent_sha256,omitempty"`
	URL          string                      `json:"url"`
	SHA256       string                      `json:"sha256"`
	Count        int64                       `json:"count"`
	Docs         int64                       `json:"docs"`
	Tokens       int64                       `json:"tokens"`
	Bytes        int64                       `json:"bytes"`
	EncodedBytes int64                       `json:"encoded_bytes"`
	Modalities   map[string]ModalityEvidence `json:"modalities,omitempty"`
}

type waldoConversion struct {
	Tool      string `json:"tool"`
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	Collector string `json:"collector,omitempty"`
	Profile   string `json:"profile"`
	Recipe    string `json:"recipe"`
	Tokenizer string `json:"tokenizer,omitempty"`
}

type waldoLicenseAssertion struct {
	Declaration string `json:"declaration,omitempty"`
	URL         string `json:"url,omitempty"`
}

type waldoSourceFile struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	SHA256     string `json:"sha256"`
	Bytes      int64  `json:"bytes,omitempty"`
	Format     string `json:"format,omitempty"`
	Adapter    string `json:"adapter,omitempty"`
	TextColumn string `json:"text_column,omitempty"`
}

type waldoContent struct {
	Types            []string `json:"types,omitempty"`
	Languages        []string `json:"languages,omitempty"`
	Geographies      []string `json:"geographies,omitempty"`
	Demographics     []string `json:"demographics,omitempty"`
	From             string   `json:"from,omitempty"`
	To               string   `json:"to,omitempty"`
	Selection        string   `json:"selection,omitempty"`
	PersonalData     string   `json:"personal_data,omitempty"`
	Copyrighted      string   `json:"copyrighted,omitempty"`
	MachineGenerated string   `json:"machine_generated,omitempty"`
}

type waldoCrawler struct {
	Name      string   `json:"name"`
	Purpose   string   `json:"purpose"`
	Behaviour string   `json:"behaviour"`
	Protocols []string `json:"protocols,omitempty"`
}

type waldoUserData struct {
	Service     string `json:"service"`
	Interaction string `json:"interaction"`
}

type waldoSyntheticData struct {
	Model       string `json:"model"`
	Version     string `json:"version,omitempty"`
	SummaryURL  string `json:"summary_url,omitempty"`
	Description string `json:"description,omitempty"`
}

type waldoDomainMeasure struct {
	Domain        string `json:"domain"`
	AcquiredBytes int64  `json:"acquired_bytes,omitempty"`
	RetainedBytes int64  `json:"retained_bytes,omitempty"`
}

type waldoAcquisition struct {
	Basis     string               `json:"basis,omitempty"`
	Crawler   *waldoCrawler        `json:"crawler,omitempty"`
	UserData  *waldoUserData       `json:"user_data,omitempty"`
	Synthetic *waldoSyntheticData  `json:"synthetic,omitempty"`
	Domains   []waldoDomainMeasure `json:"domains,omitempty"`
}

type waldoSource struct {
	Name            string                      `json:"name"`
	Source          string                      `json:"source"`
	Version         string                      `json:"version,omitempty"`
	License         string                      `json:"license,omitempty"`
	LicenseEvidence *waldoLicenseAssertion      `json:"license_evidence,omitempty"`
	URL             string                      `json:"url"`
	Category        string                      `json:"category,omitempty"`
	CollectedFrom   string                      `json:"collected_from,omitempty"`
	CollectedTo     string                      `json:"collected_to,omitempty"`
	SHA256          string                      `json:"sha256"`
	Files           []waldoSourceFile           `json:"files,omitempty"`
	Usage           map[string]ModalityEvidence `json:"usage,omitempty"`
	Content         *waldoContent               `json:"content,omitempty"`
	Acquisition     *waldoAcquisition           `json:"acquisition,omitempty"`
}

type waldoProcessingStep struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type waldoProcessing struct {
	Steps                     []waldoProcessingStep `json:"steps,omitempty"`
	RightsReservationMeasures []string              `json:"rights_reservation_measures,omitempty"`
	IllegalContentMeasures    []string              `json:"illegal_content_measures,omitempty"`
}

type waldoRecipeStep struct {
	Name       string `json:"name"`
	Executable string `json:"script"`
	SHA256     string `json:"sha256"`
}

type waldoRecipeEvidence struct {
	Path       string            `json:"path"`
	SHA256     string            `json:"sha256"`
	Repository string            `json:"repository,omitempty"`
	Commit     string            `json:"commit,omitempty"`
	Dirty      bool              `json:"dirty"`
	Steps      []waldoRecipeStep `json:"steps"`
}

type waldoDetectionMeasure struct {
	Detector string `json:"detector"`
	Records  int64  `json:"records"`
}

type waldoContentAssessment struct {
	EmailAddresses     *waldoDetectionMeasure `json:"email_addresses,omitempty"`
	RepetitiveContent  *waldoDetectionMeasure `json:"repetitive_content,omitempty"`
	BoilerplateContent *waldoDetectionMeasure `json:"boilerplate_content,omitempty"`
}

type waldoContentRedaction struct {
	Policy             string `json:"policy"`
	NamesRetained      bool   `json:"names_retained"`
	EmailAddresses     int64  `json:"email_addresses"`
	IPAddresses        int64  `json:"ip_addresses"`
	PhoneNumbers       int64  `json:"phone_numbers"`
	MailRoutingHeaders int64  `json:"mail_routing_headers"`
	Credentials        int64  `json:"credentials"`
}

type waldoCorpusManifest struct {
	Path         string                      `json:"path"`
	SHA256       string                      `json:"sha256"`
	Name         string                      `json:"name"`
	Title        string                      `json:"title"`
	Description  string                      `json:"description"`
	License      string                      `json:"license"`
	LicenseSet   []string                    `json:"license_set,omitempty"`
	Format       string                      `json:"format"`
	RecordSchema int                         `json:"record_schema"`
	ConvertedBy  waldoConversion             `json:"converted_by"`
	Sources      []waldoSource               `json:"sources"`
	Processing   *waldoProcessing            `json:"processing,omitempty"`
	ComposedBy   *waldoRecipeEvidence        `json:"composed_by,omitempty"`
	Assessment   *waldoContentAssessment     `json:"assessment,omitempty"`
	Redaction    *waldoContentRedaction      `json:"redaction,omitempty"`
	Totals       EvidenceMeasures            `json:"totals"`
	Modalities   map[string]ModalityEvidence `json:"modalities,omitempty"`
	Licenses     map[string]EvidenceMeasures `json:"licenses"`
}

type waldoShardValidation struct {
	CanonicalRecords  bool `json:"canonical_records"`
	ContentHashes     bool `json:"content_hashes"`
	TokenCounts       bool `json:"token_counts"`
	ExactLicenseDedup bool `json:"exact_license_partition_dedup"`
}

type waldoShardBOM struct {
	Kind                      string                `json:"kind"`
	Schema                    int                   `json:"schema"`
	Subject                   string                `json:"subject"`
	PlanSHA256                string                `json:"plan_sha256"`
	RecordSchema              int                   `json:"record_schema"`
	WriterRecipe              string                `json:"writer_recipe"`
	Tokenizer                 string                `json:"tokenizer"`
	Records                   int64                 `json:"records"`
	Tokens                    int64                 `json:"tokens"`
	ContentBytes              int64                 `json:"content_bytes"`
	EmailAddressRecords       int64                 `json:"email_address_records,omitempty"`
	RepetitiveContentRecords  int64                 `json:"repetitive_content_records,omitempty"`
	BoilerplateContentRecords int64                 `json:"boilerplate_content_records,omitempty"`
	Redaction                 waldoContentRedaction `json:"redaction,omitempty"`
	Licenses                  []string              `json:"licenses"`
	Validation                waldoShardValidation  `json:"validation"`
}

type waldoShardAttestation struct {
	Status       string         `json:"status"`
	WriterRecipe string         `json:"writer_recipe,omitempty"`
	BOMSHA256    string         `json:"bom_sha256,omitempty"`
	BOM          *waldoShardBOM `json:"bom,omitempty"`
}

type waldoCorpusShard struct {
	Manifest          string                      `json:"manifest"`
	SubManifestSHA256 string                      `json:"sub_manifest_sha256,omitempty"`
	URL               string                      `json:"url"`
	SHA256            string                      `json:"sha256"`
	Format            string                      `json:"format"`
	RecordSchema      int                         `json:"record_schema"`
	License           string                      `json:"license"`
	Licenses          []string                    `json:"licenses,omitempty"`
	LicenseUsage      map[string]EvidenceMeasures `json:"license_usage,omitempty"`
	Sources           []string                    `json:"sources,omitempty"`
	ConvertedBy       waldoConversion             `json:"converted_by"`
	RecordsRoot       string                      `json:"records_root,omitempty"`
	Docs              int64                       `json:"docs"`
	Tokens            int64                       `json:"tokens"`
	Bytes             int64                       `json:"bytes"`
	Modalities        map[string]ModalityEvidence `json:"modalities,omitempty"`
	Attestation       *waldoShardAttestation      `json:"attestation,omitempty"`
	Assessment        *waldoContentAssessment     `json:"assessment,omitempty"`
	Redaction         *waldoContentRedaction      `json:"redaction,omitempty"`
}

type corpusShardIdentity struct {
	Ordinal           int      `json:"ordinal"`
	Manifest          string   `json:"manifest"`
	SubManifestSHA256 string   `json:"sub_manifest_sha256,omitempty"`
	URL               string   `json:"url"`
	SHA256            string   `json:"sha256"`
	Format            string   `json:"format"`
	RecordSchema      int      `json:"record_schema"`
	Licenses          []string `json:"licenses"`
	Sources           []string `json:"sources,omitempty"`
	RecordsRoot       string   `json:"records_root,omitempty"`
	Docs              int64    `json:"docs"`
	Tokens            int64    `json:"tokens"`
	Bytes             int64    `json:"bytes"`
	AttestationStatus string   `json:"attestation_status,omitempty"`
	AttestationSHA256 string   `json:"attestation_sha256,omitempty"`
}

// LensCorpusBOM validates and projects a schema-1 corpus BOM. Unknown fields
// are accepted for WALDO's additive compatibility rule, while duplicate keys
// and trailing JSON values remain ambiguous and are rejected.
func LensCorpusBOM(data []byte) (CorpusEvidenceLens, error) {
	var bom waldoCorpusBOM
	if err := decodeJSON(data, &bom, false); err != nil {
		return CorpusEvidenceLens{}, fmt.Errorf("decode corpus BOM: %w", err)
	}
	if err := validateCorpusBOM(bom); err != nil {
		return CorpusEvidenceLens{}, err
	}

	canonical, err := json.Marshal(bom)
	if err != nil {
		return CorpusEvidenceLens{}, fmt.Errorf("canonicalize corpus BOM: %w", err)
	}
	manifests, sourceSetDigest, err := projectCorpusManifests(bom.Manifests)
	if err != nil {
		return CorpusEvidenceLens{}, err
	}
	shardSetDigest, attestation, err := projectCorpusShards(bom.Shards)
	if err != nil {
		return CorpusEvidenceLens{}, err
	}
	recordFilter, err := projectCorpusRecordFilter(bom.RecordFilter)
	if err != nil {
		return CorpusEvidenceLens{}, err
	}
	assessment, privacyRedaction, err := projectCorpusRowEvidence(bom.Shards)
	if err != nil {
		return CorpusEvidenceLens{}, err
	}

	index := CorpusIndexEvidence{Remote: bom.Index.Remote, Commit: bom.Index.Commit, Dirty: bom.Index.Dirty}
	var notices []string
	switch {
	case bom.Index.Commit == "":
		index.State = "MANIFEST_PINNED_NO_COMMIT"
		notices = append(notices, "index revision is not recorded; manifest and shard digests remain the available identity pins")
	case bom.Index.Dirty:
		index.State = "DIRTY_MANIFEST_PINNED"
		notices = append(notices, "index checkout was dirty; the commit alone does not describe the selected manifest bytes")
	default:
		index.State = "CLEAN_PINNED"
	}
	notices = append(notices,
		"license values are recorded assertions and policy inputs, not legal conclusions",
		"hashes bind recorded identity and integrity; they do not prove truth, quality, safety, or actual trainer consumption",
		"row assessments are deterministic classifier records, not proof of source meaning, quality, or harmlessness",
		"privacy redaction counts prove only the recorded deterministic policy; names, indirect identifiers, and detector misses may remain, so anonymity and legal compliance are not established",
	)

	lens := CorpusEvidenceLens{
		Schema:           CorpusEvidenceLensSchema,
		State:            CorpusEvidenceStateReady,
		BOMSHA256:        digestBytes(canonical),
		DocumentSHA256:   digestBytes(data),
		Index:            index,
		Paths:            append([]string(nil), bom.Paths...),
		LicensePolicy:    cloneLicensePolicy(bom.Policy),
		RecordFilter:     &recordFilter,
		Manifests:        manifests,
		SourceSetSHA256:  sourceSetDigest,
		ShardSetSHA256:   shardSetDigest,
		Totals:           bom.Totals,
		Modalities:       cloneModalitiesEvidence(bom.Modalities),
		Licenses:         projectCorpusLicenses(bom.Licenses),
		Attestation:      attestation,
		Assessment:       &assessment,
		PrivacyRedaction: &privacyRedaction,
		Notices:          notices,
		Authority:        Authority{},
	}
	lens.ReceiptSHA256, err = corpusLensReceiptDigest(lens)
	if err != nil {
		return CorpusEvidenceLens{}, err
	}
	if err := lens.Validate(); err != nil {
		return CorpusEvidenceLens{}, fmt.Errorf("generated corpus evidence lens: %w", err)
	}
	return lens, nil
}

func (lens CorpusEvidenceLens) Validate() error {
	if !oneOf(lens.Schema, CorpusEvidenceLensSchemaV1, CorpusEvidenceLensSchema) || lens.State != CorpusEvidenceStateReady {
		return fmt.Errorf("unsupported corpus evidence lens identity %q state %q", lens.Schema, lens.State)
	}
	if lens.Schema == CorpusEvidenceLensSchemaV1 {
		if lens.RecordFilter != nil || lens.Assessment != nil || lens.PrivacyRedaction != nil {
			return errors.New("legacy corpus evidence lens must not carry v0.2 row-policy evidence")
		}
	} else {
		if lens.RecordFilter == nil || lens.Assessment == nil || lens.PrivacyRedaction == nil {
			return errors.New("corpus evidence lens v0.2 requires record-filter, assessment, and privacy-redaction evidence")
		}
		if err := validateCorpusRecordFilterEvidence(*lens.RecordFilter); err != nil {
			return err
		}
		if err := validateCorpusAssessmentEvidence(*lens.Assessment, lens.Totals.Shards); err != nil {
			return err
		}
		if err := validateCorpusPrivacyEvidence(*lens.PrivacyRedaction, lens.Totals.Shards); err != nil {
			return err
		}
	}
	for _, item := range []struct{ name, value string }{
		{"corpus lens.bom_sha256", lens.BOMSHA256},
		{"corpus lens.document_sha256", lens.DocumentSHA256},
		{"corpus lens.receipt_sha256", lens.ReceiptSHA256},
		{"corpus lens.source_set_sha256", lens.SourceSetSHA256},
		{"corpus lens.shard_set_sha256", lens.ShardSetSHA256},
	} {
		if err := validateSHA256(item.name, item.value); err != nil {
			return err
		}
	}
	if !oneOf(lens.Index.State, "CLEAN_PINNED", "DIRTY_MANIFEST_PINNED", "MANIFEST_PINNED_NO_COMMIT") {
		return fmt.Errorf("unsupported corpus index evidence state %q", lens.Index.State)
	}
	if lens.Index.Commit != "" && !validLowerGitHash(lens.Index.Commit) {
		return errors.New("corpus lens index commit is not a lowercase Git object hash")
	}
	if !sort.StringsAreSorted(lens.Paths) || hasAdjacentDuplicate(lens.Paths) {
		return errors.New("corpus lens paths must be sorted and unique")
	}
	for i, selected := range lens.Paths {
		if err := validateCorpusSelectionPath(fmt.Sprintf("corpus lens paths[%d]", i), selected); err != nil {
			return err
		}
	}
	if err := validateLicensePolicy(lens.LicensePolicy); err != nil {
		return err
	}
	expectedIndexState := "CLEAN_PINNED"
	if lens.Index.Commit == "" {
		expectedIndexState = "MANIFEST_PINNED_NO_COMMIT"
	} else if lens.Index.Dirty {
		expectedIndexState = "DIRTY_MANIFEST_PINNED"
	}
	if lens.Index.State != expectedIndexState {
		return fmt.Errorf("corpus index evidence state %q does not match its recorded identity", lens.Index.State)
	}
	if lens.Totals.Shards < 0 || lens.Totals.Docs < 0 || lens.Totals.Tokens < 0 || lens.Totals.Bytes < 0 {
		return errors.New("corpus lens has negative totals")
	}
	if err := validateModalities("corpus lens", lens.Modalities); err != nil {
		return err
	}
	previousManifest := ""
	for i, manifest := range lens.Manifests {
		if err := validateRelativePath(fmt.Sprintf("corpus lens manifests[%d].path", i), manifest.Path); err != nil {
			return err
		}
		if i > 0 && manifest.Path <= previousManifest {
			return errors.New("corpus lens manifests must be sorted and unique")
		}
		previousManifest = manifest.Path
		if err := validateSHA256(fmt.Sprintf("corpus lens manifest %s.sha256", manifest.Path), manifest.SHA256); err != nil {
			return err
		}
		if strings.TrimSpace(manifest.Name) == "" || len(manifest.LicenseAssertions) == 0 {
			return fmt.Errorf("corpus lens manifest %s has incomplete identity or license assertions", manifest.Path)
		}
		if err := validateSortedNonemptySet("corpus lens manifest "+manifest.Path+" license assertions", manifest.LicenseAssertions); err != nil {
			return err
		}
		previousSource := ""
		for j, source := range manifest.Sources {
			if strings.TrimSpace(source.Name) == "" || strings.TrimSpace(source.Source) == "" || strings.TrimSpace(source.URL) == "" {
				return fmt.Errorf("corpus lens manifest %s source %d has incomplete identity", manifest.Path, j)
			}
			if j > 0 && source.Name <= previousSource {
				return fmt.Errorf("corpus lens manifest %s sources must be sorted and unique", manifest.Path)
			}
			previousSource = source.Name
			if err := validateSHA256(fmt.Sprintf("corpus lens manifest %s source %s.sha256", manifest.Path, source.Name), source.SHA256); err != nil {
				return err
			}
			if source.SourceFileCount < 0 || source.SourceFileCount == 0 && source.SourceFilesSHA256 != "" || source.SourceFileCount > 0 && source.SourceFilesSHA256 == "" {
				return fmt.Errorf("corpus lens manifest %s source %s has inconsistent source-file evidence", manifest.Path, source.Name)
			}
			if source.SourceFilesSHA256 != "" {
				if err := validateSHA256(fmt.Sprintf("corpus lens manifest %s source %s.source_files_sha256", manifest.Path, source.Name), source.SourceFilesSHA256); err != nil {
					return err
				}
			}
			switch source.License.State {
			case "MISSING":
				if source.License.Normalized != "" || source.License.Declaration != "" || source.License.EvidenceURL != "" {
					return fmt.Errorf("corpus lens manifest %s source %s marks present license evidence MISSING", manifest.Path, source.Name)
				}
			case "RECORDED_ASSERTION":
				if source.License.Normalized == "" && source.License.Declaration == "" && source.License.EvidenceURL == "" {
					return fmt.Errorf("corpus lens manifest %s source %s has an empty license assertion", manifest.Path, source.Name)
				}
			default:
				return fmt.Errorf("corpus lens manifest %s source %s has unsupported license state %q", manifest.Path, source.Name, source.License.State)
			}
		}
	}
	previousLicense := ""
	for i, license := range lens.Licenses {
		if strings.TrimSpace(license.ID) == "" || license.State != "RECORDED_ASSERTION" || license.Measures.Shards < 0 || license.Measures.Docs < 0 || license.Measures.Tokens < 0 || license.Measures.Bytes < 0 {
			return fmt.Errorf("corpus lens license %d is invalid", i)
		}
		if i > 0 && license.ID <= previousLicense {
			return errors.New("corpus lens licenses must be sorted and unique")
		}
		previousLicense = license.ID
	}
	if lens.Attestation.Embedded < 0 || lens.Attestation.ImplicitV4 < 0 || lens.Attestation.DeepValidated < 0 || lens.Attestation.NotRecorded < 0 {
		return errors.New("corpus lens has negative attestation counts")
	}
	attested := lens.Attestation.Embedded + lens.Attestation.ImplicitV4 + lens.Attestation.DeepValidated
	if attested+lens.Attestation.NotRecorded != lens.Totals.Shards {
		return errors.New("corpus lens attestation counts do not match its shard total")
	}
	expectedAttestationState := "RECORDED_COMPLETE"
	switch {
	case lens.Totals.Shards == 0:
		expectedAttestationState = "NO_SHARDS"
	case lens.Attestation.NotRecorded == lens.Totals.Shards:
		expectedAttestationState = "NOT_RECORDED"
	case lens.Attestation.NotRecorded > 0:
		expectedAttestationState = "RECORDED_PARTIAL"
	}
	if lens.Attestation.State != expectedAttestationState {
		return fmt.Errorf("corpus lens attestation state %q does not match its counts", lens.Attestation.State)
	}
	if len(lens.Notices) == 0 {
		return errors.New("corpus evidence lens must state its evidence boundaries")
	}
	if !lens.Authority.closed() {
		return errors.New("corpus evidence lens must carry closed authority")
	}
	actual, err := corpusLensReceiptDigest(lens)
	if err != nil {
		return err
	}
	if actual != lens.ReceiptSHA256 {
		return fmt.Errorf("corpus evidence lens receipt digest mismatch: expected %s, got %s", lens.ReceiptSHA256, actual)
	}
	return nil
}

func corpusLensReceiptDigest(lens CorpusEvidenceLens) (string, error) {
	lens.ReceiptSHA256 = ""
	data, err := json.Marshal(lens)
	if err != nil {
		return "", fmt.Errorf("encode corpus evidence lens receipt: %w", err)
	}
	return digestBytes(data), nil
}

func validateCorpusBOM(bom waldoCorpusBOM) error {
	if bom.Kind != "openwaldo-bom" || bom.Schema != 1 || bom.Subject != "corpus" {
		return fmt.Errorf("unsupported corpus BOM identity %q schema %d subject %q", bom.Kind, bom.Schema, bom.Subject)
	}
	if bom.Index.Commit != "" && !validLowerGitHash(bom.Index.Commit) {
		return fmt.Errorf("corpus index commit %q is not a lowercase Git object hash", bom.Index.Commit)
	}
	if !sort.StringsAreSorted(bom.Paths) || hasAdjacentDuplicate(bom.Paths) {
		return errors.New("corpus BOM paths must be sorted and unique")
	}
	for i, selected := range bom.Paths {
		if err := validateCorpusSelectionPath(fmt.Sprintf("corpus paths[%d]", i), selected); err != nil {
			return err
		}
	}
	if err := validateLicensePolicy(bom.Policy); err != nil {
		return err
	}
	if err := validateWaldoRecordFilterPolicy(bom.RecordFilter, bom.Paths); err != nil {
		return err
	}

	manifests := make(map[string]waldoCorpusManifest, len(bom.Manifests))
	manifestSources := make(map[string]map[string]bool, len(bom.Manifests))
	manifestTotals := make(map[string]EvidenceMeasures, len(bom.Manifests))
	manifestLicenses := make(map[string]map[string]EvidenceMeasures, len(bom.Manifests))
	manifestModalities := make(map[string]map[string]ModalityEvidence, len(bom.Manifests))
	manifestEmailRecords := make(map[string]int64, len(bom.Manifests))
	manifestRepetitiveRecords := make(map[string]int64, len(bom.Manifests))
	manifestBoilerplateRecords := make(map[string]int64, len(bom.Manifests))
	for i, manifest := range bom.Manifests {
		if err := validateRelativePath(fmt.Sprintf("corpus manifests[%d].path", i), manifest.Path); err != nil {
			return err
		}
		if _, exists := manifests[manifest.Path]; exists {
			return fmt.Errorf("duplicate corpus manifest path %q", manifest.Path)
		}
		if err := validateSHA256(fmt.Sprintf("corpus manifest %s.sha256", manifest.Path), manifest.SHA256); err != nil {
			return err
		}
		if strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Title) == "" || strings.TrimSpace(manifest.Description) == "" || strings.TrimSpace(manifest.Format) == "" || manifest.RecordSchema <= 0 {
			return fmt.Errorf("corpus manifest %s is missing resolved identity or format fields", manifest.Path)
		}
		if (manifest.License == "") == (len(manifest.LicenseSet) == 0) {
			return fmt.Errorf("corpus manifest %s must declare exactly one license or license_set form", manifest.Path)
		}
		if err := validateSortedNonemptySet("corpus manifest "+manifest.Path+" license_set", manifest.LicenseSet); err != nil {
			return err
		}
		if err := validateConversion("corpus manifest "+manifest.Path, manifest.ConvertedBy, manifest.RecordSchema); err != nil {
			return err
		}
		if manifest.RecordSchema >= waldoTextRecordSchema {
			if err := validateWaldoContentAssessment(manifest.Assessment, manifest.Totals.Docs); err != nil {
				return fmt.Errorf("corpus manifest %s assessment: %w", manifest.Path, err)
			}
		}
		if manifest.ConvertedBy.Recipe == waldoTextWriterRecipe {
			if err := validateWaldoContentRedaction(manifest.Redaction); err != nil {
				return fmt.Errorf("corpus manifest %s redaction: %w", manifest.Path, err)
			}
		}
		sources := map[string]bool{}
		for j, source := range manifest.Sources {
			if strings.TrimSpace(source.Name) == "" || strings.TrimSpace(source.Source) == "" || strings.TrimSpace(source.URL) == "" {
				return fmt.Errorf("corpus manifest %s source %d is missing name, source, or url", manifest.Path, j)
			}
			if sources[source.Name] {
				return fmt.Errorf("corpus manifest %s has duplicate source %q", manifest.Path, source.Name)
			}
			sources[source.Name] = true
			if err := validateSHA256(fmt.Sprintf("corpus manifest %s source %s.sha256", manifest.Path, source.Name), source.SHA256); err != nil {
				return err
			}
			for k, file := range source.Files {
				if strings.TrimSpace(file.Name) == "" || strings.TrimSpace(file.URL) == "" {
					return fmt.Errorf("corpus manifest %s source %s file %d is missing name or url", manifest.Path, source.Name, k)
				}
				if err := validateSHA256(fmt.Sprintf("corpus manifest %s source %s file %d.sha256", manifest.Path, source.Name, k), file.SHA256); err != nil {
					return err
				}
				if file.Bytes < 0 {
					return fmt.Errorf("corpus manifest %s source %s file %d has negative bytes", manifest.Path, source.Name, k)
				}
			}
			if err := validateModalities("corpus manifest "+manifest.Path+" source "+source.Name+" usage", source.Usage); err != nil {
				return err
			}
		}
		if err := validateModalities("corpus manifest "+manifest.Path, manifest.Modalities); err != nil {
			return err
		}
		if err := validateMeasureMap("corpus manifest "+manifest.Path+" licenses", manifest.Licenses); err != nil {
			return err
		}
		manifests[manifest.Path] = manifest
		manifestSources[manifest.Path] = sources
		manifestLicenses[manifest.Path] = map[string]EvidenceMeasures{}
		manifestModalities[manifest.Path] = map[string]ModalityEvidence{}
	}

	subManifestKeys, err := validateCorpusSubManifests(bom.SubManifests, manifests)
	if err != nil {
		return err
	}
	calculated := EvidenceMeasures{}
	calculatedLicenses := map[string]EvidenceMeasures{}
	calculatedModalities := map[string]ModalityEvidence{}
	for i, shard := range bom.Shards {
		manifest, exists := manifests[shard.Manifest]
		if !exists {
			return fmt.Errorf("corpus shard %d refers to unknown manifest %q", i+1, shard.Manifest)
		}
		if strings.TrimSpace(shard.URL) == "" || strings.TrimSpace(shard.Format) == "" || shard.RecordSchema <= 0 {
			return fmt.Errorf("corpus shard %d has incomplete resolved identity", i+1)
		}
		if err := validateSHA256(fmt.Sprintf("corpus shard %d.sha256", i+1), shard.SHA256); err != nil {
			return err
		}
		if (shard.License == "") == (len(shard.Licenses) == 0) {
			return fmt.Errorf("corpus shard %d must declare exactly one license or licenses form", i+1)
		}
		if err := validateSortedNonemptySet(fmt.Sprintf("corpus shard %d licenses", i+1), shard.Licenses); err != nil {
			return err
		}
		if err := validateConversion(fmt.Sprintf("corpus shard %d", i+1), shard.ConvertedBy, shard.RecordSchema); err != nil {
			return err
		}
		if shard.RecordsRoot != "" {
			if err := validateSHA256(fmt.Sprintf("corpus shard %d.records_root", i+1), shard.RecordsRoot); err != nil {
				return err
			}
		}
		if shard.SubManifestSHA256 != "" && !subManifestKeys[shard.Manifest+"\x00"+shard.SubManifestSHA256] {
			return fmt.Errorf("corpus shard %d refers to an unpinned submanifest", i+1)
		}
		if shard.Docs <= 0 || shard.Tokens < 0 || shard.Bytes <= 0 {
			return fmt.Errorf("corpus shard %d has non-positive totals", i+1)
		}
		if err := validateModalities(fmt.Sprintf("corpus shard %d", i+1), shard.Modalities); err != nil {
			return err
		}
		if len(shard.Modalities) > 0 && modalityTokens(shard.Modalities) != shard.Tokens {
			return fmt.Errorf("corpus shard %d modality tokens do not match its token total", i+1)
		}
		if shard.RecordSchema >= waldoTextRecordSchema {
			if err := validateWaldoContentAssessment(shard.Assessment, shard.Docs); err != nil {
				return fmt.Errorf("corpus shard %d assessment: %w", i+1, err)
			}
			manifestEmailRecords[shard.Manifest] += shard.Assessment.EmailAddresses.Records
			manifestRepetitiveRecords[shard.Manifest] += shard.Assessment.RepetitiveContent.Records
			manifestBoilerplateRecords[shard.Manifest] += shard.Assessment.BoilerplateContent.Records
		}
		if shard.ConvertedBy.Recipe == waldoTextWriterRecipe {
			if err := validateWaldoContentRedaction(shard.Redaction); err != nil {
				return fmt.Errorf("corpus shard %d redaction: %w", i+1, err)
			}
		}
		seenSources := map[string]bool{}
		for _, source := range shard.Sources {
			if !manifestSources[shard.Manifest][source] || seenSources[source] {
				return fmt.Errorf("corpus shard %d has unknown or duplicate source %q", i+1, source)
			}
			seenSources[source] = true
		}
		licenses := effectiveShardLicenses(shard)
		for _, license := range licenses {
			if !licensePolicyAllows(bom.Policy, license) {
				return fmt.Errorf("corpus shard %d license %q violates the BOM policy", i+1, license)
			}
		}
		if err := validateLicenseUsage(fmt.Sprintf("corpus shard %d", i+1), shard, licenses); err != nil {
			return err
		}
		if err := validateCorpusAttestation(i+1, shard, licenses); err != nil {
			return err
		}

		measure := EvidenceMeasures{Shards: 1, Docs: shard.Docs, Tokens: shard.Tokens, Bytes: shard.Bytes}
		addEvidenceMeasures(&calculated, measure)
		addCorpusLicenseMeasures(calculatedLicenses, shard, licenses, measure)
		addModalityMeasures(calculatedModalities, shard.Modalities)
		manifestMeasure := manifestTotals[manifest.Path]
		addEvidenceMeasures(&manifestMeasure, measure)
		manifestTotals[manifest.Path] = manifestMeasure
		addCorpusLicenseMeasures(manifestLicenses[manifest.Path], shard, licenses, measure)
		addModalityMeasures(manifestModalities[manifest.Path], shard.Modalities)
	}
	if calculated != bom.Totals || !maps.Equal(calculatedLicenses, bom.Licenses) || !maps.Equal(calculatedModalities, bom.Modalities) {
		return errors.New("corpus BOM totals, modality totals, or license totals do not match its shards")
	}
	for path, manifest := range manifests {
		if manifest.Totals != manifestTotals[path] || !maps.Equal(manifest.Licenses, manifestLicenses[path]) || !maps.Equal(manifest.Modalities, manifestModalities[path]) {
			return fmt.Errorf("corpus manifest %s totals do not match its selected shards", path)
		}
		if manifest.RecordSchema >= waldoTextRecordSchema && (manifest.Assessment.EmailAddresses.Records != manifestEmailRecords[path] || manifest.Assessment.RepetitiveContent.Records != manifestRepetitiveRecords[path] || manifest.Assessment.BoilerplateContent.Records != manifestBoilerplateRecords[path]) {
			return fmt.Errorf("corpus manifest %s assessment does not match its selected shards", path)
		}
	}
	return nil
}

func validateCorpusSubManifests(pins []waldoCorpusSubManifest, manifests map[string]waldoCorpusManifest) (map[string]bool, error) {
	seen := map[string]bool{}
	parents := map[string]string{}
	roots := map[string]int{}
	for i, pin := range pins {
		if _, exists := manifests[pin.Manifest]; !exists {
			return nil, fmt.Errorf("corpus submanifest %d refers to unknown manifest %q", i+1, pin.Manifest)
		}
		key := pin.Manifest + "\x00" + pin.SHA256
		if seen[key] || strings.TrimSpace(pin.URL) == "" {
			return nil, fmt.Errorf("invalid or duplicate corpus submanifest %d", i+1)
		}
		if err := validateSHA256(fmt.Sprintf("corpus submanifest %d.sha256", i+1), pin.SHA256); err != nil {
			return nil, err
		}
		if pin.ParentSHA256 != "" {
			if err := validateSHA256(fmt.Sprintf("corpus submanifest %d.parent_sha256", i+1), pin.ParentSHA256); err != nil {
				return nil, err
			}
		}
		if pin.Count <= 0 || pin.Docs <= 0 || pin.Tokens < 0 || pin.Bytes <= 0 || pin.EncodedBytes <= 0 {
			return nil, fmt.Errorf("corpus submanifest %d has non-positive totals", i+1)
		}
		if err := validateModalities(fmt.Sprintf("corpus submanifest %d", i+1), pin.Modalities); err != nil {
			return nil, err
		}
		if len(pin.Modalities) > 0 && modalityTokens(pin.Modalities) != pin.Tokens {
			return nil, fmt.Errorf("corpus submanifest %d modality tokens do not match its token total", i+1)
		}
		seen[key] = true
		if pin.ParentSHA256 == "" {
			roots[pin.Manifest]++
		} else {
			parents[key] = pin.Manifest + "\x00" + pin.ParentSHA256
		}
	}
	for key, parent := range parents {
		if !seen[parent] {
			return nil, fmt.Errorf("corpus submanifest %s refers to an unpinned parent", strings.Split(key, "\x00")[1])
		}
		chain := map[string]bool{}
		for current := key; current != ""; current = parents[current] {
			if chain[current] {
				return nil, errors.New("corpus submanifest parent cycle")
			}
			chain[current] = true
		}
	}
	for manifest, count := range roots {
		if count != 1 {
			return nil, fmt.Errorf("corpus manifest %s has %d submanifest roots, want one", manifest, count)
		}
	}
	return seen, nil
}

func validateCorpusAttestation(position int, shard waldoCorpusShard, licenses []string) error {
	attestation := shard.Attestation
	if attestation == nil {
		return nil
	}
	switch attestation.Status {
	case "embedded":
		if attestation.BOM == nil {
			return fmt.Errorf("corpus shard %d has incomplete embedded BOM evidence", position)
		}
		if err := validateSHA256(fmt.Sprintf("corpus shard %d attestation.bom_sha256", position), attestation.BOMSHA256); err != nil {
			return err
		}
		bom := attestation.BOM
		if bom.Kind != "openwaldo-bom" || bom.Schema != 1 || bom.Subject != "shard" || !supportedWaldoShardWriter(bom.RecordSchema, bom.WriterRecipe) || strings.TrimSpace(bom.Tokenizer) == "" || bom.RecordSchema != shard.RecordSchema || bom.Records != shard.Docs || bom.Tokens != shard.Tokens || !slices.Equal(bom.Licenses, licenses) {
			return fmt.Errorf("corpus shard %d embedded BOM differs from its corpus pin", position)
		}
		assessment := waldoContentAssessment{}
		if shard.Assessment != nil {
			assessment = *shard.Assessment
		}
		emailRecords, repetitiveRecords, boilerplateRecords := assessmentCounts(assessment)
		redaction := waldoContentRedaction{}
		if shard.Redaction != nil {
			redaction = *shard.Redaction
		}
		if bom.EmailAddressRecords != emailRecords || bom.RepetitiveContentRecords != repetitiveRecords || bom.BoilerplateContentRecords != boilerplateRecords || bom.Redaction != redaction {
			return fmt.Errorf("corpus shard %d embedded BOM row evidence differs from its corpus pin", position)
		}
		if bom.WriterRecipe == waldoTextWriterRecipe {
			if err := validateWaldoContentRedaction(&bom.Redaction); err != nil {
				return fmt.Errorf("corpus shard %d embedded BOM redaction: %w", position, err)
			}
		}
		if err := validateSHA256(fmt.Sprintf("corpus shard %d embedded plan_sha256", position), bom.PlanSHA256); err != nil {
			return err
		}
		if bom.ContentBytes <= 0 || !bom.Validation.CanonicalRecords || !bom.Validation.ContentHashes || !bom.Validation.TokenCounts || !bom.Validation.ExactLicenseDedup || attestation.WriterRecipe != bom.WriterRecipe {
			return fmt.Errorf("corpus shard %d embedded BOM has incomplete validation evidence", position)
		}
		encoded, err := json.Marshal(bom)
		if err != nil {
			return err
		}
		if digestBytes(encoded) != attestation.BOMSHA256 {
			return fmt.Errorf("corpus shard %d embedded BOM digest differs", position)
		}
	case "implicit-v4":
		if attestation.WriterRecipe != waldoFormerTextRecipe || attestation.BOM != nil || attestation.BOMSHA256 != "" {
			return fmt.Errorf("corpus shard %d has invalid implicit-v4 evidence", position)
		}
	case "deep-validated":
		if attestation.BOM != nil || attestation.BOMSHA256 != "" {
			return fmt.Errorf("corpus shard %d has invalid deep-validation evidence", position)
		}
	default:
		return fmt.Errorf("corpus shard %d has unsupported attestation status %q", position, attestation.Status)
	}
	return nil
}

func validateLicenseUsage(name string, shard waldoCorpusShard, licenses []string) error {
	if len(licenses) > 1 {
		if len(shard.LicenseUsage) != len(licenses) {
			return fmt.Errorf("%s license usage must cover every represented license", name)
		}
		var docs, tokens int64
		for _, license := range licenses {
			usage, exists := shard.LicenseUsage[license]
			if !exists || usage.Docs <= 0 || usage.Tokens < 0 || usage.Shards != 0 || usage.Bytes != 0 {
				return fmt.Errorf("%s has invalid usage for license %q", name, license)
			}
			docs += usage.Docs
			tokens += usage.Tokens
		}
		if docs != shard.Docs || tokens != shard.Tokens {
			return fmt.Errorf("%s license usage does not match shard totals", name)
		}
		return nil
	}
	if len(shard.LicenseUsage) == 0 {
		return nil
	}
	usage, exists := shard.LicenseUsage[licenses[0]]
	if len(shard.LicenseUsage) != 1 || !exists || usage.Docs != shard.Docs || usage.Tokens != shard.Tokens || usage.Shards != 0 || usage.Bytes != 0 {
		return fmt.Errorf("%s has invalid usage for license %q", name, licenses[0])
	}
	return nil
}

func projectCorpusManifests(input []waldoCorpusManifest) ([]CorpusManifestEvidence, string, error) {
	output := make([]CorpusManifestEvidence, 0, len(input))
	for _, manifest := range input {
		licenses := append([]string(nil), manifest.LicenseSet...)
		if manifest.License != "" {
			licenses = []string{manifest.License}
		}
		sources := make([]CorpusSourceEvidence, 0, len(manifest.Sources))
		for _, source := range manifest.Sources {
			license := SourceLicenseAssertion{Normalized: source.License, State: "MISSING"}
			if source.LicenseEvidence != nil {
				license.Declaration = source.LicenseEvidence.Declaration
				license.EvidenceURL = source.LicenseEvidence.URL
			}
			if license.Normalized != "" || license.Declaration != "" || license.EvidenceURL != "" {
				license.State = "RECORDED_ASSERTION"
			}
			sourceEvidence := CorpusSourceEvidence{
				Name: source.Name, Source: source.Source, Version: source.Version, URL: source.URL,
				SHA256: source.SHA256, License: license, SourceFileCount: len(source.Files),
			}
			if len(source.Files) > 0 {
				encoded, err := json.Marshal(source.Files)
				if err != nil {
					return nil, "", fmt.Errorf("encode source files for %s/%s: %w", manifest.Path, source.Name, err)
				}
				sourceEvidence.SourceFilesSHA256 = digestBytes(encoded)
			}
			sources = append(sources, sourceEvidence)
		}
		sort.Slice(sources, func(i, j int) bool { return sources[i].Name < sources[j].Name })
		output = append(output, CorpusManifestEvidence{Path: manifest.Path, SHA256: manifest.SHA256, Name: manifest.Name, LicenseAssertions: licenses, Sources: sources})
	}
	sort.Slice(output, func(i, j int) bool { return output[i].Path < output[j].Path })
	encoded, err := json.Marshal(output)
	if err != nil {
		return nil, "", fmt.Errorf("encode corpus source evidence: %w", err)
	}
	return output, digestBytes(encoded), nil
}

func projectCorpusShards(shards []waldoCorpusShard) (string, CorpusAttestationEvidence, error) {
	identities := make([]corpusShardIdentity, 0, len(shards))
	attestation := CorpusAttestationEvidence{}
	for i, shard := range shards {
		identity := corpusShardIdentity{
			Ordinal: i + 1, Manifest: shard.Manifest, SubManifestSHA256: shard.SubManifestSHA256,
			URL: shard.URL, SHA256: shard.SHA256, Format: shard.Format, RecordSchema: shard.RecordSchema,
			Licenses: append([]string(nil), effectiveShardLicenses(shard)...), Sources: append([]string(nil), shard.Sources...),
			RecordsRoot: shard.RecordsRoot, Docs: shard.Docs, Tokens: shard.Tokens, Bytes: shard.Bytes,
		}
		if shard.Attestation == nil {
			attestation.NotRecorded++
		} else {
			identity.AttestationStatus = shard.Attestation.Status
			identity.AttestationSHA256 = shard.Attestation.BOMSHA256
			switch shard.Attestation.Status {
			case "embedded":
				attestation.Embedded++
			case "implicit-v4":
				attestation.ImplicitV4++
			case "deep-validated":
				attestation.DeepValidated++
			}
		}
		identities = append(identities, identity)
	}
	switch {
	case len(shards) == 0:
		attestation.State = "NO_SHARDS"
	case attestation.NotRecorded == int64(len(shards)):
		attestation.State = "NOT_RECORDED"
	case attestation.NotRecorded > 0:
		attestation.State = "RECORDED_PARTIAL"
	default:
		attestation.State = "RECORDED_COMPLETE"
	}
	encoded, err := json.Marshal(identities)
	if err != nil {
		return "", CorpusAttestationEvidence{}, fmt.Errorf("encode corpus shard identities: %w", err)
	}
	return digestBytes(encoded), attestation, nil
}

func projectCorpusRecordFilter(policy *waldoRecordFilterPolicy) (CorpusRecordFilterEvidence, error) {
	encoded, err := json.Marshal(policy)
	if err != nil {
		return CorpusRecordFilterEvidence{}, fmt.Errorf("encode corpus record filter: %w", err)
	}
	evidence := CorpusRecordFilterEvidence{State: "NOT_DECLARED", PolicySHA256: digestBytes(encoded)}
	if policy != nil {
		evidence.State = "DECLARED"
		evidence.Schema = policy.Schema
		evidence.GlobalDeclared = policy.Global != nil
		evidence.CorpusFilterCount = len(policy.Corpora)
	}
	return evidence, nil
}

func projectCorpusRowEvidence(shards []waldoCorpusShard) (CorpusAssessmentEvidence, CorpusPrivacyRedactionEvidence, error) {
	assessment := CorpusAssessmentEvidence{}
	privacy := CorpusPrivacyRedactionEvidence{}
	detectors := map[string]bool{}
	for i, shard := range shards {
		if shard.RecordSchema >= waldoTextRecordSchema {
			if err := validateWaldoContentAssessment(shard.Assessment, shard.Docs); err != nil {
				return CorpusAssessmentEvidence{}, CorpusPrivacyRedactionEvidence{}, fmt.Errorf("project corpus shard %d assessment: %w", i+1, err)
			}
			assessment.AssessedShards++
			detectors["email_addresses\x00"+shard.Assessment.EmailAddresses.Detector] = true
			detectors["repetitive_content\x00"+shard.Assessment.RepetitiveContent.Detector] = true
			detectors["boilerplate_content\x00"+shard.Assessment.BoilerplateContent.Detector] = true
			assessment.EmailAddressRecords += shard.Assessment.EmailAddresses.Records
			assessment.RepetitiveContentRecords += shard.Assessment.RepetitiveContent.Records
			assessment.BoilerplateContentRecords += shard.Assessment.BoilerplateContent.Records
		} else {
			assessment.LegacyShards++
		}

		if shard.ConvertedBy.Recipe == waldoTextWriterRecipe {
			if err := validateWaldoContentRedaction(shard.Redaction); err != nil {
				return CorpusAssessmentEvidence{}, CorpusPrivacyRedactionEvidence{}, fmt.Errorf("project corpus shard %d redaction: %w", i+1, err)
			}
			privacy.RedactedShards++
			privacy.Policy = shard.Redaction.Policy
			privacy.NamesRetained = privacy.NamesRetained || shard.Redaction.NamesRetained
			privacy.EmailAddresses += shard.Redaction.EmailAddresses
			privacy.IPAddresses += shard.Redaction.IPAddresses
			privacy.PhoneNumbers += shard.Redaction.PhoneNumbers
			privacy.MailRoutingHeaders += shard.Redaction.MailRoutingHeaders
			privacy.Credentials += shard.Redaction.Credentials
		} else {
			privacy.UnredactedCompatibleShards++
		}
	}
	detectorSet := make([]string, 0, len(detectors))
	for detector := range detectors {
		detectorSet = append(detectorSet, detector)
	}
	sort.Strings(detectorSet)
	encodedDetectors, err := json.Marshal(detectorSet)
	if err != nil {
		return CorpusAssessmentEvidence{}, CorpusPrivacyRedactionEvidence{}, fmt.Errorf("encode corpus detector set: %w", err)
	}
	assessment.DetectorSetSHA256 = digestBytes(encodedDetectors)
	switch {
	case len(shards) == 0:
		assessment.State = "NO_SHARDS"
		privacy.State = "NO_SHARDS"
	case assessment.AssessedShards == 0:
		assessment.State = "NOT_APPLICABLE_LEGACY"
	case assessment.LegacyShards == 0:
		assessment.State = "RECORDED_COMPLETE"
	default:
		assessment.State = "RECORDED_WITH_LEGACY"
	}
	if len(shards) > 0 {
		switch {
		case privacy.RedactedShards == 0:
			privacy.State = "NOT_RECORDED"
		case privacy.UnredactedCompatibleShards == 0:
			privacy.State = "RECORDED_COMPLETE"
		default:
			privacy.State = "RECORDED_PARTIAL"
		}
	}
	return assessment, privacy, nil
}

func validateCorpusRecordFilterEvidence(evidence CorpusRecordFilterEvidence) error {
	if err := validateSHA256("corpus record filter.policy_sha256", evidence.PolicySHA256); err != nil {
		return err
	}
	if evidence.CorpusFilterCount < 0 {
		return errors.New("corpus record filter has a negative corpus-filter count")
	}
	switch evidence.State {
	case "NOT_DECLARED":
		if evidence.Schema != 0 || evidence.GlobalDeclared || evidence.CorpusFilterCount != 0 {
			return errors.New("undeclared corpus record filter invents policy fields")
		}
	case "DECLARED":
		if evidence.Schema != 1 || !evidence.GlobalDeclared && evidence.CorpusFilterCount == 0 {
			return errors.New("declared corpus record filter has an incomplete schema or policy")
		}
	default:
		return fmt.Errorf("unsupported corpus record-filter evidence state %q", evidence.State)
	}
	return nil
}

func validateCorpusAssessmentEvidence(evidence CorpusAssessmentEvidence, shards int64) error {
	if err := validateSHA256("corpus assessment.detector_set_sha256", evidence.DetectorSetSHA256); err != nil {
		return err
	}
	if evidence.AssessedShards < 0 || evidence.LegacyShards < 0 || evidence.AssessedShards+evidence.LegacyShards != shards || evidence.EmailAddressRecords < 0 || evidence.RepetitiveContentRecords < 0 || evidence.BoilerplateContentRecords < 0 {
		return errors.New("corpus assessment evidence has inconsistent shard or record counts")
	}
	expected := "RECORDED_WITH_LEGACY"
	switch {
	case shards == 0:
		expected = "NO_SHARDS"
	case evidence.AssessedShards == 0:
		expected = "NOT_APPLICABLE_LEGACY"
	case evidence.LegacyShards == 0:
		expected = "RECORDED_COMPLETE"
	}
	if evidence.State != expected {
		return fmt.Errorf("corpus assessment state %q does not match its shard counts", evidence.State)
	}
	return nil
}

func validateCorpusPrivacyEvidence(evidence CorpusPrivacyRedactionEvidence, shards int64) error {
	if evidence.RedactedShards < 0 || evidence.UnredactedCompatibleShards < 0 || evidence.RedactedShards+evidence.UnredactedCompatibleShards != shards || evidence.EmailAddresses < 0 || evidence.IPAddresses < 0 || evidence.PhoneNumbers < 0 || evidence.MailRoutingHeaders < 0 || evidence.Credentials < 0 {
		return errors.New("corpus privacy-redaction evidence has inconsistent shard or transformation counts")
	}
	expected := "RECORDED_PARTIAL"
	switch {
	case shards == 0:
		expected = "NO_SHARDS"
	case evidence.RedactedShards == 0:
		expected = "NOT_RECORDED"
	case evidence.UnredactedCompatibleShards == 0:
		expected = "RECORDED_COMPLETE"
	}
	if evidence.State != expected {
		return fmt.Errorf("corpus privacy-redaction state %q does not match its shard counts", evidence.State)
	}
	if evidence.RedactedShards == 0 {
		if evidence.Policy != "" || evidence.NamesRetained || evidence.EmailAddresses != 0 || evidence.IPAddresses != 0 || evidence.PhoneNumbers != 0 || evidence.MailRoutingHeaders != 0 || evidence.Credentials != 0 {
			return errors.New("unrecorded corpus privacy evidence invents a policy or transformation counts")
		}
	} else if evidence.Policy != waldoPrivacyRedactionPolicy || !evidence.NamesRetained {
		return errors.New("recorded corpus privacy evidence has an unsupported policy or names-retained state")
	}
	return nil
}

func projectCorpusLicenses(input map[string]EvidenceMeasures) []CorpusLicenseEvidence {
	licenses := make([]CorpusLicenseEvidence, 0, len(input))
	for id, measure := range input {
		licenses = append(licenses, CorpusLicenseEvidence{ID: id, State: "RECORDED_ASSERTION", Measures: measure})
	}
	sort.Slice(licenses, func(i, j int) bool { return licenses[i].ID < licenses[j].ID })
	return licenses
}

func validateWaldoRecordFilterPolicy(policy *waldoRecordFilterPolicy, paths []string) error {
	if policy == nil {
		return nil
	}
	if policy.Schema != 1 {
		return fmt.Errorf("unsupported corpus record-filter schema %d", policy.Schema)
	}
	if policy.Global == nil && len(policy.Corpora) == 0 {
		return errors.New("corpus record-filter policy must declare a global or corpus filter")
	}
	if policy.Global != nil {
		if err := validateWaldoRecordFilter(*policy.Global); err != nil {
			return fmt.Errorf("global corpus record filter: %w", err)
		}
	}
	selected := make(map[string]bool, len(paths))
	for _, path := range paths {
		selected[path] = true
	}
	for path, filter := range policy.Corpora {
		if !selected[path] {
			return fmt.Errorf("corpus record filter declares unselected corpus %q", path)
		}
		if err := validateWaldoRecordFilter(filter); err != nil {
			return fmt.Errorf("corpus record filter %s: %w", path, err)
		}
	}
	return nil
}

func validateWaldoRecordFilter(filter waldoRecordFilter) error {
	if filter.MainContent == nil && filter.Exclude == nil && filter.Licenses == nil && filter.Languages == nil && filter.Sources == nil && filter.Date == nil {
		return errors.New("record filter must declare at least one condition")
	}
	if filter.Exclude != nil {
		if filter.Exclude.RepetitiveContent == nil && filter.Exclude.BoilerplateContent == nil && len(filter.Exclude.Licenses) == 0 {
			return errors.New("record-filter exclude requires at least one condition")
		}
		if err := validateWaldoPatterns(filter.Exclude.Licenses); err != nil {
			return fmt.Errorf("record-filter exclude licenses: %w", err)
		}
		if len(filter.Exclude.Licenses) > 0 && filter.Licenses != nil {
			return errors.New("record-filter exclude licenses cannot be combined with legacy license filtering")
		}
	}
	for _, item := range []struct {
		name   string
		filter *waldoValueFilter
	}{
		{name: "licenses", filter: filter.Licenses},
		{name: "languages", filter: filter.Languages},
		{name: "sources", filter: filter.Sources},
	} {
		if item.filter == nil {
			continue
		}
		if len(item.filter.Include) == 0 && len(item.filter.Exclude) == 0 {
			return fmt.Errorf("record-filter %s requires include or exclude", item.name)
		}
		patterns := append(append([]string(nil), item.filter.Include...), item.filter.Exclude...)
		if err := validateWaldoPatterns(patterns); err != nil {
			return fmt.Errorf("record-filter %s: %w", item.name, err)
		}
	}
	if filter.Date != nil {
		if err := validateWaldoDateFilter(*filter.Date); err != nil {
			return err
		}
	}
	return nil
}

func validateWaldoPatterns(patterns []string) error {
	seen := map[string]bool{}
	for _, pattern := range patterns {
		if pattern == "" || seen[pattern] {
			return errors.New("record-filter patterns must be non-empty and unique")
		}
		seen[pattern] = true
		if _, err := pathpkg.Match(pattern, "probe"); err != nil {
			return fmt.Errorf("invalid record-filter pattern %q: %w", pattern, err)
		}
	}
	return nil
}

func validateWaldoDateFilter(filter waldoDateFilter) error {
	if filter.From == "" && filter.To == "" {
		return errors.New("record-filter date requires from or to")
	}
	var from, to time.Time
	if filter.From != "" {
		start, _, err := waldoDateInterval(filter.From)
		if err != nil {
			return fmt.Errorf("record-filter date.from: %w", err)
		}
		from = start
	}
	if filter.To != "" {
		_, end, err := waldoDateInterval(filter.To)
		if err != nil {
			return fmt.Errorf("record-filter date.to: %w", err)
		}
		to = end
	}
	if !from.IsZero() && !to.IsZero() && from.After(to) {
		return errors.New("record-filter date.from must not be after date.to")
	}
	return nil
}

func waldoDateInterval(value string) (time.Time, time.Time, error) {
	var start time.Time
	var err error
	switch len(value) {
	case len("2006"):
		start, err = time.Parse("2006", value)
		if err == nil {
			return start, start.AddDate(1, 0, 0).Add(-time.Nanosecond), nil
		}
	case len("2006-01"):
		start, err = time.Parse("2006-01", value)
		if err == nil {
			return start, start.AddDate(0, 1, 0).Add(-time.Nanosecond), nil
		}
	case len("2006-01-02"):
		start, err = time.Parse("2006-01-02", value)
		if err == nil {
			return start, start.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
		}
	default:
		start, err = time.Parse(time.RFC3339Nano, value)
		if err == nil {
			return start, start, nil
		}
	}
	return time.Time{}, time.Time{}, fmt.Errorf("%q must be YYYY, YYYY-MM, YYYY-MM-DD, or RFC 3339", value)
}

func validateWaldoContentAssessment(assessment *waldoContentAssessment, documents int64) error {
	if assessment == nil {
		return errors.New("content assessment is required")
	}
	for _, item := range []struct {
		name    string
		measure *waldoDetectionMeasure
	}{
		{name: "email_addresses", measure: assessment.EmailAddresses},
		{name: "repetitive_content", measure: assessment.RepetitiveContent},
		{name: "boilerplate_content", measure: assessment.BoilerplateContent},
	} {
		if item.measure == nil || strings.TrimSpace(item.measure.Detector) == "" {
			return fmt.Errorf("%s detector is required", item.name)
		}
		if item.measure.Records < 0 || item.measure.Records > documents {
			return fmt.Errorf("%s record count is invalid", item.name)
		}
	}
	return nil
}

func validateWaldoContentRedaction(redaction *waldoContentRedaction) error {
	if redaction == nil || redaction.Policy != waldoPrivacyRedactionPolicy || !redaction.NamesRetained {
		return errors.New("privacy policy and names_retained are required")
	}
	if redaction.EmailAddresses < 0 || redaction.IPAddresses < 0 || redaction.PhoneNumbers < 0 || redaction.MailRoutingHeaders < 0 || redaction.Credentials < 0 {
		return errors.New("privacy-redaction counts must be non-negative")
	}
	return nil
}

func assessmentCounts(assessment waldoContentAssessment) (int64, int64, int64) {
	var email, repetitive, boilerplate int64
	if assessment.EmailAddresses != nil {
		email = assessment.EmailAddresses.Records
	}
	if assessment.RepetitiveContent != nil {
		repetitive = assessment.RepetitiveContent.Records
	}
	if assessment.BoilerplateContent != nil {
		boilerplate = assessment.BoilerplateContent.Records
	}
	return email, repetitive, boilerplate
}

func supportedWaldoShardWriter(recordSchema int, writerRecipe string) bool {
	if recordSchema == waldoTextRecordSchema {
		return oneOf(writerRecipe, waldoTextWriterRecipe, waldoFormerMainContentRecipe, waldoFormerAssessmentRecipe)
	}
	return recordSchema == waldoFormerTextRecordSchema && writerRecipe == waldoFormerTextBOMRecipe
}

func validateLicensePolicy(policy CorpusLicensePolicy) error {
	for label, values := range map[string][]string{"include": policy.Include, "exclude": policy.Exclude} {
		seen := map[string]bool{}
		for _, pattern := range values {
			if pattern == "" || seen[pattern] {
				return fmt.Errorf("corpus license policy %s patterns must be non-empty and unique", label)
			}
			seen[pattern] = true
			if _, err := pathpkg.Match(pattern, "probe"); err != nil {
				return fmt.Errorf("invalid corpus license pattern %q: %w", pattern, err)
			}
		}
	}
	return nil
}

func licensePolicyAllows(policy CorpusLicensePolicy, license string) bool {
	for _, pattern := range policy.Exclude {
		if matched, _ := pathpkg.Match(pattern, license); matched {
			return false
		}
	}
	if len(policy.Include) == 0 {
		return true
	}
	for _, pattern := range policy.Include {
		if matched, _ := pathpkg.Match(pattern, license); matched {
			return true
		}
	}
	return false
}

func validateConversion(name string, conversion waldoConversion, recordSchema int) error {
	if strings.TrimSpace(conversion.Tool) == "" || strings.TrimSpace(conversion.Version) == "" || strings.TrimSpace(conversion.Profile) == "" || strings.TrimSpace(conversion.Recipe) == "" || recordSchema != 1 && strings.TrimSpace(conversion.Tokenizer) == "" {
		return fmt.Errorf("%s has incomplete conversion provenance", name)
	}
	return nil
}

func validateSortedNonemptySet(name string, values []string) error {
	if len(values) == 0 {
		return nil
	}
	if !sort.StringsAreSorted(values) || hasAdjacentDuplicate(values) {
		return fmt.Errorf("%s must be sorted and unique", name)
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s contains an empty value", name)
		}
	}
	return nil
}

func validateMeasureMap(name string, values map[string]EvidenceMeasures) error {
	for id, measure := range values {
		if strings.TrimSpace(id) == "" || measure.Shards < 0 || measure.Docs < 0 || measure.Tokens < 0 || measure.Bytes < 0 {
			return fmt.Errorf("%s contains an invalid measure for %q", name, id)
		}
	}
	return nil
}

func validateModalities(name string, modalities map[string]ModalityEvidence) error {
	for modality, measure := range modalities {
		if strings.TrimSpace(modality) == "" || measure.Samples < 0 || measure.Items < 0 || measure.Tokens < 0 || measure.DurationMS < 0 || measure.ContentBytes < 0 || measure == (ModalityEvidence{}) {
			return fmt.Errorf("%s contains invalid modality %q", name, modality)
		}
	}
	return nil
}

func validateCorpusSelectionPath(name, value string) error {
	if value == "" { // WALDO uses the empty string for an index-root selection.
		return nil
	}
	return validateRelativePath(name, value)
}

func validLowerGitHash(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				return false
			}
		}
	}
	return true
}

func hasAdjacentDuplicate(values []string) bool {
	for i := 1; i < len(values); i++ {
		if values[i] == values[i-1] {
			return true
		}
	}
	return false
}

func effectiveShardLicenses(shard waldoCorpusShard) []string {
	if len(shard.Licenses) > 0 {
		return shard.Licenses
	}
	return []string{shard.License}
}

func addEvidenceMeasures(target *EvidenceMeasures, value EvidenceMeasures) {
	target.Shards += value.Shards
	target.Docs += value.Docs
	target.Tokens += value.Tokens
	target.Bytes += value.Bytes
}

func addCorpusLicenseMeasures(target map[string]EvidenceMeasures, shard waldoCorpusShard, licenses []string, fallback EvidenceMeasures) {
	if len(licenses) == 1 || len(shard.LicenseUsage) == 0 {
		value := target[licenses[0]]
		addEvidenceMeasures(&value, fallback)
		target[licenses[0]] = value
		return
	}
	for _, license := range licenses {
		usage := shard.LicenseUsage[license]
		usage.Shards = 1
		value := target[license]
		addEvidenceMeasures(&value, usage)
		target[license] = value
	}
}

func addModalityMeasures(target, values map[string]ModalityEvidence) {
	for name, value := range values {
		current := target[name]
		current.Samples += value.Samples
		current.Items += value.Items
		current.Tokens += value.Tokens
		current.DurationMS += value.DurationMS
		current.ContentBytes += value.ContentBytes
		target[name] = current
	}
}

func modalityTokens(modalities map[string]ModalityEvidence) int64 {
	var tokens int64
	for _, value := range modalities {
		tokens += value.Tokens
	}
	return tokens
}

func cloneLicensePolicy(input CorpusLicensePolicy) CorpusLicensePolicy {
	return CorpusLicensePolicy{Include: append([]string(nil), input.Include...), Exclude: append([]string(nil), input.Exclude...)}
}

func cloneModalitiesEvidence(input map[string]ModalityEvidence) map[string]ModalityEvidence {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]ModalityEvidence, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
