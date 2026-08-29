package axmmirror

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

func WitnessHermesExposureV021Contract(data []byte) (HermesExposureV021Witness, error) {
	c, err := VerifyHermesExposureV021Contract(data)
	if err != nil {
		return HermesExposureV021Witness{}, err
	}
	profiles := make([]string, 0, len(c.Profiles))
	for _, p := range c.Profiles {
		profiles = append(profiles, p.ProfileID)
	}
	sort.Strings(profiles)
	w := HermesExposureV021Witness{
		Schema:                 HermesExposureV021WitnessSchema,
		SourceV020Receipt:      c.Source.SourceV020Receipt,
		SourceV020Head:         c.Source.SourceV020Head,
		HermesLocalDonorCommit: c.Source.LocalDonorCommit,
		HermesReviewedCommit:   c.Source.ReviewedCommit,
		DonorFileCount:         len(c.Source.DonorFiles),
		SurfaceProfiles:        profiles,
		EvaluatedCases:         len(c.Cases),
		RefusalCases:           len(c.RefusalCases),
		SourceReceiptSHA256:    c.ReceiptDigest,
		Authority:              "NONE",
	}
	raw, err := json.Marshal(w)
	if err != nil {
		return HermesExposureV021Witness{}, err
	}
	sum := sha256.Sum256(raw)
	w.WitnessDigest = hex.EncodeToString(sum[:])
	return w, nil
}
