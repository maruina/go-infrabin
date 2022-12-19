package aws

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
		expectedId  string
		expectedErr bool
	}{
		{
			name:        "missing role",
			role:        "",
			session:     fakeSession,
			expectedId:  "",
			expectedErr: true,
		},
		{
			name:        "invalid role",
			role:        "my_invalid_role",
			session:     fakeSession,
			expectedId:  "",
			expectedErr: true,
		},
		{
			name:        "missing session",
			role:        fakeArn,
			session:     "",
			expectedId:  "",
			expectedErr: true,
		},
		{
			name:        "valid role",
			role:        fakeArn,
			session:     fakeSession,
			expectedId:  fakeAssumedRoleId,
			expectedErr: false,
		},
	}

	for _, tt := range testCases {
		ctx := context.Background()
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			gotRoleId, gotErr := STSAssumeRole(ctx, FakeSTSClient{}, tt.role, tt.session)
			if (gotErr != nil) != tt.expectedErr {
				t.Errorf("AssumeRole error = %v, expectedErr = %v", gotErr, tt.expectedErr)
			}
			g.Expect(gotRoleId).To(Equal(tt.expectedId))
		})
	}
}
