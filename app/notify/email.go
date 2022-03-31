package notify

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/smtp"
	"regexp"
	"strings"
)

type Email struct {
	BaseNotify

	SMTP            string   `yaml:"smtp"`
	UserName        string   `yaml:"userName"`
	Pass            string   `yaml:"pass"`
	Subject         string   `yaml:"subject"`
	Recipients      []string `yaml:"recipients"`
	TemplateMessage string   `yaml:"templateMessage"`
}

func (n *Email) Init(logger *logrus.Entry) *Email {
	n.logger = logger.WithField("notifyType", "Email")
	return n
}

func (n *Email) Notify(items []interface{}) {

	if len(items) == 0 {
		return
	}

	n.logger.Info("оповещение на email")

	for _, item := range items {
		n.Subject = n.buildMessages(n.Subject, item)
		n.TemplateMessage = n.buildMessages(n.TemplateMessage, item)
		n.send()
	}

}

func (n *Email) CheckParams() error {
	n.SMTP = strings.Trim(n.SMTP, " ")
	if len(n.SMTP) == 0 || len(strings.Split(n.SMTP, ":")) != 2 {
		return errors.New("SMTP должен быть задан в формате 'host:port'")
	}

	n.TemplateMessage = strings.Trim(n.TemplateMessage, " ")
	if len(n.TemplateMessage) == 0 {
		return errors.New("пустая строка 'TemplateMessage'")
	}

	mailRegexp := regexp.MustCompile(`\A[\w+\-.]+@[a-z\d\-]+(\.[a-z]+)*\.[a-z]+\z`)

	if !mailRegexp.MatchString(n.UserName) {
		return errors.New("неверный формат адреса в поле 'UserName'")
	}

	if len(n.Recipients) == 0 {
		return errors.New("пустой список получателей")
	}

	errMsg := ""
	for i := len(n.Recipients) - 1; i >= 0; i-- {
		n.Recipients[i] = strings.Trim(n.Recipients[i], " ")
		if !mailRegexp.MatchString(n.Recipients[i]) {
			errMsg = fmt.Sprintf("%s, ", n.Recipients[i]) + errMsg
			n.Recipients = append(n.Recipients[:i], n.Recipients[i+1:]...)
		}
	}

	if len(errMsg) > 0 {
		errMsg = fmt.Sprintf("некорректно заполнены адреса получателей: %s", strings.TrimSuffix(errMsg, ", "))
		if len(n.Recipients) > 0 {
			// на часть адресов можно отправить
			n.logger.Info(errMsg)
		} else {
			return fmt.Errorf("проверка не выполнена, %s", errMsg)
		}
	}

	return nil
}

func (n *Email) send() {
	n.logger.Debugf("отправка сообщения %v", n.Recipients)

	auth := smtp.PlainAuth("", n.UserName, n.Pass, strings.Split(n.SMTP, ":")[0])

	header := make(map[string]string)
	header["From"] = n.UserName
	header["Subject"] = n.Subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"
	header["Importance"] = "High"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(n.TemplateMessage))

	err := smtp.SendMail(n.SMTP, auth, n.UserName, n.Recipients, []byte(message))
	if err != nil {
		n.logger.Errorf("ошибка при отправке email: %s", strings.Join(n.Recipients, "; "))
	} else {
		n.logger.Infof("оповещение отправлено, получатели %s", strings.Join(n.Recipients, "; "))
	}
}
