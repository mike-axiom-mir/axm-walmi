package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"syscall/js"

	"github.com/openwaldo/waldo/internal/axmmirror"
)

var callbacks []js.Func
var done = make(chan struct{})

func strictJSON(raw string, target any) error {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func encoded(value any, err error) string {
	packet := map[string]any{
		"schema":    "walmi.axm-mirror-browser-result/v1",
		"ok":        err == nil,
		"authority": "NONE",
	}
	if err != nil {
		packet["error"] = err.Error()
	} else {
		packet["result"] = value
	}
	raw, marshalErr := json.Marshal(packet)
	if marshalErr != nil {
		return `{"schema":"walmi.axm-mirror-browser-result/v1","ok":false,"error":"result encoding failed","authority":"NONE"}`
	}
	return string(raw)
}

func arg(args []js.Value, index int) (string, error) {
	if index >= len(args) {
		return "", fmt.Errorf("argument %d is required", index)
	}
	return args[index].String(), nil
}

func runtimeInfo(this js.Value, args []js.Value) any {
	return encoded(map[string]any{
		"schema":             "walmi.axm-mirror-browser-runtime/v1",
		"go_version":         runtime.Version(),
		"goos":               runtime.GOOS,
		"goarch":             runtime.GOARCH,
		"source_commit":      "c6011afbb456b8fe1e6c7c6eda7857aec681cb8f",
		"network_required":   false,
		"filesystem_required": false,
		"neural_generation":  false,
		"authority":          "NONE",
	}, nil)
}

func invoke(this js.Value, args []js.Value) any {
	op, err := arg(args, 0)
	if err != nil {
		return encoded(nil, err)
	}

	switch op {
	case "census-capabilities":
		raw, err := arg(args, 1)
		if err != nil { return encoded(nil, err) }
		var request axmmirror.SelfCapabilityCensusRequest
		if err := strictJSON(raw, &request); err != nil { return encoded(nil, err) }
		result, err := axmmirror.CensusSelfCapabilities(request)
		return encoded(result, err)

	case "seal-tool-experience":
		raw, err := arg(args, 1)
		if err != nil { return encoded(nil, err) }
		var experience axmmirror.IdentityToolExperience
		if err := strictJSON(raw, &experience); err != nil { return encoded(nil, err) }
		result, err := axmmirror.SealIdentityToolExperience(experience)
		return encoded(result, err)

	case "start-tool-memory":
		raw, err := arg(args, 1)
		if err != nil { return encoded(nil, err) }
		var experience axmmirror.IdentityToolExperience
		if err := strictJSON(raw, &experience); err != nil { return encoded(nil, err) }
		result, err := axmmirror.StartIdentityToolMemory(experience)
		return encoded(result, err)

	case "grow-tool-memory":
		currentRaw, err := arg(args, 1)
		if err != nil { return encoded(nil, err) }
		experienceRaw, err := arg(args, 2)
		if err != nil { return encoded(nil, err) }
		var current axmmirror.IdentityToolMemoryShard
		var experience axmmirror.IdentityToolExperience
		if err := strictJSON(currentRaw, &current); err != nil { return encoded(nil, err) }
		if err := strictJSON(experienceRaw, &experience); err != nil { return encoded(nil, err) }
		result, err := axmmirror.GrowIdentityToolMemory(current, experience)
		return encoded(result, err)

	case "recall-tool-wisdom":
		shardRaw, err := arg(args, 1)
		if err != nil { return encoded(nil, err) }
		queryRaw, err := arg(args, 2)
		if err != nil { return encoded(nil, err) }
		var shard axmmirror.IdentityToolMemoryShard
		var query axmmirror.IdentityWisdomQuery
		if err := strictJSON(shardRaw, &shard); err != nil { return encoded(nil, err) }
		if err := strictJSON(queryRaw, &query); err != nil { return encoded(nil, err) }
		result, err := axmmirror.RecallIdentityToolWisdom(shard, query)
		return encoded(result, err)

	case "anchor-bom":
		raw, err := arg(args, 1)
		if err != nil { return encoded(nil, err) }
		result, err := axmmirror.AnchorWALDOBOM([]byte(raw))
		return encoded(result, err)

	case "lens-corpus":
		raw, err := arg(args, 1)
		if err != nil { return encoded(nil, err) }
		result, err := axmmirror.LensCorpusBOM([]byte(raw))
		return encoded(result, err)

	default:
		return encoded(nil, fmt.Errorf("unsupported browser AXM Mirror operation %q", op))
	}
}

func shutdown(this js.Value, args []js.Value) any {
	select {
	case <-done:
	default:
		close(done)
	}
	return encoded(map[string]any{"state": "SHUTDOWN_REQUESTED"}, nil)
}

func main() {
	api := js.Global().Get("Object").New()
	invokeFn := js.FuncOf(invoke)
	infoFn := js.FuncOf(runtimeInfo)
	shutdownFn := js.FuncOf(shutdown)
	callbacks = append(callbacks, invokeFn, infoFn, shutdownFn)
	api.Set("invoke", invokeFn)
	api.Set("info", infoFn)
	api.Set("shutdown", shutdownFn)
	js.Global().Set("WALMIAxmMirror", api)
	js.Global().Set("WALMIAxmMirrorReady", true)
	<-done
	js.Global().Set("WALMIAxmMirrorReady", false)
	js.Global().Set("WALMIAxmMirror", js.Undefined())
	for _, callback := range callbacks {
		callback.Release()
	}
}
