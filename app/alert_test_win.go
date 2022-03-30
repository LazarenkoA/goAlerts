//go:build windows

package alert

import (
	"fmt"
	"os"
	"path"
)

func elastic_rule(batFile, email string) string {
	body := fmt.Sprintf(`index: "techlog-*" # индекс эластика
rule_name: "Test_elastic" # имя правила
ctxField: "aggregations.errors.buckets"

condition: # правила срабатывания оповещения
  expression: "[timelock.value] > 10 && key == \"key2\"" 

notify:
  email:
    smtp: "smtp.mail.ru:587"
    userName: "mika.temp25@mail.ru"
    pass: "8Qm2jF2KRBwBwGUb8n7x"
    subject: "Ошибка в базе %%key%%"
    templateMessage: "Ошибка в базе %%key%%"
    recipients:
      - %s
      - "bademail"
  cli:
    comand: "cmd"
    args:
      - /C
      - %s "%%timelock.value%%, %%key%%"
shedule: "@every 1m"

# текст запроса в формате
request: ''`, email, batFile)

	dirPath := path.Join(os.TempDir(), "elastic")
	os.Mkdir(dirPath, os.ModePerm)
	tmpFile, _ := os.CreateTemp(dirPath, "*.yaml")
	tmpFile.WriteString(body)
	tmpFile.Close()

	return tmpFile.Name()
}

func click_rule(batFile string) string {

	body := fmt.Sprintf(`rule_name: "Test_click" # имя правила
ctxField: "data" 

condition: # правила срабатывания оповещения
  expression: "value >= 50 && Name == \"key2\"" 

notify:
  cli:
    comand: "cmd"
    args:
      - /C
      - %s "%%value%%, %%Name%%"
shedule: "@every 1m"

# текст запроса в формате
request: ''`, batFile)

	dirPath := path.Join(os.TempDir(), "Clickhouse")
	os.Mkdir(dirPath, os.ModePerm)
	tmpFile, _ := os.CreateTemp(dirPath, "*.yaml")
	tmpFile.WriteString(body)
	tmpFile.Close()

	return tmpFile.Name()
}

func createScriptFile(outFile string) string {
	tmpFile, _ := os.CreateTemp("", "*.bat")
	tmpFile.WriteString(fmt.Sprintf("@echo %%1 > %s", outFile))
	tmpFile.Close()

	return tmpFile.Name()
}
