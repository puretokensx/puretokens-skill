package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const taskRecordFormat = "puretokens-task-v1"

// An explicit user artifact, never a hidden task database. Keep only the
// metadata needed to continue or deliver an existing task.
type taskRecord struct {
	Format                 string                `json:"format"`
	Kind                   string                `json:"kind"`
	TaskID                 string                `json:"task_id,omitempty"`
	Model                  string                `json:"model,omitempty"`
	OriginalOperation      string                `json:"original_operation,omitempty"`
	RequestedCount         int                   `json:"requested_count,omitempty"`
	Parameters             map[string]any        `json:"parameters,omitempty"`
	Status                 string                `json:"status,omitempty"`
	SubmissionOutcome      string                `json:"submission_outcome,omitempty"`
	ReconciliationRequired bool                  `json:"reconciliation_required,omitempty"`
	RetryNotBefore         string                `json:"retry_not_before,omitempty"`
	WaitWindowsCompleted   int                   `json:"wait_windows_completed,omitempty"`
	OutputDir              string                `json:"output_dir,omitempty"`
	Downloaded             map[int]string        `json:"downloaded,omitempty"`
	DownloadProofs         map[int]downloadProof `json:"download_proofs,omitempty"`
	Delivered              []int                 `json:"delivered,omitempty"`
	LastFailure            *taskFailure          `json:"last_failure,omitempty"`
	PendingRedeliveryIndex *int                  `json:"pending_redelivery_index,omitempty"`
}

type recordReceiptWriter struct {
	output         io.Writer
	path           string
	record         taskRecord
	err            error
	downloadProofs map[string]downloadProof
}

func (writer *recordReceiptWriter) Write(data []byte) (int, error) {
	return writer.output.Write(data)
}

func (writer *recordReceiptWriter) writeReceipt(result receipt) {
	record := writer.record
	if result.TaskID != "" {
		record.TaskID = result.TaskID
	}
	if result.Model != "" {
		record.Model = result.Model
	}
	if result.OriginalOperation != "" {
		record.OriginalOperation = result.OriginalOperation
	}
	if result.RequestedCount > 0 {
		record.RequestedCount = result.RequestedCount
	}
	if len(result.Parameters) > 0 {
		record.Parameters = result.Parameters
	}
	if result.Status != "" {
		record.Status = result.Status
	}
	if failure := taskFailureFromReceipt(result); failure != nil {
		// A later read may omit the original terminal error. Keep the known
		// cause for that same terminal state until more specific evidence arrives.
		if failure.Category != "unknown" || record.LastFailure == nil ||
			record.LastFailure.Category == "unknown" || record.LastFailure.TaskStatus != record.Status {
			record.LastFailure = failure
		}
	} else if record.LastFailure != nil && record.LastFailure.TaskStatus != record.Status {
		record.LastFailure = nil
	}
	if result.SubmissionOutcome != "" {
		record.SubmissionOutcome = result.SubmissionOutcome
	}
	if record.TaskID == "" && result.FailurePhase == "validation" {
		record.SubmissionOutcome = "not_submitted"
	}
	record.ReconciliationRequired = result.ReconciliationRequired
	record.RetryNotBefore = result.RetryNotBefore
	record.WaitWindowsCompleted = result.WaitWindowsCompleted
	if record.Downloaded == nil {
		record.Downloaded = make(map[int]string)
	}
	if record.DownloadProofs == nil {
		record.DownloadProofs = make(map[int]downloadProof)
	}
	for index, path := range result.DownloadedPaths {
		if index < len(result.DownloadedIndexes) {
			i := result.DownloadedIndexes[index]
			if proof, ok := writer.downloadProofs[path]; ok && timedDownloadProof(timingsForWriter(writer.output), path, proof) {
				record.Downloaded[i] = path
				record.DownloadProofs[i] = proof
			} else {
				writer.err = errors.New("download integrity changed")
				result.OK = false
				result.FailurePhase = "content"
				result.LocalErrorCode = "download_integrity_unverified"
				result.ErrorMessage = "The downloaded file changed before its integrity could be recorded."
				result.NextAction = "Preserve the file and retrieve this same task into another output directory."
				result.DownloadedPaths = nil
				result.DownloadedIndexes = nil
				result.DeliveryStatus = ""
				break
			}
		}
	}
	result.RecordPath = writer.path
	result.DeliveredIndexes = record.Delivered
	result.RedeliveryIndex = record.PendingRedeliveryIndex
	if result.TaskID != "" && len(result.DownloadedIndexes) == 0 {
		// Resume receipts expose only pending handoffs. The host can attach an
		// already downloaded file before issuing another content request.
		for index := range record.Downloaded {
			if record.PendingRedeliveryIndex != nil && *record.PendingRedeliveryIndex != index {
				continue
			}
			delivered := false
			for _, done := range record.Delivered {
				if done == index {
					delivered = true
					break
				}
			}
			path := record.Downloaded[index]
			format := recordedDownloadFormat(record.Kind, record.TaskID, index, path)
			proof, known := record.DownloadProofs[index]
			pendingRedelivery := record.PendingRedeliveryIndex != nil && *record.PendingRedeliveryIndex == index
			if (!delivered || pendingRedelivery) && format != "" && known && proof.MediaType == format && timedDownloadProof(timingsForWriter(writer.output), path, proof) {
				result.DownloadedIndexes = append(result.DownloadedIndexes, index)
			}
		}
		sort.Ints(result.DownloadedIndexes)
		for _, index := range result.DownloadedIndexes {
			result.DownloadedPaths = append(result.DownloadedPaths, record.Downloaded[index])
		}
		if len(result.DownloadedIndexes) > 0 {
			result.DeliveryStatus = "downloaded_awaiting_host_delivery"
		}
	}
	if result.OK && terminalSuccess(result.Status) && !result.ReconciliationRequired && record.deliveryComplete() {
		result.DeliveryStatus = "delivered"
		result.NextAction = "All requested indexes have already been handed off. No further content request is needed."
	}
	if err := saveTaskRecord(writer.path, record, false); err != nil {
		writer.err = err
		result.OK = false
		result.LocalErrorCode = "task_record_write_failed"
		if result.FailurePhase == "" {
			result.FailurePhase = "status"
			if result.Operation == "generate" || result.Operation == "edit" {
				result.FailurePhase = "submission"
			}
			if len(result.DownloadedPaths) > 0 || result.DeliveryStatus != "" {
				result.FailurePhase = "content"
			}
		}
		result.ErrorMessage = "The task receipt could not be saved to its explicit record."
		result.NextAction = "Keep this receipt and task ID. Repair the record location before continuing; do not repeat the submission."
	} else {
		writer.record = record
	}
	writeJSON(writer.output, guideReceipt(result))
}

