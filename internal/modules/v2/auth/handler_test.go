package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"serveoapi/internal/testutil"
)

func TestLogin_Success(t *testing.T) {
	db, err := testutil.SetupTestDB(&User{})
	require.NoError(t, err)

	user := User{
		Username: "admin",
	}
	require.NoError(t, user.HashPassword("password123"))
	require.NoError(t, db.Create(&user).Error)

	h := &Handler{DB: db}

	reqBody := LoginRequest{
		Username: "admin",
		Password: "password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v2/auth/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response LoginResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Token)

	// Verify that user status is updated
	var updatedUser User
	db.First(&updatedUser, user.ID)
	assert.Equal(t, "online", updatedUser.Status)
	assert.NotNil(t, updatedUser.LastConnection)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	db, err := testutil.SetupTestDB(&User{})
	require.NoError(t, err)

	user := User{
		Username: "admin",
	}
	require.NoError(t, user.HashPassword("password123"))
	require.NoError(t, db.Create(&user).Error)

	h := &Handler{DB: db}

	reqBody := LoginRequest{
		Username: "admin",
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/v2/auth/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestLogout_Success(t *testing.T) {
	db, err := testutil.SetupTestDB(&User{})
	require.NoError(t, err)

	user := User{
		Username: "admin",
		Status:   "online",
	}
	require.NoError(t, db.Create(&user).Error)

	h := &Handler{DB: db}

	req := testutil.NewAuthenticatedRequest("POST", "/v2/auth/logout", nil, user.ID, nil)
	rr := httptest.NewRecorder()

	h.Logout(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Verify that user status is updated to offline
	var updatedUser User
	db.First(&updatedUser, user.ID)
	assert.Equal(t, "offline", updatedUser.Status)
}
