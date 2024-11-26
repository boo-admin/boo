package booclient

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"golang.org/x/exp/slog"
)

type FieldGetter interface {
	Data(ctx context.Context, name string) interface{}
}

func ReadField(ctx context.Context, name string, v interface{}) interface{} {
	if v == nil {
		return nil
	}

	getter, ok := v.(FieldGetter)
	if ok {
		return getter.Data(ctx, name)
	}

	switch u := v.(type) {
	case map[string]interface{}:
		if u == nil {
			return nil
		}

		o, ok := u[name]
		if ok {
			return o
		}

		embededObject := u["attributes"]
		if embededObject == nil {
			embededObject = u["fields"]
			if embededObject == nil {
				return nil
			}
		}

		if s, ok := embededObject.(string); ok && strings.HasPrefix(s, "{") {
			var s2o map[string]interface{}
			err := json.Unmarshal([]byte(s), &s2o)
			if err != nil {
				slog.Warn("ReadField() fail", slog.Any("err", err))
			}
			if s2o != nil {
				return s2o[name]
			}
		}
		if bs, ok := embededObject.([]byte); ok && bytes.HasPrefix(bs, []byte("{")) {
			var s2o map[string]interface{}
			err := json.Unmarshal(bs, &s2o)
			if err != nil {
				slog.Warn("ReadField() fail", slog.Any("err", err))
			}
			if s2o != nil {
				return s2o[name]
			}
		}
	case map[string]string:
		if u == nil {
			return nil
		}
		return u[name]
	}
	return nil
}
