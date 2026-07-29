package metadata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"serveoapi/internal/testutil"
)

func TestGetMetadata_Success(t *testing.T) {
	// Metadata has no DB dependency, we can pass nil or an empty db
	db, err := testutil.SetupTestDB()
	require.NoError(t, err)

	h := &Handler{DB: db}

	req := testutil.NewAuthenticatedRequest("GET", "/v2/metadata/", nil, 1, nil)
	rr := httptest.NewRecorder()

	h.GetMetadata(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response MetadataResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "iSweat", response.Author)
	assert.False(t, response.Deprecated)
}
