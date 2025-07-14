package jwtprovider_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	jwtprovider "github.com/parta4ok/kvs/auth/internal/adapter/jwt_provider"
	"github.com/parta4ok/kvs/auth/internal/entities"
)

func TestNewProvider_Success(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret")
	aud := []string{"test-audience"}
	iss := "test-issuer"
	ttl := time.Hour

	provider, err := jwtprovider.NewProvider(secret, aud, iss, ttl)

	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestNewProvider_ValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		secret      []byte
		aud         []string
		iss         string
		ttl         time.Duration
		expectedErr string
	}{
		{
			name:        "empty_secret",
			secret:      []byte{},
			aud:         []string{"test-audience"},
			iss:         "test-issuer",
			ttl:         time.Hour,
			expectedErr: "secret not set",
		},
		{
			name:        "nil_secret",
			secret:      nil,
			aud:         []string{"test-audience"},
			iss:         "test-issuer",
			ttl:         time.Hour,
			expectedErr: "secret not set",
		},
		{
			name:        "empty_issuer",
			secret:      []byte("test-secret"),
			aud:         []string{"test-audience"},
			iss:         "",
			ttl:         time.Hour,
			expectedErr: "iss not set",
		},
		{
			name:        "zero_ttl",
			secret:      []byte("test-secret"),
			aud:         []string{"test-audience"},
			iss:         "test-issuer",
			ttl:         time.Duration(0),
			expectedErr: "jwt ttl not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider, err := jwtprovider.NewProvider(tt.secret, tt.aud, tt.iss, tt.ttl)

			require.Error(t, err)
			require.Nil(t, provider)
			require.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestProvider_Generate_Success(t *testing.T) {
	t.Parallel()

	provider, err := jwtprovider.NewProvider(
		[]byte("test-secret"),
		[]string{"test-audience"},
		"test-issuer",
		time.Hour,
	)
	require.NoError(t, err)

	user := &entities.User{
		ID:       "123",
		Username: "testuser",
		Rights:   []string{"read", "write"},
	}

	token, err := provider.Generate(user)

	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Contains(t, token, ".")
}

func TestProvider_Generate_NilUser(t *testing.T) {
	t.Parallel()

	provider, err := jwtprovider.NewProvider(
		[]byte("test-secret"),
		[]string{"test-audience"},
		"test-issuer",
		time.Hour,
	)
	require.NoError(t, err)

	token, err := provider.Generate(nil)

	require.Error(t, err)
	require.Empty(t, token)
}

func TestProvider_Introspect_Success(t *testing.T) {
	t.Skip()
	t.Parallel()

	provider, err := jwtprovider.NewProvider(
		[]byte("test-secret"),
		[]string{"test-audience"},
		"test-issuer",
		time.Hour,
	)
	require.NoError(t, err)

	originalUser := &entities.User{
		ID:       "123",
		Username: "testuser",
		Rights:   []string{"read", "write"},
	}

	// Генерируем токен
	token, err := provider.Generate(originalUser)
	require.NoError(t, err)

	// Извлекаем claims
	claims, err := provider.Introspect(token)

	require.NoError(t, err)
	require.NotNil(t, claims)
	require.Equal(t, originalUser.ID, claims.Subject)
	require.Equal(t, originalUser.Username, claims.Username)
	require.Equal(t, originalUser.Rights, claims.Rights)
	require.Equal(t, "test-issuer", claims.Issuer)
	require.Equal(t, []string{"test-audience"}, claims.Audience)
}

func TestProvider_Introspect_InvalidToken(t *testing.T) {
	t.Parallel()

	provider, err := jwtprovider.NewProvider(
		[]byte("test-secret"),
		[]string{"test-audience"},
		"test-issuer",
		time.Hour,
	)
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectedErr string
	}{
		{
			name:        "empty_token",
			token:       "",
			expectedErr: "jwt parse failure",
		},
		{
			name:        "invalid_format",
			token:       "invalid.token.format",
			expectedErr: "jwt parse failure",
		},
		{
			name:        "malformed_token",
			token:       "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
			expectedErr: "jwt parse failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			claims, err := provider.Introspect(tt.token)

			require.Error(t, err)
			require.Nil(t, claims)
			require.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestProvider_Introspect_WrongSecret(t *testing.T) {
	t.Parallel()

	// Создаем провайдер с одним секретом
	provider1, err := jwtprovider.NewProvider(
		[]byte("secret-1"),
		[]string{"test-audience"},
		"test-issuer",
		time.Hour,
	)
	require.NoError(t, err)

	// Создаем провайдер с другим секретом
	provider2, err := jwtprovider.NewProvider(
		[]byte("secret-2"),
		[]string{"test-audience"},
		"test-issuer",
		time.Hour,
	)
	require.NoError(t, err)

	user := &entities.User{
		ID:       "123",
		Username: "testuser",
		Rights:   []string{"read"},
	}

	// Генерируем токен первым провайдером
	token, err := provider1.Generate(user)
	require.NoError(t, err)

	// Пытаемся проверить вторым провайдером
	claims, err := provider2.Introspect(token)

	require.Error(t, err)
	require.Nil(t, claims)
	require.Contains(t, err.Error(), "jwt parse failure")
}

func TestProvider_Introspect_ExpiredToken(t *testing.T) {
	t.Parallel()

	// Создаем провайдер с очень коротким TTL
	provider, err := jwtprovider.NewProvider(
		[]byte("test-secret"),
		[]string{"test-audience"},
		"test-issuer",
		time.Millisecond*10, // 10ms TTL
	)
	require.NoError(t, err)

	user := &entities.User{
		ID:       "123",
		Username: "testuser",
		Rights:   []string{"read"},
	}

	// Генерируем токен
	token, err := provider.Generate(user)
	require.NoError(t, err)

	// Ждем истечения токена
	time.Sleep(time.Millisecond * 20)

	// Пытаемся проверить истекший токен
	claims, err := provider.Introspect(token)

	require.Error(t, err)
	require.Nil(t, claims)
	require.Contains(t, err.Error(), "jwt parse failure")
}

func TestProvider_GenerateAndIntrospect_RoundTrip(t *testing.T) {
	t.Skip()
	t.Parallel()

	tests := []struct {
		name string
		user *entities.User
	}{
		{
			name: "user_with_multiple_rights",
			user: &entities.User{
				ID:       "456",
				Username: "admin",
				Rights:   []string{"read", "write", "delete", "admin"},
			},
		},
		{
			name: "user_with_single_right",
			user: &entities.User{
				ID:       "789",
				Username: "reader",
				Rights:   []string{"read"},
			},
		},
		{
			name: "user_with_no_rights",
			user: &entities.User{
				ID:       "999",
				Username: "guest",
				Rights:   []string{},
			},
		},
		{
			name: "user_with_special_characters",
			user: &entities.User{
				ID:       "111",
				Username: "user@example.com",
				Rights:   []string{"read", "write"},
			},
		},
	}

	provider, err := jwtprovider.NewProvider(
		[]byte("test-secret"),
		[]string{"test-audience"},
		"test-issuer",
		time.Hour,
	)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Generate token
			token, err := provider.Generate(tt.user)
			require.NoError(t, err)
			require.NotEmpty(t, token)

			// Introspect token
			claims, err := provider.Introspect(token)
			require.NoError(t, err)
			require.NotNil(t, claims)

			// Verify claims
			require.Equal(t, tt.user.ID, claims.Subject)
			require.Equal(t, tt.user.Username, claims.Username)
			require.Equal(t, tt.user.Rights, claims.Rights)
			require.Equal(t, "test-issuer", claims.Issuer)
			require.Equal(t, []string{"test-audience"}, claims.Audience)
		})
	}
}
