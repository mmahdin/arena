package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestStrictJSONEnforcesRequestShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, contentType, body string
		status                  int
		code                    string
		unknownLength           bool
	}{
		{name: "valid Persian name", contentType: "application/json", body: `{"displayName":"تستر"}`, status: http.StatusOK},
		{name: "trailing whitespace", contentType: "application/json; charset=utf-8", body: "{\"displayName\":\"Tester\"} \n\t", status: http.StatusOK},
		{name: "unknown field", contentType: "application/json", body: `{"displayName":"Tester","conversationStyle":"guidance"}`, status: http.StatusBadRequest, code: "INVALID_JSON"},
		{name: "second JSON object", contentType: "application/json", body: `{"displayName":"Tester"} {}`, status: http.StatusBadRequest, code: "INVALID_JSON"},
		{name: "trailing garbage", contentType: "application/json", body: `{"displayName":"Tester"} invalid`, status: http.StatusBadRequest, code: "INVALID_JSON"},
		{name: "malformed JSON", contentType: "application/json", body: `{"displayName":`, status: http.StatusBadRequest, code: "INVALID_JSON"},
		{name: "wrong media type", contentType: "text/plain", body: `{}`, status: http.StatusUnsupportedMediaType, code: "UNSUPPORTED_MEDIA_TYPE"},
		{name: "known oversized body", contentType: "application/json", body: `{"displayName":"` + strings.Repeat("a", 16*1024) + `"}`, status: http.StatusRequestEntityTooLarge, code: "PAYLOAD_TOO_LARGE"},
		{name: "unknown oversized body", contentType: "application/json", body: `{"displayName":"` + strings.Repeat("a", 16*1024) + `"}`, unknownLength: true, status: http.StatusBadRequest, code: "INVALID_JSON"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			handled := false
			r.POST("/", func(c *gin.Context) {
				var body struct {
					DisplayName string `json:"displayName"`
				}
				if StrictJSON(c, &body) {
					handled = true
					c.JSON(http.StatusOK, body)
				}
			})
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			if tc.unknownLength {
				req.ContentLength = -1
			}
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("status: got %d, want %d", response.Code, tc.status)
			}
			if handled != (tc.status == http.StatusOK) {
				t.Fatal("invalid input reached request handling")
			}
			if tc.code != "" {
				var problem Problem
				if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
					t.Fatal(err)
				}
				if problem.Code != tc.code {
					t.Fatalf("code: got %s, want %s", problem.Code, tc.code)
				}
			}
		})
	}
}
