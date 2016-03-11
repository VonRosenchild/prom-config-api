package main

import (
	"errors"
	"io/ioutil"
	"sync"

	"gopkg.in/yaml.v2"
)

var (
	ErrHostNotFound = errors.New("host not found")
)

type Target struct {
	Port     string
	Filename string
}

type TargetsFile struct {
	hostsFile string
	targets   []Target
	*sync.Mutex
}

func NewTargetsFile(hostsFile string, targets []Target) *TargetsFile {
	f := &TargetsFile{
		hostsFile: hostsFile,
		targets:   targets,
		Mutex:     &sync.Mutex{},
	}
	return f
}

func (f *TargetsFile) List() ([]Host, error) {
	f.Lock()
	defer f.Unlock()

	return f.open()
}

func (f *TargetsFile) Add(host Host) error {
	f.Lock()
	defer f.Unlock()

	hosts, err := f.open()
	if err != nil {
		return err
	}

	hosts = append(hosts, host)

	return f.writeFiles(hosts)
}

func (f *TargetsFile) Remove(alias string) error {
	f.Lock()
	defer f.Unlock()

	hosts, err := f.open()
	if err != nil {
		return err
	}

	for i, host := range hosts {
		if host.Alias != alias {
			continue
		}
		hosts = append(hosts[:i], hosts[i+1:]...)
		return f.writeFiles(hosts)
	}

	return ErrHostNotFound
}

// --------------------------------------------------------------------------

func (f *TargetsFile) open() ([]Host, error) {
	yamlData, err := ioutil.ReadFile(f.hostsFile)
	if err != nil {
		return nil, err
	}

	hosts := []Host{}
	if err := yaml.Unmarshal(yamlData, &hosts); err != nil {
		return nil, err
	}

	return hosts, nil
}

func (f *TargetsFile) writeFiles(hosts []Host) error {
	yamlData, _ := yaml.Marshal(&hosts)
	if err := ioutil.WriteFile(f.hostsFile, yamlData, 0644); err != nil {
		return err
	}

	for _, target := range f.targets {
		var endPoints []Endpoint
		for _, host := range hosts {
			ep := Endpoint{
				Targets: []string{host.Address + ":" + target.Port},
				Labels:  map[string]string{"alias": host.Alias},
			}
			endPoints = append(endPoints, ep)
		}
		yamlData, _ = yaml.Marshal(&endPoints)
		if err := ioutil.WriteFile(target.Filename, yamlData, 0644); err != nil {
			return err
		}
	}

	return nil
}
