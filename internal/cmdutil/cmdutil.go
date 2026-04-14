package cmdutil

import (
	"context"
	"strconv"

	"dappco.re/go/core"
	"dappco.re/go/core/cli/pkg/cli"
)

// ContextOrBackground returns the active CLI context when available.
func ContextOrBackground() context.Context {
	if ctx, ok := currentCLIContext(); ok && ctx != nil {
		return ctx
	}

	return context.Background()
}

func currentCLIContext() (ctx context.Context, ok bool) {
	defer func() {
		if recover() != nil {
			ctx = nil
			ok = false
		}
	}()

	return cli.Context(), true
}

// OptionString returns the first non-empty option value for the provided keys.
func OptionString(opts core.Options, keys ...string) string {
	for _, key := range keys {
		if value := opts.String(key); value != "" {
			return value
		}
	}
	return ""
}

// OptionBoolDefault returns the parsed boolean value for the first matching key.
// Missing values fall back to defaultValue.
func OptionBoolDefault(opts core.Options, defaultValue bool, keys ...string) bool {
	for _, key := range keys {
		result := opts.Get(key)
		if !result.OK {
			continue
		}

		switch value := result.Value.(type) {
		case bool:
			return value
		case string:
			parsed, err := strconv.ParseBool(value)
			if err == nil {
				return parsed
			}
		}
	}

	return defaultValue
}

// OptionBool returns the parsed boolean value for the first matching key.
func OptionBool(opts core.Options, keys ...string) bool {
	return OptionBoolDefault(opts, false, keys...)
}

// ResultFromError adapts a Go error into a Core result.
func ResultFromError(err error) core.Result {
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
}
