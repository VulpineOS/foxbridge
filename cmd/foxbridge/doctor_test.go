package main

import "testing"

func TestRunDoctorCommand_JSON(t *testing.T) {
	if code := runDoctorCommand([]string{"--format", "json"}); code != 0 {
		t.Fatalf("runDoctorCommand returned %d, want 0", code)
	}
}

func TestRunDoctorCommand_InvalidFormat(t *testing.T) {
	if code := runDoctorCommand([]string{"--format", "yaml"}); code != 2 {
		t.Fatalf("runDoctorCommand returned %d, want 2", code)
	}
}
