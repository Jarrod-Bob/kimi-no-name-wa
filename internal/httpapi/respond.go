package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"
	"strconv"
)

// errorBody is the one error shape the API ever returns. Code is a stable,
// machine-readable reason (see the README's API reference); Message is a
// sentence for a person.
type errorBody struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("writing response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	var body errorBody
	body.Error.Message = message
	body.Error.Code = code
	writeJSON(w, status, body)
}

const maxBody = 256 << 10

// decodeJSON reads a JSON request body into dst, answering the request
// itself and returning false if it can't. It insists on a JSON content type:
// a web page on another origin can't send one without a CORS preflight,
// which this server never grants, so no other site can drive it.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send the request body as JSON with Content-Type: application/json.")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, http.StatusRequestEntityTooLarge, "too_large", "That request is too large.")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid_json", "The request body isn't valid JSON: "+err.Error())
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "not_found", "Not found.")
		return 0, false
	}
	return id, true
}
