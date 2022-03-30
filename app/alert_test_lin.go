//go:build linux

package alert

import (
	"fmt"
	"os"
	"path"
)

func elastic_rule(shellFile string) string {
	body := fmt.Sprintf(`index: "techlog-*" # индекс эластика
rule_name: "Test_elastic" # имя правила
ctxField: "aggregations.errors.buckets"

condition: # правила срабатывания оповещения
  expression: "[timelock.value] > 10 && key == \"key2\"" 

notify:
  cli:
    comand: %s
    args:
      - "%%timelock.value%%, %%key%%"
shedule: "@every 1s"

# текст запроса в формате
request: ''`, shellFile)

	dirPath := path.Join(os.TempDir(), "elastic")
	os.Mkdir(dirPath, os.ModePerm)
	tmpFile, _ := os.CreateTemp(dirPath, "*.yaml")
	tmpFile.WriteString(body)
	tmpFile.Close()

	return tmpFile.Name()
}

func click_rule(shellFile string) string {

	body := fmt.Sprintf(`rule_name: "Test_click" # имя правила
ctxField: "data" 

condition: # правила срабатывания оповещения
  expression: "value >= 50 && Name == \"key2\"" 

notify:
  cli:
    comand: %s
    args:
      - "%%value%%, %%Name%%"
shedule: "@every 1s"

# текст запроса в формате
request: ''`, shellFile)

	dirPath := path.Join(os.TempDir(), "Clickhouse")
	os.Mkdir(dirPath, os.ModePerm)
	tmpFile, _ := os.CreateTemp(dirPath, "*.yaml")
	tmpFile.WriteString(body)
	tmpFile.Close()

	return tmpFile.Name()
}

func createScriptFile(outFile string) string {
	tmpFile, _ := os.CreateTemp("", "*.sh")
	tmpFile.WriteString(fmt.Sprintf("#!/bin/sh \n"+
		"echo $1 > %s", outFile))
	tmpFile.Close()
	os.Chmod(tmpFile.Name(), os.ModePerm|os.ModeExclusive)

	return tmpFile.Name()
}
