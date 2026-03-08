package responses

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type oaiToResponsesStateReasoning struct {
	ReasoningID   string
	ReasoningData string
}
type oaiToResponsesState struct {
	Seq                    int
	ResponseID             string
	Created                int64
	Started                bool
	CompletedRequestFields string
	ReasoningID            string
	ReasoningIndex         int
	// aggregation buffers for response.output
	// Per-output message text buffers by index
	MsgTextBuf   map[int]*strings.Builder
	ReasoningBuf strings.Builder
	Reasonings   []oaiToResponsesStateReasoning
	FuncArgsBuf  map[int]*strings.Builder // index -> args
	FuncNames    map[int]string           // index -> name
	FuncCallIDs  map[int]string           // index -> call_id
	// message item state per output index
	MsgItemAdded    map[int]bool // whether response.output_item.added emitted for message
	MsgContentAdded map[int]bool // whether response.content_part.added emitted for message
	MsgItemDone     map[int]bool // whether message done events were emitted
	// function item done state
	FuncArgsDone map[int]bool
	FuncItemDone map[int]bool
	// usage aggregation
	PromptTokens     int64
	CachedTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	ReasoningTokens  int64
	UsageSeen        bool
}

// responseIDCounter provides a process-wide unique counter for synthesized response identifiers.
var responseIDCounter uint64

func emitRespEvent(event string, payload string) string {
	var builder strings.Builder
	builder.Grow(len(event) + len(payload) + len("event: \ndata: "))
	builder.WriteString("event: ")
	builder.WriteString(event)
	builder.WriteString("\ndata: ")
	builder.WriteString(payload)
	return builder.String()
}

func appendSSEPrefix(buf []byte, event string) []byte {
	buf = append(buf, "event: "...)
	buf = append(buf, event...)
	buf = append(buf, "\ndata: "...)
	return buf
}

func messageItemID(responseID string, idx int) string {
	return "msg_" + responseID + "_" + strconv.Itoa(idx)
}

func reasoningItemID(responseID string, idx int) string {
	return "rs_" + responseID + "_" + strconv.Itoa(idx)
}

func functionItemID(callID string) string {
	return "fc_" + callID
}

