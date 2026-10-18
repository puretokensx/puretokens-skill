package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const maxRetrievalRequestBytes = 1 << 20
const maxEmbeddingDimensions = 65536

// Reviewed protocol support is separate from authenticated model visibility.
var retrievalModels = map[string]string{
	"qwen3.7-text-embedding":       "embedding",
	"qwen3.7-text-embedding-flash": "embedding",
	"qwen3.7-text-rerank":          "rerank",
}

type retrievalRequest struct {
	Model      string
	Input      []string
	Query      string
	Documents  []string
	TopN       int
	OutputFile string
}

type embeddingRow struct {
	Index      int       `json:"index"`
	Dimensions int       `json:"dimensions"`
	Embedding  []float64 `json:"embedding"`
}

type rankingRow struct {
	Rank           int     `json:"rank"`
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type rankingResult struct {
	Results []rankingRow `json:"results"`
}

type retrievalArtifact struct {
	Path       string `json:"path"`
	Format     string `json:"format"`
	ItemCount  int    `json:"item_count"`
	Dimensions int    `json:"dimensions,omitempty"`
	SHA256     string `json:"sha256"`
	Bytes      int    `json:"bytes"`
}

type retrievalReceipt struct {
	OK                bool               `json:"ok"`
	Command           string             `json:"command"`
	SubmissionOutcome string             `json:"submission_outcome"`
	NextStep          string             `json:"next_step"`
	OutputVerified    bool               `json:"output_verified"`
	Artifact          *retrievalArtifact `json:"artifact,omitempty"`
	Result            *rankingResult     `json:"result,omitempty"`
	FailurePhase      string             `json:"failure_phase,omitempty"`
	LocalErrorCode    string             `json:"local_error_code,omitempty"`
	HTTPStatus        int                `json:"http_status,omitempty"`
	APIErrorCode      string             `json:"api_error_code,omitempty"`
	ErrorMessage      string             `json:"error_message,omitempty"`
	NextAction        string             `json:"next_action,omitempty"`
	RetryAfterSecs    int                `json:"retry_after_seconds,omitempty"`
}

func isRetrievalCommand(command string) bool { return command == "embedding" || command == "rerank" }

func retrievalFailure(command, outcome, phase, code, message string, status int) retrievalReceipt {
	if status < 100 || status > 599 {
		status = 0
	}
	return retrievalReceipt{Command: command, SubmissionOutcome: outcome, NextStep: "await_user",
		FailurePhase: phase, LocalErrorCode: code, ErrorMessage: message, HTTPStatus: status,
		NextAction: "Stop. Do not automatically repeat the request, switch hosts or poll a media task. An unknown response or a local save failure does not prove that no charge occurred."}
}

func decodeRetrievalRequest(kind string, data []byte) (retrievalRequest, error) {
	request := retrievalRequest{}
	fail := errors.New("invalid retrieval request")
	value, err := strictEvaluationJSON(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}), maxRetrievalRequestBytes)
	body, ok := value.(map[string]any)
	if err != nil || !ok || !isRetrievalCommand(kind) {
		return request, fail
	}
	request.Model, _ = body["model"].(string)
	request.OutputFile, _ = body["output_file"].(string)
	if retrievalModels[request.Model] != kind || !filepath.IsAbs(request.OutputFile) ||
		strings.IndexFunc(request.OutputFile, unicode.IsControl) >= 0 {
		return request, fail
	}
	allowed := map[string]bool{"model": true, "output_file": true}
	extension := ".jsonl"
	if kind == "embedding" {
		allowed["input"] = true
		request.Input, ok = retrievalTexts(body["input"])
	} else {
		extension = ".json"
		allowed["query"], allowed["documents"], allowed["top_n"] = true, true, true
		request.Query, _ = body["query"].(string)
		if !retrievalText(request.Query, 8192) {
			return request, fail
		}
		request.Documents, ok = retrievalTexts(body["documents"])
		request.TopN = len(request.Documents)
		if top, present := body["top_n"]; present {
			var valid bool
			request.TopN, valid = retrievalIndex(top, 1, len(request.Documents))
			if !valid {
				return request, fail
			}
		}
	}
	if !ok || filepath.Ext(request.OutputFile) != extension {
		return request, fail
	}
	for key := range body {
		if !allowed[key] {
			return request, fail
		}
	}
	return request, nil
}

