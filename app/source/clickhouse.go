package source

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/ungerik/go-dry"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ClickhouseConf struct {
	URL               string `yaml:"url"`
	User              string `yaml:"user"`
	Password          string `yaml:"password"`
	Max_result_bytes  int    `yaml:"max_result_bytes"`
	Buffer_size       int    `yaml:"buffer_size"`
	Wait_end_of_query int    `yaml:"wait_end_of_query"`
}

type Clickhouse struct {
	conf   *ClickhouseConf
	logger *logrus.Entry
}

func (cl *Clickhouse) Init(confPath string, logger *logrus.Entry) *Clickhouse {
	cl.logger = logger

	err := cl.ReadConf(confPath)
	if err != nil {
		cl.logger.Fatal(err)
	}

	return cl
}

func (cl *Clickhouse) GetData(req string, addParams ...string) ([]byte, error) {
	return cl.sendRequest(req)
}

func (cl *Clickhouse) ReadConf(confPath string) error {
	if !dry.FileExists(confPath) {
		return fmt.Errorf("конфигурационный файл \"%s\" не найден", confPath)
	}
	data, err := dry.FileGetBytes(confPath, time.Second*5)
	if err != nil {
		return fmt.Errorf("ошибка чтения настроек: %w", err)
	}
	cl.conf = new(ClickhouseConf)
	if err = yaml.Unmarshal(data, cl.conf); err != nil {
		return fmt.Errorf("ошибка чтения настроек: %w", err)
	}

	return nil
}

func (cl *Clickhouse) sendRequest(query string) (result []byte, err error) {
	defer func() {
		if err != nil {
			cl.logger.WithError(err).Error("ошибка отправки запроса в сlickhouse")
		}
	}()

	if cl.conf == nil {
		return result, errors.New("запрос не отправлен, не заполнены настройки для работы с clickhouse")
	}

	clickhouseUrl, err := url.Parse(cl.conf.URL)
	if err != nil {
		return result, fmt.Errorf("ошибка в адресе сlickhouse: %w", err)
	}
	q := clickhouseUrl.Query()
	if query == "" {
		return result, errors.New("не заполнен запрос")
	} else {
		q.Set("query", query)
	}

	if cl.conf.Buffer_size > 0 {
		q.Set("buffer_size", strconv.Itoa(cl.conf.Buffer_size))
	}
	if cl.conf.Max_result_bytes > 0 {
		q.Set("max_result_bytes", strconv.Itoa(cl.conf.Max_result_bytes))
	}
	if cl.conf.Wait_end_of_query > 0 {
		q.Set("wait_end_of_query", strconv.Itoa(cl.conf.Wait_end_of_query))
	}

	clickhouseUrl.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", clickhouseUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	if cl.conf.User != "" && cl.conf.Password != "" {
		req.Header.Set("X-ClickHouse-User", cl.conf.User)
		req.Header.Set("X-ClickHouse-Key", cl.conf.Password)
	}

	httpClient := &http.Client{Timeout: time.Minute}
	if resp, err := httpClient.Do(req); err != nil {
		return result, err
	} else {
		if resp.StatusCode-(resp.StatusCode%100) != 200 {
			return result, fmt.Errorf("ошибка отправки запроса, код ответа %d", resp.StatusCode)
		}
		defer resp.Body.Close()
		result, _ = ioutil.ReadAll(resp.Body)
	}

	return result, nil
}

func (cl *Clickhouse) RequestCheck(request string) error {
	request = strings.Trim(request, " ")
	if request == "" {
		return RequestIsEmpty
	}

	if strings.Index(strings.ToUpper(request), "FORMAT JSON") < 0 {
		return errors.New("запрос должен быть в формате json (.... FORMAT JSON)")
	}

	return nil
}
