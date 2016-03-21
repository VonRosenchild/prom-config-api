package prom

var HostTypes = []string{"os", "mysql"}

type Host struct {
	Alias   string
	Address string
}

type Endpoint struct {
	Targets []string
	Labels  map[string]string
}

type Error struct {
	Error string
}
