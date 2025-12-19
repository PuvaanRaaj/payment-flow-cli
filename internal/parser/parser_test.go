package parser

import (
	"testing"
)

func TestParse_ValidCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantArgs []string
	}{
		{
			name:     "CREATE with all required args",
			input:    "CREATE P1001 10.00 MYR M01",
			wantName: "CREATE",
			wantArgs: []string{"P1001", "10.00", "MYR", "M01"},
		},
		{
			name:     "CREATE with inline comment after required args",
			input:    "CREATE P1001 10.00 MYR M01 # test comment",
			wantName: "CREATE",
			wantArgs: []string{"P1001", "10.00", "MYR", "M01"},
		},
		{
			name:     "AUTHORIZE with inline comment",
			input:    "AUTHORIZE P1001 # retry",
			wantName: "AUTHORIZE",
			wantArgs: []string{"P1001"},
		},
		{
			name:     "AUTHORIZE simple",
			input:    "AUTHORIZE P1001",
			wantName: "AUTHORIZE",
			wantArgs: []string{"P1001"},
		},
		{
			name:     "CAPTURE simple",
			input:    "CAPTURE P1001",
			wantName: "CAPTURE",
			wantArgs: []string{"P1001"},
		},
		{
			name:     "VOID without reason",
			input:    "VOID P1001",
			wantName: "VOID",
			wantArgs: []string{"P1001"},
		},
		{
			name:     "VOID with reason",
			input:    "VOID P1001 CUSTOMER_REQUEST",
			wantName: "VOID",
			wantArgs: []string{"P1001", "CUSTOMER_REQUEST"},
		},
		{
			name:     "VOID with reason and comment",
			input:    "VOID P1001 FRAUD # suspicious activity",
			wantName: "VOID",
			wantArgs: []string{"P1001", "FRAUD"},
		},
		{
			name:     "REFUND without amount",
			input:    "REFUND P1001",
			wantName: "REFUND",
			wantArgs: []string{"P1001"},
		},
		{
			name:     "REFUND with amount",
			input:    "REFUND P1001 5.00",
			wantName: "REFUND",
			wantArgs: []string{"P1001", "5.00"},
		},
		{
			name:     "SETTLE simple",
			input:    "SETTLE P1001",
			wantName: "SETTLE",
			wantArgs: []string{"P1001"},
		},
		{
			name:     "SETTLEMENT batch",
			input:    "SETTLEMENT BATCH001",
			wantName: "SETTLEMENT",
			wantArgs: []string{"BATCH001"},
		},
		{
			name:     "STATUS simple",
			input:    "STATUS P1001",
			wantName: "STATUS",
			wantArgs: []string{"P1001"},
		},
		{
			name:     "LIST",
			input:    "LIST",
			wantName: "LIST",
			wantArgs: []string{},
		},
		{
			name:     "LIST with comment",
			input:    "LIST # show all payments",
			wantName: "LIST",
			wantArgs: []string{},
		},
		{
			name:     "AUDIT",
			input:    "AUDIT P1001",
			wantName: "AUDIT",
			wantArgs: []string{"P1001"},
		},
		{
			name:     "EXIT",
			input:    "EXIT",
			wantName: "EXIT",
			wantArgs: []string{},
		},
		{
			name:     "EXIT with comment",
			input:    "EXIT # goodbye",
			wantName: "EXIT",
			wantArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := Parse(tt.input)
			if err != nil {
				t.Errorf("Parse() error = %v, want nil", err)
				return
			}
			if cmd.Name != tt.wantName {
				t.Errorf("Parse() Name = %v, want %v", cmd.Name, tt.wantName)
			}
			if len(cmd.Args) != len(tt.wantArgs) {
				t.Errorf("Parse() Args length = %v, want %v", len(cmd.Args), len(tt.wantArgs))
				return
			}
			for i, arg := range cmd.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("Parse() Args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestParse_MalformedInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "hash at beginning of line is NOT a comment",
			input: "# CREATE P1002 11.00 MYR M01",
		},
		{
			name:  "hash as second token before required args met",
			input: "CREATE # P1003 10.00 MYR M01",
		},
		{
			name:  "hash interrupting required args",
			input: "CREATE P1001 # 10.00 MYR M01",
		},
		{
			name:  "insufficient args for CREATE",
			input: "CREATE P1001 10.00",
		},
		{
			name:  "insufficient args for AUTHORIZE",
			input: "AUTHORIZE",
		},
		{
			name:  "unknown command",
			input: "UNKNOWN P1001",
		},
		{
			name:  "empty input",
			input: "",
		},
		{
			name:  "whitespace only",
			input: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Errorf("Parse(%q) expected error, got nil", tt.input)
			}
		})
	}
}

func TestParse_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantArgs []string
		wantErr  bool
	}{
		{
			name:     "extra whitespace",
			input:    "  CREATE   P1001   10.00   MYR   M01  ",
			wantName: "CREATE",
			wantArgs: []string{"P1001", "10.00", "MYR", "M01"},
			wantErr:  false,
		},
		{
			name:     "comment with hash symbol repeated",
			input:    "CREATE P1001 10.00 MYR M01 ## double hash comment",
			wantName: "CREATE",
			wantArgs: []string{"P1001", "10.00", "MYR", "M01"},
			wantErr:  false,
		},
		{
			name:    "amount with hash in wrong position",
			input:   "CREATE P1001 10.00 # MYR M01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if cmd.Name != tt.wantName {
				t.Errorf("Parse() Name = %v, want %v", cmd.Name, tt.wantName)
			}
			if len(cmd.Args) != len(tt.wantArgs) {
				t.Errorf("Parse() Args length = %v, want %v", len(cmd.Args), len(tt.wantArgs))
				return
			}
			for i, arg := range cmd.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("Parse() Args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestIsValidCommand(t *testing.T) {
	validCommands := []string{"CREATE", "AUTHORIZE", "CAPTURE", "VOID", "REFUND", "SETTLE", "SETTLEMENT", "STATUS", "LIST", "AUDIT", "EXIT"}
	for _, cmd := range validCommands {
		if !IsValidCommand(cmd) {
			t.Errorf("IsValidCommand(%s) = false, want true", cmd)
		}
	}

	invalidCommands := []string{"create", "INVALID", "DELETE", ""}
	for _, cmd := range invalidCommands {
		if IsValidCommand(cmd) {
			t.Errorf("IsValidCommand(%s) = true, want false", cmd)
		}
	}
}

func TestGetRequiredArgCount(t *testing.T) {
	tests := []struct {
		cmd   string
		count int
		ok    bool
	}{
		{"CREATE", 4, true},
		{"AUTHORIZE", 1, true},
		{"LIST", 0, true},
		{"EXIT", 0, true},
		{"UNKNOWN", 0, false},
	}

	for _, tt := range tests {
		count, ok := GetRequiredArgCount(tt.cmd)
		if ok != tt.ok || count != tt.count {
			t.Errorf("GetRequiredArgCount(%s) = (%d, %v), want (%d, %v)", tt.cmd, count, ok, tt.count, tt.ok)
		}
	}
}
