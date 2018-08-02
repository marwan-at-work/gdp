package vanity

import "testing"

func TestDeduceVanity(t *testing.T) {
	str, err := deduceVanity("go.opencensus.io")
	if err != nil {
		t.Fatal(err)
	}

	t.Fatal(str)
}
