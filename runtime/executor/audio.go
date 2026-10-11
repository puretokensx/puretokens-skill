package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const maxAudioInput = 8 << 20
const maxAudioOutput = 16 << 20
const maxAudioRequest = 64 << 10

//go:embed audio-profiles.json
var audioProfileJSON []byte

type audioProfile struct {
	Model                    string            `json:"model"`
	Operation                string            `json:"operation"`
	Path                     string            `json:"path"`
	MaxInputCharacters       int               `json:"max_input_characters"`
	MaxInstructionCharacters int               `json:"max_instruction_characters"`
	Voices                   map[string]string `json:"voices"`
	SupportsInstruction      bool              `json:"supports_instruction"`
}

func audioProfiles() map[string]audioProfile {
	var catalog struct {
		Models map[string]audioProfile `json:"models"`
	}
	if json.Unmarshal(audioProfileJSON, &catalog) != nil {
		return nil
	}
	return catalog.Models
}

type audioRequest struct {
	Operation      string   `json:"operation"`
	Model          string   `json:"model"`
	Input          string   `json:"input,omitempty"`
	Instruction    string   `json:"instruction,omitempty"`
	Voice          string   `json:"voice,omitempty"`
	Speed          *float64 `json:"speed,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"`
	File           string   `json:"file,omitempty"`
	OutputDir      string   `json:"output_dir,omitempty"`
}

type audioArtifact struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Bytes     int    `json:"bytes"`
	MediaType string `json:"media_type"`
}

type audioReceipt struct {
	OK                bool              `json:"ok"`
	Command           string            `json:"command"`
	Operation         string            `json:"operation,omitempty"`
	Model             string            `json:"model,omitempty"`
	SubmissionOutcome string            `json:"submission_outcome"`
	NextStep          string            `json:"next_step"`
	DeliveryStatus    string            `json:"delivery_status,omitempty"`
	Artifact          *audioArtifact    `json:"artifact,omitempty"`
	Result            map[string]string `json:"result,omitempty"`
	FailurePhase      string            `json:"failure_phase,omitempty"`
	LocalErrorCode    string            `json:"local_error_code,omitempty"`
	HTTPStatus        int               `json:"http_status,omitempty"`
	APIErrorCode      string            `json:"api_error_code,omitempty"`
	ErrorMessage      string            `json:"error_message,omitempty"`
	NextAction        string            `json:"next_action,omitempty"`
	Validation        *audioValidation  `json:"validation,omitempty"`
}

func audioFailure(outcome, phase, code, message string, status int) audioReceipt {
	if status < 100 || status > 599 {
		status = 0
	}
	return audioReceipt{Command: "audio", SubmissionOutcome: outcome, NextStep: "await_user",
		FailurePhase: phase, LocalErrorCode: code, HTTPStatus: status, ErrorMessage: message,
		NextAction: "Stop. Do not automatically repeat the POST, poll a media task, switch hosts or infer charges. Reattach an existing verified output if available; unknown results cannot be retrieved by this command."}
}

func readAudioRequestFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !filepath.IsAbs(path) || !info.Mode().IsRegular() || info.Size() > maxAudioRequest {
		return nil, errors.New("invalid audio request file")
	}
	return readBoundedFile(path, maxAudioRequest)
}

