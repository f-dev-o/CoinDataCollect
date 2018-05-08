package exchanges

/**
 * websocketで接続先からデータを取得し、該当パスにファイルを落とす
 * 最後に処理した時間を指定Interval事に通知
 * 接続が強制終了された場合、再接続を試みる
**/
import (
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/f-dev-o/CoinDataCollect/collect/common"
	"github.com/f-dev-o/CoinDataCollect/collect/config"
	"github.com/gorilla/websocket"
)

// bitflyerRPC
// type bitflyerRPC struct {
// 	/** 公式のサンプル… */
// 	Version string      `json:"jsonrpc"`
// 	Method  string      `json:"method"`
// 	Params  interface{} `json:"params"`
// 	Result  interface{} `json:"result"`
// 	ID      *int        `json:"id"`
// }

// bitflyerRPCRequest リクエスト用JSON
type bitflyerRPCRequest struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// bitflyerRPCResponse
type bitflyerRPCResponse struct {
	// ID/Version/Resultは何の意味もないので棄てる
	Method string `json:"method"`
	Params struct {
		Channel string      `json:"channel"`
		Message interface{} `json:"message"`
	} `json:"params"`
}

type subscribeParams struct {
	Channel string `json:"channel"`
}

// BitflyerGoroutine struct
type BitflyerGoroutine struct {
	config  config.CollectConfig
	channel chan struct{}
	done    chan struct{}

	fileout  *common.MultiFileWriter
	timeUtil *common.UtilUnixTime
}

// Initialize constructor
func (t *BitflyerGoroutine) Initialize(config config.CollectConfig) *BitflyerGoroutine {
	t.config = config
	t.done = make(chan struct{})

	t.fileout = new(common.MultiFileWriter).Initialize(len(t.config.Channels))
	t.timeUtil = new(common.UtilUnixTime)
	t.createDirs()
	return t
}

func (t *BitflyerGoroutine) createDirs() {
	if err := os.MkdirAll(t.config.OutputDir, 0777); err != nil {
		log.Println("create dir faild")
		panic(err)
	}
	for _, channel := range t.config.Channels {
		if err := os.Mkdir(t.config.OutputDir+"/"+channel, 0777); err != nil {
			log.Println("create dir faild")
			panic(err)
		}
	}
}

// Start CollectGoroutine I/F start recive
func (t *BitflyerGoroutine) Start() <-chan struct{} {
	t.channel = make(chan struct{})
	go t.startRecive()
	return t.channel
}

// Stop CollectGoroutine I/F finalize
func (t *BitflyerGoroutine) Stop() {
	// 受信のループのチャンネルを終了
	close(t.done)
}

//
func (t *BitflyerGoroutine) startRecive() {

	client := t.connect()

	// このメソッドが終了するときに、clientを終了させる(Hookみたいなもの)
	defer client.Close()

	// このメソッドが終了するまで実施
	go func() {
		for {
			message := new(bitflyerRPCResponse)
			// 「An existing connection was forcibly closed by the remote host.」は即再接続対象
			if err := client.ReadJSON(message); err != nil {
				log.Println("read:", err)
				client.Close()
				// wait
				<-time.After(1000 * time.Millisecond)
				client = t.connect()
				continue
			}

			if message.Method == "channelMessage" {
				t.writeFile(message)
			}
		}
	}()

	// チャンネルが閉じられるまでブロッキング
	for {
		select {
		case _, isOpen := <-t.done:
			if !isOpen {
				err := client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					log.Println("write close:", err)
					return
				}

				t.fileout.Finalize()

				// 全ての処理が終わったら、親とのチャンネルも閉じる
				close(t.channel)
				return
			}
		}
	}
}

// connect retry つながらない場合に待ち続けるが、上限を設けても管理者が気が付けなければ同じなので一旦このまま
func (t *BitflyerGoroutine) connect() *websocket.Conn {
	client := t._connect()
	for client == nil {
		// wait
		<-time.After(1000 * time.Millisecond)
		t._connect()
	}
	return client
}
func (t *BitflyerGoroutine) _connect() *websocket.Conn {
	_url := url.URL{Scheme: "wss", Host: t.config.Endpoint, Path: "/json-rpc"}
	log.Printf("connecting to %s", _url.String())

	client, _, err := websocket.DefaultDialer.Dial(_url.String(), nil)

	// 初期接続でエラー、ログに出力して、リトライが必要 FIXME LOG出力
	if err != nil {
		log.Fatal("dial:", err)
		return nil
	}

	for _, channel := range t.config.Channels {
		// エラーになった場合、パラメータが正しいと仮定するならば、数秒毎にリトライさせたい FIXME LOG出力
		if err := client.WriteJSON(&bitflyerRPCRequest{Version: "2.0", Method: "subscribe", Params: &subscribeParams{channel}}); err != nil {
			log.Fatal("subscribe:", err)

			return nil
		}
	}
	return client
}

func (t *BitflyerGoroutine) writeFile(message *bitflyerRPCResponse) {
	// outputdir/{exchange}/channel/unixtime_min|hour.json
	// root:outputdir/{exchange}/
	// append: channel
	// filename: unixtime_min.json
	// queueで書き込む形式に換えない限り、名前の生成を効率化する意味がなさそうなので保留
	min := t.timeUtil.GetMinitusFloorTime(time.Now().UnixNano() / 1000000)
	filename := t.config.OutputDir + "/" + message.Params.Channel + "/" + strconv.FormatInt(min, 10) + ".json"

	// どこかのサードパーティ製の変換ツールは値がロストしたので大人しく手動で変換させる
	// map[string]interface{} で中身を強引に使用

	var tmp map[string]interface{}
	reciveTime := time.Now().UnixNano() / 1000000

	channel := message.Params.Channel
	switch {
	// ticker
	case channel == "lightning_ticker_FX_BTC_JPY":
		fallthrough
	case channel == "lightning_ticker_BTC_JPY":
		tmp = message.Params.Message.(map[string]interface{})
		tmp["recivetime"] = reciveTime
		tmp["timestamp"] = t.timeUtil.GetUnixTimeMills(tmp["timestamp"].(string))
		delete(tmp, "product_code")
		t.fileout.WriteJSONLine(filename, tmp)
	// board
	case channel == "lightning_board_FX_BTC_JPY":
		fallthrough
	case channel == "lightning_board_BTC_JPY":
		fallthrough
	case channel == "lightning_board_snapshot_FX_BTC_JPY":
		fallthrough
	case channel == "lightning_board_snapshot_BTC_JPY":
		tmp = message.Params.Message.(map[string]interface{})
		tmp["recivetime"] = reciveTime
		t.fileout.WriteJSONLine(filename, tmp)
	// executions
	case channel == "lightning_executions_FX_BTC_JPY":
		fallthrough
	case channel == "lightning_executions_BTC_JPY":
		for _, recode := range message.Params.Message.([]interface{}) {
			t.fileout.WriteJSONLine(filename, recode)
		}
	}
}
