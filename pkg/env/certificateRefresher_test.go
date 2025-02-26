package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"testing"
	"time"

	azsecretssdk "github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	mock_azsecrets "github.com/Azure/ARO-RP/pkg/util/mocks/azureclient/azuresdk/azsecrets"
	utilpem "github.com/Azure/ARO-RP/pkg/util/pem"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

const testCertBundle1 = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDwFPCd+9yh5Qyx
ctlSvtG6D/5Y1ClB33NSaJ2x7ZwW+xJH2NkWaR4eAP0vzrlV9zdfc5PxF0+skOEd
DuyhFQwgzGXWfoIFwaxtkvibWo8+qz1PDzxdWnlK2Qk9BbF44I3J0cCYE0+6NXLR
/8gmUVzt2Rk1xFez8dVvFEEJDK6OiVEeduMAeTfjmUcHBUqsHzPtDbxMsG4fhhuE
yWH58f9QKIk9Q9SPZ07jAnSKcVuWY+0Tox4797hPNTrn2/7c6BqsitOijUD/55RB
Jo3wpiLcyyIWxLgkMgIlg01nDAY0qhOjFPoPlUCKXdRr7BhazanGCArK0z2FzIWA
5S4CEDflAgMBAAECggEABX+lRykGk5qoYMQNoCyIpydInwY077JLdN66heG4Snpz
n7uitTWxH+TL57VnX0WrOf9uqv3qsDwdO8okt0fBIFsuFeyN083swhG0qfI4B6pq
XA4wRr8UuhcgdApWVztlY/Lu40zF7bDdsVuXXPFOHJB1WFrn21I1njariqaEtPT6
z5bWELFP5Syq9WfXY5ug4MhNXLMuoMQTLXtspQ0M0gCldJEb7dQzczgiODI/q+JU
uMROZ5xssAr8C880fapvaxo3dcBqt4W22ya7zhahduLgfG4FsMUjYOBvWhe+gBA0
YPGz9Qaej/pLjYQgiKUOSkR0d8tkT50AfZXD1d4kHQKBgQDx3rarPnAk+1B2HruI
Duxy8DG7hGCzd5PqlaSpQpDKqJJY6DaoZ+tuV0CkqqODNe+L+i69wk1JwFR8HOyr
ReILU0AvN1DxexRvIc4KTCRjC4xOTbrMOOtSMRchVIaGOnCN5ElfJxKEuoBYCLPi
Xw7QVkrI7lV1Jhc8QW6X+7EW2wKBgQD+G3u25ngsk+9scGxq34pVz3YohVyine8h
fUgsyCVdERRR5RKLrKng/GSdgTzl0+smwO7N+/9Q+QSL1SANr2zjmHVa6n02Hehm
S+4YvHsA4WPGFrlmaVgO80KwVNmm5Xey6p0PnNt9zKYi4SRrEHPRMd8+EQuqL2Nn
KkVlI4pIPwKBgQC06Kti3HnO/3bIUuZbtyXeNpBMPJCDy+4EKVeXDmX0Xy/Pdijj
v47V4kdEoylYS/BXl5J8dqeOgV/v0UaoOMYBSIyahFpztGatVPCivR7+QjX4n6UX
eX9x46v0Tx+rqGxlhRnoJPZx9nlm32OE7yrKY7DeJ34d+JaqiBprbWOgvwKBgHEV
7hLRsn20QIMz7SwK29eggmc6IqXEP53Z0XsMf4RRi4d+uKgsaVXVPTnTQDTQAQC4
MA6/rTpt+BX6/U7Z2U3YlbGmVZ715G1SMV4U03Dq3apUhqILE8NjgzRSLqLV0FVx
kABYwF3V68HuDHURV1msJjvK/jP47vYEm+mMzYelAoGAK2zS87DVUMbwiQOHveGQ
lhvC6HZsja/CEk2C3RTo+0IsEzc/O8/qyDZd6+rqCmJ9ViKHccvtYwRTzRrrDtP1
XQ9FR4VO2eH2YLAREzCgfykEzwBPFinM1kBhtpbZID4n8YizSODJTI9pasC1z/zL
x5MTZmC2cAtxmyx3pY/KFa8=
-----END PRIVATE KEY-----
-----BEGIN CERTIFICATE-----
MIIDLjCCAhagAwIBAgIQHkfoyE/KTKuEhOTumdk4+zANBgkqhkiG9w0BAQsFADAU
MRIwEAYDVQQDEwlwZXRyLnRlc3QwHhcNMjEwNTI2MDkzNjU5WhcNMjEwNjI2MDk0
NjU5WjAUMRIwEAYDVQQDEwlwZXRyLnRlc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IB
DwAwggEKAoIBAQDwFPCd+9yh5QyxctlSvtG6D/5Y1ClB33NSaJ2x7ZwW+xJH2NkW
aR4eAP0vzrlV9zdfc5PxF0+skOEdDuyhFQwgzGXWfoIFwaxtkvibWo8+qz1PDzxd
WnlK2Qk9BbF44I3J0cCYE0+6NXLR/8gmUVzt2Rk1xFez8dVvFEEJDK6OiVEeduMA
eTfjmUcHBUqsHzPtDbxMsG4fhhuEyWH58f9QKIk9Q9SPZ07jAnSKcVuWY+0Tox47
97hPNTrn2/7c6BqsitOijUD/55RBJo3wpiLcyyIWxLgkMgIlg01nDAY0qhOjFPoP
lUCKXdRr7BhazanGCArK0z2FzIWA5S4CEDflAgMBAAGjfDB6MA4GA1UdDwEB/wQE
AwIFoDAJBgNVHRMEAjAAMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAf
BgNVHSMEGDAWgBSiCXS05A51/N/eKXEBLiNi/W5ZwDAdBgNVHQ4EFgQUogl0tOQO
dfzf3ilxAS4jYv1uWcAwDQYJKoZIhvcNAQELBQADggEBADPwcFnUq8NYMlyZriF1
Yk3tTiLwlMrwHQViz143t/C+lMlQfVME515xxn1SUdEG0JAseCOGiIsqLVpwc042
cFkBgbCAkIMg3BIJoKMIFMEXdlcbQ9TlGY0QQPxvfT+L1giGaK6mcmBuIBi9iM9k
8ClJkAbkMgX3A68eWbI8PEaV/KyyHD/zHX/UmnyquXYxUZ9Cdazt3rG7vmt+NTw1
LlsPZY/5jnNhkjNt+qgruByc5/XNcVJE4VZcEUiDaAwhi5XDigIPx7eM9tF+yuIf
C+cV51VbGUPQBvJTr0LjGaUDbryrVOAwma5qNi4ekdcOvSrBBs4hflrn8up+HUoX
zB0=
-----END CERTIFICATE-----`

const testCertBundle2 = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDGwJS2bDR2ugfs
ODr1/VkhX9A2b6/w66r4habosbIeX9meMsKZbtsMOrWH2uten9H7L0o2rYbU5c3L
AtEaeEplGG4AamtF4s35axIF8/jgMvkswHjodb2f79iImOx96tE/Odhu3T/uoR1V
543Q1f80/HZzdpqnIpNZd/NrDMv2Hjm1LcLFMpcZFBpdF4CmAm6J59LjfZWm5qq5
aJf3MFEdxsX0VwEURmP0lrsvZTJhsu+httv+soFareiOk2W5nU0UhFSpKfOrFfld
cHxNPX6T2wr7JTYLDfgeRG6VoboPwcdVCCllrKWUlP9zdq+A6zbByHqC+QG6IP2c
zzXUAMR9AgMBAAECggEAOHAjSpH7a+NzsH5fL88bakC87VhVy8IAMMYzGUysWbe6
IhQj8lHqXdSmC8f8okgb5ooNNu2bpgUDpSxNmIikB4UiZ3fQsm2vM65V1d7rwy52
V2DodIpKqIoBIPjb3v25IY0ZipHFP8v8epJgUPcTm0Y9lJgPXnqRAQzw0Gs319FI
wxVy/RkwJ/p6BYufJs/T7HjCjCSsBA0JPx46qiCGA90u4gmYOwQFmZoEjlcMzPaL
Oinbfs2WN5mSazev/4zY/bEnFuO1vbpvx1qAEZpoDvN2CeYjAHvhmMe5zQwEFgsY
Bjhfd8pHuujbkIvHNo8mcRY8D7tFggw0QC8YZBl9+QKBgQDNcJQPQPaLUy8ZUxfL
8HMfpnVPAt9g0mctMo52CiF95WqMjXhJ5tbn7J2R6JFawCoIunNAWDwCkOg9dzgE
SwkGAQjSvHNlLOAIx1ej+vjeLvjyCgXqiv66bG16JK4ZA3K/Dewe/o27Tbpny0/7
AkfjrknhyRjK4pnnZ/kWj6wJtwKBgQD3qqrXHBJD78lo0HnZmNLeI8noM5ruJD23
9dfTEkr99/cpe6mdD2OJKNsBfUB4y/Ces7EHPa6UEPLl36vIHhg+5chrqrNbRfIv
LeCeRPkjVTxEuPCtUx7Bt1LqlM1tPfOp67V6160aroLnpIdUsWSOQFuAvGqHKLhG
Nau31SDzawKBgQC9PIYlxuFTVTx9R10ULljdPqewMCUzOpxvtbIkaRCQt1J+RZIY
ANrUp9A9Js09muUdRSIEk0Iz2ucSN08SJUwai7lk5NIm0D9N1tGT6wpzHzGRQkpQ
0dfyQQ5XBJKZ1+NKubhWlIRZlC+wjEcQH/m4cEL+CA8eU70Qu2VmstD14QKBgEmo
PWT6WUhRMUJ19jdL5zLfy/W+G07GAoEKoaSJpToBHEX/HEO0xvKM7w1zVdBXPvnE
EVtI8fnhTIwnSGyc3rMeHcw/mVYE6HE1oL8RXlMuz1zU7+dseBI+1m8j0DC0Ixqf
GnstV7M+wXnpCcKbe39/DnesEbae2qcu4SIsRb9/AoGBAJw6iTZm2kjk/AK3i/ZN
RpuP6/8jbHZNTvLz8Aw9J4wxoM8wp3kpJe0kMOgzWZPiya4QX7kCma+uoibiR3v/
ohPVpYqZtLFdfEd9pDXMZCg/q1sOcL94gNiv9w0QCbuvBhAno7OHk9JZMIr3pJ0h
tT1hverLghBfQ8KhO9Gk+FSM
-----END PRIVATE KEY-----
-----BEGIN CERTIFICATE-----
MIIDLjCCAhagAwIBAgIQD/cl3WNvT3ynxkUONemaPjANBgkqhkiG9w0BAQsFADAU
MRIwEAYDVQQDEwlwZXRyLnRlc3QwHhcNMjEwNTI2MDkzNDQzWhcNMjEwNjI2MDk0
NDQzWjAUMRIwEAYDVQQDEwlwZXRyLnRlc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IB
DwAwggEKAoIBAQDGwJS2bDR2ugfsODr1/VkhX9A2b6/w66r4habosbIeX9meMsKZ
btsMOrWH2uten9H7L0o2rYbU5c3LAtEaeEplGG4AamtF4s35axIF8/jgMvkswHjo
db2f79iImOx96tE/Odhu3T/uoR1V543Q1f80/HZzdpqnIpNZd/NrDMv2Hjm1LcLF
MpcZFBpdF4CmAm6J59LjfZWm5qq5aJf3MFEdxsX0VwEURmP0lrsvZTJhsu+httv+
soFareiOk2W5nU0UhFSpKfOrFfldcHxNPX6T2wr7JTYLDfgeRG6VoboPwcdVCCll
rKWUlP9zdq+A6zbByHqC+QG6IP2czzXUAMR9AgMBAAGjfDB6MA4GA1UdDwEB/wQE
AwIFoDAJBgNVHRMEAjAAMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAf
BgNVHSMEGDAWgBTVoGOzKcn+Y+vypnh2e5BzIcQ6NzAdBgNVHQ4EFgQU1aBjsynJ
/mPr8qZ4dnuQcyHEOjcwDQYJKoZIhvcNAQELBQADggEBAGREAWRpsAUlhLtnqgyh
qhRDd0xuzlBte5uMWYJxojwK7jmJlb4YbIgm6OfSMW3cd1xN497egji5AiiW37wN
UDONqYUL4qF3nnPJl9yjvptJCdpZQC1Ma5b2BLF8X8kurh8rYtyx2NNncj6/p/IA
D/R4qcuA0UaXLdARsppjLTCzk6T/Z8v4qaaXQJBfdHLY2fZo4vSI1yLyTcGia703
Zn30eMz0PLFCkMEYhZADUia0peReH08WO7RTtE5Nf8r5ZiFSfuJJlBmURMU3MAxJ
PeihXgcD511I3eMq7A8r7+xlNFEFYaleYwmoYSOt0cBGtoZttXDYacR/xvEkVaxO
1LQ=
-----END CERTIFICATE-----`

