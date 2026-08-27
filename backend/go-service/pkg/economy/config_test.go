package economy

import "testing"

func TestPriceTableChecksumIsIndependentOfMapInsertionOrder(t *testing.T) {
	original := weaponPrices
	t.Cleanup(func() { weaponPrices = original })

	forward := make(map[string]int, len(original))
	for item, price := range original {
		forward[item] = price
	}
	weaponPrices = forward
	first := PriceTableChecksum()

	reverseKeys := make([]string, 0, len(original))
	for item := range original {
		reverseKeys = append([]string{item}, reverseKeys...)
	}
	reverse := make(map[string]int, len(original))
	for _, item := range reverseKeys {
		reverse[item] = original[item]
	}
	weaponPrices = reverse
	second := PriceTableChecksum()

	if first != second {
		t.Fatalf("price-table checksum depends on map order: %s != %s", first, second)
	}
}

func TestPriceQuotesDistinguishKnownUnknownAndRealZero(t *testing.T) {
	if price, status := PriceQuote("AK-47"); price != 2700 || status != "known" {
		t.Fatalf("known price = %d/%s", price, status)
	}
	if price, status := PriceQuote("C4"); price != 0 || status != "known_zero" {
		t.Fatalf("real zero = %d/%s", price, status)
	}
	if price, status := PriceQuote("future-item"); price != 0 || status != "unknown" {
		t.Fatalf("unknown price = %d/%s", price, status)
	}
}

func TestLossBonusUsesVersionedConsecutiveLevels(t *testing.T) {
	want := []int{1400, 1900, 2400, 2900, 3400}
	for level, expected := range want {
		if actual := LossBonus(level); actual != expected {
			t.Fatalf("level %d = %d, want %d", level, actual, expected)
		}
	}
}
