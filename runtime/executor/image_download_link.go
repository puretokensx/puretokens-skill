package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type imageDownloadLink struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
	Index     int    `json:"index"`
}

func executeImageDownloadLink(output io.Writer, svc service, request taskRequest, result receipt) error {
	if request.Kind != "image" || !terminalSuccess(request.TaskStatus) || request.ReconciliationRequired ||
		request.RequestedCount < 1 || !validOutputIndex(request, request.Index) {
		writeReceipt(output, mergeFailure(result, validationFailure("A download link requires one completed image task and its original count/index. Do not generate again.")))
		return errors.New("invalid download link request")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	body, status, retry, code, message, err := svc.request(ctx, http.MethodGet, statusPath("image", request.TaskID)+"/download-link?index="+strconv.Itoa(request.Index), nil, "")
	if err != nil || status != http.StatusOK {
		failure := apiFailure("content", status, retry, code, message, "The same-task download link is unavailable. Keep the original task and download diagnostics; do not retry automatically or generate again.")
		failure.LocalErrorCode = "download_link_unavailable"
		writeReceipt(output, withRetry(mergeFailure(result, failure), retry, svc.clock()))
		return errors.New("download link unavailable")
	}
	var response struct {
		TaskID    string `json:"task_id"`
		Index     *int   `json:"index"`
		URL       string `json:"url"`
		ExpiresAt string `json:"expires_at"`
	}
	_, logicalErr := readAPIObject(body)
	if logicalErr != nil || json.Unmarshal(body, &response) != nil || response.TaskID != request.TaskID || response.Index == nil || *response.Index != request.Index ||
		!validImageDownloadLink(response.URL, response.ExpiresAt, request.TaskID, svc.clock()) {
		writeReceipt(output, mergeFailure(result, receipt{FailurePhase: "content", HTTPStatus: status, LocalErrorCode: "download_link_invalid",
			ErrorMessage: "The service did not return a valid bounded download link.", NextAction: "Keep the original task and download diagnostics. Do not expose the response, follow its URL, retry automatically or generate again."}))
		return errors.New("invalid download link")
	}
	result.OK, result.HTTPStatus = true, status
	result.Status, result.DeliveryStatus = "completed", "link_available"
	result.DownloadLink = &imageDownloadLink{URL: response.URL, ExpiresAt: response.ExpiresAt, Index: request.Index}
	result.NextAction = "Present this temporary link for the user to click, with its expiry. The image is generated but no local attachment was delivered. Do not open or download this link, mark delivered, refresh automatically or generate again. Refresh only on explicit user request."
	writeReceipt(output, result)
	return nil
}

func validImageDownloadLink(raw, expiry, taskID string, now time.Time) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "api.puretokensx.com" || u.User != nil || u.Fragment != "" || u.RawPath != "" || u.Opaque != "" {
		return false
	}
	prefix := "/v1/tasks/" + taskID + "/artifacts/"
	key, ok := strings.CutPrefix(u.Path, prefix)
	key, suffix := strings.CutSuffix(key, "/content")
	if !ok || !suffix || !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~-]{0,127}$`).MatchString(key) {
		return false
	}
	access, ok := strings.CutPrefix(u.RawQuery, "access=v2.")
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(access)
	if !ok || err != nil || len(decoded) != 44 || base64.RawURLEncoding.EncodeToString(decoded) != access {
		return false
	}
	expires, err := time.Parse(time.RFC3339, expiry)
	return err == nil && expires.After(now) && !expires.After(now.Add(10*time.Minute+5*time.Second))
}

func imageLinkFallbackEligible(result receipt) bool {
	if result.OK || result.Kind != "image" || !validTaskID(result.TaskID) || !terminalSuccess(result.Status) ||
		result.ReconciliationRequired || result.SubmissionOutcome == "unknown" || result.FailurePhase != "content" || result.RequestedCount < 1 ||
		result.APIErrorCode != "" || result.RetryAfterSecs != 0 || result.RetryNotBefore != "" ||
		(result.HTTPStatus != 0 && result.HTTPStatus != http.StatusOK) {
		return false
	}
	switch result.LocalErrorCode {
	case "media_download_stalled", "media_download_timeout", "media_content_response_timeout":
		return true
	case "media_download_incomplete", "output_permission_denied":
		return result.HTTPStatus == http.StatusOK
	}
	return false
}
