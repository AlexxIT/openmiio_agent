package rpc

import (
	"encoding/json"
	"strconv"
)

// Message - miIO has no respect for the specs, so we can't define prebuild struct for it
type Message map[string]json.RawMessage

func NewMessage(data []byte) (Message, error) {
	var m Message
	return m, json.Unmarshal(data, &m)
}

func (m Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m Message) String() string {
	b, _ := json.Marshal(m)
	return string(b)
}

func (m Message) ID() int {
	return m.GetInt("id")
}

func (m Message) Method() string {
	return m.GetString("method")
}

func (m Message) GetString(key string) string {
	if v, ok := m[key]; ok {
		// without any checks
		return string(v[1 : len(v)-1])
	}
	return ""
}

func (m Message) GetInt(key string) int {
	if v, ok := m[key]; ok {
		return Atoi(v)
	}
	return 0
}

func (m Message) SetInt(key string, value int) {
	m[key] = json.RawMessage(strconv.FormatInt(int64(value), 10))
}

// Atoi - simple string to int for unsigned 2,147,483,647 without any checks
func Atoi(b []byte) (i int) {
	for _, ch := range b {
		ch -= '0'
		if ch > 9 {
			return 0
		}
		i = i*10 + int(ch)
	}
	return
}
