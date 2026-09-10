package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

type reviewArtifact struct {
	ID        string
	Role      string
	Filename  string
	MIME      string
	Format    string
	Editable  bool
	Width     int
	Height    int
	SizeBytes int
	SHA256    string
	DataURL   string
}

type reviewPage struct {
	Candidate   axmmirror.InnerAssetCandidate
	Recipe      axmmirror.InnerAssetRecipe
	Validation  axmmirror.InnerAssetValidationReceipt
	Primary     reviewArtifact
	Preview     reviewArtifact
	Artifacts   []axmmirror.InnerAssetArtifact
	NextAction  string
	LicenseNote string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/inner-asset-review <candidate.axmasset> <new-review.html>")
		os.Exit(2)
	}
	candidate, err := writeReviewFile(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("OK REVIEW", candidate.State, candidate.CandidateSHA256)
}

func writeReviewFile(bundlePath, outputPath string) (axmmirror.InnerAssetCandidate, error) {
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		return axmmirror.InnerAssetCandidate{}, fmt.Errorf("read candidate %s: %w", bundlePath, err)
	}
	html, candidate, err := buildReviewHTML(data)
	if err != nil {
		return axmmirror.InnerAssetCandidate{}, fmt.Errorf("verify candidate before review: %w", err)
	}
	file, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return axmmirror.InnerAssetCandidate{}, fmt.Errorf("create new review %s: %w", outputPath, err)
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(outputPath)
		}
	}()
	if _, err := file.Write(html); err != nil {
		return axmmirror.InnerAssetCandidate{}, fmt.Errorf("write review %s: %w", outputPath, err)
	}
	if err := file.Sync(); err != nil {
		return axmmirror.InnerAssetCandidate{}, fmt.Errorf("sync review %s: %w", outputPath, err)
	}
	if err := file.Close(); err != nil {
		return axmmirror.InnerAssetCandidate{}, fmt.Errorf("close review %s: %w", outputPath, err)
	}
	remove = false
	return candidate, nil
}

func buildReviewHTML(bundle []byte) ([]byte, axmmirror.InnerAssetCandidate, error) {
	candidate, err := axmmirror.VerifyInnerAssetBundle(bundle)
	if err != nil {
		return nil, axmmirror.InnerAssetCandidate{}, err
	}
	entries, err := readVerifiedEntries(bundle)
	if err != nil {
		return nil, axmmirror.InnerAssetCandidate{}, err
	}

	recipeArtifact, err := requireArtifact(candidate, "normalized-recipe")
	if err != nil {
		return nil, axmmirror.InnerAssetCandidate{}, err
	}
	var recipe axmmirror.InnerAssetRecipe
	if err := axmmirror.DecodeStrictJSON(entries[recipeArtifact.Filename], &recipe); err != nil {
		return nil, axmmirror.InnerAssetCandidate{}, fmt.Errorf("decode verified recipe %s: %w", recipeArtifact.Filename, err)
	}

	var validation axmmirror.InnerAssetValidationReceipt
	if err := axmmirror.DecodeStrictJSON(entries[candidate.ValidationFilename], &validation); err != nil {
		return nil, axmmirror.InnerAssetCandidate{}, fmt.Errorf("decode verified validation %s: %w", candidate.ValidationFilename, err)
	}
	primaryArtifact, err := requireArtifact(candidate, candidate.PrimaryArtifactID)
	if err != nil {
		return nil, axmmirror.InnerAssetCandidate{}, err
	}
	previewArtifact, err := requireArtifact(candidate, candidate.PreviewArtifactID)
	if err != nil {
		return nil, axmmirror.InnerAssetCandidate{}, err
	}
	primary, err := makeReviewArtifact(*primaryArtifact, entries[primaryArtifact.Filename])
	if err != nil {
		return nil, axmmirror.InnerAssetCandidate{}, err
	}
	preview, err := makeReviewArtifact(*previewArtifact, entries[previewArtifact.Filename])
	if err != nil {
		return nil, axmmirror.InnerAssetCandidate{}, err
	}

	nextAction := "Visually inspect both verified realizations. If you choose to consume this READY candidate locally, use the separate materialize-asset command; this review cannot materialize, install, approve, promote, publish, or make CANON."
	if candidate.State != axmmirror.InnerAssetStateReady {
		nextAction = "Technical HOLD. Inspect the retained hold reasons and recipe before any consumer step; this review cannot clear the hold or materialize the candidate."
	}
	page := reviewPage{
		Candidate:   candidate,
		Recipe:      recipe,
		Validation:  validation,
		Primary:     primary,
		Preview:     preview,
		Artifacts:   append([]axmmirror.InnerAssetArtifact(nil), candidate.Artifacts...),
		NextAction:  nextAction,
		LicenseNote: "Not declared by the inner-asset recipe/candidate v0.1 contract; do not infer license fitness from technical verification.",
	}
	var output bytes.Buffer
	if err := reviewTemplate.Execute(&output, page); err != nil {
		return nil, axmmirror.InnerAssetCandidate{}, fmt.Errorf("render verified asset review: %w", err)
	}
	return output.Bytes(), candidate, nil
}

