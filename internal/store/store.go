package store

import (
	"encoding/json"
	"github.com/AlexxIT/openmiio_agent/internal/app"
	"os"
)

func Init() {
	log := app.GetLogger("store")

	name, _ = os.Executable()
	name += ".json"

	if data, _ := os.ReadFile(name); data != nil {
		if err := json.Unmarshal(data, &Store); err != nil {
			log.Warn().Err(err).Caller().Send()
		}
	}
}

var Store = map[string]json.RawMessage{}

var name string

func Get(key string, v interface{}) error {
	if raw, ok := Store[key]; ok {
		return json.Unmarshal(raw, v)
	}

	return nil
}

func Set(key string, v interface{}) (err error) {
	Store[key], err = json.Marshal(v)
	if err != nil {
		return err
	}

	if name == "" {
		return nil
	}

	data, err := json.Marshal(Store)
	if err != nil {
		return err
	}

	return os.WriteFile(name, data, 0644)
}
