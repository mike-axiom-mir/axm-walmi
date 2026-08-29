package cli

import "github.com/openwaldo/waldo/internal/inference"

// v0.35 routes the existing model-chat consumer through the experimental AXM
// session wrapper. The wrapper itself preserves an explicit raw fallback via
// WALDO_AXM_HYBRID=raw/off/0/false. No model, training, or worker implementation
// is replaced here.
func init() {
	openModelChat = inference.OpenAXMHybrid
}
