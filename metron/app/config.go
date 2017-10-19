package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/net/idna"
)

type GRPC struct {
	Port         uint16
	CAFile       string
	CertFile     string
	KeyFile      string
	CipherSuites []string
}

type Config struct {
	Deployment string
	Zone       string
	Job        string
	Index      string
	IP         string

	Tags map[string]string

	DisableUDP         bool
	IncomingUDPPort    int
	HealthEndpointPort uint

	GRPC GRPC

	DopplerAddr       string
	DopplerAddrWithAZ string

	MetricBatchIntervalMilliseconds  uint
	RuntimeStatsIntervalMilliseconds uint

	PPROFPort uint32
}

func ParseConfig(configFile string) (*Config, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return Parse(file)
}

func Parse(reader io.Reader) (*Config, error) {
	config := &Config{
		MetricBatchIntervalMilliseconds:  60000,
		RuntimeStatsIntervalMilliseconds: 60000,
	}
	err := json.NewDecoder(reader).Decode(config)
	if err != nil {
		return nil, err
	}

	if config.DopplerAddr == "" {
		return nil, fmt.Errorf("DopplerAddr is required")
	}

	if config.DopplerAddrWithAZ == "" {
		return nil, fmt.Errorf("DopplerAddrWithAZ is required")
	}

	config.DopplerAddrWithAZ, err = idna.ToASCII(config.DopplerAddrWithAZ)
	if err != nil {
		return nil, err
	}
	config.DopplerAddrWithAZ = strings.Replace(config.DopplerAddrWithAZ, "@", "-", -1)

	return config, nil
}
