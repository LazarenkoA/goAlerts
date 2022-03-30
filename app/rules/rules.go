package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	nfy "github.com/LazarenkoA/goAlerts/app/notify"
	"github.com/LazarenkoA/goAlerts/app/source"
	"github.com/hanagantig/cron"
	"github.com/sirupsen/logrus"
	"github.com/ungerik/go-dry"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Isource interface {
	GetData(string, ...string) ([]byte, error)
	ReadConf(string) error
	RequestCheck(string) error
}

type Inotify interface {
	Notify([]interface{})
	CheckParams() error
}

type notifyConf struct {
	Telegram *nfy.Telegram `yaml:"telegram"`
	Email    *nfy.Email    `yaml:"email"`
	CLI      *nfy.CLI      `yaml:"cli"`
}

type Rule struct {
	RuleName string `yaml:"rule_name"`

	// индекс эластика
	Index string `yaml:"index"`

	// расписание в формате cron
	Shedule string `yaml:"shedule"`

	// текст запроса
	Request string `yaml:"request"`

	Condition *Сondition `yaml:"condition"`

	CtxField string `yaml:"ctxField"`

	Notify *notifyConf `yaml:"notify"`

	logger *logrus.Entry
}

const envName = "RULES_DIR"

type Rules struct {
	logger *logrus.Entry
	rules  []*Rule
}

func (r *Rules) Init(logger *logrus.Entry) *Rules {
	r.logger = logger.WithField("name", "Rules")
	r.rules = []*Rule{}

	return r
}

func (r *Rules) Rules() []*Rule {
	return r.rules
}

func (r *Rules) RulesLoad(checkMode bool) error {
	rules_dir := os.Getenv(envName)
	if rules_dir == "" {
		r.logger.Warnf("не задан каталог с правилами (правила будут искаться в текущем каталоге). Путь задается в переменной окружения %s", envName)
		return errors.New("не задан каталог с правилами ")
	}
	if !dry.FileExists(rules_dir) {
		return fmt.Errorf("каталог %q не найден", rules_dir)
	}
	r.logger.Debugf("чтение правил из каталога %q", rules_dir)

	if files, err := FileFind(rules_dir, "*.yaml"); err == nil {
		for _, filePath := range files {
			if err = r.appendRule(filePath); err != nil && checkMode {
				return fmt.Errorf("ошибка загрузки роли: %w", err)
			}
		}
	} else {
		return err
	}

	return nil
}

func (r *Rules) appendRule(filePath string) (err error) {
	defer func() {
		if err != nil {
			r.logger.WithError(err).Error()
		}
	}()

	if !dry.FileExists(filePath) {
		return fmt.Errorf("файл правил %q не найден", filePath)
	}
	data, err := dry.FileGetBytes(filePath, time.Second*5)
	if err != nil {
		return fmt.Errorf("файл правил %q пропущен", filePath)
	}

	newRule := new(Rule)
	if err = yaml.Unmarshal(data, newRule); err != nil {
		return fmt.Errorf("файл правил %q пропущен", filePath)
	}
	if newRule.RuleName == "" {
		return fmt.Errorf("файл правил %q пропущен, не заполнено имя", filePath)
	}
	if newRule.Notify == nil {
		return fmt.Errorf("файл правил %q пропущен, не заполнен параметр notify", filePath)
	}

	newRule.logger = r.logger.WithField("name", newRule.RuleName)
	//newRule.data = out
	//newRule.createNotify()

	r.rules = append(r.rules, newRule)
	r.logger.Infof("Правило %q успешно загружено", newRule.RuleName)

	return nil
}

func (r *Rule) Run(ctx context.Context, wg *sync.WaitGroup, src Isource) {
	defer wg.Done()

	if r.Shedule == "" {
		r.logger.Errorf("для правила %q не задано расписание", r.RuleName)
		return
	}

	// проверка request
	if err := src.RequestCheck(r.Request); err != nil && !errors.Is(err, source.RequestIsEmpty) {
		r.logger.WithError(err).Error("в настройках некорректно задано поле request")
		return
	}

	r.invoke(src) // запускаем первый раз, что б отработало при старте приложения и не ждало наступления расписания
	myCron := cron.New()
	if _, err := myCron.AddFunc(r.Shedule, r.RuleName, func() { r.invoke(src) }); err != nil {
		r.logger.WithError(err).Error("ошибка добавления задания")
		return
	}
	myCron.Start()
	r.logger.Infof("задание по правилу %q запланировано", r.RuleName)

	select {
	case <-ctx.Done():
		myCron.Stop()
		return
	}
}

func (r *Rule) invoke(src Isource) {
	r.logger.Infof("плановое выполнения правила %q", r.RuleName)

	raw_data, err := src.GetData(r.Request, r.Index)
	if err != nil {
		r.logger.WithError(err).Error("ошибка выполнения запроса")
		return
	}

	var data map[string]interface{}
	err = json.Unmarshal(raw_data, &data)
	if err != nil {
		r.logger.WithError(err).Error("ошибка чтения результата запроса")
		return
	}

	filteredData := nfy.GetValue(data, r.CtxField)
	if r.Condition != nil {
		r.Condition.Init(r.logger)
		filteredData = r.Condition.filterByCondition(filteredData)
	}
	if r.Notify != nil {
		if r.Notify.CLI != nil {
			r.notify(r.Notify.CLI.Init(r.logger), filteredData)
		}
		if r.Notify.Telegram != nil {
			r.notify(r.Notify.Telegram.Init(r.logger), filteredData)
		}
		if r.Notify.Email != nil {
			r.notify(r.Notify.Email.Init(r.logger), filteredData)
		}
	}
}

func (r *Rule) notify(ntf Inotify, filteredData []interface{}) {
	if ntf == nil {
		return
	}

	if err := ntf.CheckParams(); err == nil {
		ntf.Notify(filteredData)
	}
}

//func (r *Rule) createNotify() {
//	if r.NotifyConf == nil {
//		r.logger.Fatalf("у правила %q не заданы настройки оповещения", r.RuleName)
//	}
//
//	if r.NotifyConf.CLI != nil {
//		r.Notifys = append(r.Notifys, &nfy.CLI{})
//	}
//	if r.NotifyConf.Telegram != nil {
//		r.Notifys = append(r.Notifys, &nfy.Telegram{})
//	}
//	if r.NotifyConf.Email != nil {
//		r.Notifys = append(r.Notifys, &nfy.Email{})
//	}
//
//}

func FileFind(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}
