package common

import (
	"encoding/json"
	"os"

	"github.com/golang/groupcache/lru"
)

// bufio.Writer
// io/ioutil WriteFile:
// 現状ではバッファリングするほどのデータ量を1分毎のファイルは持ち合わせていないので後回し

// MultiFileWriter ファイルハンドルの節約
type MultiFileWriter struct {
	cache *lru.Cache
	_lf   []byte
}

// Initialize 初期化処理
func (t *MultiFileWriter) Initialize(maxOpenFile int) *MultiFileWriter {
	t.cache = lru.New(maxOpenFile)
	t.cache.OnEvicted = t.cacheRemoveEvent

	// constで宣言できない…
	t._lf = []byte("\n")
	return t
}

// WriteJSONLine JSONに変換して行出力
func (t *MultiFileWriter) WriteJSONLine(filepath string, obj interface{}) {
	// TODO errorの場合、中間ファイルを作って復旧材料にするか検討
	// この段階でこけるなら　openfile系かdisk fulllの場合も…
	file, _ := t.getFile(filepath)
	json, _ := json.Marshal(obj)
	file.Write(json)
	file.Write(t._lf)
}

func (t *MultiFileWriter) getFile(filepath string) (*os.File, error) {
	cacheObj, hasValue := t.cache.Get(filepath)
	if hasValue {
		file, _ := cacheObj.(*os.File)
		return file, nil
	}
	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err == nil {
		t.cache.Add(filepath, file)
	}
	return file, err
}

// Finalize 全てのFileを閉じる
func (t *MultiFileWriter) Finalize() {
	t.cache.Clear()
}

// 上限を超えた場合や、終了処理でClearが呼ばれる際に、閉じていないリソースをCloseする(FLUSH)
func (t *MultiFileWriter) cacheRemoveEvent(key lru.Key, value interface{}) {
	file, _ := value.(*os.File)
	file.Close()
}
