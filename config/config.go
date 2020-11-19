package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	defaultDaemonEndpoint   = "localhost:9000"
	defaultKrakenWsEndpoint = "ws.kraken.com"
	defaultBaseAsset        = "lbtc"
	defaultQuoteAsset       = "usd"
	defaultKrakenTicker     = "XBT/USD"
	defaultInterval         = 30
)

type Config struct {
	DaemonEndpoint   string   `json:"daemon_endpoint,required"`
	DaemonMacaroon   string   `json:"daemon_macaroon"`
	KrakenWsEndpoint string   `json:"kraken_ws_endpoint,required"`
	Markets          []Market `json:"markets,required"`
}

type Market struct {
	BaseAsset    string `json:"base_asset,required"`
	QuoteAsset   string `json:"quote_asset,required"`
	KrakenTicker string `json:"kraken_ticker,required"`
	Interval     int    `json:"interval,required"`
}

// DefaultConfig returns the datastructure needed
// for a default connection.
func defaultConfig() Config {
	return Config{
		DaemonEndpoint:   defaultDaemonEndpoint,
		KrakenWsEndpoint: defaultKrakenWsEndpoint,
		Markets: []Market{
			Market{
				BaseAsset:    defaultBaseAsset,
				QuoteAsset:   defaultQuoteAsset,
				KrakenTicker: defaultKrakenTicker,
				Interval:     defaultInterval,
			},
		},
	}
}

// LoadConfigFromFile reads a file with the intended running behaviour
// and returns a Config struct with the respective configurations.
func loadConfigFromFile(filePath string) (Config, error) {
	jsonFile, err := os.Open(filePath)
	if err != nil {
		return Config{}, err
	}
	defer jsonFile.Close()

	var config Config

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return Config{}, err
	}
	json.Unmarshal(byteValue, &config)

	err = checkConfigParsing(config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

// checkConfigParsing checks if all the required fields
// were correctly loaded into the Config struct.
func checkConfigParsing(config Config) error {
	fields := reflect.ValueOf(config)
	for i := 0; i < fields.NumField(); i++ {
		tags := fields.Type().Field(i).Tag
		if strings.Contains(string(tags), "required") && fields.Field(i).IsZero() {
			return errors.New("Config required field is missing: " + string(tags))
		}
	}
	for _, market := range config.Markets {
		fields := reflect.ValueOf(market)
		for i := 0; i < fields.NumField(); i++ {
			tags := fields.Type().Field(i).Tag
			if strings.Contains(string(tags), "required") && fields.Field(i).IsZero() {
				return errors.New("Config required field is missing: " + string(tags))
			}
		}
	}
	return nil
}

// LoadConfig handles the default behaviour for loading
// config.json files. In case the file is not found,
// it loads the default config.
func LoadConfig(filePath string) (Config, error) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		log.Println("File not found. Loading default config.")
		return defaultConfig(), nil
	}
	return loadConfigFromFile(filePath)
}
