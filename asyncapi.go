package shiftapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	spec "github.com/swaggest/go-asyncapi/spec-2.4.0"
)

// addWSChannel registers a WebSocket endpoint in the AsyncAPI spec.
// It creates a channel with subscribe (server→client) and publish
// (client→server) operations, and registers schemas in both the
// AsyncAPI and OpenAPI specs.
func (a *API) addWSChannel(
	path string,
	sendType, recvType reflect.Type,
	sendVariants []WSMessageVariant,
	recvVariants []WSMessageVariant,
	info *RouteInfo,
	pathFields map[string]reflect.StructField,
) error {
	channelItem := spec.ChannelItem{}

	// Path parameters.
	for _, match := range pathParamRe.FindAllStringSubmatch(path, -1) {
		name := match[1]
		paramSchema := map[string]interface{}{"type": "string"}
		if field, ok := pathFields[name]; ok {
			paramSchema = goTypeToJSONSchema(field.Type)
		}
		if channelItem.Parameters == nil {
			channelItem.Parameters = make(map[string]spec.Parameter)
		}
		channelItem.Parameters[name] = spec.Parameter{Schema: paramSchema}
	}

	// Subscribe = what clients receive = our Send type (server→client).
	if sendType != nil || len(sendVariants) > 0 {
		subMsg, err := a.buildWSMessage(sendType, sendVariants)
		if err != nil {
			return fmt.Errorf("send message: %w", err)
		}
		channelItem.Subscribe = &spec.Operation{
			ID:      operationID("subscribe", path),
			Message: subMsg,
		}
	}

	// Publish = what clients send = our Recv type (client→server).
	if recvType != nil || len(recvVariants) > 0 {
		pubMsg, err := a.buildWSMessage(recvType, recvVariants)
		if err != nil {
			return fmt.Errorf("recv message: %w", err)
		}
		channelItem.Publish = &spec.Operation{
			ID:      operationID("publish", path),
			Message: pubMsg,
		}
	}

	if info != nil {
		channelItem.Description = info.Description
		for _, op := range []*spec.Operation{channelItem.Subscribe, channelItem.Publish} {
			if op == nil {
				continue
			}
			op.Summary = info.Summary
			for _, t := range info.Tags {
				op.Tags = append(op.Tags, spec.Tag{Name: t})
			}
		}
	}

	a.asyncSpec.WithChannelsItem(path, channelItem)
	return nil
}

// buildWSMessage builds an AsyncAPI Message for a single direction of a
// WebSocket channel. For single-type endpoints it produces a direct message
// reference. For multi-type endpoints (variants) it produces a oneOf wrapper.
func (a *API) buildWSMessage(t reflect.Type, variants []WSMessageVariant) (*spec.Message, error) {
	if len(variants) > 0 {
		return a.buildWSOneOfMessage(variants)
	}
	return a.buildWSSingleMessage(t)
}

// buildWSSingleMessage creates a message with an inline payload reference to
// the schema in components/schemas. No components/messages entry is created
// for the simple single-type case.
func (a *API) buildWSSingleMessage(t reflect.Type) (*spec.Message, error) {
	name, err := a.registerWSSchema(t)
	if err != nil {
		return nil, err
	}

	msg := &spec.Message{}
	msg.OneOf1Ens().WithMessageEntity(spec.MessageEntity{
		Name:    name,
		Payload: map[string]interface{}{"$ref": "#/components/schemas/" + name},
	})
	return msg, nil
}

// buildWSOneOfMessage creates a oneOf message from discriminated variants.
// Each variant gets an envelope schema {type, data} registered in components.
func (a *API) buildWSOneOfMessage(variants []WSMessageVariant) (*spec.Message, error) {
	var msgs []spec.Message

	for _, v := range variants {
		// Register the payload schema.
		payloadName, err := a.registerWSSchema(v.messagePayloadType())
		if err != nil {
			return nil, err
		}

		// Build envelope schema: {"type": name, "data": payload}
		envelopeName := v.messageName() + "_" + payloadName
		envelopeSchema := map[string]interface{}{
			"type":     "object",
			"required": []string{"type", "data"},
			"properties": map[string]interface{}{
				"type": map[string]interface{}{
					"type": "string",
					"enum": []interface{}{v.messageName()},
				},
				"data": map[string]interface{}{
					"$ref": "#/components/schemas/" + payloadName,
				},
			},
		}
		a.asyncSpec.ComponentsEns().WithSchemasItem(envelopeName, envelopeSchema)

		// Register envelope message in components.
		envelopeMsg := spec.Message{}
		envelopeMsg.OneOf1Ens().WithMessageEntity(spec.MessageEntity{
			Name:    v.messageName(),
			Payload: map[string]interface{}{"$ref": "#/components/schemas/" + envelopeName},
		})
		a.asyncSpec.ComponentsEns().WithMessagesItem(envelopeName, envelopeMsg)

		msgs = append(msgs, spec.Message{
			Reference: &spec.Reference{Ref: "#/components/messages/" + envelopeName},
		})
	}

	result := &spec.Message{}
	result.OneOf1Ens().WithOneOf0(spec.MessageOneOf1OneOf0{OneOf: msgs})
	return result, nil
}

// registerWSSchema registers a Go type as a schema in both the AsyncAPI and
// OpenAPI component sections, returning the schema name.
func (a *API) registerWSSchema(t reflect.Type) (string, error) {
	schema, err := a.generateSchemaRef(t)
	if err != nil {
		return "", err
	}
	if schema == nil {
		return "", fmt.Errorf("could not generate schema for %v", t)
	}

	name := schema.Ref
	if name == "" {
		name = t.Name()
	}

	// Register in OpenAPI components (for openapi-typescript type generation).
	if schema.Ref != "" && len(schema.Value.Properties) > 0 {
		a.spec.Components.Schemas[name] = &openapi3.SchemaRef{Value: schema.Value}
	}

	// Register in AsyncAPI components.
	asyncSchema, err := openAPISchemaToMap(schema)
	if err != nil {
		return "", err
	}
	a.asyncSpec.ComponentsEns().WithSchemasItem(name, asyncSchema)

	return name, nil
}

// openAPISchemaToMap converts a kin-openapi SchemaRef to a plain map for use
// in the AsyncAPI spec's JSON Schema fields.
func openAPISchemaToMap(s *openapi3.SchemaRef) (map[string]interface{}, error) {
	b, err := json.Marshal(s.Value)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// goTypeToJSONSchema returns a minimal JSON Schema map for a scalar Go type.
func goTypeToJSONSchema(t reflect.Type) map[string]interface{} {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	default:
		return map[string]interface{}{"type": "string"}
	}
}

func (a *API) serveAsyncSpec(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(a.asyncSpec); err != nil {
		http.Error(w, "error encoding async spec", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}