func retrievalText(text string, limit int) bool {
	return strings.TrimSpace(text) != "" && utf8.RuneCountInString(text) <= limit
}

func retrievalTexts(value any) ([]string, bool) {
	values, ok := value.([]any)
	if !ok || len(values) < 1 || len(values) > 64 {
		return nil, false
	}
	items := make([]string, len(values))
	for index, value := range values {
		text, ok := value.(string)
		if !ok || !retrievalText(text, 32768) {
			return nil, false
		}
		items[index] = text
	}
	return items, true
}

func retrievalNumber(value any) (float64, bool) {
	number, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	result, err := number.Float64()
	return result, err == nil && !math.IsNaN(result) && !math.IsInf(result, 0)
}

func retrievalIndex(value any, minimum, maximum int) (int, bool) {
	number, ok := retrievalNumber(value)
	if !ok || number < float64(minimum) || number > float64(maximum) || math.Trunc(number) != number {
		return 0, false
	}
	return int(number), true
}

func runRetrievalCommand(output io.Writer, svc service, host, kind, file string) error {
	fail := func(code, message string) error {
		writeJSON(output, retrievalFailure(kind, "not_submitted", "validation", code, message, 0))
		return errors.New(code)
	}
	// A regular, explicit request file avoids waiting on a pipe/device before
	// the bounded HTTP command has even begun.
	info, err := os.Stat(file)
	if !filepath.IsAbs(file) || err != nil || !info.Mode().IsRegular() {
		return fail(kind+"_request_invalid", "Use one absolute regular UTF-8 JSON request file. No API request was sent.")
	}
	data, err := readBoundedFile(file, maxRetrievalRequestBytes)
	request, decodeErr := decodeRetrievalRequest(kind, data)
	if err != nil || decodeErr != nil {
		return fail(kind+"_request_invalid", "Use a reviewed exact model, 1–64 nonempty texts within the installed limits, and an absolute new output file. No API request was sent.")
	}
	if _, err := os.Lstat(request.OutputFile); !os.IsNotExist(err) {
		return fail(kind+"_output_unavailable", "The output file already exists or cannot be safely inspected. Choose a new file. No API request was sent.")
	}
	if _, err := os.Lstat(request.OutputFile + ".incomplete"); !os.IsNotExist(err) {
		return fail(kind+"_output_unavailable", "The output file has an unresolved incomplete write. Choose a new file. No API request was sent.")
	}
	// Reserve writable space before reading a credential or incurring a charge.
	temporary, err := os.CreateTemp(filepath.Dir(request.OutputFile), ".puretokens-retrieval-part-*")
	if err != nil {
		return fail(kind+"_output_unavailable", "Choose a new file in an existing writable output directory. No API request was sent.")
	}
	defer func() { temporary.Close(); os.Remove(temporary.Name()) }()
	token, err := credentialForHost(host)
	if err != nil {
		code, message, next := credentialFailureDetails(err)
		result := retrievalFailure(kind, "not_submitted", "validation", code, message, 0)
		result.NextAction = next
		writeJSON(output, result)
		return err
	}
	defer clearString(&token)
	svc.token = token
	return executeRetrieval(output, svc, kind, request, temporary)
}

