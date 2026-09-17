package rsa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRSAKey_Lifecycle(t *testing.T) {
	testCases := []struct {
		name string
		bits int
	}{
		{
			name: "默认 2048 位生成与 PEM 编解码闭环",
			bits: 2048,
		},
		{
			name: "非标参数缺省生成 2048 位",
			bits: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			priv, err := GenerateRSAKey(tc.bits)
			require.NoError(t, err)
			require.NotNil(t, priv)

			// 私钥 PEM
			privPEM := EncodePrivateKeyPEM(priv)
			require.NotEmpty(t, privPEM)

			parsedPriv, err := ParsePrivateKeyPEM(privPEM)
			require.NoError(t, err)
			assert.Equal(t, priv.N, parsedPriv.N)
			assert.Equal(t, priv.D, parsedPriv.D)

			// 公钥 PEM
			pubPEM, err := EncodePublicKeyPEM(&priv.PublicKey)
			require.NoError(t, err)
			require.NotEmpty(t, pubPEM)

			parsedPub, err := ParsePublicKeyPEM(pubPEM)
			require.NoError(t, err)
			assert.Equal(t, priv.PublicKey.N, parsedPub.N)
			assert.Equal(t, priv.PublicKey.E, parsedPub.E)
		})
	}
}

func TestParseRSA_InvalidPEM(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr string
	}{
		{
			name:      "空输入解析私钥",
			input:     "",
			expectErr: "无效的 PEM 格式",
		},
		{
			name:      "非法格式输入解析私钥",
			input:     "invalid pem content",
			expectErr: "无效的 PEM 格式",
		},
		{
			name:      "空输入解析公钥",
			input:     "",
			expectErr: "无效的 PEM 格式",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, errPriv := ParsePrivateKeyPEM(tc.input)
			assert.ErrorContains(t, errPriv, tc.expectErr)

			_, errPub := ParsePublicKeyPEM(tc.input)
			assert.ErrorContains(t, errPub, tc.expectErr)
		})
	}
}
