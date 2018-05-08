package main

import (
	"encoding/json"
	"flag"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/f-dev-o/CoinDataCollect/common"
)

const apiURL string = "https://api.bitflyer.jp"

var productCode = flag.String("product_code", "BTC_JPY", "-product_code {BTC_JPY|FX_BTC_JPY}")
var before = flag.String("before", "0", "-before {before execution id}")
var after = flag.String("after", "0", "-after {after execution id}")

func main() {
	flag.Parse()

	beforeID, err := strconv.ParseInt(*before, 10, 64)
	if err != nil {
		log.Println("before not decimal")
		return
	}

	afterID, err := strconv.ParseInt(*after, 10, 64)
	if err != nil {
		log.Println("after not decimal")
		return
	}
	if beforeID > 0 && beforeID < afterID {
		log.Println("after is small")
	}

	if err := os.MkdirAll("./execution_history/"+*productCode, 0777); err != nil {
		log.Println("create dir faild")
		return
	}
	exportExecutionHistoryList(*productCode, beforeID, afterID)
}

// 1リクエスト500件が上限
func exportExecutionHistoryList(productCode string, before int64, after int64) {
	// before 10000	after 10000	NG
	// after 100 OKbefore 1000
	// before 1000 || after 1000 OK
	// 遡れる件数が知れてるので、全部とってからソートしてファイルに出力する
	filewriter := new(common.MultiFileWriter).Initialize(1)
	utilTime := new(common.UtilUnixTime)
	var all []*ExecutionsResponseJSON
	var minitus int64
	var counter int
	for {
		// wait
		<-time.After(150 * time.Millisecond)
		list := getExecutionHistoryList(productCode, before, after)
		size := len(*list)
		if size > 0 {
			// 最後の行が一番若い行
			before = (*list)[size-1].ID

			for i := size - 1; i >= 0; i-- {
				// REST API は規約違反な日付文字列なのでUTC相当のZを取り付けてごまかす
				timeStr, _ := (*list)[i].ExecDate.(string)
				time := utilTime.GetUnixTimeMills(timeStr + "Z")
				(*list)[i].ExecDate = time

				// interfaceとして受け取り、floatとして取り出す
				price, _ := (*list)[i].Price.(float64)
				// float切り捨てで、intに変換し、戻す(Castだと0になる…)
				(*list)[i].Price = int(math.Floor(price))
				recodeMin := utilTime.GetMinitusFloorTime(time)

				// 最後に切り替わった最初の行を特定
				if minitus != recodeMin {
					minitus = recodeMin
					counter = i
				}
			}
			if counter > 10 {
				// 未処理分と、今回の切り替わった行までを結合
				all = append(all, ((*list)[0:counter])...)
				// 結合分をすべて処理させる
				fileoutMinitus(filewriter, utilTime, all)

				all = []*ExecutionsResponseJSON{}
				all = append(all, ((*list)[counter:size])...)
				counter = -1
			} else {
				// 全部とりあえず結合
				all = append(all, *list...)
			}
		} else {
			break
		}
	}
	fileoutMinitus(filewriter, utilTime, all)

	filewriter.Finalize()
}

func fileoutMinitus(filewriter *common.MultiFileWriter, utilTime *common.UtilUnixTime, all []*ExecutionsResponseJSON) {
	for i := len(all) - 1; i >= 0; i-- {
		time, _ := all[i].ExecDate.(int64)
		filename := strconv.FormatInt(utilTime.GetMinitusFloorTime(time), 10) + ".json"
		filewriter.WriteJSONLine("./execution_history/"+*productCode+"/"+filename, all[i])
	}
}
func getExecutionHistoryList(productCode string, before int64, after int64) *[]*ExecutionsResponseJSON {
	return _getExecutionHistoryList(productCode, before, after, 0)
}
func _getExecutionHistoryList(productCode string, before int64, after int64, retry int) *[]*ExecutionsResponseJSON {
	url := apiURL + "/v1/executions?product_code=" + productCode + "&count=99"

	// 3項演算子が使えない…
	if before > 0 {
		url = url + "&before=" + strconv.FormatInt(before, 10)
	}
	if after > 0 {
		url = url + "&after=" + strconv.FormatInt(after, 10)
	}
	log.Println(url)

	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return nil
	}

	defer resp.Body.Close()
	var recodes = new([]*ExecutionsResponseJSON)
	err = json.NewDecoder(resp.Body).Decode(recodes)
	if err != nil {
		log.Println(err)
		if retry < 5 {
			// wait
			<-time.After(1000 * time.Millisecond)
			return _getExecutionHistoryList(productCode, before, after, retry)
		}
	}
	return recodes
}

// ExecutionsResponseJSON @see: https://lightning.bitflyer.jp/docs?lang=ja#%E7%B4%84%E5%AE%9A%E5%B1%A5%E6%AD%B4
type ExecutionsResponseJSON struct {
	ID                         int64       `json:"id"`
	Side                       string      `json:"side"`
	Price                      interface{} `json:"price"`
	Size                       float64     `json:"size"`
	ExecDate                   interface{} `json:"exec_date"`
	BuyChildOrderAcceptanceID  string      `json:"buy_child_order_acceptance_id"`
	SellChildOrderAcceptanceID string      `json:"sell_child_order_acceptance_id"`
}
