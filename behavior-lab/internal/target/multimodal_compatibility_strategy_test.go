package target

import "testing"

func TestMultimodalCompatibilityStrategies(t *testing.T) {
	tests := []struct {
		name       string
		wantName   string
		wantDetail any
		wantKey    bool
	}{
		{name: CompatibilityLMStudio, wantName: CompatibilityLMStudio, wantDetail: "high", wantKey: true},
		{name: CompatibilityOpenAI, wantName: CompatibilityOpenAI, wantDetail: "auto", wantKey: true},
		{name: CompatibilityMinimal, wantName: CompatibilityMinimal, wantDetail: nil, wantKey: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, err := ResolveMultimodalCompatibility(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if s.Name() != tc.wantName {
				t.Fatalf("name=%q want=%q", s.Name(), tc.wantName)
			}
			part := s.ImageURLPart("data:image/png;base64,abc")
			if part["url"] != "data:image/png;base64,abc" {
				t.Fatalf("unexpected url: %#v", part["url"])
			}
			detail, ok := part["detail"]
			if ok != tc.wantKey {
				t.Fatalf("detail presence=%t want=%t", ok, tc.wantKey)
			}
			if ok && detail != tc.wantDetail {
				t.Fatalf("detail=%#v want=%#v", detail, tc.wantDetail)
			}
		})
	}
}

func TestResolveMultimodalCompatibilityRejectsUnknown(t *testing.T) {
	if _, err := ResolveMultimodalCompatibility("vendor-magic"); err == nil {
		t.Fatal("expected unsupported strategy error")
	}
}
