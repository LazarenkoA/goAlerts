## goAlerts
goAlerts - это приложение для настройки уведомлений по данным систем логирования, таких как Elasticsearch, Clickhouse и прочих. Архитектура приложения разработана таким образом, чтобы легко можно было расширять ее функционал, а именно, новые способы уведомления, новые источники данных.
На текущий момент goAlerts умеет работать с elasticsearch и сlickhouse.

**Способы уведомления**:
- command-line interface (CLI)
- telegram
- email



### Начать использовать
- Скачать актуальный [релиз](https://github.com/LazarenkoA/goAlerts/releases )
- Собрать в ручном режиме. Ставим [Go](https://blog.golang.org/), `git clone https://github.com/LazarenkoA/goAlerts`, переходим в каталог, выполняем `go build -o "goAlerts"` для linux или `go build -o "goAlerts.exe"` для windows.


**Запуск**

`./goAlerts --confpath=elastic::C:\\app\\config.yaml --v`

❗ перед запуском в переменную окружения `RULES_DIR` нужно указать путь к каталогу с правилами оповещений.




**Параметры запуска**
- `--v` - подробный вывод (info) в stdout
- `--vv` - максимально подробный вывод (debug) в stdout
- `--confpath` - путь к конфигурационному файлу. Путь задается по шаблону `sourcetype::путь к файлу *.yaml`, например `--confpath=elastic::C:\\app\\elastic_config.yaml`, `--confpath=clickhouse::C:\\app\\click_config.yaml`
- `--checkMode` - если параметр присутствует приложение будет останавливаться в случае ошибок хотя бы в одном правиле, без этого флага "плохие" правила будут пропускаться


### Структура конфига
Структура конфигурационного файла для разных источников данных может отличаться.
Пример [elasticsearch](https://github.com/LazarenkoA/goAlerts/blob/main/app/elastic_config.yaml) и [сlickhouse](https://github.com/LazarenkoA/goAlerts/blob/main/app/click_config.yaml)

### Структура правил
```yaml
index: "techlog-*" # индекс эластика (используется только для эластика)  
rule_name: "QERR" # имя правила  
ctxField: "aggregations.group" # поле, в контексте которого будет выполняться expression, так же дочерние поля будут доступны для notify (например, для формирования текста сообщения)  
  
conditions: # правила срабатывания оповещения (настройка может отсутствовать, тогда уведомление будет по всем данным, которые вернет система логирования)
  expression: "spike(value / doc_cout) > 2" # Доступны функции average, median, spike.
  
notify:  
  telegram:  
    bot_token: ""  
    chatID:  # массив ID чатов для уведомления
      - ""  
    proxy: ""  # SOCKS5/HTTP прокси
    templateMessage: "❗Ошибки запроса\nЗа последние 8 часов выявлено %doc_count% ошибок запроса, ошибка \"%key%\""
  email:  
    smtp: ""  
    userName: ""  
    pass: ""  
    subject: ""  
    templateMessage: ""  
    recipients:  # массив email адресов для уведомления
      - ""  
  cli:  
    comand: "echo"  
  args:  
      - "Ошибка в базе %key%"   
  
shedule: "*/5 * * * *" # расписание в формате cron или @every <duration> (https://pkg.go.dev/github.com/hanagantig/cron?utm_source=godoc#hdr-CRON_Expression_Format)  
#shedule: "@every 8h"
  
# текст запроса (для эластика в формате json), если используется ELK то запрос можно получить через "Inspect"
request: ''
```


#### Подробнее про настройку conditions.expression
expression - выражение, которое должно вернуть булево, при выполнении выражения доступны поля из контекста `ctxFieldctxField`. Например, система источник возвращает такой ответ:
```json
{
  "took": 349,
  "timed_out": false,
  "_shards": {
    "total": 54,
    "successful": 54,
    "skipped": 0,
    "failed": 0
  },
  "aggregations": {
    "dbname": {
      "doc_count_error_upper_bound": 0,
      "sum_other_doc_count": 0,
      "buckets": [
        {
          "key": "acc-n39",
          "doc_count": 2
        },
        {
          "key": "acc-n41",
          "doc_count": 1
        },
        {
          "key": "hrmprof-n35",
          "doc_count": 1
        }
      ]
    }
  }
}
```

нам нужно написать выражение, чтобы уведомления приходили по элементам buckets где doc_count больше 1, в таком случае в  `ctxFieldctxField` следует указать значение `"aggregations.dbname.buckets"` и  `expression` такой `"doc_cout > 1"`.
В expression доступны выражения со скобками, доступны операторы `||` - или, `&&` - и
Так же доступны функции:
- `average` - среднее арифметическое
- `median`  - медиана
- `spike`  - функция возвращает показатель на сколько больше текущее значение от среднего, т.е. факт того что произошел всплеск показаний.
  Все функции считаются по "окну" в 10 последних значений. Например в нашем буфере такие значения `[233 100 272666 1519 532 278 226 112 112]` среднее 30642 и приходит новое значение 465093 в этом случае spike будет равен 15 

Если бы buckets имел вложенную структуру, и в выражении нужно было бы обратиться к `timelock.value`, то в этом случае поля следует заключать в квадратные скобки. Например, `expression: "[timelock.value] > 10 && key == \"key2\""`
```json
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
```

НО при формировании шаблона текста сообщения квадратные скобки использовать не нужно.