func (record taskRecord) deliveryComplete() bool {
	count := record.RequestedCount
	if record.Kind == "video" || record.Kind == "music" {
		count = 1
	}
	return count > 0 && len(record.Delivered) == count && record.PendingRedeliveryIndex == nil
}

func (record taskRecord) request() taskRequest {
	return taskRequest{
		Kind: record.Kind, Operation: "continue", TaskID: record.TaskID,
		Model: record.Model, OriginalOperation: record.OriginalOperation,
		RequestedCount: record.RequestedCount, Parameters: record.Parameters,
		TaskStatus: record.Status, RetryNotBefore: record.RetryNotBefore,
		ReconciliationRequired: record.ReconciliationRequired, OutputDir: record.OutputDir,
		WaitWindowsCompleted: record.WaitWindowsCompleted,
	}
}

func recordFromRequest(request taskRequest) taskRecord {
	result := taskReceipt(request, request.TaskID, request.TaskStatus)
	return taskRecord{
		Format: taskRecordFormat, Kind: result.Kind, TaskID: result.TaskID,
		Model: result.Model, OriginalOperation: result.OriginalOperation,
		RequestedCount: result.RequestedCount, Parameters: result.Parameters,
		Status: result.Status, RetryNotBefore: result.RetryNotBefore,
		ReconciliationRequired: result.ReconciliationRequired, OutputDir: request.OutputDir,
		WaitWindowsCompleted: result.WaitWindowsCompleted,
	}
}

