//go:build windows

package alert

import (
	"fmt"
	"os"
	"path"
)

func elastic_rule(batFile string) string {
	body := fmt.Sprintf(`index: "techlog-*" # индекс эластика
rule_name: "Test_elastic" # имя правила
ctxField: "aggregations.groupcontext.buckets"

condition: # правила срабатывания оповещения
  expression: "doc_count >= 50" 

notify:
  cli:
    command: %s
    args:
      - "Выявлено %%doc_count%% блокировок с ожиданием более 10 секунд, последняя строка стека %%key%%. Проблема наблюдается в базах %%groupdb.buckets.key%%"
shedule: "@every 1m"

# текст запроса в формате
request: ''`, batFile)

	dirPath := path.Join(os.TempDir(), "elastic")
	os.Mkdir(dirPath, os.ModePerm)
	tmpFile, _ := os.CreateTemp(dirPath, "*.yaml")
	tmpFile.WriteString(body)
	tmpFile.Close()

	return tmpFile.Name()
}

func elastic_rule_with_email(email string) string {
	body := fmt.Sprintf(`index: "techlog-*" # индекс эластика
rule_name: "Test_elastic" # имя правила
ctxField: "aggregations.groupcontext.buckets"

condition: # правила срабатывания оповещения
  expression: "doc_count >= 50" 

notify:
  email:
    smtp: "smtp.mail.ru:587"
    userName: "mika.temp25@mail.ru"
    pass: "8Qm2jF2KRBwBwGUb8n7x"
    subject: "Ошибка в базах %%groupdb.buckets.key%%"
    templateMessage: "Выявлено %%doc_count%% блокировок с ожиданием более 10 секунд, последняя строка стека %%key%%. Проблема наблюдается в базах %%groupdb.buckets.key%%"
    recipients:
      - %s
      - "bademail"
shedule: "@every 1m"

# текст запроса в формате
request: ''`, email)

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
  http:
    url: "https://1c-fresh.parma.tech/TelegramRequestsProxy/toChat"
    method: "GET"
    params:
      - "txt=%%value%%, %%Name%%"
      - "botToken=832480422:AAGO9Lm8ekPofPGKTg31oswzRTNcH4ffhCw"
      - "chatID=-599333313,251159934"
  cli:
    command: "cmd"
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
