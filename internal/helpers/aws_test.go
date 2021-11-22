package helpers

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
)

func TestAssumeRole(t *testing.T) {
	testCases := []struct {
		name        string
		role        string
		session     string
		expectedArn string
		expectErr   bool
	}{
		{
			name:        "missing role",
			role:        "",
			session:     "my_session",
			expectedArn: "",
			expectErr:   true,
		},
		{
			name:        "invalid role",
			role:        "my_invalid_role",
			session:     "my_session",
			expectedArn: "",
			expectErr:   true,
		},
		{
			name:        "missing session",
			role:        "arn:aws:iam::123456789012:user/Development/product_1234",
			session:     "",
			expectedArn: "",
			expectErr:   true,
		},
	}

	for _, tt := range testCases {
		ctx := context.Background()
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			gotArn, gotErr := AssumeRole(ctx, tt.role, tt.session)
			g.Expect(gotArn).To(Equal(tt.expectedArn))
			if (gotErr != nil) != tt.expectErr {
				t.Errorf("AssumeRole error = %v, expectedErr = %v", gotErr, tt.expectErr)
			}

		})
	}

}
