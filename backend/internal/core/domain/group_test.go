package domain

import "testing"

func TestAccessTypeAllows(t *testing.T) {
	cases := []struct {
		name     string
		access   AccessType
		required AccessType
		want     bool
	}{
		{name: "edit allows edit", access: AccessViewEdit, required: AccessViewEdit, want: true},
		{name: "edit allows view", access: AccessViewEdit, required: AccessViewOnly, want: true},
		{name: "view allows view", access: AccessViewOnly, required: AccessViewOnly, want: true},
		{name: "view rejects edit", access: AccessViewOnly, required: AccessViewEdit, want: false},
		{name: "unknown rejects view", access: AccessType("unknown"), required: AccessViewOnly, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.access.Allows(tc.required); got != tc.want {
				t.Fatalf("%q.Allows(%q) = %v, want %v", tc.access, tc.required, got, tc.want)
			}
		})
	}
}