// Recheck an explicit artifact receipt locally before another attachment
// handoff. This command never resolves a credential or contacts an endpoint.
func runAudioVerify(output io.Writer, file string) error {
	fail := func() error {
		result := audioFailure("not_submitted", "validation", "audio_artifact_invalid",
			"The saved audio artifact no longer matches its receipt. Preserve existing files; do not generate again automatically.", 0)
		result.Command = "audio-verify"
		writeJSON(output, result)
		return errors.New("audio artifact verification failed")
	}
	if !filepath.IsAbs(file) {
		return fail()
	}
	data, err := readAudioRequestFile(file)
	if err != nil {
		return fail()
	}
	value, err := strictEvaluationJSON(data, maxAudioRequest)
	document, ok := value.(map[string]any)
	if err != nil || !ok || len(document) != 1 {
		return fail()
	}
	fields, ok := document["artifact"].(map[string]any)
	if !ok || len(fields) != 4 {
		return fail()
	}
	for _, key := range []string{"path", "sha256", "bytes", "media_type"} {
		if fields[key] == nil {
			return fail()
		}
	}
	encoded, _ := json.Marshal(fields)
	var artifact audioArtifact
	if json.Unmarshal(encoded, &artifact) != nil || !filepath.IsAbs(artifact.Path) ||
		artifact.Bytes < 1 || artifact.Bytes > maxAudioOutput || len(artifact.SHA256) != 64 {
		return fail()
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(artifact.Path)), ".")
	if (format != "mp3" && format != "wav") || audioMIME(format) != artifact.MediaType {
		return fail()
	}
	finishVerificationTiming := timingsForWriter(output).enter("verification")
	defer finishVerificationTiming()
	info, err := os.Lstat(artifact.Path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != int64(artifact.Bytes) {
		return fail()
	}
	if _, err := os.Lstat(artifact.Path + ".incomplete"); !os.IsNotExist(err) {
		return fail()
	}
	input, err := os.Open(artifact.Path)
	if err != nil {
		return fail()
	}
	defer input.Close()
	opened, err := input.Stat()
	if err != nil || !sameAttachment(info, opened) {
		return fail()
	}
	bytes, err := io.ReadAll(io.LimitReader(input, maxAudioOutput+1))
	after, statErr := input.Stat()
	digest := sha256.Sum256(bytes)
	if err != nil || statErr != nil || !sameAttachment(opened, after) ||
		hex.EncodeToString(digest[:]) != artifact.SHA256 || !validAudioBytes(bytes, format) {
		return fail()
	}
	writeJSON(output, audioReceipt{OK: true, Command: "audio-verify", SubmissionOutcome: "not_submitted",
		NextStep: "deliver", DeliveryStatus: "downloaded", Artifact: &artifact})
	return nil
}

func decodeAudioRequest(data []byte) (audioRequest, error) {
	request := audioRequest{}
	value, err := strictEvaluationJSON(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}), maxAudioRequest)
	fields, ok := value.(map[string]any)
	if err != nil || !ok {
		return request, audioIssue("request", "invalid_json", maxAudioRequest)
	}
	encoded, _ := json.Marshal(fields)
	if json.Unmarshal(encoded, &request) != nil {
		return request, audioIssue("request", "invalid_type", 0)
	}
	profile, ok := audioProfiles()[request.Model]
	if !ok {
		return request, audioIssue("model", "unsupported", 0)
	}
	if request.Operation != profile.Operation {
		return request, audioIssue("operation", "model_mismatch", 0)
	}
	allowed := map[string]bool{"operation": true, "model": true}
	switch request.Operation {
	case "speech":
		for _, key := range []string{"input", "voice", "speed", "response_format", "output_dir", "instruction"} {
			allowed[key] = true
		}
		if strings.TrimSpace(request.Input) == "" {
			return request, audioIssue("input", "required", 0)
		}
		if utf8.RuneCountInString(request.Input) > profile.MaxInputCharacters {
			return request, audioIssue("input", "max_characters", profile.MaxInputCharacters)
		}
		if profile.Voices[request.Voice] == "" {
			return request, audioIssue("voice", "unsupported", 0)
		}
		if request.Speed != nil && (*request.Speed < 0.5 || *request.Speed > 2) {
			return request, audioIssue("speed", "out_of_range", 0)
		}
		if _, exists := fields["instruction"]; exists {
			if !profile.SupportsInstruction {
				return request, audioIssue("instruction", "unsupported", 0)
			}
			if strings.TrimSpace(request.Instruction) == "" {
				return request, audioIssue("instruction", "required", 0)
			}
			if utf8.RuneCountInString(request.Instruction) > 500 {
				return request, audioIssue("instruction", "max_characters", 500)
			}
		}
	case "generate":
		for _, key := range []string{"instruction", "response_format", "output_dir"} {
			allowed[key] = true
		}
		if strings.TrimSpace(request.Instruction) == "" {
			return request, audioIssue("instruction", "required", 0)
		}
		if utf8.RuneCountInString(request.Instruction) > profile.MaxInstructionCharacters {
			return request, audioIssue("instruction", "max_characters", profile.MaxInstructionCharacters)
		}
	case "transcribe":
		allowed["file"] = true
		if !filepath.IsAbs(request.File) {
			return request, audioIssue("file", "absolute_path_required", 0)
		}
	default:
		return request, audioIssue("operation", "unsupported", 0)
	}
	for key, value := range fields {
		if !allowed[key] {
			return request, audioIssue("request", "undeclared_field", 0)
		}
		if value == nil {
			return request, audioIssue(key, "invalid_type", 0)
		}
	}
	if request.Operation != "transcribe" {
		if !filepath.IsAbs(request.OutputDir) {
			return request, audioIssue("output_dir", "absolute_path_required", 0)
		}
		if request.ResponseFormat != "mp3" && request.ResponseFormat != "wav" {
			return request, audioIssue("response_format", "unsupported", 0)
		}
	}
	return request, nil
}

