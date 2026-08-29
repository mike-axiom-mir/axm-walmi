package axmmirror

import "testing"

func TestIdentityToolMemoryGrowsPerToolAndRecallsBoundedWisdom(t *testing.T) {
	first := toolMemoryExperience(t, "experience-0001", "2026-08-16T07:00:00Z", "SUCCESS")
	shard, err := StartIdentityToolMemory(first)
	if err != nil {
		t.Fatal(err)
	}
	second := toolMemoryExperience(t, "experience-0002", "2026-08-16T07:05:00Z", "FAILURE")
	shard, err = GrowIdentityToolMemory(shard, second)
	if err != nil {
		t.Fatal(err)
	}
	query := IdentityWisdomQuery{
		Schema: IdentityWisdomQuerySchema, QueryID: "query-0001",
		TargetAnsweringIdentitySHA256: toolMemoryRepeat("1"), ToolID: "image-render-hand", ToolVersion: "v1",
		TaskClass: "asset-render", UseTags: []string{"pixel-art", "transparent-background"},
		AsOf: "2026-08-16T07:06:00Z", MaxEntries: 1, Authority: Authority{},
	}
	view, err := RecallIdentityToolWisdom(shard, query)
	if err != nil {
		t.Fatal(err)
	}
	if view.State != ToolWisdomReady || len(view.Selected) != 1 || view.Selected[0].ExperienceID != "experience-0002" || view.OmittedRelevantCount != 1 {
		t.Fatalf("wisdom view = %+v", view)
	}
	if view.Selected[0].WisdomClass != ToolWisdomAvoid {
		t.Fatalf("selected wisdom = %+v", view.Selected[0])
	}
}

func TestIdentityToolMemoryRefusesCrossIdentityOrToolGrowth(t *testing.T) {
	shard, err := StartIdentityToolMemory(toolMemoryExperience(t, "experience-0001", "2026-08-16T07:00:00Z", "SUCCESS"))
	if err != nil {
		t.Fatal(err)
	}
	crossTool := toolMemoryExperience(t, "experience-0002", "2026-08-16T07:01:00Z", "SUCCESS")
	crossTool.ToolID = "different-tool"
	crossTool.ExperienceSHA256 = ""
	crossTool, err = SealIdentityToolExperience(crossTool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GrowIdentityToolMemory(shard, crossTool); err == nil {
		t.Fatal("cross-tool experience entered the shard")
	}
}

func TestIdentityToolMemoryRefusesRawOrUnverifiedContent(t *testing.T) {
	experience := toolMemoryExperience(t, "experience-0001", "2026-08-16T07:00:00Z", "SUCCESS")
	experience.ExperienceSHA256 = ""
	experience.RawContentIncluded = true
	if _, err := SealIdentityToolExperience(experience); err == nil {
		t.Fatal("raw content entered identity tool memory")
	}
	experience.RawContentIncluded = false
	experience.VerificationVerdict = "UNVERIFIED"
	if _, err := SealIdentityToolExperience(experience); err == nil {
		t.Fatal("unverified experience entered identity tool memory")
	}
}

func TestIdentityToolWisdomHoldsWhenOnlyMatchingEntryIsStale(t *testing.T) {
	experience := toolMemoryExperience(t, "experience-0001", "2026-08-16T07:00:00Z", "SUCCESS")
	experience.ExperienceSHA256 = ""
	experience.TTLMillis = 1_000
	sealed, err := SealIdentityToolExperience(experience)
	if err != nil {
		t.Fatal(err)
	}
	shard, err := StartIdentityToolMemory(sealed)
	if err != nil {
		t.Fatal(err)
	}
	query := IdentityWisdomQuery{
		Schema: IdentityWisdomQuerySchema, QueryID: "query-stale-0001",
		TargetAnsweringIdentitySHA256: toolMemoryRepeat("1"), ToolID: "image-render-hand", ToolVersion: "v1",
		TaskClass: "asset-render", UseTags: []string{"pixel-art"}, AsOf: "2026-08-16T07:00:02Z",
		MaxEntries: 2, Authority: Authority{},
	}
	view, err := RecallIdentityToolWisdom(shard, query)
	if err != nil {
		t.Fatal(err)
	}
	if view.State != ToolWisdomNoRelevantHold || len(view.Selected) != 0 {
		t.Fatalf("wisdom view = %+v", view)
	}
}

func TestIdentityToolMemoryAndWisdomDetectMutation(t *testing.T) {
	shard, err := StartIdentityToolMemory(toolMemoryExperience(t, "experience-0001", "2026-08-16T07:00:00Z", "SUCCESS"))
	if err != nil {
		t.Fatal(err)
	}
	mutated := shard
	mutated.Entries = append([]IdentityToolExperience(nil), shard.Entries...)
	mutated.Entries[0].Wisdom = "mutated"
	if err := mutated.Validate(); err == nil {
		t.Fatal("identity tool memory accepted mutated wisdom")
	}
	query := IdentityWisdomQuery{
		Schema: IdentityWisdomQuerySchema, QueryID: "query-tamper-0001",
		TargetAnsweringIdentitySHA256: toolMemoryRepeat("1"), ToolID: "image-render-hand", ToolVersion: "v1",
		TaskClass: "asset-render", UseTags: []string{"pixel-art"}, AsOf: "2026-08-16T07:01:00Z",
		MaxEntries: 2, Authority: Authority{},
	}
	view, err := RecallIdentityToolWisdom(shard, query)
	if err != nil {
		t.Fatal(err)
	}
	view.Selected[0].Wisdom = "mutated"
	if err := view.Validate(); err == nil {
		t.Fatal("identity wisdom view accepted mutated wisdom")
	}
}

func toolMemoryExperience(t *testing.T, id, observedAt, outcome string) IdentityToolExperience {
	t.Helper()
	experience := IdentityToolExperience{
		Schema: IdentityToolExperienceSchema, MemoryID: "waldo-image-render-memory", ExperienceID: id,
		TargetAnsweringIdentitySHA256: toolMemoryRepeat("1"), ToolID: "image-render-hand", ToolVersion: "v1",
		TaskClass: "asset-render", UseTags: []string{"transparent-background", "pixel-art"},
		InputArtifactSHA256: toolMemoryRepeat("2"), OutputArtifactSHA256: toolMemoryRepeat("3"),
		ToolReceiptSHA256: toolMemoryRepeat("4"), IndependentVerificationSHA256: toolMemoryRepeat("5"),
		Outcome: outcome, ObservedAt: observedAt, TTLMillis: 86_400_000,
		ContentClass: ToolMemoryContentPublicSafe, Authority: Authority{},
	}
	if outcome == ToolOutcomeSuccess {
		experience.VerificationVerdict = ToolVerificationSuccess
		experience.WisdomClass = ToolWisdomReuse
		experience.Wisdom = "Reuse the bounded transparent-background recipe when the exact pixel-art constraints match."
	} else {
		experience.VerificationVerdict = ToolVerificationFailure
		experience.WisdomClass = ToolWisdomAvoid
		experience.Wisdom = "Avoid the high-detail preset for this pixel-art task because independent verification found edge contamination."
	}
	sealed, err := SealIdentityToolExperience(experience)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func toolMemoryRepeat(value string) string {
	result := ""
	for len(result) < 64 {
		result += value
	}
	return result[:64]
}