func executeRetrieval(output io.Writer, svc service, kind string, request retrievalRequest, temporary *os.File) error {
	payload := map[string]any{"model": request.Model}
	path := "/v1/embeddings"
	if kind == "embedding" {
		payload["input"], payload["encoding_format"] = request.Input, "float"
	} else {
		path = "/v1/rerank"
		payload["query"], payload["documents"], payload["top_n"], payload["return_documents"] = request.Query, request.Documents, request.TopN, false
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err // Validated strings and integers only.
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	response, status, retry, code, message, err := svc.request(ctx, http.MethodPost, path, bytes.NewReader(body), "application/json")
	if err != nil || status < 200 || status >= 300 {
		outcome, local := "unknown", kind+"_request_failed"
		if err == nil && status >= 400 && status < 500 {
			outcome = "rejected"
		}
		if status >= 200 && status < 300 {
			local = kind + "_response_invalid"
		}
		if message == "" {
			message = "The synchronous request did not return a verified result."
		}
		failure := retrievalFailure(kind, outcome, "submission", local, message, status)
		failure.APIErrorCode, failure.RetryAfterSecs = code, retry
		writeJSON(output, failure)
		return errors.New(local)
	}
	data, count, dimensions, result, valid := projectRetrievalResult(kind, response, request)
	if !valid {
		writeJSON(output, retrievalFailure(kind, "unknown", "submission", kind+"_response_invalid",
			"The response could not be verified against the input indices and numeric result contract. This does not prove that no charge occurred.", status))
		return errors.New("invalid retrieval response")
	}
	failOutput := func() error {
		writeJSON(output, retrievalFailure(kind, "accepted", "content", kind+"_output_unavailable",
			"The API result was verified but the local output could not be saved and verified without overwriting a file. Do not generate it again automatically.", status))
		return errors.New("retrieval output unavailable")
	}
	if len(data) > maxResponseBytes+65536 {
		return failOutput()
	}
	if _, err := temporary.Write(data); err != nil {
		return failOutput()
	}
	if err := temporary.Sync(); err != nil {
		return failOutput()
	}
	if err := temporary.Close(); err != nil {
		return failOutput()
	}
	if err := copyDownloadExclusive(temporary.Name(), request.OutputFile); err != nil {
		return failOutput()
	}
	saved, err := readBoundedFile(request.OutputFile, maxResponseBytes+65536)
	if err != nil || !bytes.Equal(saved, data) {
		return failOutput()
	}
	digest := sha256.Sum256(data)
	format := "jsonl"
	if kind == "rerank" {
		format = "json"
	}
	writeJSON(output, retrievalReceipt{OK: true, Command: kind, SubmissionOutcome: "accepted", NextStep: "done", OutputVerified: true,
		Artifact: &retrievalArtifact{Path: request.OutputFile, Format: format, ItemCount: count, Dimensions: dimensions,
			SHA256: hex.EncodeToString(digest[:]), Bytes: len(data)}, Result: result})
	return nil
}

func projectRetrievalResult(kind string, data []byte, request retrievalRequest) ([]byte, int, int, *rankingResult, bool) {
	invalid := func() ([]byte, int, int, *rankingResult, bool) { return nil, 0, 0, nil, false }
	value, err := strictEvaluationJSON(data, maxResponseBytes)
	body, ok := value.(map[string]any)
	if err != nil || !ok || body["error"] != nil {
		return invalid()
	}
	key, count := "data", len(request.Input)
	if kind == "rerank" {
		key, count = "results", request.TopN
	}
	entries, ok := body[key].([]any)
	if !ok || len(entries) != count {
		return invalid()
	}
	seen := map[int]bool{}
	var output bytes.Buffer
	if kind == "embedding" {
		rows, dimensions := make([]embeddingRow, count), 0
		for _, entry := range entries {
			object := jsonObject(entry)
			index, valid := retrievalIndex(object["index"], 0, count-1)
			vector, ok := object["embedding"].([]any)
			if !valid || seen[index] || !ok || len(vector) < 1 || len(vector) > maxEmbeddingDimensions || (dimensions != 0 && len(vector) != dimensions) {
				return invalid()
			}
			seen[index], dimensions = true, len(vector)
			values := make([]float64, dimensions)
			for position, value := range vector {
				var valid bool
				values[position], valid = retrievalNumber(value)
				if !valid {
					return invalid()
				}
			}
			rows[index] = embeddingRow{Index: index, Dimensions: dimensions, Embedding: values}
		}
		for _, row := range rows {
			if json.NewEncoder(&output).Encode(row) != nil {
				return invalid()
			}
		}
		return output.Bytes(), count, dimensions, nil, true
	}
	rows := make([]rankingRow, 0, count)
	for _, entry := range entries {
		object := jsonObject(entry)
		index, valid := retrievalIndex(object["index"], 0, len(request.Documents)-1)
		score, ok := retrievalNumber(object["relevance_score"])
		if !valid || seen[index] || !ok {
			return invalid()
		}
		seen[index] = true
		rows = append(rows, rankingRow{Index: index, RelevanceScore: score})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].RelevanceScore > rows[j].RelevanceScore })
	for index := range rows {
		rows[index].Rank = index + 1
	}
	result := &rankingResult{Results: rows}
	if json.NewEncoder(&output).Encode(result) != nil {
		return invalid()
	}
	return output.Bytes(), count, 0, result, true
}
