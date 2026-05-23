package types

import "unsafe"

type ConfigRequest struct {
	Token      string `json:"token"`
	Country    string `json:"country"`
	City       string `json:"city"`
	Name       string `json:"name"`
	PrivateKey string `json:"privateKey"`
	DNS        string `json:"dns"`
	Endpoint   string `json:"endpoint"`
	KeepAlive  *int   `json:"keepalive"`
}

type BatchConfigReq struct {
	Token      string `json:"token"`
	PrivateKey string `json:"privateKey"`
	DNS        string `json:"dns"`
	Endpoint   string `json:"endpoint"`
	KeepAlive  *int   `json:"keepalive"`
	Country    string `json:"country"`
	City       string `json:"city"`
}

type ValidatedConfig struct {
	Name       string
	PrivateKey string
	DNS        string
	UseStation bool
	KeepAlive  int
}

type ServerLoc struct {
	Country struct {
		Name string `json:"name"`
		Code string `json:"code"`
		City struct {
			Name string `json:"name"`
		} `json:"city"`
	} `json:"country"`
}

type ServerTech struct {
	ID       string `json:"identifier"`
	Metadata []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"metadata"`
}

type ServerSpec struct {
	ID     string `json:"identifier"`
	Values []struct {
		Value string `json:"value"`
	} `json:"values"`
}

type RawServer struct {
	Name           string       `json:"name"`
	Station        string       `json:"station"`
	Hostname       string       `json:"hostname"`
	Load           int          `json:"load"`
	Locations      []ServerLoc  `json:"locations"`
	Technologies   []ServerTech `json:"technologies"`
	Specifications []ServerSpec `json:"specifications"`
}

type ProcessedServer struct {
	Name            [16]byte
	Station         [48]byte
	Hostname        [64]byte
	Country         [32]byte
	City            [32]byte
	Code            [4]byte
	LowCode         [4]byte
	Number          [8]byte
	FileName        [24]byte
	CityDedupSuffix [8]byte
	KeyID           uint16
}

type ServerPayload struct {
	Headers []string                              `json:"h"`
	List    map[string]map[string][][]interface{} `json:"l"`
}

type Asset struct {
	Content []byte
	Brotli  []byte
	Gzip    []byte
	Mime    string
	Etag    string
}

func getString(b []byte) string {
	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			if i == 0 {
				return ""
			}
			return unsafe.String(&b[0], i)
		}
	}
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}

func (p *ProcessedServer) GetName() string {
	return getString(p.Name[:])
}

func (p *ProcessedServer) GetStation() string {
	return getString(p.Station[:])
}

func (p *ProcessedServer) GetHostname() string {
	return getString(p.Hostname[:])
}

func (p *ProcessedServer) GetCountry() string {
	return getString(p.Country[:])
}

func (p *ProcessedServer) GetCity() string {
	return getString(p.City[:])
}

func (p *ProcessedServer) GetCode() string {
	return getString(p.Code[:])
}

func (p *ProcessedServer) GetLowCode() string {
	return getString(p.LowCode[:])
}

func (p *ProcessedServer) GetNumber() string {
	return getString(p.Number[:])
}

func (p *ProcessedServer) GetFileName() string {
	return getString(p.FileName[:])
}

func (p *ProcessedServer) GetCityDedupSuffix() string {
	return getString(p.CityDedupSuffix[:])
}
