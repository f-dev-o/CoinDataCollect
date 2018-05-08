package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/f-dev-o/CoinDataCollect/collect/config"
	"github.com/f-dev-o/CoinDataCollect/collect/exchanges"
)

func main() {
	var filePath = flag.String("conf", "./collect_config.json", "[file path] collect_config.json")
	flag.Parse()

	// 設定ファイルを読み込む
	configArr, err := config.ReadCollectConfig(*filePath)
	if err != nil {
		log.Println(err)
		return
	}

	var waitChanList []<-chan struct{}

	// 設定ファイルの項目分、goroutineを起動
	goroutineList := []exchanges.CollectGoroutine{}
	for _, config := range *configArr {
		goroutine := exchangesSelector(config)
		if goroutine != nil {
			goroutineList = append(goroutineList, goroutine)
			waitChanList = append(waitChanList, goroutine.Start())
		}
	}

	// 割り込みを待ち受ける(Ctrl+C|kill (nomal))
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// 終了時は順次Stop命令を出していく(同期)
	for {
		select {
		case <-interrupt:
			log.Println("interrupt")

			for _, goroutine := range goroutineList {
				goroutine.Stop()
			}
			close(interrupt)

			// 終了待機(後で、各チャンネルの待ち受けに変更する)
			<-time.After(1000 * time.Millisecond)

			// 全部終了するまで待機
			for _, channel := range waitChanList {
				<-channel
			}
			return
		}
	}
}

func exchangesSelector(config config.CollectConfig) exchanges.CollectGoroutine {
	name := strings.ToLower(config.Name)
	var instance exchanges.CollectGoroutine
	switch {
	case name == "coincheck":
		return nil
	case name == "bitflyer":
		instance = new(exchanges.BitflyerGoroutine).Initialize(config)
	case name == "bitmex":
		instance = new(exchanges.BitMexGoroutine).Initialize(config)
	case name == "binance":
		return nil
	case name == "okex":
		return nil
	case name == "houbipro":
		return nil
	case name == "bitfinex":
		return nil
	case name == "hitbtc":
		return nil
	case name == "quoine":
		return nil
	case name == "gdax":
		return nil
	case name == "kraken":
		return nil
	case name == "bitstamp":
		return nil
	case name == "bithumb":
		return nil
	case name == "btcbox":
		return nil
	default:
		return nil
	}
	return instance
}
