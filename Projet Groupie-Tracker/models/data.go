package models

import (
	"encoding/json"
	"os"
)

// Collection represents the top-level structure of data.json (Postman collection).
type Collection struct {
	Info     Info       `json:"info"`
	Item     []Item     `json:"item"`
	Auth     *Auth      `json:"auth,omitempty"`
	Event    []Event    `json:"event,omitempty"`
	Variable []Variable `json:"variable,omitempty"`
}

type Info struct {
	PostmanID string `json:"_postman_id"`
	Name      string `json:"name"`
	Schema    string `json:"schema"`
}

type Item struct {
	Name                    string                 `json:"name"`
	ID                      string                 `json:"id,omitempty"`
	ProtocolProfileBehavior map[string]interface{} `json:"protocolProfileBehavior,omitempty"`
	Event                   []Event                `json:"event,omitempty"`
	Request                 *Request               `json:"request,omitempty"`
	Response                []interface{}          `json:"response,omitempty"`
}

type Request struct {
	Method string   `json:"method,omitempty"`
	Header []Header `json:"header,omitempty"`
	
	URL json.RawMessage `json:"url,omitempty"`
}

type Header struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Type     string `json:"type,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

type QueryParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Event struct {
	Listen string  `json:"listen,omitempty"`
	Script *Script `json:"script,omitempty"`
}

type Script struct {
	ID   string   `json:"id,omitempty"`
	Exec []string `json:"exec,omitempty"`
	Type string   `json:"type,omitempty"`
}

type Auth struct {
	Type   string  `json:"type,omitempty"`
	Apikey *APIKey `json:"apikey,omitempty"`
}

type APIKey struct {
	Value string `json:"value,omitempty"`
	Key   string `json:"key,omitempty"`
}

type Variable struct {
	ID    string `json:"id,omitempty"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
	Type  string `json:"type,omitempty"`
}

// LoadCollectionFromFile lit un fichier JSON et le désérialise dans une
// structure `Collection`. Elle renvoie la collection et une erreur le cas échéant.
func LoadCollectionFromFile(path string) (*Collection, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Collection
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
