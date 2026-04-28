package watp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type Field struct {
	Key   string
	Value any
}

func KV(key string, value any) Field {
	return Field{Key: key, Value: value}
}

type Payload struct {
	fields []Field
}

func NewPayload(fields ...Field) Payload {
	return Payload{fields: append([]Field(nil), fields...)}
}

func (p Payload) Clone() Payload {
	return Payload{fields: append([]Field(nil), p.fields...)}
}

func (p Payload) Fields() []Field {
	return append([]Field(nil), p.fields...)
}

func (p Payload) Len() int {
	return len(p.fields)
}

func (p Payload) Get(key string) (any, bool) {
	for index := len(p.fields) - 1; index >= 0; index-- {
		if p.fields[index].Key == key {
			return p.fields[index].Value, true
		}
	}

	return nil, false
}

func (p *Payload) Set(key string, value any) {
	if p == nil {
		return
	}

	for index := len(p.fields) - 1; index >= 0; index-- {
		if p.fields[index].Key == key {
			p.fields[index].Value = value
			return
		}
	}

	p.fields = append(p.fields, Field{Key: key, Value: value})
}

func (p Payload) MarshalJSON() ([]byte, error) {
	return marshalOrderedObject(p.fields)
}

func (p *Payload) UnmarshalJSON(data []byte) error {
	value, err := unmarshalOrderedValue(data)
	if err != nil {
		return err
	}

	payload, ok := value.(Payload)
	if !ok {
		return fmt.Errorf("payload must be a JSON object")
	}

	*p = payload
	return nil
}

func marshalOrderedObject(fields []Field) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteByte('{')

	for index, field := range fields {
		if index > 0 {
			buffer.WriteByte(',')
		}

		keyBytes, err := marshalJSONValue(field.Key)
		if err != nil {
			return nil, err
		}

		valueBytes, err := marshalJSONValue(field.Value)
		if err != nil {
			return nil, err
		}

		buffer.Write(keyBytes)
		buffer.WriteByte(':')
		buffer.Write(valueBytes)
	}

	buffer.WriteByte('}')
	return buffer.Bytes(), nil
}

func marshalJSONValue(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(value); err != nil {
		return nil, err
	}

	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

func unmarshalOrderedValue(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	value, err := decodeOrderedValue(decoder)
	if err != nil {
		return nil, err
	}

	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("unexpected trailing JSON content")
		}

		return nil, err
	}

	return value, nil
}

func decodeOrderedValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	switch token := token.(type) {
	case json.Delim:
		switch token {
		case '{':
			fields := make([]Field, 0)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}

				key, ok := keyToken.(string)
				if !ok {
					return nil, fmt.Errorf("expected object key, got %T", keyToken)
				}

				value, err := decodeOrderedValue(decoder)
				if err != nil {
					return nil, err
				}

				fields = append(fields, Field{Key: key, Value: value})
			}

			endToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}

			if endDelimiter, ok := endToken.(json.Delim); !ok || endDelimiter != '}' {
				return nil, fmt.Errorf("expected object terminator, got %v", endToken)
			}

			return Payload{fields: fields}, nil
		case '[':
			items := make([]any, 0)
			for decoder.More() {
				value, err := decodeOrderedValue(decoder)
				if err != nil {
					return nil, err
				}

				items = append(items, value)
			}

			endToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}

			if endDelimiter, ok := endToken.(json.Delim); !ok || endDelimiter != ']' {
				return nil, fmt.Errorf("expected array terminator, got %v", endToken)
			}

			return items, nil
		default:
			return nil, fmt.Errorf("unexpected JSON delimiter %q", token)
		}
	case string, bool, nil, json.Number:
		return token, nil
	case float64:
		return token, nil
	default:
		return nil, fmt.Errorf("unsupported JSON token type %T", token)
	}
}
