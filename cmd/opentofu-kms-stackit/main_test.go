package main

import "testing"

func TestHasVersionFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "single dash", args: []string{"-version"}, want: true},
		{name: "double dash", args: []string{"--version"}, want: true},
		{name: "after other flags", args: []string{"-key-id", "key", "-version"}, want: true},
		{name: "no args", args: nil, want: false},
		{name: "key flags only", args: []string{"-project-id", "proj", "-key-id", "key"}, want: false},
		{name: "not a flag", args: []string{"version"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasVersionFlag(tt.args); got != tt.want {
				t.Fatalf("hasVersionFlag(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
