package alert

import (
	"context"
	"strings"
	"sync"

	"github.com/LazarenkoA/goAlerts/app/rules"
	src "github.com/LazarenkoA/goAlerts/app/source"
	"github.com/sirupsen/logrus"
)

type Alert struct {
	logger    *logrus.Entry
	source    rules.Isource
	checkMode bool
}

/////////////////////////////////////////////

func (a *Alert) Init(confPath string, checkMode bool) (result *Alert, err error) {
	a.logger = logrus.WithField("name", "main")
	a.checkMode = checkMode

	if parts := strings.Split(confPath, "::"); len(parts) != 2 {
		a.logger.Info("путь должен задаваться как type::filePath. Например, elastic::C:\\config.yaml")
		a.logger.Fatalf("путь к конфигурационному файлу задан не корректно")
	} else {
		a.source = a.factory(parts[0], parts[1])
	}

	return a, err
}

func (a *Alert) factory(stype, confPath string) rules.Isource {
	switch strings.ToLower(stype) {
	case "elastic":
		return new(src.Elasticsearch).Init(confPath, a.logger)
	case "clickhouse":
		return new(src.Clickhouse).Init(confPath, a.logger)
	default:
		a.logger.Fatalf("тип источника \"%s\" не поддерживается", stype)
		return nil
	}
}

func (a *Alert) Run(ctx context.Context) error {
	r := new(rules.Rules).Init(a.logger)
	if err := r.RulesLoad(a.checkMode); err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	for _, r := range r.Rules() {
		wg.Add(1)
		go r.Run(ctx, wg, a.source)
	}
	wg.Wait()

	return nil
}
