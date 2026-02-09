package workloads

import "testing"

func TestParseMixValid(t *testing.T) {
	mix, err := ParseMix("small=2,med=3,large=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mix["small"] != 2 || mix["medium"] != 3 || mix["large"] != 1 {
		t.Fatalf("unexpected mix: %#v", mix)
	}
}

func TestParseMixInvalid(t *testing.T) {
	if _, err := ParseMix("small=1,unknown=2"); err == nil {
		t.Fatal("expected error for unknown size class")
	}
	if _, err := ParseMix("small=notint"); err == nil {
		t.Fatal("expected error for invalid count")
	}
}

func TestMixTotal(t *testing.T) {
	mix := Mix{"small": 2, "medium": 3, "large": 4}
	if total := MixTotal(mix); total != 9 {
		t.Fatalf("expected total 9, got %d", total)
	}
}
