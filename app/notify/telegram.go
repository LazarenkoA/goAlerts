package notify

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Telegram struct {
	BaseNotify

	Bot_token       string   `yaml:"bot_token"`
	ChatID          []string `yaml:"chatID"`
	Proxy           string   `yaml:"proxy"`
	TemplateMessage string   `yaml:"templateMessage"`

	httpClient *http.Client
}

const telegramAPI = "https://api.telegram.org"

func (tel *Telegram) Notify(items []interface{}, l *logrus.Entry) {
	if len(items) == 0 {
		return
	}
	if tel.logger == nil {
		tel.logger = l
	}
	tel.logger.Info("оповещение Telegram")

	if tel.Bot_token == "" {
		tel.logger.Error("отправка не выполнена, не заполнен Bot_token")
		return
	}

	for _, item := range items {
		message := tel.buildMessages(tel.TemplateMessage, item)
		tel.send(message)
	}
}

func (tel *Telegram) send(message string) {
	for _, id := range tel.ChatID {
		tel.sendRequest(id, message)
		tel.logger.Debugf("отправка сообщения в чат %v", id)
	}
}

func (tel *Telegram) sendRequest(id, message string) {
	if tel.httpClient == nil {
		tel.createHttpClient()
	}

	apiUrl, err := url.Parse(fmt.Sprintf("%s/bot%s/sendMessage", telegramAPI, tel.Bot_token))
	if err != nil {
		return
	}
	q := apiUrl.Query()
	q.Set("chat_id", id)
	q.Set("text", message)

	apiUrl.RawQuery = q.Encode()
	if resp, err := tel.httpClient.Post(apiUrl.String(), "application/x-www-form-urlencoded", nil); err != nil {
		tel.logger.WithError(err).WithField("chat_id", id).Error("ошибка отправки сообщения")
		return
	} else {
		if resp.StatusCode-(resp.StatusCode%100) != 200 {
			tel.logger.WithField("chat_id", id).Errorf("ошибка отправки сообщения, код ответа %d", resp.StatusCode)
			return
		}
	}
}

func (tel *Telegram) createHttpClient() {
	tel.logger.Debug("создание http клиента")

	httpTransport := &http.Transport{}
	if tel.Proxy != "" {
		tel.logger.Debug("используем прокси " + tel.Proxy)
		httpTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			select {
			case <-ctx.Done():
				return nil, nil
			default:
			}

			dialer, err := proxy.SOCKS5("tcp", tel.Proxy, nil, proxy.Direct)
			if err != nil {
				tel.logger.WithField("Прокси", tel.Proxy).WithError(err).Error("ошибка соединения с прокси")
				return nil, err
			}

			return dialer.Dial(network, addr)
		}
	}

	//cookieJar, _ := cookiejar.New(nil)
	tel.httpClient = &http.Client{
		Transport: httpTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
		//Jar:     cookieJar,
		Timeout: time.Minute,
	}
}
