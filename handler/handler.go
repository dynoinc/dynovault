package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/handlers"
)

// DynamoDB error response format
type ddbError struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

type state struct {
	kv KVStore

	partitionKey sync.Map
}

type ddbHandler struct {
	s *state
}

func New(kv KVStore) http.Handler {
	var r http.Handler = &ddbHandler{
		s: &state{kv: kv},
	}

	r = handlers.CustomLoggingHandler(os.Stdout, r, func(writer io.Writer, params handlers.LogFormatterParams) {
		_, _ = fmt.Fprintf(
			writer,
			"[%s] %s -> %d (%vB)\n",
			params.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
			params.Request.Header.Get("x-amz-target"),
			params.StatusCode,
			params.Size,
		)
	})

	return r
}

func (d *ddbHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	target := request.Header.Get("x-amz-target")
	switch target {
	case "DynamoDB_20120810.CreateTable":
		handle(writer, request, d.s, CreateTable)
	case "DynamoDB_20120810.DeleteTable":
		handle(writer, request, d.s, DeleteTable)
	case "DynamoDB_20120810.PutItem":
		handle(writer, request, d.s, PutItem)
	case "DynamoDB_20120810.GetItem":
		handle(writer, request, d.s, GetItem)
	case "DynamoDB_20120810.DeleteItem":
		handle(writer, request, d.s, DeleteItem)
	case "DynamoDB_20120810.DescribeTable":
		handle(writer, request, d.s, DescribeTable)
	case "DynamoDB_20120810.BatchWriteItem":
		handle(writer, request, d.s, BatchWriteItem)
	case "DynamoDB_20120810.BatchGetItem":
		handle(writer, request, d.s, BatchGetItem)
	default:
		sendResponse(writer, 404, fmt.Sprintf("Unknown target method: %v", target))
	}
}

type validatable interface {
	Validate() error
}

func handle[I validatable, O any](
	writer http.ResponseWriter,
	request *http.Request,
	s *state,
	fn func(context.Context, *state, I) (O, error),
) {
	body, err := io.ReadAll(request.Body)
	_ = request.Body.Close()
	if err != nil {
		sendResponse(writer, 500, err.Error())
		return
	}

	var i I
	if err := json.Unmarshal(body, &i); err != nil {
		sendResponse(writer, 400, err.Error())
		return
	}

	if err := i.Validate(); err != nil {
		sendResponse(writer, 400, err.Error())
		return
	}

	resp, err := fn(request.Context(), s, i)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			sendDDBError(writer, 400, "com.amazonaws.dynamodb.v20120810#ResourceNotFoundException", "Requested resource not found")
		} else {
			sendDDBError(writer, 500, "com.amazonaws.dynamodb.v20120810#InternalServerError", err.Error())
		}
		return
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		sendResponse(writer, 500, err.Error())
		return
	}

	sendResponse(writer, 200, string(jsonResp))
}

func sendResponse(writer http.ResponseWriter, statusCode int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_, _ = writer.Write([]byte(message))
}

func sendDDBError(writer http.ResponseWriter, statusCode int, errType, message string) {
	resp, _ := json.Marshal(ddbError{
		Type:    errType,
		Message: message,
	})
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_, _ = writer.Write(resp)
}