func readVerifiedEntries(bundle []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		return nil, fmt.Errorf("reopen verified bundle: %w", err)
	}
	entries := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		stream, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open verified entry %s: %w", file.Name, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, int64(axmmirror.MaxInnerAssetBundleBytes)+1))
		closeErr := stream.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read verified entry %s: %w", file.Name, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close verified entry %s: %w", file.Name, closeErr)
		}
		if len(data) > axmmirror.MaxInnerAssetBundleBytes {
			return nil, fmt.Errorf("verified entry %s exceeds review byte bound", file.Name)
		}
		entries[file.Name] = data
	}
	return entries, nil
}

func requireArtifact(candidate axmmirror.InnerAssetCandidate, id string) (*axmmirror.InnerAssetArtifact, error) {
	for index := range candidate.Artifacts {
		if candidate.Artifacts[index].ID == id {
			return &candidate.Artifacts[index], nil
		}
	}
	return nil, fmt.Errorf("verified candidate lacks artifact %q", id)
}

func makeReviewArtifact(artifact axmmirror.InnerAssetArtifact, data []byte) (reviewArtifact, error) {
	if artifact.MIME != "image/png" {
		return reviewArtifact{}, fmt.Errorf("review image %s has unsupported MIME %q", artifact.ID, artifact.MIME)
	}
	return reviewArtifact{
		ID: artifact.ID, Role: artifact.Role, Filename: artifact.Filename, MIME: artifact.MIME,
		Format: artifact.Format, Editable: artifact.Editable, Width: artifact.Width, Height: artifact.Height,
		SizeBytes: artifact.SizeBytes, SHA256: artifact.SHA256,
		DataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data),
	}, nil
}

func join(values []string) string {
	return strings.Join(values, " · ")
}

