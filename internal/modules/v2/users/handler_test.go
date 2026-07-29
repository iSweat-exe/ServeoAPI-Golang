package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"serveoapi/internal/modules/v2/auth"
	"serveoapi/internal/testutil"
)

func TestGetMe_Success(t *testing.T) {
	// Setup DB
	db, err := testutil.SetupTestDB(&auth.User{})
	require.NoError(t, err)

	// Create a test user
	user := auth.User{
		Username: "testuser",
	}
	require.NoError(t, db.Create(&user).Error)

	// Create handler
	h := &Handler{DB: db}

	// Create request
	req := testutil.NewAuthenticatedRequest("GET", "/v2/users/me", nil, user.ID, nil)
	rr := httptest.NewRecorder()

	// Call handler
	h.GetMe(rr, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rr.Code)

	var response UserResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "testuser", response.Username)
}

func TestUpdateMePassword_Success(t *testing.T) {
	db, err := testutil.SetupTestDB(&auth.User{})
	require.NoError(t, err)

	user := auth.User{
		Username:     "testuser",
		TokenVersion: 1,
	}
	user.HashPassword("oldpassword123")
	require.NoError(t, db.Create(&user).Error)

	h := &Handler{DB: db}

	// Body
	reqBody := UpdatePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "newpassword123",
	}
	body, _ := json.Marshal(reqBody)

	req := testutil.NewAuthenticatedRequest("PUT", "/v2/users/me/password", bytes.NewReader(body), user.ID, nil)
	rr := httptest.NewRecorder()

	h.UpdateMePassword(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Check if token version was incremented
	var updatedUser auth.User
	db.First(&updatedUser, user.ID)
	assert.Equal(t, 2, updatedUser.TokenVersion)
	
	// Check if password was actually changed
	assert.NoError(t, updatedUser.CheckPassword("newpassword123"))
}

func TestUpdateMePassword_InvalidInput(t *testing.T) {
	db, err := testutil.SetupTestDB(&auth.User{})
	require.NoError(t, err)

	user := auth.User{
		Username:     "testuser",
		TokenVersion: 1,
	}
	user.HashPassword("oldpassword123")
	require.NoError(t, db.Create(&user).Error)

	h := &Handler{DB: db}

	// Body with missing new password
	reqBody := UpdatePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "short", // less than 8 chars
	}
	body, _ := json.Marshal(reqBody)

	req := testutil.NewAuthenticatedRequest("PUT", "/v2/users/me/password", bytes.NewReader(body), user.ID, nil)
	rr := httptest.NewRecorder()

	h.UpdateMePassword(rr, req)

	// Should fail validation (from go-playground/validator)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Check that token version was NOT incremented
	var updatedUser auth.User
	db.First(&updatedUser, user.ID)
	assert.Equal(t, 1, updatedUser.TokenVersion)
}
