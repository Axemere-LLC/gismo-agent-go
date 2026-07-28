package agent

import (
	"os"
	"testing"
)

func TestDefaultAddr(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		portSet  bool
		fallback string
		want     string
	}{
		{name: "port set", port: "8080", portSet: true, fallback: ":8081", want: ":8080"},
		{name: "port unset", portSet: false, fallback: ":8081", want: ":8081"},
		{name: "port set empty", port: "", portSet: true, fallback: ":8081", want: ":8081"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.portSet {
				t.Setenv("PORT", tt.port)
			} else if prev, ok := os.LookupEnv("PORT"); ok {
				os.Unsetenv("PORT")
				t.Cleanup(func() { os.Setenv("PORT", prev) })
			}

			if got := DefaultAddr(tt.fallback); got != tt.want {
				t.Errorf("DefaultAddr(%q) = %q, want %q", tt.fallback, got, tt.want)
			}
		})
	}
}
