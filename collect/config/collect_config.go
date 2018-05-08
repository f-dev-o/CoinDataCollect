package config

import (
	"encoding/json"
	"io/ioutil"
)

// TODO Viper

type extendCollectConfig struct {
	Name          string `json:"name"`
	OutputOptions struct {
		Dir             string `json:"dir"`
		ConvertUnixTime bool   `json:"convert_unixtime"`
		// 出力単位の制御を…
		// OutputPeriod int `json:output_period`
	} `json:"output_options"`
	Targets []CollectConfig `json:"targets"`
}

// CollectConfig collect_config.json struct
type CollectConfig struct {
	Name            string   `json:"name"`
	Endpoint        string   `json:"endpoint"`
	Channels        []string `json:"channels"`
	OutputDir       string   `json:"output_dir"`
	ConvertUnixTime bool     `json:"convert_unixtime"`
}

// ReadCollectConfig read config file (errors io|decode
func ReadCollectConfig(filepath string) (*[]CollectConfig, error) {
	var config []CollectConfig
	var extendCollectConfig = new(extendCollectConfig)

	// JSONファイル読み込み
	bytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	// JSONデコード FIXME json.NewDecoder
	if err := json.Unmarshal(bytes, &extendCollectConfig); err != nil {
		return nil, err
	}

	defaultOptions := extendCollectConfig.OutputOptions

	// 拡張設定を各設定に反映する
	for _, target := range extendCollectConfig.Targets {
		if len(target.OutputDir) == 0 {
			// path builderを作って、winとlinuxのフォーマットに対応させる
			target.OutputDir = defaultOptions.Dir + target.Name + "/"
		}
		if defaultOptions.ConvertUnixTime {
			target.ConvertUnixTime = defaultOptions.ConvertUnixTime
		}
		config = append(config, target)
	}
	return &config, nil
}