var reviewTemplate = template.Must(template.New("review").Funcs(template.FuncMap{"join": join}).Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="color-scheme" content="dark">
<meta name="axm-candidate-sha256" content="{{.Candidate.CandidateSHA256}}">
<title>WALMI Inner Asset Review · {{.Recipe.Title}}</title>
<style>
:root{font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;color:#eef7ff;background:#071016;line-height:1.45;--line:rgba(160,221,255,.24);--muted:#9fb2bf;--cyan:#67e8f9;--glass:rgba(13,28,37,.82)}
*{box-sizing:border-box}body{margin:0;min-height:100vh;background:radial-gradient(circle at 14% 0%,rgba(42,137,164,.22),transparent 32rem),radial-gradient(circle at 100% 30%,rgba(119,82,178,.16),transparent 35rem),#071016}button{font:inherit}button:focus-visible{outline:3px solid #f8f5a2;outline-offset:3px}[hidden]{display:none!important}.shell{width:min(1180px,calc(100% - 32px));margin:0 auto;padding:34px 0 56px}.mast{display:grid;gap:16px;margin-bottom:20px}.eyebrow{font-size:.76rem;letter-spacing:.17em;text-transform:uppercase;color:var(--cyan);font-weight:800}.title-row{display:flex;justify-content:space-between;gap:20px;align-items:end;flex-wrap:wrap}h1{font-size:clamp(2rem,5vw,4.4rem);line-height:.95;letter-spacing:-.045em;margin:0;max-width:16ch}.intent{max-width:62ch;color:#c7d7e1;font-size:1.02rem;margin:8px 0 0}.guard{border:1px solid var(--line);background:linear-gradient(120deg,rgba(12,45,58,.9),rgba(21,24,42,.86));padding:13px 15px;border-radius:14px;display:flex;gap:10px 18px;align-items:center;flex-wrap:wrap;box-shadow:inset 0 1px rgba(255,255,255,.06)}.guard strong{color:#c9fbff;letter-spacing:.08em}.guard span{color:#b7c8d1}.status{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px}.metric,.panel{border:1px solid var(--line);background:var(--glass);box-shadow:0 18px 50px rgba(0,0,0,.2),inset 0 1px rgba(255,255,255,.04);border-radius:18px}.metric{padding:14px}.metric b{display:block;font-size:.72rem;text-transform:uppercase;letter-spacing:.12em;color:var(--muted);margin-bottom:4px}.metric strong{font-size:1rem;overflow-wrap:anywhere}.grid{display:grid;grid-template-columns:minmax(0,1.35fr) minmax(310px,.65fr);gap:16px;margin-top:16px}.panel{padding:18px;min-width:0}.panel h2{margin:0 0 5px;font-size:1rem;letter-spacing:.07em;text-transform:uppercase}.panel-intro{color:var(--muted);margin:0 0 14px}.view-switch{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:14px}.view-switch button{min-height:44px;border-radius:12px;border:1px solid var(--line);background:#0d1b24;color:#dfeef6;padding:9px 14px;cursor:pointer;font-weight:750}.view-switch button[aria-pressed="true"]{background:#d9fbff;color:#071016;border-color:#d9fbff}.stage{position:relative;display:grid;place-items:center;min-height:410px;overflow:hidden;border:1px solid rgba(255,255,255,.09);border-radius:16px;background:linear-gradient(45deg,rgba(255,255,255,.035) 25%,transparent 25%,transparent 75%,rgba(255,255,255,.035) 75%),linear-gradient(45deg,rgba(255,255,255,.035) 25%,transparent 25%,transparent 75%,rgba(255,255,255,.035) 75%),#091219;background-position:0 0,8px 8px;background-size:16px 16px;padding:20px}.stage figure{margin:0;width:100%;text-align:center}.stage img{display:block;max-width:min(100%,720px);max-height:58vh;width:auto;height:auto;margin:auto;image-rendering:pixelated;image-rendering:crisp-edges;filter:drop-shadow(0 16px 28px rgba(0,0,0,.44))}.artifact-line{display:grid;grid-template-columns:auto 1fr;gap:4px 10px;text-align:left;margin:15px auto 0;max-width:720px;font-size:.82rem}.artifact-line dt{color:var(--muted)}.artifact-line dd{margin:0;overflow-wrap:anywhere}.live{min-height:1.5em;color:#c6f7ff;margin:10px 0 0}.next{border-left:3px solid var(--cyan);padding:11px 13px;background:rgba(36,133,153,.11);border-radius:0 10px 10px 0;margin-bottom:15px}.next b{display:block;color:#9ff7ff;font-size:.72rem;letter-spacing:.11em;text-transform:uppercase;margin-bottom:4px}.facts{display:grid;gap:9px;margin:0}.facts div{display:grid;grid-template-columns:125px minmax(0,1fr);gap:10px;border-top:1px solid rgba(255,255,255,.07);padding-top:9px}.facts dt{color:var(--muted)}.facts dd{margin:0;overflow-wrap:anywhere}.subhead{margin:20px 0 8px;color:#dcebf4;font-size:.8rem;letter-spacing:.09em;text-transform:uppercase}.palette{display:flex;flex-wrap:wrap;gap:8px}.chip{border:1px solid rgba(255,255,255,.12);background:rgba(255,255,255,.035);border-radius:999px;padding:6px 9px;display:inline-flex;align-items:center;gap:7px;font-size:.78rem}.swatch{width:15px;height:15px;border-radius:50%;border:1px solid rgba(255,255,255,.4);background:#111}.checks{display:grid;gap:7px}.check{display:grid;grid-template-columns:auto 1fr;gap:8px 10px;border-top:1px solid rgba(255,255,255,.07);padding-top:8px}.pass{color:#9ff7cc;font-weight:800}.hold{color:#ffd28b;font-weight:800}.detail{color:var(--muted);font-size:.84rem}.artifact-table{display:grid;gap:8px;margin-top:8px}.artifact-row{border:1px solid rgba(255,255,255,.08);border-radius:12px;padding:10px}.artifact-row b{display:block}.artifact-row small{color:var(--muted);overflow-wrap:anywhere;display:block}.ceiling{margin-top:16px;padding:14px;border:1px dashed rgba(255,216,139,.4);background:rgba(91,62,16,.13);border-radius:14px;color:#ecdcbf}.ceiling strong{color:#ffe1a6}.hash{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:.82em;overflow-wrap:anywhere}.footer{color:var(--muted);font-size:.8rem;margin-top:16px}
@media(max-width:760px){.shell{width:min(100% - 20px,1180px);padding-top:18px}.status{grid-template-columns:1fr 1fr}.grid{grid-template-columns:1fr}.panel{padding:14px}.stage{min-height:300px;padding:12px}.facts div{grid-template-columns:1fr;gap:2px}.view-switch{display:grid;grid-template-columns:1fr 1fr}.view-switch button{width:100%}}
@media(prefers-reduced-motion:reduce){*{scroll-behavior:auto!important;transition:none!important;animation:none!important}}
@media(prefers-contrast:more){:root{--line:rgba(206,242,255,.7);--muted:#cbdce5}.metric,.panel,.guard{border-width:2px}.stage{border-color:rgba(255,255,255,.6)}}
@media(forced-colors:active){button,[aria-pressed="true"],.metric,.panel,.guard,.stage{forced-color-adjust:auto}.pass,.hold{color:CanvasText}}
</style>
</head>
<body data-state="{{.Candidate.State}}" data-candidate-sha="{{.Candidate.CandidateSHA256}}">
<main class="shell">
<header class="mast">
<div class="eyebrow">WALMI · Inner Asset Review Desk</div>
<div class="title-row"><div><h1>{{.Recipe.Title}}</h1><p class="intent">{{.Recipe.IntendedUse}}</p></div></div>
<div class="guard"><strong>VERIFIED BUNDLE</strong><span>LOCAL / OFFLINE · DISPLAY ≠ APPROVAL · CANDIDATE ONLY · NO AUTO-MATERIALIZE</span></div>
<div class="status" aria-label="Candidate state">
<div class="metric"><b>Candidate</b><strong>{{.Candidate.State}}</strong></div>
<div class="metric"><b>Technical</b><strong>{{.Candidate.TechnicalStatus}}</strong></div>
<div class="metric"><b>Visual review</b><strong>{{.Candidate.VisualStatus}}</strong></div>
<div class="metric"><b>Authority</b><strong>NONE / HUMAN DECIDES</strong></div>
</div>
</header>
<section class="grid">
<section class="panel" aria-labelledby="surface-title">
<h2 id="surface-title">Verified realizations</h2>
<p class="panel-intro">Switch between the exact PNG artifacts already bound into this candidate. The review layer does not redraw or regenerate them.</p>
<div class="view-switch" role="group" aria-label="Asset realization">
<button type="button" data-view-button="preview" aria-pressed="true">Preview atlas</button>
<button type="button" data-view-button="primary" aria-pressed="false">Primary atlas</button>
</div>
<div class="stage">
<figure data-view="preview">
<img src="{{.Preview.DataURL}}" alt="Verified preview atlas for {{.Recipe.Title}}">
<figcaption><dl class="artifact-line"><dt>File</dt><dd>{{.Preview.Filename}}</dd><dt>Pixels</dt><dd>{{.Preview.Width}} × {{.Preview.Height}}</dd><dt>SHA-256</dt><dd class="hash">{{.Preview.SHA256}}</dd></dl></figcaption>
</figure>
<figure data-view="primary" hidden>
<img src="{{.Primary.DataURL}}" alt="Verified primary atlas for {{.Recipe.Title}}">
<figcaption><dl class="artifact-line"><dt>File</dt><dd>{{.Primary.Filename}}</dd><dt>Pixels</dt><dd>{{.Primary.Width}} × {{.Primary.Height}}</dd><dt>SHA-256</dt><dd class="hash">{{.Primary.SHA256}}</dd></dl></figcaption>
</figure>
</div>
<p class="live" aria-live="polite" data-view-status>Showing verified preview atlas.</p>
</section>
<aside class="panel" aria-labelledby="evidence-title">
<h2 id="evidence-title">Review evidence</h2>
<div class="next"><b>Next explicit action</b>{{.NextAction}}</div>
<dl class="facts">
<div><dt>Recipe</dt><dd>{{.Recipe.RecipeID}} · {{.Recipe.Kind}} · {{.Recipe.Profile}}</dd></div>
<div><dt>Created</dt><dd>{{.Recipe.CreatedAt}}</dd></div>
<div><dt>Seed</dt><dd class="hash">{{.Recipe.Seed}}</dd></div>
<div><dt>Candidate SHA</dt><dd class="hash">{{.Candidate.CandidateSHA256}}</dd></div>
<div><dt>Recipe SHA</dt><dd class="hash">{{.Candidate.RecipeSHA256}}</dd></div>
<div><dt>Validation receipt</dt><dd class="hash">{{.Candidate.ValidationReceiptSHA256}}</dd></div>
<div><dt>Frames / colours</dt><dd>{{.Candidate.Measures.FrameCount}} frames · {{.Candidate.Measures.ColoursUsed}} colours used</dd></div>
<div><dt>Semantic core</dt><dd>{{if .Candidate.Measures.ProtectedSemanticLayers}}{{join .Candidate.Measures.ProtectedSemanticLayers}}{{else}}none declared{{end}}</dd></div>
<div><dt>License fitness</dt><dd>{{.LicenseNote}}</dd></div>
</dl>
<h3 class="subhead">Palette</h3>
<div class="palette">{{range .Recipe.Palette}}<span class="chip"><span class="swatch" data-color="{{.RGBA}}" aria-hidden="true"></span>{{.ID}} · {{.Role}} · <span class="hash">{{.RGBA}}</span></span>{{end}}</div>
<h3 class="subhead">Technical checks</h3>
<div class="checks">{{range .Validation.Checks}}<div class="check"><span class="{{if .Pass}}pass{{else}}hold{{end}}">{{if .Pass}}PASS{{else}}HOLD{{end}}</span><div><b>{{.Name}}</b><div class="detail">{{.Detail}}</div></div></div>{{end}}</div>
<h3 class="subhead">Bound artifacts</h3>
<div class="artifact-table">{{range .Artifacts}}<div class="artifact-row"><b>{{.ID}} · {{.Role}}</b><small>{{.Filename}} · {{.Format}} · {{.SizeBytes}} bytes</small><small class="hash">{{.SHA256}}</small></div>{{end}}</div>
<div class="ceiling"><strong>Evidence ceiling.</strong> Exact bundle verification proves deterministic bounded compilation and recorded byte identity. It does not prove appearance quality, originality, accessibility, game fitness, license rights, human preference, approval, installation, promotion, publication, or CANON.</div>
</aside>
</section>
<p class="footer">Generated from one locally verified .axmasset. No network request, model call, source retrieval, install step, asset rewrite, or state promotion is performed by this review page.</p>
</main>
<script>
(() => {
  const buttons = [...document.querySelectorAll('[data-view-button]')];
  const views = [...document.querySelectorAll('[data-view]')];
  const status = document.querySelector('[data-view-status]');
  const labels = { preview: 'Showing verified preview atlas.', primary: 'Showing verified primary atlas.' };
  function show(name) {
    buttons.forEach((button) => button.setAttribute('aria-pressed', String(button.dataset.viewButton === name)));
    views.forEach((view) => { view.hidden = view.dataset.view !== name; });
    status.textContent = labels[name];
  }
  buttons.forEach((button) => button.addEventListener('click', () => show(button.dataset.viewButton)));
  document.querySelectorAll('[data-color]').forEach((swatch) => { swatch.style.backgroundColor = swatch.dataset.color; });
})();
</script>
</body>
</html>
`))
