package domain_test

import (
	"testing"

	"trust_wallet_homework/internal/core/domain"
)

func TestNewAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantVal string
	}{
		{
			name:    "Valid lowercase address",
			input:   "0x71c7656ec7ab88b098defb751b7401b5f6d8976f",
			wantErr: false,
			wantVal: "0x71c7656ec7ab88b098defb751b7401b5f6d8976f",
		},
		{
			name:    "Valid uppercase address (expect lowercase)",
			input:   "0X71C7656EC7AB88B098DEFB751B7401B5F6D8976F",
			wantErr: false,
			wantVal: "0x71c7656ec7ab88b098defb751b7401b5f6d8976f",
		},
		{
			name:    "Valid mixed case address (expect lowercase)",
			input:   "0x71c7656Ec7aB88b098dEfb751B7401b5f6d8976f",
			wantErr: false,
			wantVal: "0x71c7656ec7ab88b098defb751b7401b5f6d8976f",
		},
		{
			name:    "Invalid address (too short)",
			input:   "0x71c7656ec7ab88b098defb751b7401b5f6d8",
			wantErr: true,
		},
		{
			name:    "Invalid address (too long)",
			input:   "0x71c7656ec7ab88b098defb751b7401b5f6d8976f00",
			wantErr: true,
		},
		{
			name:    "Invalid address (missing 0x)",
			input:   "71c7656ec7ab88b098defb751b7401b5f6d8976f",
			wantErr: true,
		},
		{
			name:    "Invalid address (invalid characters)",
			input:   "0x71c7656ec7ab88b098defb751b7401b5f6d8976g",
			wantErr: true,
		},
		{
			name:    "Empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Address with whitespace (expect trimmed)",
			input:   "  0x71c7656ec7ab88b098defb751b7401b5f6d8976f  ",
			wantErr: false,
			wantVal: "0x71c7656ec7ab88b098defb751b7401b5f6d8976f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewAddress(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.String() != tt.wantVal {
				t.Errorf("NewAddress() got = %v, want %v", got.String(), tt.wantVal)
			}
		})
	}
}
