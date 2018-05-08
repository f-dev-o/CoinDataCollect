package common

import "time"

// UtilUnixTime 多量に類似性の高い時間変換を行うのは無駄なので、キャッシュする
// hour単位までの分割であれば、UTC±に柔軟に対応できるので、金を積む環境でない限りこの単位がいいと判断
type UtilUnixTime struct {
	unixTime     int64
	unixHourTime int64
}

// GetUnixTimeMills convert "2018-05-06T15:23:23.2320573Z" => 1525620203232
func (_this *UtilUnixTime) GetUnixTimeMills(src string) int64 {
	t, _ := time.Parse(time.RFC3339, src)
	return t.UnixNano() / 1000000 // 1000/1000
}

// GetMinitusFloorTime "2018-05-06T15:23:23.2320573Z" => 1525620180000(2018-05-07T00:23:00+09:00)
func (_this *UtilUnixTime) GetMinitusFloorTime(srcUnixTimeMills int64) int64 {
	var result int64
	if _this.unixTime <= srcUnixTimeMills && srcUnixTimeMills < _this.unixTime+60000 {
		result = _this.unixTime
	} else {
		tmp := time.Unix(srcUnixTimeMills/1000, 0)
		// 他言語みたく、該当の数値だけ0にする等ができない…
		newTime := time.Date(tmp.Year(), tmp.Month(), tmp.Day(), tmp.Hour(), tmp.Minute(), 0, 0, tmp.Location())
		// a = b = 0 のような記述もできない…
		result = newTime.UnixNano() / 1000000
		_this.unixTime = result
	}
	return result
}

// GetHourFloorTime "2018-05-06T15:23:23.2320573Z" => 1525618800000(2018-05-07T00:00:00+09:00)
func (_this *UtilUnixTime) GetHourFloorTime(srcUnixTimeMills int64) int64 {
	var result int64
	if _this.unixHourTime <= srcUnixTimeMills && srcUnixTimeMills < _this.unixHourTime+3600000 {
		result = _this.unixHourTime
	} else {
		tmp := time.Unix(srcUnixTimeMills/1000, 0)
		newTime := time.Date(tmp.Year(), tmp.Month(), tmp.Day(), tmp.Hour(), 0, 0, 0, tmp.Location())
		result = newTime.UnixNano() / 1000000
		_this.unixHourTime = result
	}
	return result
}
