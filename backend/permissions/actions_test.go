package permissions

import "testing"

func TestActionCatalogHasUniqueStableCodes(t *testing.T) {
	seen := map[string]bool{}
	for _, definition := range Definitions() {
		if definition.Code == "" || definition.Resource == "" || definition.Action == "" || definition.Label == "" {
			t.Fatalf("incomplete action definition: %+v", definition)
		}
		if seen[definition.Code] {
			t.Fatalf("duplicate action code: %s", definition.Code)
		}
		seen[definition.Code] = true
		if IsReadOnly(definition.Code) != definition.ReadOnly {
			t.Fatalf("read-only classification mismatch: %+v", definition)
		}
	}
	if len(seen) != len(AllCodes()) {
		t.Fatalf("catalog size mismatch: definitions=%d codes=%d", len(seen), len(AllCodes()))
	}
	for _, code := range DefaultRoleCodes() {
		if !IsReadOnly(code) {
			t.Fatalf("default role received write action: %s", code)
		}
	}
	for _, required := range []string{UsersCreate, RolesUpdate, ArticlesDelete, FilesPermanentDelete} {
		if !IsKnown(required) || IsReadOnly(required) {
			t.Fatalf("write action misclassified: %s", required)
		}
	}
}