func sortedIntKeys[V any](input map[int]V) []int {
	keys := make([]int, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func jsonArrayFromRawItems(items []string) string {
	if len(items) == 0 {
		return ""
	}
	totalLen := 2
	for idx := range items {
		totalLen += len(items[idx])
		if idx > 0 {
			totalLen++
		}
	}
	var builder strings.Builder
	builder.Grow(totalLen)
	builder.WriteByte('[')
	for idx := range items {
		if idx > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(items[idx])
	}
	builder.WriteByte(']')
	return builder.String()
}

func responseCreatedPayload(seq int, responseID string, createdAt int64) string {
	buf := make([]byte, 0, len(responseID)+160)
	buf = append(buf, `{"type":"response.created","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"response":{"id":`...)
	buf = strconv.AppendQuote(buf, responseID)
	buf = append(buf, `,"object":"response","created_at":`...)
	buf = strconv.AppendInt(buf, createdAt, 10)
	buf = append(buf, `,"status":"in_progress","background":false,"error":null,"output":[]}}`...)
	return string(buf)
}

func responseInProgressPayload(seq int, responseID string, createdAt int64) string {
	buf := make([]byte, 0, len(responseID)+128)
	buf = append(buf, `{"type":"response.in_progress","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"response":{"id":`...)
	buf = strconv.AppendQuote(buf, responseID)
	buf = append(buf, `,"object":"response","created_at":`...)
	buf = strconv.AppendInt(buf, createdAt, 10)
	buf = append(buf, `,"status":"in_progress"}}`...)
	return string(buf)
}

func outputTextDeltaPayload(seq int, itemID string, outputIndex int, delta string) string {
	buf := make([]byte, 0, len(itemID)+len(delta)+128)
	buf = append(buf, `{"type":"response.output_text.delta","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"content_index":0,"delta":`...)
	buf = strconv.AppendQuote(buf, delta)
	buf = append(buf, `,"logprobs":[]}`...)
	return string(buf)
}

func outputTextDonePayload(seq int, itemID string, outputIndex int, text string) string {
	buf := make([]byte, 0, len(itemID)+len(text)+128)
	buf = append(buf, `{"type":"response.output_text.done","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"content_index":0,"text":`...)
	buf = strconv.AppendQuote(buf, text)
	buf = append(buf, `,"logprobs":[]}`...)
	return string(buf)
}

func messageOutputItemAddedPayload(seq int, itemID string, outputIndex int) string {
	buf := make([]byte, 0, len(itemID)+152)
	buf = append(buf, `{"type":"response.output_item.added","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"item":{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"message","status":"in_progress","content":[],"role":"assistant"}}`...)
	return string(buf)
}

func contentPartAddedPayload(seq int, itemID string, outputIndex int) string {
	buf := make([]byte, 0, len(itemID)+176)
	buf = append(buf, `{"type":"response.content_part.added","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"content_index":0,"part":{"type":"output_text","annotations":[],"logprobs":[],"text":""}}`...)
	return string(buf)
}

func contentPartDonePayload(seq int, itemID string, outputIndex int, text string) string {
	buf := make([]byte, 0, len(itemID)+len(text)+176)
	buf = append(buf, `{"type":"response.content_part.done","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"content_index":0,"part":{"type":"output_text","annotations":[],"logprobs":[],"text":`...)
	buf = strconv.AppendQuote(buf, text)
	buf = append(buf, `}}`...)
	return string(buf)
}

func messageOutputItemDonePayload(seq int, itemID string, outputIndex int, text string) string {
	buf := make([]byte, 0, len(itemID)+len(text)+192)
	buf = append(buf, `{"type":"response.output_item.done","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"item":{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"message","status":"completed","content":[{"type":"output_text","annotations":[],"logprobs":[],"text":`...)
	buf = strconv.AppendQuote(buf, text)
	buf = append(buf, `}],"role":"assistant"}}`...)
	return string(buf)
}

func responseCreatedEvent(seq int, responseID string, createdAt int64) string {
	buf := make([]byte, 0, len(responseID)+192)
	buf = appendSSEPrefix(buf, "response.created")
	buf = append(buf, `{"type":"response.created","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"response":{"id":`...)
	buf = strconv.AppendQuote(buf, responseID)
	buf = append(buf, `,"object":"response","created_at":`...)
	buf = strconv.AppendInt(buf, createdAt, 10)
	buf = append(buf, `,"status":"in_progress","background":false,"error":null,"output":[]}}`...)
	return string(buf)
}

func responseInProgressEvent(seq int, responseID string, createdAt int64) string {
	buf := make([]byte, 0, len(responseID)+168)
	buf = appendSSEPrefix(buf, "response.in_progress")
	buf = append(buf, `{"type":"response.in_progress","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"response":{"id":`...)
	buf = strconv.AppendQuote(buf, responseID)
	buf = append(buf, `,"object":"response","created_at":`...)
	buf = strconv.AppendInt(buf, createdAt, 10)
	buf = append(buf, `,"status":"in_progress"}}`...)
	return string(buf)
}

func messageOutputItemAddedEvent(seq int, itemID string, outputIndex int) string {
	buf := make([]byte, 0, len(itemID)+192)
	buf = appendSSEPrefix(buf, "response.output_item.added")
	buf = append(buf, `{"type":"response.output_item.added","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"item":{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"message","status":"in_progress","content":[],"role":"assistant"}}`...)
	return string(buf)
}

func contentPartAddedEvent(seq int, itemID string, outputIndex int) string {
	buf := make([]byte, 0, len(itemID)+216)
	buf = appendSSEPrefix(buf, "response.content_part.added")
	buf = append(buf, `{"type":"response.content_part.added","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"content_index":0,"part":{"type":"output_text","annotations":[],"logprobs":[],"text":""}}`...)
	return string(buf)
}

func outputTextDeltaEvent(seq int, itemID string, outputIndex int, delta string) string {
	buf := make([]byte, 0, len(itemID)+len(delta)+168)
	buf = appendSSEPrefix(buf, "response.output_text.delta")
	buf = append(buf, `{"type":"response.output_text.delta","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"content_index":0,"delta":`...)
	buf = strconv.AppendQuote(buf, delta)
	buf = append(buf, `,"logprobs":[]}`...)
	return string(buf)
}

func outputTextDoneEvent(seq int, itemID string, outputIndex int, text string) string {
	buf := make([]byte, 0, len(itemID)+len(text)+168)
	buf = appendSSEPrefix(buf, "response.output_text.done")
	buf = append(buf, `{"type":"response.output_text.done","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"content_index":0,"text":`...)
	buf = strconv.AppendQuote(buf, text)
	buf = append(buf, `,"logprobs":[]}`...)
	return string(buf)
}

func contentPartDoneEvent(seq int, itemID string, outputIndex int, text string) string {
	buf := make([]byte, 0, len(itemID)+len(text)+216)
	buf = appendSSEPrefix(buf, "response.content_part.done")
	buf = append(buf, `{"type":"response.content_part.done","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"content_index":0,"part":{"type":"output_text","annotations":[],"logprobs":[],"text":`...)
	buf = strconv.AppendQuote(buf, text)
	buf = append(buf, `}}`...)
	return string(buf)
}

func messageOutputItemDoneEvent(seq int, itemID string, outputIndex int, text string) string {
	buf := make([]byte, 0, len(itemID)+len(text)+232)
	buf = appendSSEPrefix(buf, "response.output_item.done")
	buf = append(buf, `{"type":"response.output_item.done","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"item":{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"message","status":"completed","content":[{"type":"output_text","annotations":[],"logprobs":[],"text":`...)
	buf = strconv.AppendQuote(buf, text)
	buf = append(buf, `}],"role":"assistant"}}`...)
	return string(buf)
}

func functionOutputItemAddedPayload(seq int, itemID string, outputIndex int, callID string, name string) string {
	buf := make([]byte, 0, len(itemID)+len(callID)+len(name)+192)
	buf = append(buf, `{"type":"response.output_item.added","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"item":{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"function_call","status":"in_progress","arguments":"","call_id":`...)
	buf = strconv.AppendQuote(buf, callID)
	buf = append(buf, `,"name":`...)
	buf = strconv.AppendQuote(buf, name)
	buf = append(buf, `}}`...)
	return string(buf)
}

func functionCallArgumentsDeltaPayload(seq int, itemID string, outputIndex int, delta string) string {
	buf := make([]byte, 0, len(itemID)+len(delta)+144)
	buf = append(buf, `{"type":"response.function_call_arguments.delta","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"delta":`...)
	buf = strconv.AppendQuote(buf, delta)
	buf = append(buf, `}`...)
	return string(buf)
}

func functionCallArgumentsDonePayload(seq int, itemID string, outputIndex int, arguments string) string {
	buf := make([]byte, 0, len(itemID)+len(arguments)+152)
	buf = append(buf, `{"type":"response.function_call_arguments.done","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"arguments":`...)
	buf = strconv.AppendQuote(buf, arguments)
	buf = append(buf, `}`...)
	return string(buf)
}

func functionOutputItemDonePayload(seq int, itemID string, outputIndex int, arguments string, callID string, name string) string {
	buf := make([]byte, 0, len(itemID)+len(arguments)+len(callID)+len(name)+208)
	buf = append(buf, `{"type":"response.output_item.done","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"item":{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"function_call","status":"completed","arguments":`...)
	buf = strconv.AppendQuote(buf, arguments)
	buf = append(buf, `,"call_id":`...)
	buf = strconv.AppendQuote(buf, callID)
	buf = append(buf, `,"name":`...)
	buf = strconv.AppendQuote(buf, name)
	buf = append(buf, `}}`...)
	return string(buf)
}

func functionOutputItemAddedEvent(seq int, itemID string, outputIndex int, callID string, name string) string {
	buf := make([]byte, 0, len(itemID)+len(callID)+len(name)+224)
	buf = appendSSEPrefix(buf, "response.output_item.added")
	buf = append(buf, `{"type":"response.output_item.added","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"item":{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"function_call","status":"in_progress","arguments":"","call_id":`...)
	buf = strconv.AppendQuote(buf, callID)
	buf = append(buf, `,"name":`...)
	buf = strconv.AppendQuote(buf, name)
	buf = append(buf, `}}`...)
	return string(buf)
}

func functionCallArgumentsDeltaEvent(seq int, itemID string, outputIndex int, delta string) string {
	buf := make([]byte, 0, len(itemID)+len(delta)+176)
	buf = appendSSEPrefix(buf, "response.function_call_arguments.delta")
	buf = append(buf, `{"type":"response.function_call_arguments.delta","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"delta":`...)
	buf = strconv.AppendQuote(buf, delta)
	buf = append(buf, `}`...)
	return string(buf)
}

func functionCallArgumentsDoneEvent(seq int, itemID string, outputIndex int, arguments string) string {
	buf := make([]byte, 0, len(itemID)+len(arguments)+184)
	buf = appendSSEPrefix(buf, "response.function_call_arguments.done")
	buf = append(buf, `{"type":"response.function_call_arguments.done","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"item_id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"arguments":`...)
	buf = strconv.AppendQuote(buf, arguments)
	buf = append(buf, `}`...)
	return string(buf)
}

func functionOutputItemDoneEvent(seq int, itemID string, outputIndex int, arguments string, callID string, name string) string {
	buf := make([]byte, 0, len(itemID)+len(arguments)+len(callID)+len(name)+240)
	buf = appendSSEPrefix(buf, "response.output_item.done")
	buf = append(buf, `{"type":"response.output_item.done","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"output_index":`...)
	buf = strconv.AppendInt(buf, int64(outputIndex), 10)
	buf = append(buf, `,"item":{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"function_call","status":"completed","arguments":`...)
	buf = strconv.AppendQuote(buf, arguments)
	buf = append(buf, `,"call_id":`...)
	buf = strconv.AppendQuote(buf, callID)
	buf = append(buf, `,"name":`...)
	buf = strconv.AppendQuote(buf, name)
	buf = append(buf, `}}`...)
	return string(buf)
}

func completedReasoningItemPayload(itemID string, text string) string {
	buf := make([]byte, 0, len(itemID)+len(text)+104)
	buf = append(buf, `{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"reasoning","summary":[{"type":"summary_text","text":`...)
	buf = strconv.AppendQuote(buf, text)
	buf = append(buf, `}]}`...)
	return string(buf)
}

func completedMessageItemPayload(itemID string, text string) string {
	buf := make([]byte, 0, len(itemID)+len(text)+160)
	buf = append(buf, `{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"message","status":"completed","content":[{"type":"output_text","annotations":[],"logprobs":[],"text":`...)
	buf = strconv.AppendQuote(buf, text)
	buf = append(buf, `}],"role":"assistant"}`...)
	return string(buf)
}

func completedFunctionItemPayload(itemID string, arguments string, callID string, name string) string {
	buf := make([]byte, 0, len(itemID)+len(arguments)+len(callID)+len(name)+176)
	buf = append(buf, `{"id":`...)
	buf = strconv.AppendQuote(buf, itemID)
	buf = append(buf, `,"type":"function_call","status":"completed","arguments":`...)
	buf = strconv.AppendQuote(buf, arguments)
	buf = append(buf, `,"call_id":`...)
	buf = strconv.AppendQuote(buf, callID)
	buf = append(buf, `,"name":`...)
	buf = strconv.AppendQuote(buf, name)
	buf = append(buf, `}`...)
	return string(buf)
}

func appendJSONStringField(buf []byte, field string, value string) []byte {
	buf = append(buf, ',')
	buf = strconv.AppendQuote(buf, field)
	buf = append(buf, ':')
	buf = strconv.AppendQuote(buf, value)
	return buf
}

func appendJSONIntField(buf []byte, field string, value int64) []byte {
	buf = append(buf, ',')
	buf = strconv.AppendQuote(buf, field)
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, value, 10)
	return buf
}

func appendJSONBoolField(buf []byte, field string, value bool) []byte {
	buf = append(buf, ',')
	buf = strconv.AppendQuote(buf, field)
	buf = append(buf, ':')
	buf = strconv.AppendBool(buf, value)
	return buf
}

func appendJSONFloatField(buf []byte, field string, value float64) []byte {
	buf = append(buf, ',')
	buf = strconv.AppendQuote(buf, field)
	buf = append(buf, ':')
	buf = strconv.AppendFloat(buf, value, 'f', -1, 64)
	return buf
}

func appendJSONRawField(buf []byte, field string, raw string) []byte {
	buf = append(buf, ',')
	buf = strconv.AppendQuote(buf, field)
	buf = append(buf, ':')
	buf = append(buf, raw...)
	return buf
}

func buildCompletedRequestFieldsFromResult(req gjson.Result) string {
	if !req.Exists() {
		return ""
	}

	buf := make([]byte, 0, len(req.Raw))

	if v := req.Get("instructions"); v.Exists() {
		buf = appendJSONStringField(buf, "instructions", v.String())
	}
	if v := req.Get("max_output_tokens"); v.Exists() {
		buf = appendJSONIntField(buf, "max_output_tokens", v.Int())
	}
	if v := req.Get("max_tool_calls"); v.Exists() {
		buf = appendJSONIntField(buf, "max_tool_calls", v.Int())
	}
	if v := req.Get("model"); v.Exists() {
		buf = appendJSONStringField(buf, "model", v.String())
	}
	if v := req.Get("parallel_tool_calls"); v.Exists() {
		buf = appendJSONBoolField(buf, "parallel_tool_calls", v.Bool())
	}
	if v := req.Get("previous_response_id"); v.Exists() {
		buf = appendJSONStringField(buf, "previous_response_id", v.String())
	}
	if v := req.Get("prompt_cache_key"); v.Exists() {
		buf = appendJSONStringField(buf, "prompt_cache_key", v.String())
	}
	if v := req.Get("reasoning"); v.Exists() {
		buf = appendJSONRawField(buf, "reasoning", v.Raw)
	}
	if v := req.Get("safety_identifier"); v.Exists() {
		buf = appendJSONStringField(buf, "safety_identifier", v.String())
	}
	if v := req.Get("service_tier"); v.Exists() {
		buf = appendJSONStringField(buf, "service_tier", v.String())
	}
	if v := req.Get("store"); v.Exists() {
		buf = appendJSONBoolField(buf, "store", v.Bool())
	}
	if v := req.Get("temperature"); v.Exists() {
		buf = appendJSONFloatField(buf, "temperature", v.Float())
	}
	if v := req.Get("text"); v.Exists() {
		buf = appendJSONRawField(buf, "text", v.Raw)
	}
	if v := req.Get("tool_choice"); v.Exists() {
		buf = appendJSONRawField(buf, "tool_choice", v.Raw)
	}
	if v := req.Get("tools"); v.Exists() {
		buf = appendJSONRawField(buf, "tools", v.Raw)
	}
	if v := req.Get("top_logprobs"); v.Exists() {
		buf = appendJSONIntField(buf, "top_logprobs", v.Int())
	}
	if v := req.Get("top_p"); v.Exists() {
		buf = appendJSONFloatField(buf, "top_p", v.Float())
	}
	if v := req.Get("truncation"); v.Exists() {
		buf = appendJSONStringField(buf, "truncation", v.String())
	}
	if v := req.Get("user"); v.Exists() {
		buf = appendJSONRawField(buf, "user", v.Raw)
	}
	if v := req.Get("metadata"); v.Exists() {
		buf = appendJSONRawField(buf, "metadata", v.Raw)
	}

	return string(buf)
}

func buildCompletedRequestFields(requestRawJSON []byte) string {
	if len(requestRawJSON) == 0 {
		return ""
	}
	return buildCompletedRequestFieldsFromResult(gjson.ParseBytes(requestRawJSON))
}

func completedPayload(seq int, responseID string, created int64, requestFields string, outputArray string, promptTokens int64, cachedTokens int64, completionTokens int64, totalTokens int64, reasoningTokens int64, usageSeen bool) string {
	buf := make([]byte, 0, len(responseID)+len(requestFields)+len(outputArray)+384)
	buf = append(buf, `{"type":"response.completed","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"response":{"id":`...)
	buf = strconv.AppendQuote(buf, responseID)
	buf = append(buf, `,"object":"response","created_at":`...)
	buf = strconv.AppendInt(buf, created, 10)
	buf = append(buf, `,"status":"completed","background":false,"error":null`...)
	buf = append(buf, requestFields...)
	if outputArray != "" {
		buf = append(buf, `,"output":`...)
		buf = append(buf, outputArray...)
	}
	if usageSeen {
		total := totalTokens
		if total == 0 {
			total = promptTokens + completionTokens
		}
		buf = append(buf, `,"usage":{"input_tokens":`...)
		buf = strconv.AppendInt(buf, promptTokens, 10)
		buf = append(buf, `,"input_tokens_details":{"cached_tokens":`...)
		buf = strconv.AppendInt(buf, cachedTokens, 10)
		buf = append(buf, `},"output_tokens":`...)
		buf = strconv.AppendInt(buf, completionTokens, 10)
		if reasoningTokens > 0 {
			buf = append(buf, `,"output_tokens_details":{"reasoning_tokens":`...)
			buf = strconv.AppendInt(buf, reasoningTokens, 10)
			buf = append(buf, '}')
		}
		buf = append(buf, `,"total_tokens":`...)
		buf = strconv.AppendInt(buf, total, 10)
		buf = append(buf, '}')
	}
	buf = append(buf, `}}`...)
	return string(buf)
}

func completedEvent(seq int, responseID string, created int64, requestFields string, outputArray string, promptTokens int64, cachedTokens int64, completionTokens int64, totalTokens int64, reasoningTokens int64, usageSeen bool) string {
	buf := make([]byte, 0, len(responseID)+len(requestFields)+len(outputArray)+416)
	buf = appendSSEPrefix(buf, "response.completed")
	buf = append(buf, `{"type":"response.completed","sequence_number":`...)
	buf = strconv.AppendInt(buf, int64(seq), 10)
	buf = append(buf, `,"response":{"id":`...)
	buf = strconv.AppendQuote(buf, responseID)
	buf = append(buf, `,"object":"response","created_at":`...)
	buf = strconv.AppendInt(buf, created, 10)
	buf = append(buf, `,"status":"completed","background":false,"error":null`...)
	buf = append(buf, requestFields...)
	if outputArray != "" {
		buf = append(buf, `,"output":`...)
		buf = append(buf, outputArray...)
	}
	if usageSeen {
		total := totalTokens
		if total == 0 {
			total = promptTokens + completionTokens
		}
		buf = append(buf, `,"usage":{"input_tokens":`...)
		buf = strconv.AppendInt(buf, promptTokens, 10)
		buf = append(buf, `,"input_tokens_details":{"cached_tokens":`...)
		buf = strconv.AppendInt(buf, cachedTokens, 10)
		buf = append(buf, `},"output_tokens":`...)
		buf = strconv.AppendInt(buf, completionTokens, 10)
		if reasoningTokens > 0 {
			buf = append(buf, `,"output_tokens_details":{"reasoning_tokens":`...)
			buf = strconv.AppendInt(buf, reasoningTokens, 10)
			buf = append(buf, '}')
		}
		buf = append(buf, `,"total_tokens":`...)
		buf = strconv.AppendInt(buf, total, 10)
		buf = append(buf, '}')
	}
	buf = append(buf, `}}`...)
	return string(buf)
}

func nonStreamResponsePayload(id string, created int64, requestFields string, outputArray string, usage gjson.Result) string {
	buf := make([]byte, 0, len(id)+len(requestFields)+len(outputArray)+256)
	buf = append(buf, `{"id":`...)
	buf = strconv.AppendQuote(buf, id)
	buf = append(buf, `,"object":"response","created_at":`...)
	buf = strconv.AppendInt(buf, created, 10)
	buf = append(buf, `,"status":"completed","background":false,"error":null,"incomplete_details":null`...)
	buf = append(buf, requestFields...)
	if outputArray != "" {
		buf = append(buf, `,"output":`...)
		buf = append(buf, outputArray...)
	}
	if usage.Exists() {
		if usage.Get("prompt_tokens").Exists() || usage.Get("completion_tokens").Exists() || usage.Get("total_tokens").Exists() {
			buf = append(buf, `,"usage":{"input_tokens":`...)
			buf = strconv.AppendInt(buf, usage.Get("prompt_tokens").Int(), 10)
			if cached := usage.Get("prompt_tokens_details.cached_tokens"); cached.Exists() {
				buf = append(buf, `,"input_tokens_details":{"cached_tokens":`...)
				buf = strconv.AppendInt(buf, cached.Int(), 10)
				buf = append(buf, `}`...)
			}
			buf = append(buf, `,"output_tokens":`...)
			buf = strconv.AppendInt(buf, usage.Get("completion_tokens").Int(), 10)
			if reasoning := usage.Get("output_tokens_details.reasoning_tokens"); reasoning.Exists() {
				buf = append(buf, `,"output_tokens_details":{"reasoning_tokens":`...)
				buf = strconv.AppendInt(buf, reasoning.Int(), 10)
				buf = append(buf, `}`...)
			}
			buf = append(buf, `,"total_tokens":`...)
			buf = strconv.AppendInt(buf, usage.Get("total_tokens").Int(), 10)
			buf = append(buf, `}`...)
		} else {
			buf = append(buf, `,"usage":`...)
			buf = append(buf, usage.Raw...)
		}
	}
	buf = append(buf, '}')
	return string(buf)
}

func appendMessageDoneEvents(out []string, responseID string, outputIndex int, text string, nextSeq func() int) []string {
	itemID := messageItemID(responseID, outputIndex)
	out = append(out, outputTextDoneEvent(nextSeq(), itemID, outputIndex, text))
	out = append(out, contentPartDoneEvent(nextSeq(), itemID, outputIndex, text))
	out = append(out, messageOutputItemDoneEvent(nextSeq(), itemID, outputIndex, text))
	return out
}

func appendFunctionDoneEvents(out []string, callID string, outputIndex int, arguments string, name string, nextSeq func() int) []string {
	itemID := functionItemID(callID)
	out = append(out, functionCallArgumentsDoneEvent(nextSeq(), itemID, outputIndex, arguments))
	out = append(out, functionOutputItemDoneEvent(nextSeq(), itemID, outputIndex, arguments, callID, name))
	return out
}

// ConvertOpenAIChatCompletionsResponseToOpenAIResponses converts OpenAI Chat Completions streaming chunks
// to OpenAI Responses SSE events (response.*).
func ConvertOpenAIChatCompletionsResponseToOpenAIResponses(ctx context.Context, modelName string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, param *any) []string {
	if *param == nil {
		*param = &oaiToResponsesState{
			FuncArgsBuf:     make(map[int]*strings.Builder),
			FuncNames:       make(map[int]string),
			FuncCallIDs:     make(map[int]string),
			MsgTextBuf:      make(map[int]*strings.Builder),
			MsgItemAdded:    make(map[int]bool),
			MsgContentAdded: make(map[int]bool),
			MsgItemDone:     make(map[int]bool),
			FuncArgsDone:    make(map[int]bool),
			FuncItemDone:    make(map[int]bool),
			Reasonings:      make([]oaiToResponsesStateReasoning, 0),
		}
	}
	st := (*param).(*oaiToResponsesState)

	if bytes.HasPrefix(rawJSON, []byte("data:")) {
		rawJSON = bytes.TrimSpace(rawJSON[5:])
	}

	rawJSON = bytes.TrimSpace(rawJSON)
	if len(rawJSON) == 0 {
		return []string{}
	}
	if bytes.Equal(rawJSON, []byte("[DONE]")) {
		return []string{}
	}

	root := gjson.ParseBytes(rawJSON)
	obj := root.Get("object")
	if obj.Exists() && obj.String() != "" && obj.String() != "chat.completion.chunk" {
		return []string{}
	}
	if !root.Get("choices").Exists() || !root.Get("choices").IsArray() {
		return []string{}
	}

	if usage := root.Get("usage"); usage.Exists() {
		if v := usage.Get("prompt_tokens"); v.Exists() {
			st.PromptTokens = v.Int()
			st.UsageSeen = true
		}
		if v := usage.Get("prompt_tokens_details.cached_tokens"); v.Exists() {
			st.CachedTokens = v.Int()
			st.UsageSeen = true
		}
		if v := usage.Get("completion_tokens"); v.Exists() {
			st.CompletionTokens = v.Int()
			st.UsageSeen = true
		} else if v := usage.Get("output_tokens"); v.Exists() {
			st.CompletionTokens = v.Int()
			st.UsageSeen = true
		}
		if v := usage.Get("output_tokens_details.reasoning_tokens"); v.Exists() {
			st.ReasoningTokens = v.Int()
			st.UsageSeen = true
		} else if v := usage.Get("completion_tokens_details.reasoning_tokens"); v.Exists() {
			st.ReasoningTokens = v.Int()
			st.UsageSeen = true
		}
		if v := usage.Get("total_tokens"); v.Exists() {
			st.TotalTokens = v.Int()
			st.UsageSeen = true
		}
	}

	nextSeq := func() int { st.Seq++; return st.Seq }
	out := make([]string, 0, 8)

	if !st.Started {
		st.ResponseID = root.Get("id").String()
		st.Created = root.Get("created").Int()
		st.CompletedRequestFields = buildCompletedRequestFields(requestRawJSON)
		// reset aggregation state for a new streaming response
		st.MsgTextBuf = make(map[int]*strings.Builder)
		st.ReasoningBuf.Reset()
		st.ReasoningID = ""
		st.ReasoningIndex = 0
		st.FuncArgsBuf = make(map[int]*strings.Builder)
		st.FuncNames = make(map[int]string)
		st.FuncCallIDs = make(map[int]string)
		st.MsgItemAdded = make(map[int]bool)
		st.MsgContentAdded = make(map[int]bool)
		st.MsgItemDone = make(map[int]bool)
		st.FuncArgsDone = make(map[int]bool)
		st.FuncItemDone = make(map[int]bool)
		st.PromptTokens = 0
		st.CachedTokens = 0
		st.CompletionTokens = 0
		st.TotalTokens = 0
		st.ReasoningTokens = 0
		st.UsageSeen = false
		out = append(out, responseCreatedEvent(nextSeq(), st.ResponseID, st.Created))
		out = append(out, responseInProgressEvent(nextSeq(), st.ResponseID, st.Created))
		st.Started = true
	}

	stopReasoning := func(text string) {
		// Emit reasoning done events
		textDone := `{"type":"response.reasoning_summary_text.done","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"text":""}`
		textDone, _ = sjson.Set(textDone, "sequence_number", nextSeq())
		textDone, _ = sjson.Set(textDone, "item_id", st.ReasoningID)
		textDone, _ = sjson.Set(textDone, "output_index", st.ReasoningIndex)
		textDone, _ = sjson.Set(textDone, "text", text)
		out = append(out, emitRespEvent("response.reasoning_summary_text.done", textDone))
		partDone := `{"type":"response.reasoning_summary_part.done","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"part":{"type":"summary_text","text":""}}`
		partDone, _ = sjson.Set(partDone, "sequence_number", nextSeq())
		partDone, _ = sjson.Set(partDone, "item_id", st.ReasoningID)
		partDone, _ = sjson.Set(partDone, "output_index", st.ReasoningIndex)
		partDone, _ = sjson.Set(partDone, "part.text", text)
		out = append(out, emitRespEvent("response.reasoning_summary_part.done", partDone))
		outputItemDone := `{"type":"response.output_item.done","item":{"id":"","type":"reasoning","encrypted_content":"","summary":[{"type":"summary_text","text":""}]},"output_index":0,"sequence_number":0}`
		outputItemDone, _ = sjson.Set(outputItemDone, "sequence_number", nextSeq())
		outputItemDone, _ = sjson.Set(outputItemDone, "item.id", st.ReasoningID)
		outputItemDone, _ = sjson.Set(outputItemDone, "output_index", st.ReasoningIndex)
		outputItemDone, _ = sjson.Set(outputItemDone, "item.summary.text", text)
		out = append(out, emitRespEvent("response.output_item.done", outputItemDone))

		st.Reasonings = append(st.Reasonings, oaiToResponsesStateReasoning{ReasoningID: st.ReasoningID, ReasoningData: text})
		st.ReasoningID = ""
	}

	// choices[].delta content / tool_calls / reasoning_content
	if choices := root.Get("choices"); choices.Exists() && choices.IsArray() {
		choices.ForEach(func(_, choice gjson.Result) bool {
			idx := int(choice.Get("index").Int())
			delta := choice.Get("delta")
			if delta.Exists() {
				if c := delta.Get("content"); c.Exists() && c.String() != "" {
					contentText := c.String()
					// Ensure the message item and its first content part are announced before any text deltas
					if st.ReasoningID != "" {
						stopReasoning(st.ReasoningBuf.String())
						st.ReasoningBuf.Reset()
					}
					itemID := messageItemID(st.ResponseID, idx)
					if !st.MsgItemAdded[idx] {
						out = append(out, messageOutputItemAddedEvent(nextSeq(), itemID, idx))
						st.MsgItemAdded[idx] = true
					}
					if !st.MsgContentAdded[idx] {
						out = append(out, contentPartAddedEvent(nextSeq(), itemID, idx))
						st.MsgContentAdded[idx] = true
					}

					out = append(out, outputTextDeltaEvent(nextSeq(), itemID, idx, contentText))
					// aggregate for response.output
					if st.MsgTextBuf[idx] == nil {
						st.MsgTextBuf[idx] = &strings.Builder{}
					}
					st.MsgTextBuf[idx].WriteString(contentText)
				}

				// reasoning_content (OpenAI reasoning incremental text)
				if rc := delta.Get("reasoning_content"); rc.Exists() && rc.String() != "" {
					reasoningText := rc.String()
					// On first appearance, add reasoning item and part
					if st.ReasoningID == "" {
						st.ReasoningID = reasoningItemID(st.ResponseID, idx)
						st.ReasoningIndex = idx
						item := `{"type":"response.output_item.added","sequence_number":0,"output_index":0,"item":{"id":"","type":"reasoning","status":"in_progress","summary":[]}}`
						item, _ = sjson.Set(item, "sequence_number", nextSeq())
						item, _ = sjson.Set(item, "output_index", idx)
						item, _ = sjson.Set(item, "item.id", st.ReasoningID)
						out = append(out, emitRespEvent("response.output_item.added", item))
						part := `{"type":"response.reasoning_summary_part.added","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"part":{"type":"summary_text","text":""}}`
						part, _ = sjson.Set(part, "sequence_number", nextSeq())
						part, _ = sjson.Set(part, "item_id", st.ReasoningID)
						part, _ = sjson.Set(part, "output_index", st.ReasoningIndex)
						out = append(out, emitRespEvent("response.reasoning_summary_part.added", part))
					}
					// Append incremental text to reasoning buffer
					st.ReasoningBuf.WriteString(reasoningText)
					msg := `{"type":"response.reasoning_summary_text.delta","sequence_number":0,"item_id":"","output_index":0,"summary_index":0,"delta":""}`
					msg, _ = sjson.Set(msg, "sequence_number", nextSeq())
					msg, _ = sjson.Set(msg, "item_id", st.ReasoningID)
					msg, _ = sjson.Set(msg, "output_index", st.ReasoningIndex)
					msg, _ = sjson.Set(msg, "delta", reasoningText)
					out = append(out, emitRespEvent("response.reasoning_summary_text.delta", msg))
				}

				// tool calls
				if tcs := delta.Get("tool_calls"); tcs.Exists() && tcs.IsArray() {
					if st.ReasoningID != "" {
						stopReasoning(st.ReasoningBuf.String())
						st.ReasoningBuf.Reset()
					}
					// Before emitting any function events, if a message is open for this index,
					// close its text/content to match Codex expected ordering.
					if st.MsgItemAdded[idx] && !st.MsgItemDone[idx] {
						fullText := ""
						if b := st.MsgTextBuf[idx]; b != nil {
							fullText = b.String()
						}
						out = appendMessageDoneEvents(out, st.ResponseID, idx, fullText, nextSeq)
						st.MsgItemDone[idx] = true
					}

					// Only emit item.added once per tool call and preserve call_id across chunks.
					firstToolCall := tcs.Get("0")
					functionCall := firstToolCall.Get("function")
					newCallID := firstToolCall.Get("id").String()
					nameChunk := functionCall.Get("name").String()
					if nameChunk != "" {
						st.FuncNames[idx] = nameChunk
					}
					existingCallID := st.FuncCallIDs[idx]
					effectiveCallID := existingCallID
					shouldEmitItem := false
					if existingCallID == "" && newCallID != "" {
						// First time seeing a valid call_id for this index
						effectiveCallID = newCallID
						st.FuncCallIDs[idx] = newCallID
						shouldEmitItem = true
					}

					if shouldEmitItem && effectiveCallID != "" {
						name := st.FuncNames[idx]
						out = append(out, functionOutputItemAddedEvent(nextSeq(), functionItemID(effectiveCallID), idx, effectiveCallID, name))
					}

					// Ensure args buffer exists for this index
					if st.FuncArgsBuf[idx] == nil {
						st.FuncArgsBuf[idx] = &strings.Builder{}
					}

					// Append arguments delta if available and we have a valid call_id to reference
					argsText := functionCall.Get("arguments").String()
					if argsText != "" {
						// Prefer an already known call_id; fall back to newCallID if first time
						refCallID := st.FuncCallIDs[idx]
						if refCallID == "" {
							refCallID = newCallID
						}
						if refCallID != "" {
							out = append(out, functionCallArgumentsDeltaEvent(nextSeq(), functionItemID(refCallID), idx, argsText))
						}
						st.FuncArgsBuf[idx].WriteString(argsText)
					}
				}
			}

			// finish_reason triggers finalization, including text done/content done/item done,
			// reasoning done/part.done, function args done/item done, and completed
			if fr := choice.Get("finish_reason"); fr.Exists() && fr.String() != "" {
				// Emit message done events for all indices that started a message
				if len(st.MsgItemAdded) > 0 {
					// sort indices for deterministic order
					idxs := sortedIntKeys(st.MsgItemAdded)
					for _, i := range idxs {
						if st.MsgItemAdded[i] && !st.MsgItemDone[i] {
							fullText := ""
							if b := st.MsgTextBuf[i]; b != nil {
								fullText = b.String()
							}
							out = appendMessageDoneEvents(out, st.ResponseID, i, fullText, nextSeq)
							st.MsgItemDone[i] = true
						}
					}
				}

				if st.ReasoningID != "" {
					stopReasoning(st.ReasoningBuf.String())
					st.ReasoningBuf.Reset()
				}

				// Emit function call done events for any active function calls
				if len(st.FuncCallIDs) > 0 {
					idxs := sortedIntKeys(st.FuncCallIDs)
					for _, i := range idxs {
						callID := st.FuncCallIDs[i]
						if callID == "" || st.FuncItemDone[i] {
							continue
						}
						args := "{}"
						if b := st.FuncArgsBuf[i]; b != nil && b.Len() > 0 {
							args = b.String()
						}
						out = appendFunctionDoneEvents(out, callID, i, args, st.FuncNames[i], nextSeq)
						st.FuncItemDone[i] = true
						st.FuncArgsDone[i] = true
					}
				}
				// Build response.output using aggregated buffers
				outputItems := make([]string, 0, len(st.Reasonings)+len(st.MsgItemAdded)+len(st.FuncArgsBuf))
				if len(st.Reasonings) > 0 {
					for _, r := range st.Reasonings {
						outputItems = append(outputItems, completedReasoningItemPayload(r.ReasoningID, r.ReasoningData))
					}
				}
				// Append message items in ascending index order
				if len(st.MsgItemAdded) > 0 {
					midxs := sortedIntKeys(st.MsgItemAdded)
					for _, i := range midxs {
						txt := ""
						if b := st.MsgTextBuf[i]; b != nil {
							txt = b.String()
						}
						outputItems = append(outputItems, completedMessageItemPayload(messageItemID(st.ResponseID, i), txt))
					}
				}
				if len(st.FuncArgsBuf) > 0 {
					idxs := sortedIntKeys(st.FuncArgsBuf)
					for _, i := range idxs {
						args := ""
						if b := st.FuncArgsBuf[i]; b != nil {
							args = b.String()
						}
						callID := st.FuncCallIDs[i]
						name := st.FuncNames[i]
						outputItems = append(outputItems, completedFunctionItemPayload(functionItemID(callID), args, callID, name))
					}
				}
				outputArray := jsonArrayFromRawItems(outputItems)
				out = append(out, completedEvent(nextSeq(), st.ResponseID, st.Created, st.CompletedRequestFields, outputArray, st.PromptTokens, st.CachedTokens, st.CompletionTokens, st.TotalTokens, st.ReasoningTokens, st.UsageSeen))
			}

			return true
		})
	}

	return out
}

// ConvertOpenAIChatCompletionsResponseToOpenAIResponsesNonStream builds a single Responses JSON
// from a non-streaming OpenAI Chat Completions response.
func ConvertOpenAIChatCompletionsResponseToOpenAIResponsesNonStream(_ context.Context, _ string, originalRequestRawJSON, requestRawJSON, rawJSON []byte, _ *any) string {
	root := gjson.ParseBytes(rawJSON)

	// id: use provider id if present, otherwise synthesize
	id := root.Get("id").String()
	if id == "" {
		id = fmt.Sprintf("resp_%x_%d", time.Now().UnixNano(), atomic.AddUint64(&responseIDCounter, 1))
	}

	// created_at: map from chat.completion created
	created := root.Get("created").Int()
	if created == 0 {
		created = time.Now().Unix()
	}
	var requestRoot gjson.Result
	if len(requestRawJSON) > 0 {
		requestRoot = gjson.ParseBytes(requestRawJSON)
	}
	requestFields := buildCompletedRequestFieldsFromResult(requestRoot)
	if len(requestRawJSON) > 0 {
		if !requestRoot.Get("max_output_tokens").Exists() {
			if v := requestRoot.Get("max_tokens"); v.Exists() {
				requestFields = string(appendJSONIntField([]byte(requestFields), "max_output_tokens", v.Int()))
			}
		}
		if !requestRoot.Get("model").Exists() {
			if v := root.Get("model"); v.Exists() {
				requestFields = string(appendJSONStringField([]byte(requestFields), "model", v.String()))
			}
		}
	} else if v := root.Get("model"); v.Exists() {
		requestFields = string(appendJSONStringField(nil, "model", v.String()))
	}

	// Build output list from choices[...]
	outputItems := make([]string, 0, 4)
	// Detect and capture reasoning content if present
	rcText := gjson.GetBytes(rawJSON, "choices.0.message.reasoning_content").String()
	includeReasoning := rcText != ""
	if !includeReasoning && len(requestRawJSON) > 0 {
		includeReasoning = gjson.GetBytes(requestRawJSON, "reasoning").Exists()
	}
	if includeReasoning {
		rid := id
		if strings.HasPrefix(rid, "resp_") {
			rid = strings.TrimPrefix(rid, "resp_")
		}
		outputItems = append(outputItems, completedReasoningItemPayload("rs_"+rid, rcText))
	}

	if choices := root.Get("choices"); choices.Exists() && choices.IsArray() {
		choices.ForEach(func(_, choice gjson.Result) bool {
			msg := choice.Get("message")
			if msg.Exists() {
				// Text message part
				if c := msg.Get("content"); c.Exists() && c.String() != "" {
					outputItems = append(outputItems, completedMessageItemPayload("msg_"+id+"_"+strconv.Itoa(int(choice.Get("index").Int())), c.String()))
				}

				// Function/tool calls
				if tcs := msg.Get("tool_calls"); tcs.Exists() && tcs.IsArray() {
					tcs.ForEach(func(_, tc gjson.Result) bool {
						callID := tc.Get("id").String()
						name := tc.Get("function.name").String()
						args := tc.Get("function.arguments").String()
						outputItems = append(outputItems, completedFunctionItemPayload("fc_"+callID, args, callID, name))
						return true
					})
				}
			}
			return true
		})
	}

	return nonStreamResponsePayload(id, created, requestFields, jsonArrayFromRawItems(outputItems), root.Get("usage"))
}
