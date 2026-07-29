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

func TestGetMeSuccess(t *testing.T) {
	db, err := testutil.SetupTestDB(&auth.User{})
	require.NoError(t, err)

	user := auth.User{
		Username: "testuser",
	}
	require.NoError(t, db.Create(&user).Error)

	h := &Handler{DB: db}

	req := testutil.NewAuthenticatedRequest("GET", "/v2/users/me", nil, user.ID, nil)
	rr := httptest.NewRecorder()

	h.GetMe(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response UserResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "testuser", response.Username)
}

func TestUpdateMePasswordSuccess(t *testing.T) {
	db, err := testutil.SetupTestDB(&auth.User{})
	require.NoError(t, err)

	user := auth.User{
		Username:     "testuser",
		TokenVersion: 1,
	}
	user.HashPassword("oldpassword123")
	require.NoError(t, db.Create(&user).Error)

	h := &Handler{DB: db}

	reqBody := UpdatePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "newpassword123",
	}
	body, _ := json.Marshal(reqBody)

	req := testutil.NewAuthenticatedRequest("PUT", "/v2/users/me/password", bytes.NewReader(body), user.ID, nil)
	rr := httptest.NewRecorder()

	h.UpdateMePassword(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)

	// Vérifier si la version du token a été incrémentée
	var updatedUser auth.User
	db.First(&updatedUser, user.ID)
	assert.Equal(t, 2, updatedUser.TokenVersion)
	
	// Vérifier si le mot de passe a bien été changé
	assert.NoError(t, updatedUser.CheckPassword("newpassword123"))
}

func TestUpdateMePasswordInvalidInput(t *testing.T) {
	db, err := testutil.SetupTestDB(&auth.User{})
	require.NoError(t, err)

	user := auth.User{
		Username:     "testuser",
		TokenVersion: 1,
	}
	user.HashPassword("oldpassword123")
	require.NoError(t, db.Create(&user).Error)

	h := &Handler{DB: db}

	reqBody := UpdatePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "short", // less than 8 chars
	}
	body, _ := json.Marshal(reqBody)

	req := testutil.NewAuthenticatedRequest("PUT", "/v2/users/me/password", bytes.NewReader(body), user.ID, nil)
	rr := httptest.NewRecorder()

	h.UpdateMePassword(rr, req)

	// Doit échouer à la validation (go-playground/validator)
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	// Vérifier que la version du token n'a PAS été incrémentée
	var updatedUser auth.User
	db.First(&updatedUser, user.ID)
	assert.Equal(t, 1, updatedUser.TokenVersion)
}
