package hoverctl

import (
	_ "net"
	"net/http"
)

type Dataplane struct {
	client  *http.Client
	baseUrl string
	id      string
}

func NewDataplane() *Dataplane {
	client := &http.Client{}
	d := &Dataplane{
		client: client,
	}

	return d
}

func (d *Dataplane) Init(baseUrl string) error {
	d.baseUrl = baseUrl
	return nil
}

type Module struct {
	Id          string                 `json:"id"`
	ModuleType  string                 `json:"module_type"`
	DisplayName string                 `json:"display_name"`
	Perm        string                 `json:"permissions"`
	Config      map[string]interface{} `json:"config"`
}

type ExternalInterface struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Link struct {
	Id     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
	FromId int    `json:"from-id"`
	ToId   int    `json:"to-id"`
}

type TableEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
