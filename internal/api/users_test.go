package api

import "testing"

func TestUsernameValidationAllowsDots(t *testing.T) {
	valid := []string{"wajahat.ali", "user_name", "user-name", "abc"}
	for _, username := range valid {
		if !usernameRe.MatchString(username) {
			t.Errorf("expected %q to be valid", username)
		}
	}

	invalid := []string{"ab", "user name", "user@example"}
	for _, username := range invalid {
		if usernameRe.MatchString(username) {
			t.Errorf("expected %q to be invalid", username)
		}
	}
}