func loadTaskRecord(path string) (taskRecord, error) {
	var record taskRecord
	if !filepath.IsAbs(path) {
		return record, errors.New("record requires an absolute path")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return record, errors.New("record must be an existing regular file")
	}
	data, err := readBoundedFile(path, maxResponseBytes)
	if err != nil {
		return record, errors.New("record unreadable")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return record, errors.New("record invalid")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return record, errors.New("record contains additional content")
	}
	if record.Format != taskRecordFormat || (record.Kind != "image" && record.Kind != "video" && record.Kind != "music") || (record.TaskID != "" && !validTaskID(record.TaskID)) || record.RequestedCount < 0 || record.RequestedCount > maxImageCount || record.WaitWindowsCompleted < 0 || record.WaitWindowsCompleted > 2 {
		return record, errors.New("record has unsupported metadata")
	}
	if !validTaskFailure(record.LastFailure, record.Status) {
		return record, errors.New("record has invalid failure evidence")
	}
	if index := record.PendingRedeliveryIndex; index != nil && (*index < 0 || *index >= max(1, record.RequestedCount) || ((record.Kind == "video" || record.Kind == "music") && *index != 0)) {
		return record, errors.New("record has invalid redelivery metadata")
	}
	if record.TaskID != "" {
		if err := validateTaskRequest(record.request()); err != nil {
			return record, errors.New("record has invalid task metadata")
		}
	}
	// Re-project any copied record before persisting or displaying it.
	safe := taskReceipt(record.request(), record.TaskID, record.Status)
	record.Model, record.OriginalOperation, record.Parameters = safe.Model, safe.OriginalOperation, safe.Parameters
	for index, path := range record.Downloaded {
		if index < 0 || index >= max(1, record.RequestedCount) || recordedDownloadFormat(record.Kind, record.TaskID, index, path) == "" {
			return record, errors.New("record has invalid download metadata")
		}
	}
	for index, proof := range record.DownloadProofs {
		if format, ok := record.Parameters["response_format"].(string); record.Kind == "music" && ok && proof.MediaType != audioMIME(format) {
			return record, errors.New("music format differs from original request")
		}
		path, ok := record.Downloaded[index]
		if !ok || !validDownloadProof(proof, record.Kind) ||
			recordedDownloadFormat(record.Kind, record.TaskID, index, path) != proof.MediaType {
			return record, errors.New("record has invalid download proof")
		}
	}
	seen := map[int]bool{}
	for _, index := range record.Delivered {
		if _, ok := record.Downloaded[index]; !ok || seen[index] {
			return record, errors.New("record has invalid delivery metadata")
		}
		seen[index] = true
	}
	return record, nil
}

func recordedDownloadFormat(kind, taskID string, index int, filePath string) string {
	name := recordedPathBase(filePath)
	if !validTaskID(taskID) || name == "" {
		return ""
	}
	stem := fmt.Sprintf("puretokens-%x", sha256.Sum256([]byte(contentPath(kind, taskID, index))))
	for _, format := range []string{"image/png", "image/jpeg", "image/webp", "image/gif", "image/avif", "video/mp4", "video/webm", "audio/mpeg", "audio/wav"} {
		if strings.HasPrefix(format, mediaKindPrefix(kind)) && name == stem+"."+extensionFor(format, kind) {
			return format
		}
	}
	return ""
}

// A copied task may name files on another OS. Validate their identity without
// resolving them locally; byte verification still requires a native absolute path.
func recordedPathBase(value string) string {
	if strings.ContainsAny(value, "\x00\r\n") {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Base(value)
	}
	if path.IsAbs(value) {
		return path.Base(value)
	}
	windows := strings.ReplaceAll(value, "\\", "/")
	if len(windows) >= 3 && windows[1:3] == ":/" &&
		(windows[0] >= 'A' && windows[0] <= 'Z' || windows[0] >= 'a' && windows[0] <= 'z') {
		return path.Base(windows)
	}
	if strings.HasPrefix(windows, "//") {
		parts := strings.Split(windows[2:], "/")
		if len(parts) >= 3 && parts[0] != "" && parts[0] != "?" && parts[0] != "." && parts[1] != "" {
			return path.Base(windows)
		}
	}
	return ""
}

func saveTaskRecord(path string, record taskRecord, create bool) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if create {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(data)
		syncErr := file.Sync()
		closeErr := file.Close()
		return errors.Join(writeErr, syncErr, closeErr)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("record changed or removed")
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".puretokens-record-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func lockTaskRecord(path string) (func(), error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("record requires an absolute path")
	}
	lock := path + ".lock"
	file, err := os.OpenFile(lock, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, errors.New("record locked or location unwritable")
	}
	if _, err := fmt.Fprintln(file, os.Getpid()); err != nil {
		file.Close()
		os.Remove(lock)
		return nil, err
	}
	if err := file.Close(); err != nil {
		os.Remove(lock)
		return nil, err
	}
	return func() { _ = os.Remove(lock) }, nil
}

