package provider

import (
	"strings"
	"testing"
)

func TestExtractVaultName(t *testing.T) {
	t.Parallel()

	const keyVaultIDPrefix = "/subscriptions/subscription/resourceGroups/group/providers/Microsoft.KeyVault/vaults/"

	tests := []struct {
		name      string
		vaultName string
		want      string
		wantErr   bool
	}{
		{
			name:      "minimum length",
			vaultName: "abc",
			want:      "abc",
		},
		{
			name:      "maximum length",
			vaultName: strings.Repeat("a", 24),
			want:      strings.Repeat("a", 24),
		},
		{
			name:      "allowed characters",
			vaultName: "Vault-123",
			want:      "Vault-123",
		},
		{
			name:      "too short",
			vaultName: "ab",
			wantErr:   true,
		},
		{
			name:      "too long",
			vaultName: strings.Repeat("a", 25),
			wantErr:   true,
		},
		{
			name:      "with invalid characters",
			vaultName: "example.com?a=",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractVaultName(keyVaultIDPrefix + tt.vaultName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("extractVaultName() error = nil, want an error")
				}
				if got != "" {
					t.Errorf("extractVaultName() = %q, want empty", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractVaultName() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("extractVaultName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractVaultNameAndName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            string
		wantVaultName string
		wantName      string
		wantErr       bool
	}{
		{
			name:          "without version",
			id:            "https://vault-name.vault.azure.net/secrets/secret",
			wantVaultName: "vault-name",
			wantName:      "secret",
		},
		{
			name:          "with version",
			id:            "https://vault-name.vault.azure.net/secrets/secret/version",
			wantVaultName: "vault-name",
			wantName:      "secret",
		},
		{
			name:          "minimum length",
			id:            "https://abc.vault.azure.net/secrets/secret/version",
			wantVaultName: "abc",
			wantName:      "secret",
		},
		{
			name:          "maximum length",
			id:            "https://" + strings.Repeat("a", 24) + ".vault.azure.net/secrets/secret/version",
			wantVaultName: strings.Repeat("a", 24),
			wantName:      "secret",
		},
		{
			name:    "too short",
			id:      "https://ab.vault.azure.net/secrets/secret",
			wantErr: true,
		},
		{
			name:    "too long",
			id:      "https://" + strings.Repeat("a", 25) + ".vault.azure.net/secrets/secret",
			wantErr: true,
		},
		{
			name:    "with invalid characters",
			id:      "https://example.com?a=.vault.azure.net/secrets/secret",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotVaultName, gotName, err := extractVaultNameAndName(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Errorf("extractVaultNameAndName() error = nil, want an error")
				}
				if gotVaultName != "" {
					t.Errorf("extractVaultNameAndName() vaultName = %q, want empty", gotVaultName)
				}
				if gotName != "" {
					t.Errorf("extractVaultNameAndName() name = %q, want empty", gotName)
				}
				return
			}

			if err != nil {
				t.Fatalf("extractVaultNameAndName() error = %v", err)
			}

			if gotVaultName != tt.wantVaultName {
				t.Errorf("extractVaultNameAndName() vaultName = %q, want %q", gotVaultName, tt.wantVaultName)
			}
			if gotName != tt.wantName {
				t.Errorf("extractVaultNameAndName() name = %q, want %q", gotName, tt.wantName)
			}
		})
	}
}
