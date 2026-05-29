package models

type Server struct {
	Name        string
	Hostname    string
	Station     string
	Load        int
	Country     string
	CountryCode string
	City        string
	Latitude    float64
	Longitude   float64
	PublicKey   string
	Distance    float64
	Combo       string
}

type UserPreferences struct {
	DNS              string
	UseIP            bool
	Keepalive        int
	Groups           []string
	ExcludeDedicated bool
}

type GenerationStats struct {
	Total int
	Best  int
}

type RawServer struct {
	Hostname     string          `json:"hostname"`
	Station      string          `json:"station"`
	Load         int             `json:"load"`
	Locations    []RawLocation   `json:"locations"`
	Groups       []RawGroup      `json:"groups"`
	Technologies []RawTechnology `json:"technologies"`
}

type RawLocation struct {
	Latitude  float64    `json:"latitude"`
	Longitude float64    `json:"longitude"`
	Country   RawCountry `json:"country"`
}

type RawCountry struct {
	Name string  `json:"name"`
	Code string  `json:"code"`
	City RawCity `json:"city"`
}

type RawCity struct {
	Name string `json:"name"`
}

type RawGroup struct {
	Identifier string `json:"identifier"`
}

type RawTechnology struct {
	Metadata []RawMetadata `json:"metadata"`
}

type RawMetadata struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
