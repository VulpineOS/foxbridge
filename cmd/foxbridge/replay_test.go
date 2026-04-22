package main

import "testing"

func TestRunReplayCommand_RequiresRecording(t *testing.T) {
	if code := runReplayCommand(nil); code != 2 {
		t.Fatalf("runReplayCommand returned %d, want 2", code)
	}
}

func TestRunReplayCommand_MissingRecordingFile(t *testing.T) {
	if code := runReplayCommand([]string{"--recording", "/tmp/does-not-exist.jsonl"}); code != 1 {
		t.Fatalf("runReplayCommand returned %d, want 1", code)
	}
}
