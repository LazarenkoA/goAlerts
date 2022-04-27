package notify

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTP struct {
	BaseNotify

	URL    string   `yaml:"url"`
	Method string   `yaml:"method"`
	Body   string   `yaml:"body"`
	Params []string `yaml:"params"`
	Proxy  string   `yaml:"proxy"`
	_url   *url.URL

	httpClient *http.Client
}

func (h *HTTP) Init(logger *logrus.Entry) *HTTP {
	h.logger = logger.WithField("notifyType", "HTTP")
	return h
}

func (h *HTTP) Notify(items []interface{}) {
	if len(items) == 0 {
		return
	}
	h.logger.Info("оповещение HTTP")

	for _, item := range items {
		var params []string
		for _, p := range h.Params {
			params = append(params, h.buildMessages(p, item))
		}
		h.sendRequest(params)
	}
}

func (h *HTTP) sendRequest(params []string) {
	if h.httpClient == nil {
		h.createHttpClient()
	}

	q := h._url.Query()
	for _, param := range params {
		if parts := strings.Split(param, "="); len(parts) >= 2 {
			q.Set(parts[0], strings.Join(parts[1:], "="))
		} else {
			h.logger.WithField("param", param).WithField("parts", parts).Warning("параметр пропущен")
		}
	}

	h._url.RawQuery = q.Encode()
	req, err := http.NewRequest(h.Method, h._url.String(), strings.NewReader(h.Body))
	if err != nil {
		h.logger.WithError(err).Error("ошибка формирования запроса")
		return
	}

	if resp, err := h.httpClient.Do(req); err != nil {
		h.logger.WithError(err).Error("ошибка отправки запроса")
		return
	} else {
		if resp.StatusCode-(resp.StatusCode%100) != 200 {
			h.logger.Errorf("ошибка отправки сообщения, код ответа %d", resp.StatusCode)
			return
		}
	}
}

func (h *HTTP) createHttpClient() {
	h.logger.Debug("создание http клиента")

	httpTransport := &http.Transport{}
	if h.Proxy != "" {
		h.logger.Debug("используем прокси " + h.Proxy)

		if pURL, err := url.Parse(h.Proxy); err == nil {
			httpTransport.Proxy = http.ProxyURL(pURL)
		} else {
			httpTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				select {
				case <-ctx.Done():
					return nil, nil
				default:
				}

				dialer, err := proxy.SOCKS5("tcp", h.Proxy, nil, proxy.Direct)
				if err != nil {
					h.logger.WithField("Прокси", h.Proxy).WithError(err).Error("ошибка соединения с прокси")
					return nil, err
				}

				return dialer.Dial(network, addr)
			}
		}
	}
	//httpTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //set ssl

	//cookieJar, _ := cookiejar.New(nil)
	h.httpClient = &http.Client{
		Transport: httpTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
		//Jar:     cookieJar,
		Timeout: time.Minute,
	}
}

func (h *HTTP) CheckParams() (err error) {
	if h.Method != "POST" && h.Method != "GET" {
		return errors.New("методы поддерживаются POST и GET")
	}
	if h.URL == "" {
		return errors.New("не заполнен url")
	}

	h._url, err = url.Parse(h.URL)

	return err
}
