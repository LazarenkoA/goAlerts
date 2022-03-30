package alert

import (
	"context"
	"fmt"
	temail "github.com/LazarenkoA/temp-Email"
	"github.com/sirupsen/logrus"
	"github.com/ungerik/go-dry"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockE struct {
}

func (m *mockE) GetData(string, ...string) ([]byte, error) {
	txt := `{
  "took": 3046,
  "timed_out": false,
  "_shards": {
    "total": 45,
    "successful": 45,
    "skipped": 0,
    "failed": 0
  },
  "hits": {
    "total": {
      "value": 7,
      "relation": "eq"
    },
    "max_score": 0,
    "hits": [
      {
        "_index": "techlog-minimal-2022.03.25",
        "_type": "_doc",
        "_id": "220325171409.583001-8454168",
        "_score": 0,
        "_source": {}
      },
      {
        "_index": "techlog-minimal-2022.03.25",
        "_type": "_doc",
        "_id": "220325172549.595000-4120471",
        "_score": 0,
        "_source": {}
      },
      {
        "_index": "techlog-minimal-2022.03.25",
        "_type": "_doc",
        "_id": "220325173226.922000-5493484",
        "_score": 0,
        "_source": {}
      },
      {
        "_index": "techlog-minimal-2022.03.25",
        "_type": "_doc",
        "_id": "220325170726.315000-2333283",
        "_score": 0,
        "_source": {}
      },
      {
        "_index": "techlog-minimal-2022.03.25",
        "_type": "_doc",
        "_id": "220325171845.579000-11191818",
        "_score": 0,
        "_source": {}
      },
      {
        "_index": "techlog-minimal-2022.03.25",
        "_type": "_doc",
        "_id": "220325171230.746000-7516437",
        "_score": 0,
        "_source": {}
      },
      {
        "_index": "techlog-minimal-2022.03.25",
        "_type": "_doc",
        "_id": "220325171537.341000-13671696",
        "_score": 0,
        "_source": {}
      }
    ]
  },
  "aggregations": {
    "errors": {
      "doc_count_error_upper_bound": 0,
      "sum_other_doc_count": 0,
      "buckets": [
        {
          "key": "key1",
          "doc_count": 10,
			"timelock" : {
            	"value" : 56137900
          	}
        },
        {
          "key": "key2",
          "doc_count": 100,
			"timelock" : {
            	"value" : 56137901
          	}
        },
        {
          "key": "key3",
          "doc_count": 50,
			"timelock" : {
            	"value" : 56137902
          	}
        }
      ]
    }
  }
}`
	return []byte(txt), nil
}

func (m *mockE) ReadConf(string) error {
	return nil
}

func (m *mockE) RequestCheck(string) error {

	return nil
}

type mocCl struct {
}

func (m *mocCl) GetData(string, ...string) ([]byte, error) {
	txt := `{
	"meta":
	[
		{
			"name": "value",
			"type": "UInt8"
		},
		{
			"name": "Name",
			"type": "String"
		}
	],

	"data":
	[
		{
			"value": 1,
			"Name": "test"
		},
		{
			"value": 1,
			"Name": "test"
		},
		{
			"value": 1,
			"Name": "test"
		},
		{
			"value": 1,
			"Name": "test"
		},
		{
			"value": 1,
			"Name": "test"
		},
		{
			"value": 1,
			"Name": "test"
		},
		{
			"value": 1,
			"Name": "test"
		},
		{
			"value": 13,
			"Name": "test"
		},
		{
			"value": 13,
			"Name": "key2"
		},
		{
			"value": 13,
			"Name": "key2"
		},
		{
			"value": 13,
			"Name": "key2"
		},
		{
			"value": 50,
			"Name": "key2"
		},
		{
			"value": 13,
			"Name": "key2"
		}
	],

	"rows": 13,

	"statistics":
	{
		"elapsed": 0.0014614,
		"rows_read": 13,
		"bytes_read": 182
	}
}`
	return []byte(txt), nil
}

func (m *mocCl) ReadConf(string) error {
	return nil
}

func (m *mocCl) RequestCheck(string) error {

	return nil
}

func Test_Explorer(t *testing.T) {
	t.Run("testClickhouse", func(t *testing.T) {
		outFile, _ := os.CreateTemp("", "*.txt")
		outFile.Close()

		batFile := createScriptFile(outFile.Name())
		pathRule := click_rule(batFile)

		dir := filepath.Dir(pathRule)

		defer os.Remove(batFile)
		defer os.RemoveAll(dir)
		defer os.Remove(outFile.Name())

		os.Setenv("RULES_DIR", dir)
		testClickhouse(t, outFile.Name())
	})
	t.Run("testElasticsearch", func(t *testing.T) {
		outFile, _ := os.CreateTemp("", "*.txt")
		outFile.Close()

		email := ""
		cResult := make(chan *temail.Result, 1) // размер канала обязательно должен быть 1 или больше
		if err := createEmail(cResult); err == nil {
			// Читаем email
			email = (<-cResult).Email
		} else {
			t.Errorf("ошибка создания временного email '%s'", err.Error())
			// ждем секунду выполнения правила
			go func() {
				time.Sleep(time.Second)
				cResult <- &temail.Result{
					Error: err,
				}
			}()
		}

		batFile := createScriptFile(outFile.Name())
		pathRule := elastic_rule(batFile, email)

		dir := filepath.Dir(pathRule)

		defer os.Remove(batFile)
		defer os.Remove(outFile.Name())
		defer os.RemoveAll(dir)

		os.Setenv("RULES_DIR", dir)
		testElasticsearch(t, outFile.Name(), cResult)
	})
}

func testClickhouse(t *testing.T, outFile string) {
	confPath := click_config()
	defer os.Remove(confPath)

	e_alert, err := new(Alert).Init("clickhouse::"+confPath, false)
	if err != nil {
		t.Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		e_alert.source = new(mocCl)
		e_alert.logger = logrus.WithField("name", "test")

		if err := e_alert.Run(ctx); err != nil {
			t.Error(err)
			return
		}
	}()

	time.Sleep(time.Second)
	cancel()

	if !dry.FileExists(outFile) {
		t.Error("файл out.txt не найден")
	} else {
		if str, err := dry.FileGetString(outFile); err != nil {
			t.Error(err)
		} else if strings.Trim(str, "\r\n\"\\ ") != "50, key2" {
			t.Fatal("содержимое out.txt отличается от ожидаемого")
		}
	}
}

func testElasticsearch(t *testing.T, outFile string, cResult chan *temail.Result) {
	confPath := elastic_config()
	defer os.Remove(confPath)

	e_alert, err := new(Alert).Init("elastic::"+confPath, false)
	if err != nil {
		t.Error(err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		e_alert.source = new(mockE)
		e_alert.logger = logrus.WithField("name", "test")

		if err := e_alert.Run(ctx); err != nil {
			t.Error(err)
			return
		}
	}()

	// Ожидаем подтверждения
	if r := <-cResult; r == nil || r.Error != nil {
		t.Error("ошибка проверки оповещения на email, не дождались письма")
	}

	cancel()

	if !dry.FileExists(outFile) {
		t.Error("файл out.txt не найден")
	} else {
		if str, err := dry.FileGetString(outFile); err != nil {
			t.Error(err)
		} else if strings.Trim(str, "\r\n\"\\ ") != "56137901, key2" {
			t.Fatal("содержимое out.txt отличается от ожидаемого")
		}
	}
}

func elastic_config() string {
	body := `username: ""
password: ""
addresses:
- "http://172.18.1.44/elastic"`

	tmpFile, _ := os.CreateTemp("", "*.yaml")
	tmpFile.WriteString(body)
	tmpFile.Close()

	return tmpFile.Name()
}

func click_config() string {
	body := `url: "http://localhost:8123"
user: "default"
password: ""`

	tmpFile, _ := os.CreateTemp("", "*.yaml")
	tmpFile.WriteString(body)
	tmpFile.Close()

	return tmpFile.Name()
}

func createEmail(cResult chan *temail.Result) error {
	factivation := func(from, body string) bool {
		// Если функция возвращает true это значит что почта подтверждена и нам она больше не нужна.
		// После подтверждения или по таймауту (задается в настройках) временная почта удаляется
		return strings.Trim(body, " ") == "Ошибка в базе key2" && from == "mika.temp25@mail.ru"
	}

	newEmail := new(temail.OneSecmail).Create(&temail.TmpEmailConf{
		Result:     cResult,          // канал для результата
		Timeout:    time.Second * 30, // Таймаут, в течение которого будет ожидаться письмо с подтверждением
		Activation: factivation,      // функция для обработки входящих сообщений
	})

	if err := newEmail.NewRegistration(); err != nil {
		return fmt.Errorf("произошла ошибка при регистрации почты '%s'", err.Error())
	}

	return nil
}
