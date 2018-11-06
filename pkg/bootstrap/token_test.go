package bootstrap

import (
	"testing"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	expiration = time.Date(2018, time.October, 10, 23, 0, 0, 0, time.UTC)
)

func TestGenerateTokenSecret(t *testing.T) {
	_ = &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{},
	}
	testcases := []struct {
		name   string
		secret string

		wantSecret *v1.Secret
		wantErr    func(err error) (string, bool)
	}{
		{
			name:   "testing valid token name",
			secret: "50ydlk.7up8oiki8zp3qoyh",

			wantSecret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bootstrap-token-50ydlk",
				},
			},
			wantErr: func(err error) (string, bool) {
				return "err == nil", err == nil
			},
		},
		{
			name:   "testing invalid token name",
			secret: "fooo",

			wantSecret: nil,
			wantErr: func(err error) (string, bool) {
				return "err != nil", err != nil
			},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			secret, err := GenerateTokenSecret(testcase.secret, expiration)
			if desc, ok := testcase.wantErr(err); !ok {
				t.Errorf("expected %s, got %v", desc, err)
			}

			if testcase.wantSecret == nil && secret != nil {
				t.Errorf("expected nil secret, got %v", secret)
			}
			if testcase.wantSecret != nil && secret == nil {
				t.Errorf("expected %v, got nil", testcase.wantSecret)
			}
			if testcase.wantSecret != nil && secret != nil {
				if want, got := testcase.wantSecret.ObjectMeta.Name, secret.ObjectMeta.Name; want != got {
					t.Errorf("expected %v, got %v", want, got)
				}
			}
		})
	}
}
