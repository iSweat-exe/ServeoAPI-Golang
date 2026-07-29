package apikeys

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
    "fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"serveoapi/internal/testutil"
)

func TestCreateApiKey_Success(t *testing.T) {
	db, err := testutil.SetupTestDB(&ApiKey{})
	require.NoError(t, err)

	h := &Handler{DB: db}

	reqBody := CreateApiKeyRequest{
		Name: "Test Key",
	}
	body, _ := json.Marshal(reqBody)

	req := testutil.NewAuthenticatedRequest("POST", "/v2/apikeys/create", bytes.NewReader(body), 1, nil)
	rr := httptest.NewRecorder()

	h.CreateApiKey(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var response CreateApiKeyResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Test Key", response.Name)
	assert.NotEmpty(t, response.Token)
	
	// Check DB
	var apiKey ApiKey
	db.First(&apiKey, response.ID)
	assert.Equal(t, uint(1), apiKey.UserID)
	assert.Equal(t, "Test Key", apiKey.Name)
	assert.NotEmpty(t, apiKey.TokenHash)
	assert.NotEmpty(t, apiKey.Prefix)
}

func TestCreateApiKey_MissingName(t *testing.T) {
	db, err := testutil.SetupTestDB(&ApiKey{})
	require.NoError(t, err)

	h := &Handler{DB: db}

	reqBody := CreateApiKeyRequest{
		Name: "", // Missing name
	}
	body, _ := json.Marshal(reqBody)

	req := testutil.NewAuthenticatedRequest("POST", "/v2/apikeys/create", bytes.NewReader(body), 1, nil)
	rr := httptest.NewRecorder()

	h.CreateApiKey(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListApiKeys_Success(t *testing.T) {
	db, err := testutil.SetupTestDB(&ApiKey{})
	require.NoError(t, err)

	// Create test keys for User 1 and User 2
	db.Create(&ApiKey{UserID: 1, Name: "User1 Key1", Prefix: "srv_test", TokenHash: "hash1"})
	db.Create(&ApiKey{UserID: 1, Name: "User1 Key2", Prefix: "srv_test", TokenHash: "hash2"})
	db.Create(&ApiKey{UserID: 2, Name: "User2 Key1", Prefix: "srv_test", TokenHash: "hash3"})

	h := &Handler{DB: db}

	req := testutil.NewAuthenticatedRequest("GET", "/v2/apikeys/", nil, 1, nil)
	rr := httptest.NewRecorder()

	h.ListApiKeys(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response []ApiKeyResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should only see User 1's keys
	assert.Len(t, response, 2)
	assert.Equal(t, "User1 Key1", response[0].Name)
	assert.Equal(t, "User1 Key2", response[1].Name)
}

func TestRevokeApiKey_Success(t *testing.T) {
	db, err := testutil.SetupTestDB(&ApiKey{})
	require.NoError(t, err)

	key := ApiKey{UserID: 1, Name: "To Delete", Prefix: "srv_test", TokenHash: "hash_to_delete"}
	db.Create(&key)

	h := &Handler{DB: db}

	req := testutil.NewAuthenticatedRequest("DELETE", "/v2/apikeys/" + fmt.Sprint(key.ID), nil, 1, nil)
    req.SetPathValue("id", fmt.Sprint(key.ID))
	rr := httptest.NewRecorder()

	h.RevokeApiKey(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify it's deleted
	var count int64
	db.Model(&ApiKey{}).Where("id = ?", key.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestRevokeApiKey_NotFound(t *testing.T) {
	db, err := testutil.SetupTestDB(&ApiKey{})
	require.NoError(t, err)

	// Key belongs to user 2
	key := ApiKey{UserID: 2, Name: "Not mine", Prefix: "srv_test", TokenHash: "hash_not_mine"}
	db.Create(&key)

	h := &Handler{DB: db}

	// User 1 tries to delete User 2's key
	req := testutil.NewAuthenticatedRequest("DELETE", "/v2/apikeys/" + fmt.Sprint(key.ID), nil, 1, nil)
    req.SetPathValue("id", fmt.Sprint(key.ID))
	rr := httptest.NewRecorder()

	h.RevokeApiKey(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}
