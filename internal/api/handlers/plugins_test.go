package handlers

import (
	"testing"
)

func TestBuildConfigMap(t *testing.T) {
	cases := []struct {
		name  string
		input map[string]any
		want  map[string]string
	}{
		{
			name:  "string passthrough",
			input: map[string]any{"host": "example.com"},
			want:  map[string]string{"host": "example.com"},
		},
		{
			name:  "integer serialized",
			input: map[string]any{"port": 8080},
			want:  map[string]string{"port": "8080"},
		},
		{
			name:  "bool serialized",
			input: map[string]any{"tls": true},
			want:  map[string]string{"tls": "true"},
		},
		{
			name:  "array serialized",
			input: map[string]any{"tags": []any{"a", "b"}},
			want:  map[string]string{"tags": `["a","b"]`},
		},
		{
			name:  "object serialized",
			input: map[string]any{"instances": map[string]any{"name": "prod"}},
			want:  map[string]string{"instances": `{"name":"prod"}`},
		},
		{
			name:  "nil omitted",
			input: map[string]any{"key": nil},
			want:  map[string]string{},
		},
		{
			name:  "mixed types",
			input: map[string]any{"url": "https://x", "count": 3, "enabled": true},
			want:  map[string]string{"url": "https://x", "count": "3", "enabled": "true"},
		},
		{
			name:  "empty config",
			input: map[string]any{},
			want:  map[string]string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildConfigMap(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("buildConfigMap len: got %d, want %d (got %v, want %v)", len(got), len(tc.want), got, tc.want)
			}
			for k, wantV := range tc.want {
				if gotV, ok := got[k]; !ok {
					t.Errorf("missing key %q", k)
				} else if gotV != wantV {
					t.Errorf("key %q: got %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}

func TestIsSecretKey(t *testing.T) {
	secret := []string{
		"token", "apiToken", "api_token", "API_TOKEN",
		"secret", "clientSecret", "client_secret",
		"password", "userPassword",
		"privateKey", "private_key",
	}
	notSecret := []string{
		"url", "host", "port", "enabled", "instances",
		"syncInterval", "namespace", "name",
	}

	for _, k := range secret {
		if !isSecretKey(k) {
			t.Errorf("isSecretKey(%q) = false, want true", k)
		}
	}
	for _, k := range notSecret {
		if isSecretKey(k) {
			t.Errorf("isSecretKey(%q) = true, want false", k)
		}
	}
}

func TestRedactSecretValues(t *testing.T) {
	input := map[string]any{
		"url":      "https://example.com",
		"token":    "secret-token",
		"password": "",
		"nested": map[string]any{
			"apiToken": "hidden",
			"name":     "visible",
		},
	}

	got := redactSecretValues(input).(map[string]any)

	if got["url"] != "https://example.com" {
		t.Errorf("url: got %v, want unchanged", got["url"])
	}
	if got["token"] != redactedSecretValue {
		t.Errorf("token: got %v, want redacted", got["token"])
	}
	// Empty string secrets are not redacted (nothing to hide).
	if got["password"] != "" {
		t.Errorf("password: got %v, want empty string", got["password"])
	}
	nested, _ := got["nested"].(map[string]any)
	if nested["apiToken"] != redactedSecretValue {
		t.Errorf("nested.apiToken: got %v, want redacted", nested["apiToken"])
	}
	if nested["name"] != "visible" {
		t.Errorf("nested.name: got %v, want unchanged", nested["name"])
	}
}

func TestPreserveSecretValues(t *testing.T) {
	existing := map[string]any{
		"url":   "https://example.com",
		"token": "real-token",
	}
	// Incoming update has a redacted placeholder for token.
	updated := map[string]any{
		"url":   "https://new.example.com",
		"token": redactedSecretValue,
	}

	got := preserveSecretValues(existing, updated).(map[string]any)

	if got["url"] != "https://new.example.com" {
		t.Errorf("url: got %v, want updated value", got["url"])
	}
	if got["token"] != "real-token" {
		t.Errorf("token: got %v, want preserved real value", got["token"])
	}
}
