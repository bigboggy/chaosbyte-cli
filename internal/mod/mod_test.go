package mod

import "testing"

// TestQuestionTagDetectsTrailingQuestion asserts the simplest case: a
// normal message that ends in '?' gets a tag with the moderator's marker.
func TestQuestionTagDetectsTrailingQuestion(t *testing.T) {
	tag := QuestionTag("anyone know how to stream tokens from a worker into a TUI?")
	if tag == nil {
		t.Fatal("expected a tag for a question, got nil")
	}
	if tag.Marker != '✦' {
		t.Errorf("Marker = %q, want '✦'", tag.Marker)
	}
	if tag.Kind != "question" {
		t.Errorf("Kind = %q, want 'question'", tag.Kind)
	}
}

// TestQuestionTagIgnoresStatements asserts that messages without a '?'
// are not tagged. The moderator's attention is reserved for actual asks.
func TestQuestionTagIgnoresStatements(t *testing.T) {
	if tag := QuestionTag("the layout system alone"); tag != nil {
		t.Errorf("expected nil for statement, got %+v", tag)
	}
}

// TestQuestionTagIgnoresEmpty asserts whitespace-only messages are skipped.
func TestQuestionTagIgnoresEmpty(t *testing.T) {
	if tag := QuestionTag("   "); tag != nil {
		t.Error("expected nil for whitespace, got a tag")
	}
}

// TestQuestionTagTrimsTrailingWhitespace asserts the rule sees through
// trailing spaces. People type messages that end with a space sometimes.
func TestQuestionTagTrimsTrailingWhitespace(t *testing.T) {
	if tag := QuestionTag("zellij or tmux?   "); tag == nil {
		t.Error("expected a tag despite trailing whitespace")
	}
}
