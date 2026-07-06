package kern

import "testing"

func TestIntPathConstraint(t *testing.T) {
	cases := []struct {
		value string
		ok    bool
	}{
		{"42", true},
		{"-42", true},
		{"0", true},
		{"", false},
		{"-", false},
		{"4a", false},
	}

	for _, c := range cases {
		if got := IntPathConstraint(c.value); got != c.ok {
			t.Fatalf("IntPathConstraint(%q) got %v want %v", c.value, got, c.ok)
		}
	}
}

func TestUintPathConstraint(t *testing.T) {
	cases := []struct {
		value string
		ok    bool
	}{
		{"42", true},
		{"0", true},
		{"-1", false},
		{"", false},
		{"1a", false},
	}

	for _, c := range cases {
		if got := UintPathConstraint(c.value); got != c.ok {
			t.Fatalf("UintPathConstraint(%q) got %v want %v", c.value, got, c.ok)
		}
	}
}

func TestSlugPathConstraint(t *testing.T) {
	cases := []struct {
		value string
		ok    bool
	}{
		{"user-1", true},
		{"A_B_123", true},
		{"", false},
		{"user/name", false},
		{"a b", false},
	}

	for _, c := range cases {
		if got := SlugPathConstraint(c.value); got != c.ok {
			t.Fatalf("SlugPathConstraint(%q) got %v want %v", c.value, got, c.ok)
		}
	}
}

func TestUUIDPathConstraint(t *testing.T) {
	if !UUIDPathConstraint("123e4567-e89b-12d3-a456-426614174000") {
		t.Fatal("expected valid uuid")
	}
	if UUIDPathConstraint("123e4567e89b12d3a456426614174000") {
		t.Fatal("expected invalid uuid")
	}
}
