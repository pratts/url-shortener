package urls

import (
	"shortener/configs"
	"strings"
	"testing"
)

func TestValidateTargetURL(t *testing.T) {
	configs.AppConfig.ApiUrl = "https://tidylnk.com"

	valid := []string{
		"https://example.com",
		"http://example.com/path?q=1#frag",
		"  https://example.com/trimmed  ",
		"HTTPS://Example.com",
	}
	for _, u := range valid {
		if _, err := ValidateTargetURL(u); err != nil {
			t.Errorf("expected %q to be valid, got %v", u, err)
		}
	}

	invalid := []string{
		"",
		"   ",
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
		"ftp://example.com",
		"//example.com",
		"example.com",
		"https://",
		"https://user:pass@example.com",
		"https://example.com@evil.com",
		"https://tidylnk.com/abc1234",
		"https://TIDYLNK.com/abc1234",
		"https://example.com/" + strings.Repeat("a", maxUrlLen),
	}
	for _, u := range invalid {
		if _, err := ValidateTargetURL(u); err == nil {
			t.Errorf("expected %q to be rejected", u)
		}
	}
}

func TestGenerateCode(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		code, err := generateCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != codeLength {
			t.Fatalf("code %q has length %d, want %d", code, len(code), codeLength)
		}
		for _, c := range code {
			if !strings.ContainsRune(base62, c) {
				t.Fatalf("code %q contains non-base62 character %q", code, c)
			}
		}
		if seen[code] {
			t.Fatalf("duplicate code %q in 10000 draws", code)
		}
		seen[code] = true
	}
}
