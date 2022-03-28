package source

import (
	"encoding/json"
	"errors"
	"fmt"
	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/ungerik/go-dry"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
	"time"
)

type Elasticsearch struct {
	conf   *elasticsearch7.Config
	client *elasticsearch7.Client
	logger *logrus.Entry
}

var RequestIsEmpty = errors.New("Request is empty")

func (e *Elasticsearch) Init(confPath string, logger *logrus.Entry) *Elasticsearch {
	e.logger = logger

	err := e.ReadConf(confPath)
	if err != nil {
		e.logger.Fatal(err)
	}
	e.client, err = elasticsearch7.NewClient(*e.conf)
	if err != nil {
		e.logger.Fatalf("ошибка создания клиента elasticsearch: %s", err)
	}

	return e
}

func (e *Elasticsearch) ReadConf(confPath string) error {
	cfg := map[string]interface{}{}

	if !dry.FileExists(confPath) {
		return fmt.Errorf("конфигурационный файл \"%s\" не найден", confPath)
	}
	data, err := dry.FileGetBytes(confPath, time.Second*5)
	if err != nil {
		return fmt.Errorf("ошибка чтения настроек: %w", err)
	}
	if err = yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("ошибка чтения настроек: %w", err)
	}

	e.conf = new(elasticsearch7.Config)
	if err = mapstructure.Decode(cfg, e.conf); err != nil {
		return fmt.Errorf("ошибка декодирования данных в структуру: %w", err)
	}
	if cert, ok := cfg["cert"]; ok && cert != "" {
		if e.conf.CACert, err = dry.FileGetBytes(cert.(string), time.Second*5); err != nil {
			return err
		}
	}

	return nil
}

func (e *Elasticsearch) GetData(req string, addParams ...string) ([]byte, error) {
	index := ""
	if len(addParams) == 1 {
		index = addParams[0]
	}

	res, err := e.client.Search(e.client.Search.WithBody(strings.NewReader(req)), e.client.Search.WithIndex(index))
	if err != nil {
		return []byte{}, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return []byte{}, fmt.Errorf("Error: %s", res.String())
	}
	bdata, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}

	return bdata, nil
}

func (e *Elasticsearch) RequestCheck(request string) error {
	request = strings.Trim(request, " ")
	if request == "" {
		return RequestIsEmpty
	}

	//currentDir, _ := os.Getwd()
	req := map[string]interface{}{}
	if err := json.Unmarshal([]byte(request), &req); err != nil {
		return fmt.Errorf("oшибка разбора json %w", err)
	}

	return nil
}