// Preparation finishes before credential resolution or paid HTTP. Input bytes
// remain a bounded in-memory snapshot; the caller owns the original file.
func prepareAudioBody(request audioRequest) ([]byte, string, error) {
	if request.Operation != "transcribe" {
		body := map[string]any{"model": request.Model, "response_format": request.ResponseFormat}
		if request.Operation == "speech" {
			body["input"], body["voice"] = request.Input, request.Voice
			if request.Speed != nil {
				body["speed"] = *request.Speed
			}
			if request.Instruction != "" {
				body["instruction"] = request.Instruction
			}
		} else {
			body["instruction"], body["task"], body["stream_format"] = request.Instruction, "text_to_audio", "audio"
		}
		data, err := json.Marshal(body)
		return data, "application/json", err
	}
	before, err := os.Lstat(request.File)
	if err != nil || !before.Mode().IsRegular() || before.Size() < 1 || before.Size() > maxAudioInput {
		return nil, "", errors.New("invalid audio attachment")
	}
	file, err := os.Open(request.File)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !sameAttachment(before, opened) {
		return nil, "", errors.New("audio attachment changed")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxAudioInput+1))
	after, statErr := file.Stat()
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(request.File)), ".")
	if err != nil || statErr != nil || !sameAttachment(opened, after) || int64(len(data)) != opened.Size() || !validAudioBytes(data, format) {
		return nil, "", errors.New("invalid audio attachment")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("model", request.Model)
	_ = writer.WriteField("response_format", "json")
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="recording.`+format+`"`)
	header.Set("Content-Type", audioMIME(format))
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, "", err
	}
	if _, err = part.Write(data); err != nil {
		return nil, "", err
	}
	if err = writer.Close(); err != nil {
		return nil, "", err
	}
	return body.Bytes(), writer.FormDataContentType(), nil
}

func runAudioCommand(output io.Writer, svc service, host, file string) error {
	var data []byte
	var err error
	if filepath.IsAbs(file) {
		data, err = readAudioRequestFile(file)
	}
	request, decodeErr := decodeAudioRequest(data)
	fail := func(code, message string) error {
		writeJSON(output, audioFailure("not_submitted", "validation", code, message, 0))
		return errors.New("audio preparation failed")
	}
	if err != nil {
		return fail("audio_request_invalid", "Use one absolute UTF-8 request file with a reviewed audio model, operation and declared fields. No API request was sent.")
	}
	if decodeErr != nil {
		var issue *audioValidation
		if !errors.As(decodeErr, &issue) {
			issue = audioIssue("request", "invalid_json", maxAudioRequest)
		}
		result := audioFailure("not_submitted", "validation", "audio_request_invalid", issue.message(), 0)
		result.Validation = issue
		result.NextAction = "Correct only the identified request construction error before any POST. If correction changes the user's text, voice, format, speed or request count, obtain that choice first; never truncate, replace or split silently. This receipt describes only this unsubmitted attempt."
		writeJSON(output, result)
		return errors.New("audio preparation failed")
	}
	body, contentType, err := prepareAudioBody(request)
	if err != nil {
		return fail("audio_request_invalid", "The specified recording is unavailable, changed, oversized or not a supported complete audio file. No API request was sent.")
	}
	// Reserve the writable output before any network request.
	var outputFile *os.File
	if request.Operation != "transcribe" {
		info, err := os.Stat(request.OutputDir)
		if err != nil || !info.IsDir() {
			return fail("audio_request_invalid", "Choose an existing absolute output directory. No API request was sent.")
		}
		outputFile, err = os.CreateTemp(request.OutputDir, ".puretokens-audio-part-*")
		if err != nil {
			return fail("audio_output_unavailable", "The output directory is not writable. No API request was sent.")
		}
		defer func() { outputFile.Close(); os.Remove(outputFile.Name()) }()
	}
	token, err := credentialForHost(host)
	if err != nil {
		code, message, next := credentialFailureDetails(err)
		result := audioFailure("not_submitted", "validation", code, message, 0)
		result.NextAction = next
		writeJSON(output, result)
		return err
	}
	defer clearString(&token)
	svc.token = token
	return executeAudio(output, svc, request, body, contentType, outputFile)
}

func executeAudio(output io.Writer, svc service, request audioRequest, body []byte, contentType string, file *os.File) error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	path := audioProfiles()[request.Model].Path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, svc.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+svc.token)
	req.Header.Set("Content-Type", contentType)
	if request.Operation == "transcribe" {
		req.Header.Set("Accept", "application/json")
	} else {
		req.Header.Set("Accept", audioMIME(request.ResponseFormat))
	}
	svc.support.beginRequest(http.MethodPost, path)
	finishRequestTiming := svc.timePhase("request")
	defer finishRequestTiming()
	response, err := fixedClient(svc.client).Do(req)
	svc.support.observeResponse(response)
	failure := func(outcome, phase, code, message string, status int, apiCode string) error {
		result := audioFailure(outcome, phase, code, message, status)
		result.Operation, result.Model, result.APIErrorCode = request.Operation, request.Model, apiCode
		writeJSON(output, result)
		return errors.New("audio request did not complete")
	}
	if err != nil {
		return failure("unknown", "submission", "audio_request_failed", "The synchronous audio response was not received. Processing and charges are unknown.", 0, "")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
		outcome := "unknown"
		if readErr == nil && len(data) <= maxResponseBytes && response.StatusCode >= 400 && response.StatusCode < 500 {
			outcome = "rejected"
		}
		code, message := publicAPIError(sanitizeResponseJSON(data, svc.token))
		if message == "" {
			message = "The audio request did not return a verified result."
		}
		return failure(outcome, "submission", "audio_request_failed", message, response.StatusCode, code)
	}
	limit := maxAudioOutput
	if request.Operation == "transcribe" {
		limit = 128 << 10
	}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	finishRequestTiming()
	finishVerificationTiming := svc.timePhase("verification")
	defer finishVerificationTiming()
	invalid := func() error {
		return failure("unknown", "submission", "audio_response_invalid", "The audio response could not be verified. This does not prove that processing failed or that no charge occurred.", response.StatusCode, "")
	}
	if readErr != nil || len(data) > limit || len(data) == 0 {
		return invalid()
	}
	result := audioReceipt{OK: true, Command: "audio", Operation: request.Operation, Model: request.Model, SubmissionOutcome: "accepted"}
	if request.Operation == "transcribe" {
		mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			return invalid()
		}
		if _, err := strictEvaluationJSON(data, limit); err != nil {
			return invalid()
		}
		sanitized := sanitizeResponseJSON(data, svc.token)
		var document map[string]any
		if json.Unmarshal(sanitized, &document) != nil {
			return invalid()
		}
		text, ok := document["text"].(string)
		if !ok || strings.TrimSpace(text) == "" || utf8.RuneCountInString(text) > 100000 || document["error"] != nil {
			return invalid()
		}
		result.NextStep, result.Result = "done", map[string]string{"text": text}
	} else {
		mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
		if err != nil || (mediaType != audioMIME(request.ResponseFormat) && !(request.ResponseFormat == "wav" && mediaType == "audio/x-wav")) || !validAudioBytes(data, request.ResponseFormat) {
			return invalid()
		}
		if _, err = file.Write(data); err != nil {
			return failure("accepted", "content", "audio_output_unavailable", "Audio was received but could not be saved. Do not generate it again automatically.", response.StatusCode, "")
		}
		if err = file.Close(); err != nil {
			return failure("accepted", "content", "audio_output_unavailable", "Audio was received but could not be finalized.", response.StatusCode, "")
		}
		name := "puretokens-audio-" + strings.TrimPrefix(filepath.Base(file.Name()), ".puretokens-audio-part-") + "." + request.ResponseFormat
		destination := filepath.Join(request.OutputDir, name)
		if err := copyDownloadExclusive(file.Name(), destination); err != nil {
			return failure("accepted", "content", "audio_output_unavailable", "Audio was received but could not be finalized without overwriting a file.", response.StatusCode, "")
		}
		digest := sha256.Sum256(data)
		check, err := readBoundedFile(destination, maxAudioOutput)
		if err != nil || !bytes.Equal(check, data) {
			return failure("accepted", "content", "audio_output_unavailable", "The saved audio file could not be verified; do not regenerate automatically.", response.StatusCode, "")
		}
		result.Artifact = &audioArtifact{Path: destination, SHA256: hex.EncodeToString(digest[:]), Bytes: len(data), MediaType: audioMIME(request.ResponseFormat)}
		result.NextStep, result.DeliveryStatus = "deliver", "downloaded"
	}
	writeJSON(output, result)
	return nil
}
