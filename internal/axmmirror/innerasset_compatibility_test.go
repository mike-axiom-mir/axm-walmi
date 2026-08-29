package axmmirror

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInnerAssetFixturePinsInitialPortableCandidate(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "axm-mirror", "inner-asset-recipe.json"))
	if err != nil {
		t.Fatal(err)
	}
	var recipe InnerAssetRecipe
	if err := DecodeStrictJSON(data, &recipe); err != nil {
		t.Fatal(err)
	}
	build, err := ForgeInnerAsset(recipe)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := EncodeInnerAssetBundle(build)
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string][2]string{
		"recipe":     {build.Candidate.RecipeSHA256, "003e6b12423bca600c61d4fb70a132b49e0a5a74f3004789365b9d1a1ce0a7bf"},
		"candidate":  {build.Candidate.CandidateSHA256, "0eba9f0b97516305ddc8201e006d404699d057ad05ce38c472c3403f45f6a289"},
		"validation": {build.Validation.ReceiptSHA256, "1d8c51cb5d3b7b0d6527c2834b1e8ec28c9b56e8671b380a5f652f3723b5059f"},
		"bundle":     {innerAssetBytesSHA256(bundle), "82b7728d125cb66df310f05fcaf8c70b2ed5bd9615ad2324fcf8f9f9656859dd"},
	}
	for name, pair := range checks {
		if pair[0] != pair[1] {
			t.Fatalf("%s digest = %s, want %s", name, pair[0], pair[1])
		}
	}
}
