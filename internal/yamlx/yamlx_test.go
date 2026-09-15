package yamlx

import (
	"testing"
)

func TestScalarString(t *testing.T) {
	mapping, err := ParseMapping("plain: value\nsingle: 'quoted one'\ndouble: \"quoted two\"\nnumber: 42\nempty:\n")
	if err != nil {
		t.Fatalf("ParseMapping() error = %v", err)
	}

	cases := map[string]string{
		"plain":  "value",
		"single": "quoted one",
		"double": "quoted two",
		"number": "42",
	}
	for key, want := range cases {
		value := FindMappingValue(mapping, key)
		if value == nil {
			t.Fatalf("FindMappingValue(%q) = nil", key)
		}
		if got := ScalarString(value.Value); got != want {
			t.Errorf("ScalarString(%s) = %q, want %q", key, got, want)
		}
	}

	if got := ScalarString(nil); got != "" {
		t.Errorf("ScalarString(nil) = %q, want \"\"", got)
	}

	empty := FindMappingValue(mapping, "empty")
	if empty == nil {
		t.Fatal("FindMappingValue(empty) = nil")
	}
	if got := ScalarString(empty.Value); got != "" {
		t.Errorf("ScalarString(empty) = %q, want \"\"", got)
	}
}