func TestRefreshingCertificate(t *testing.T) {
	mockController := gomock.NewController(t)

	key1, certs1, err := utilpem.Parse([]byte(testCertBundle1))
	if err != nil {
		return
	}

	key2, certs2, err := utilpem.Parse([]byte(testCertBundle2))
	if err != nil {
		return
	}

	testCertName := "rotation-test-0"

	cannotPull := errors.New("Cannot pull certificates from keyvault")

	// newMockTicker inline function is used to mock time.Ticker to pull the certificate
	// return two functions:
	//
	// mockTicker := func() (tick <-chan time.Time, stop func()) returns chanel to pass ticks and stop signal
	// which is roughly equivalent to:
	//         ticker := time.NewTicker(time.Minute)
	//         ticker.Stop() // <- this
	// when stop() is called, done is passed to signal end of processing
	//
	// func(context.CancelFunc) is used to generate ticks, passed func() is used as cancel signal
	// for context.WithCancel to signal all ticks are passed
	//
	// Call order is depicted in example for two ticks bellow
	//
	//        context       mockSource        mockTicker             fetchCertificate
	//    --------------------------------------------------------------------------
	//      withCancel
	//                         1       ->       tick        ->    fetchCertificateonce
	//                         2       ->       tick        ->    fetchCertificateonce
	//                   <-  cancel
	//        Done                                          ->        stop
	//                        done     <-        stop       <-
	newMockTicker := func(n int) (func() (<-chan time.Time, func()), func(context.CancelFunc)) {
		s := make(chan time.Time)
		done := make(chan struct{})

		mockTicker := func() (tick <-chan time.Time, stop func()) {
			return s, func() {
				done <- struct{}{}
			}
		}

		mockSource := func(cancel context.CancelFunc) {
			for i := 0; i < n; i++ {
				s <- time.Time{}
			}
			cancel()
			<-done
		}

		return mockTicker, mockSource
	}

	tt := []struct {
		name           string
		tickCount      int
		managerFactory func(*gomock.Controller) azsecrets.Client
		wantKey        *rsa.PrivateKey
		wantCert       *x509.Certificate
		wantErr        error
	}{
		{
			name: "test initial certificate, pull exactly once, ticks one time",
			managerFactory: func(controller *gomock.Controller) azsecrets.Client {
				manager := mock_azsecrets.NewMockClient(controller)
				manager.EXPECT().GetSecret(gomock.Any(), testCertName, "", nil).Return(azsecretssdk.GetSecretResponse{Secret: azsecretssdk.Secret{Value: pointerutils.ToPtr(testCertBundle1)}}, nil)
				return manager
			},
			tickCount: 0,
			wantKey:   key1,
			wantCert:  certs1[0],
			wantErr:   nil,
		},
		{
			name: "test refresh certificate, pull exactly twice, first on start, second on refresh, one tick",
			managerFactory: func(controller *gomock.Controller) azsecrets.Client {
				manager := mock_azsecrets.NewMockClient(mockController)
				gomock.InOrder(
					manager.EXPECT().GetSecret(gomock.Any(), testCertName, "", nil).Return(azsecretssdk.GetSecretResponse{Secret: azsecretssdk.Secret{Value: pointerutils.ToPtr(testCertBundle1)}}, nil),
					manager.EXPECT().GetSecret(gomock.Any(), testCertName, "", nil).Return(azsecretssdk.GetSecretResponse{Secret: azsecretssdk.Secret{Value: pointerutils.ToPtr(testCertBundle2)}}, nil),
				)
				return manager
			},
			tickCount: 1,
			wantKey:   key2,
			wantCert:  certs2[0],
			wantErr:   nil,
		},
		{
			name: "test initial error, pull exactly once with an error, no tick",
			managerFactory: func(controller *gomock.Controller) azsecrets.Client {
				manager := mock_azsecrets.NewMockClient(mockController)
				manager.EXPECT().GetSecret(gomock.Any(), testCertName, "", nil).Return(azsecretssdk.GetSecretResponse{}, cannotPull)
				return manager
			},
			tickCount: 0,
			wantKey:   nil,
			wantCert:  nil,
			wantErr:   cannotPull,
		},
		{
			name: "test refresh error, pull exactly twice, first on start, second time with an error, one tick",
			managerFactory: func(controller *gomock.Controller) azsecrets.Client {
				manager := mock_azsecrets.NewMockClient(controller)
				gomock.InOrder(
					manager.EXPECT().GetSecret(gomock.Any(), testCertName, "", nil).Return(azsecretssdk.GetSecretResponse{Secret: azsecretssdk.Secret{Value: pointerutils.ToPtr(testCertBundle1)}}, nil),
					manager.EXPECT().GetSecret(gomock.Any(), testCertName, "", nil).Return(azsecretssdk.GetSecretResponse{}, cannotPull),
				)
				return manager
			},
			tickCount: 1,
			wantKey:   key1,
			wantCert:  certs1[0],
			wantErr:   nil,
		},
		{
			name: "test refresh, pull exactly 5 times, 4 ticks",
			managerFactory: func(controller *gomock.Controller) azsecrets.Client {
				manager := mock_azsecrets.NewMockClient(controller)
				manager.EXPECT().GetSecret(gomock.Any(), testCertName, "", nil).Return(azsecretssdk.GetSecretResponse{Secret: azsecretssdk.Secret{Value: pointerutils.ToPtr(testCertBundle1)}}, nil).Times(5)
				return manager
			},
			tickCount: 4,
			wantKey:   key1,
			wantCert:  certs1[0],
			wantErr:   nil,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mock, tick := newMockTicker(test.tickCount)

			refreshing := newCertificateRefresher(
				logrus.NewEntry(logrus.StandardLogger()),
				// interval is not used in tests, it is mocked
				0,
				test.managerFactory(mockController),
				testCertName,
			)
			refreshing.(*refreshingCertificate).newTicker = mock

			err := refreshing.Start(ctx)
			if err != test.wantErr {
				t.Fatal(err)
			}

			if err != nil {
				return
			}

			// call tick to do all registered ticks, once finished, cancel context and wait for done channel
			// canceled context finishes the fetchCertificate goroutine, this triggers registered stop()
			// which sends done, which finally tells tick to end and allow code to continue
			tick(cancel)

			testkey, testCerts := refreshing.GetCertificates()
			if !testkey.Equal(test.wantKey) {
				t.Error("returned private key does not match")
			}

			if !testCerts[0].Equal(test.wantCert) {
				t.Error("returned certificate does not match")
			}
		})
	}
}
