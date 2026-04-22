package repomap

import "testing"

func TestTestSortBase_DoesNotTreatContestAsTest(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "contest.java", want: "contest.java"},
		{name: "Contest.kt", want: "contest.kt"},
		{name: "MyContest.cs", want: "mycontest.cs"},
		{name: "Contest.php", want: "contest.php"},
		{name: "UserServiceTest.java", want: "userservice.java"},
		{name: "UserServiceTests.kt", want: "userservice.kt"},
		{name: "user_service_test.php", want: "user_service.php"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testSortBase(tt.name); got != tt.want {
				t.Fatalf("testSortBase(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
