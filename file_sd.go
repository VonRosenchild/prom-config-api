package main

import (
	"errors"
	"io/ioutil"
	"sync"

	"github.com/percona/platform/proto"
	"gopkg.in/yaml.v2"
)

var (
	ErrHostNotFound = errors.New("host not found")
	ErrDupeHost     = errors.New("duplicate host")
)

type Target struct {
	Port     string
	Filename string
}

type TargetsFile struct {
	hostsFile string
	targets   map[string][]Target
	*sync.Mutex
}

func NewTargetsFile(hostsFile string, targets map[string][]Target) *TargetsFile {
	f := &TargetsFile{
		hostsFile: hostsFile,
		targets:   targets,
		Mutex:     &sync.Mutex{},
	}
	return f
}

func (f *TargetsFile) List() (map[string][]proto.Host, error) {
	f.Lock()
	defer f.Unlock()

	return f.open()
}

func (f *TargetsFile) Add(hostType string, host proto.Host) error {
	f.Lock()
	defer f.Unlock()

	hosts, err := f.open()
	if err != nil {
		return err
	}

	for _, h := range hosts[hostType] {
		if h.Alias == host.Alias {
			return ErrDupeHost
		}
	}

	hosts[hostType] = append(hosts[hostType], host)

	return f.writeFiles(hosts)
}

func (f *TargetsFile) Remove(hostType, alias string) error {
	f.Lock()
	defer f.Unlock()

	hosts, err := f.open()
	if err != nil {
		return err
	}

	for i, host := range hosts[hostType] {
		if host.Alias != alias {
			continue
		}
		hosts[hostType] = append(hosts[hostType][:i], hosts[hostType][i+1:]...)
		return f.writeFiles(hosts)
	}

	return ErrHostNotFound
}

// --------------------------------------------------------------------------

func (f *TargetsFile) open() (map[string][]proto.Host, error) {
	yamlData, err := ioutil.ReadFile(f.hostsFile)
	if err != nil {
		return nil, err
	}

	hosts := map[string][]proto.Host{}
	if err := yaml.Unmarshal(yamlData, &hosts); err != nil {
		return nil, err
	}

	return hosts, nil
}

func (f *TargetsFile) writeFiles(hosts map[string][]proto.Host) error {
	yamlData, _ := yaml.Marshal(&hosts)
	if err := ioutil.WriteFile(f.hostsFile, yamlData, 0644); err != nil {
		return err
	}

	for hostType, targets := range f.targets {
		for _, target := range targets {
			var endPoints []proto.Endpoint
			for _, host := range hosts[hostType] {
				ep := proto.Endpoint{
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
	}

	return nil
}
