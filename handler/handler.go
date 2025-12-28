package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/gorilla/handlers"
)

// DynamoDB error response format
type ddbError struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

// jsonOpts returns configured JSON options for AWS SDK v2 compatibility
func jsonOpts() json.Options {
	return json.JoinOptions(
		// Marshal time.Time as Unix epoch
		json.WithMarshalers(json.JoinMarshalers(
			json.MarshalToFunc(func(enc *jsontext.Encoder, t time.Time) error {
				return enc.WriteToken(jsontext.Float(float64(t.Unix())))
			}),
			// Skip middleware.Metadata (has no exported fields)
			json.MarshalToFunc(func(enc *jsontext.Encoder, _ middleware.Metadata) error {
				return enc.WriteValue([]byte("{}"))
			}),
			// Marshal types.AttributeValue interface
			json.MarshalToFunc(marshalAttributeValue),
		)),
		json.WithUnmarshalers(json.JoinUnmarshalers(
			json.UnmarshalFromFunc(func(dec *jsontext.Decoder, t *time.Time) error {
				var epoch float64
				if err := json.UnmarshalDecode(dec, &epoch); err != nil {
					return err
				}
				*t = time.Unix(int64(epoch), 0)
				return nil
			}),
			// Skip middleware.Metadata
			json.UnmarshalFromFunc(func(dec *jsontext.Decoder, _ *middleware.Metadata) error {
				_, err := dec.ReadValue()
				return err
			}),
			// Unmarshal types.AttributeValue interface
			json.UnmarshalFromFunc(unmarshalAttributeValue),
		)),
	)
}

// marshalAttributeValue handles DynamoDB AttributeValue interface marshaling
func marshalAttributeValue(enc *jsontext.Encoder, av types.AttributeValue) error {
	switch v := av.(type) {
	case *types.AttributeValueMemberS:
		return enc.WriteValue([]byte(fmt.Sprintf(`{"S":%q}`, v.Value)))
	case *types.AttributeValueMemberN:
		return enc.WriteValue([]byte(fmt.Sprintf(`{"N":%q}`, v.Value)))
	case *types.AttributeValueMemberB:
		b, _ := json.Marshal(v.Value)
		return enc.WriteValue([]byte(fmt.Sprintf(`{"B":%s}`, b)))
	case *types.AttributeValueMemberSS:
		b, _ := json.Marshal(v.Value)
		return enc.WriteValue([]byte(fmt.Sprintf(`{"SS":%s}`, b)))
	case *types.AttributeValueMemberNS:
		b, _ := json.Marshal(v.Value)
		return enc.WriteValue([]byte(fmt.Sprintf(`{"NS":%s}`, b)))
	case *types.AttributeValueMemberBS:
		b, _ := json.Marshal(v.Value)
		return enc.WriteValue([]byte(fmt.Sprintf(`{"BS":%s}`, b)))
	case *types.AttributeValueMemberM:
		b, _ := json.Marshal(v.Value, jsonOpts())
		return enc.WriteValue([]byte(fmt.Sprintf(`{"M":%s}`, b)))
	case *types.AttributeValueMemberL:
		b, _ := json.Marshal(v.Value, jsonOpts())
		return enc.WriteValue([]byte(fmt.Sprintf(`{"L":%s}`, b)))
	case *types.AttributeValueMemberNULL:
		return enc.WriteValue([]byte(`{"NULL":true}`))
	case *types.AttributeValueMemberBOOL:
		return enc.WriteValue([]byte(fmt.Sprintf(`{"BOOL":%t}`, v.Value)))
	default:
		return enc.WriteValue([]byte("null"))
	}
}

// unmarshalAttributeValue handles DynamoDB AttributeValue interface unmarshaling
func unmarshalAttributeValue(dec *jsontext.Decoder, av *types.AttributeValue) error {
	// Read the raw JSON value
	val, err := dec.ReadValue()
	if err != nil {
		return err
	}

	// Parse to determine which type
	var raw map[string]jsontext.Value
	if err := json.Unmarshal(val, &raw); err != nil {
		return err
	}

	// DynamoDB AttributeValue has exactly one key indicating type
	for k, v := range raw {
		switch k {
		case "S":
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				return err
			}
			*av = &types.AttributeValueMemberS{Value: s}
		case "N":
			var n string
			if err := json.Unmarshal(v, &n); err != nil {
				return err
			}
			*av = &types.AttributeValueMemberN{Value: n}
		case "B":
			var b []byte
			if err := json.Unmarshal(v, &b); err != nil {
				return err
			}
			*av = &types.AttributeValueMemberB{Value: b}
		case "SS":
			var ss []string
			if err := json.Unmarshal(v, &ss); err != nil {
				return err
			}
			*av = &types.AttributeValueMemberSS{Value: ss}
		case "NS":
			var ns []string
			if err := json.Unmarshal(v, &ns); err != nil {
				return err
			}
			*av = &types.AttributeValueMemberNS{Value: ns}
		case "BS":
			var bs [][]byte
			if err := json.Unmarshal(v, &bs); err != nil {
				return err
			}
			*av = &types.AttributeValueMemberBS{Value: bs}
		case "M":
			var m map[string]types.AttributeValue
			if err := json.Unmarshal(v, &m, jsonOpts()); err != nil {
				return err
			}
			*av = &types.AttributeValueMemberM{Value: m}
		case "L":
			var l []types.AttributeValue
			if err := json.Unmarshal(v, &l, jsonOpts()); err != nil {
				return err
			}
			*av = &types.AttributeValueMemberL{Value: l}
		case "NULL":
			*av = &types.AttributeValueMemberNULL{Value: true}
		case "BOOL":
			var b bool
			if err := json.Unmarshal(v, &b); err != nil {
				return err
			}
			*av = &types.AttributeValueMemberBOOL{Value: b}
		}
		break // Only one key per AttributeValue
	}
	return nil
}

type state struct {
	kv KVStore

	keySchema sync.Map
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

func handle[I any, O any](
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
	if err := json.Unmarshal(body, &i, jsonOpts()); err != nil {
		sendDDBError(writer, 400, "com.amazonaws.dynamodb.v20120810#SerializationException", err.Error())
		return
	}

	resp, err := fn(request.Context(), s, i)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			sendDDBError(writer, 400, "com.amazonaws.dynamodb.v20120810#ResourceNotFoundException", "Requested resource not found")
		} else if errors.Is(err, ErrAlreadyExists) {
			sendDDBError(writer, 400, "com.amazonaws.dynamodb.v20120810#ResourceInUseException", "Table already exists")
		} else {
			sendDDBError(writer, 500, "com.amazonaws.dynamodb.v20120810#InternalServerError", err.Error())
		}
		return
	}

	jsonResp, err := json.Marshal(resp, jsonOpts())
	if err != nil {
		sendDDBError(writer, 500, "com.amazonaws.dynamodb.v20120810#InternalServerError", err.Error())
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