func executeRecordedTask(command, path string, request taskRequest, index int, outputDir string, output io.Writer, svc service) error {
	unlock, err := lockTaskRecord(path)
	if err != nil {
		result := mergeFailure(taskReceipt(request, request.TaskID, request.TaskStatus), validationFailure("The task record is busy or its directory is unavailable. Check whether another command is using it; after confirming no command is running, remove only an abandoned adjacent .lock file."))
		result.LocalErrorCode = "task_record_unavailable"
		writeReceipt(output, result)
		return err
	}
	defer unlock()
	var record taskRecord
	if command == "submit" {
		if request.Operation == "continue" {
			writeReceipt(output, validationFailure("Use resume with the existing record instead of submit."))
			return errors.New("record submission cannot continue")
		}
		if err := validateTaskRequest(request); err != nil {
			writeReceipt(output, validationFailure("Correct the media request before creating its task record."))
			return err
		}
		if err := prepareProfileRequest(&request, svc); err != nil {
			writeReceipt(output, validationFailure(err.Error()))
			return err
		}
		record = recordFromRequest(request)
		record.SubmissionOutcome = "unknown"
		if err := saveTaskRecord(path, record, true); err != nil {
			writeReceipt(output, validationFailure("Choose a new writable task record path. Existing files are never overwritten by submit; no POST was sent."))
			return err
		}
	} else {
		record, err = loadTaskRecord(path)
		if err != nil {
			writeReceipt(output, validationFailure("The selected task record is unreadable or unsupported. Recover the original receipt; do not submit again."))
			return err
		}
		request = record.request()
		request.Index = index
		if outputDir != "" {
			request.OutputDir = outputDir
		}
		record.OutputDir = request.OutputDir
		if !validTaskID(request.TaskID) {
			result := mergeFailure(taskReceipt(request, "", record.Status), validationFailure("This record has no confirmed task ID. Its submission cannot be safely resumed."))
			result.SubmissionOutcome = record.SubmissionOutcome
			result.NextAction = "Check the original receipt or service task history. Do not automatically submit a replacement."
			writeReceipt(output, result)
			return errors.New("task record has no id")
		}
	}
	if svc.localRecoveryOnly && ((command != "resume" && command != "wait") || (!terminalSuccess(record.Status) && !terminalFailure(record.Status)) || record.ReconciliationRequired) {
		writeReceipt(output, mergeFailure(taskReceipt(request, request.TaskID, record.Status), validationFailure("The task record changed before local recovery. Keep the record and review its current state before continuing.")))
		return errors.New("local recovery state changed")
	}
	svc.downloadProofs = make(map[string]downloadProof)
	for i, proof := range record.DownloadProofs {
		svc.downloadProofs[record.Downloaded[i]] = proof
	}
	writer := &recordReceiptWriter{output: output, path: path, record: record, downloadProofs: svc.downloadProofs}
	if (command == "content" || command == "delivered" || command == "redeliver") && record.PendingRedeliveryIndex != nil && *record.PendingRedeliveryIndex != index {
		result := mergeFailure(taskReceipt(request, request.TaskID, record.Status), validationFailure("Finish the pending redelivery index before requesting another index. Keep the existing task and delivery history."))
		result.RedeliveryIndex = record.PendingRedeliveryIndex
		writeReceipt(output, result)
		return errors.New("another redelivery is pending")
	}
	if command == "redeliver" {
		return executeRedelivery(writer, svc, request, index)
	}
	if command == "content" && record.PendingRedeliveryIndex == nil {
		for _, delivered := range record.Delivered {
			if delivered == index {
				writeReceipt(output, mergeFailure(taskReceipt(request, request.TaskID, record.Status), validationFailure("This index was already delivered. Use redeliver only when the user explicitly requests another handoff of this same result.")))
				return errors.New("explicit redelivery required")
			}
		}
	}
	if command == "delivered" {
		if _, ok := record.Downloaded[index]; !ok || index < 0 {
			writeReceipt(output, mergeFailure(taskReceipt(request, request.TaskID, record.Status), validationFailure("Only a downloaded index can be marked delivered, after the host confirms attachment handoff.")))
			return errors.New("index not downloaded")
		}
		proof, known := record.DownloadProofs[index]
		if !known || !timedDownloadProof(timingsForWriter(output), record.Downloaded[index], proof) {
			writeReceipt(output, mergeFailure(taskReceipt(request, request.TaskID, record.Status), validationFailure("The downloaded file has no matching integrity proof. Preserve it and retrieve this same task into another output directory before delivery.")))
			return errors.New("delivery integrity unverified")
		}
		found := false
		for _, done := range record.Delivered {
			if done == index {
				found = true
			}
		}
		if !found {
			writer.record.Delivered = append(writer.record.Delivered, index)
			sort.Ints(writer.record.Delivered)
		}
		if writer.record.PendingRedeliveryIndex != nil && *writer.record.PendingRedeliveryIndex == index {
			writer.record.PendingRedeliveryIndex = nil
		}
		result := taskReceipt(request, request.TaskID, record.Status)
		result.DeliveryStatus = "partially_delivered"
		if writer.record.deliveryComplete() {
			result.DeliveryStatus = "delivered"
		}
		result.NextAction = "Delivery acknowledgement saved. Continue only the remaining indexes, if any."
		writer.writeReceipt(result)
		return writer.err
	}
	if (command == "resume" || command == "wait") && terminalFailure(record.Status) && !record.ReconciliationRequired {
		result := taskReceipt(request, request.TaskID, record.Status)
		result.FailureSource = "task_record"
		result.recordedFailure = record.LastFailure
		err := finishKnownTask(writer, result)
		return errors.Join(err, writer.err)
	}
	data, _ := json.Marshal(request)
	if command == "submit" {
		err = executePreparedTask(writer, svc, request)
	} else {
		if command == "resume" {
			command = "wait"
			if record.ReconciliationRequired {
				// Explicit continuation refreshes reconciliation once. Ordinary
				// polling still stops as soon as the API requires reconciliation.
				command = "status"
			}
		}
		err = executeExistingTask(command, bytes.NewReader(data), writer, svc)
	}
	return errors.Join(err, writer.err)
}

