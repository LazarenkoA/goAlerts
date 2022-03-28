package notify

import "github.com/sirupsen/logrus"

type Email struct {
	BaseNotify

	SMTP            string   `yaml:"smtp"`
	UserName        string   `yaml:"userName"`
	Pass            string   `yaml:"pass"`
	Subject         string   `yaml:"subject"`
	Recipients      []string `yaml:"recipients"`
	TemplateMessage string   `yaml:"templateMessage"`
}

func (n *Email) Notify(items []interface{}, logger *logrus.Entry) {

}
