package axmmirror

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

func platformFabricExternalReceiptDigest(data []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", fmt.Errorf("decode platform Fabric canonical receipt: %w", err)
	}
	root, ok := value.(map[string]any)
	if !ok {
		return "", errors.New("platform Fabric receipt root must be an object")
	}
	delete(root, "receiptSha256")
	canonical, err := encodePlatformCanonicalJSON(root)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func encodePlatformCanonicalJSON(value any) ([]byte, error) {
	switch typed := value.(type) {
	case nil:
		return []byte("null"), nil
	case bool:
		if typed {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	case string:
		return json.Marshal(typed)
	case json.Number:
		return nil, errors.New("platform Fabric bridge v0.1 receipt is numberless; numeric canonicalization is refused")
	case []any:
		var buffer bytes.Buffer
		buffer.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				buffer.WriteByte(',')
			}
			encoded, err := encodePlatformCanonicalJSON(item)
			if err != nil {
				return nil, err
			}
			buffer.Write(encoded)
		}
		buffer.WriteByte(']')
		return buffer.Bytes(), nil
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var buffer bytes.Buffer
		buffer.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				buffer.WriteByte(',')
			}
			encodedKey, _ := json.Marshal(key)
			buffer.Write(encodedKey)
			buffer.WriteByte(':')
			encodedValue, err := encodePlatformCanonicalJSON(typed[key])
			if err != nil {
				return nil, err
			}
			buffer.Write(encodedValue)
		}
		buffer.WriteByte('}')
		return buffer.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported platform Fabric canonical JSON type %T", value)
	}
}

func platformFabricWitnessDigest(witness PlatformFabricWitness) (string, error) {
	witness.WitnessSHA256 = ""
	payload, err := json.Marshal(witness)
	if err != nil {
		return "", fmt.Errorf("marshal platform Fabric witness: %w", err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