// Explicit repeat handoff, always bound to one existing task and content index.
// Local proof recovery precedes credential resolution and any possible GET.
func executeRedelivery(writer *recordReceiptWriter, svc service, request taskRequest, index int) error {
	result := taskReceipt(request, request.TaskID, writer.record.Status)
	if !terminalSuccess(result.Status) || result.ReconciliationRequired || index < 0 || index >= max(1, request.RequestedCount) ||
		(request.Kind == "image" && request.RequestedCount == 0) || ((request.Kind == "video" || request.Kind == "music") && index != 0) {
		writeReceipt(writer.output, mergeFailure(result, validationFailure("Redelivery requires a completed existing task without reconciliation and a confirmed content index. Continue that same task first; never submit a replacement.")))
		return errors.New("task cannot be redelivered")
	}
	if path, exists := writer.record.Downloaded[index]; exists {
		proof, known := writer.record.DownloadProofs[index]
		if known && validDownloadProof(proof, request.Kind) && timedDownloadProof(timingsForWriter(writer.output), path, proof) {
			writer.record.PendingRedeliveryIndex = &index
			result.DownloadedPaths = []string{path}
			result.DownloadedIndexes = []int{index}
			result.DeliveryStatus = "downloaded_awaiting_host_delivery"
			result.NextAction = "Attach this verified file again as explicitly requested. Acknowledge this index with delivered only after the new host handoff; no API request or new task was made."
			writer.writeReceipt(result)
			return writer.err
		}
	}
	// Never overwrite a changed or missing-proof artifact. download retains its
	// exclusive filename/proof checks; callers may choose another output directory.
	if info, err := os.Stat(request.OutputDir); !filepath.IsAbs(request.OutputDir) || err != nil || !info.IsDir() {
		writeReceipt(writer.output, mergeFailure(result, validationFailure("No verified local file is available. Choose an existing absolute output directory to retrieve this same task index; preserve any old files.")))
		return errors.New("redelivery directory required")
	}
	if svc.token == "" {
		token, err := credentialForHost(svc.host)
		if err != nil {
			code, message, next := credentialFailureDetails(err)
			writeReceipt(writer.output, mergeFailure(result, receipt{FailurePhase: "validation", LocalErrorCode: code, ErrorMessage: message, NextAction: next}))
			return err
		}
		defer clearString(&token)
		svc.token = token
	}
	writer.record.PendingRedeliveryIndex = &index
	request.Index = index
	data, _ := json.Marshal(request)
	err := executeExistingTask("content", bytes.NewReader(data), writer, svc)
	return errors.Join(err, writer.err)
}
