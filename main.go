package main

import (
	"context"
	"os"

	app "github.com/LazarenkoA/goAlerts/app"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

func init() {

}

func main() {
	kp := kingpin.New("GO Alert", "Приложение оповещения по данным систем хранения логов (elasticsearch и другие)")
	confPath := kp.Flag("confpath", "Путь к конфирурационному файлу. Путь должен задаваться как type::filePath. Например, elastic::C:\\config.yaml").String()
	checkMode := kp.Flag("checkMode", "С данным флагом приложение будет останавливаться в случае ошибок хотя бы в одном правиле, без этого флага \"плохие\" правила будут пропускаться").Bool()
	verbose := kp.Flag("v", "Подробный вывод").Bool()
	vverbose := kp.Flag("vv", "Максимально подробный вывод").Bool()

	kp.Parse(os.Args[1:])

	// --confpath=elastic::$ContentRoot$\\app\\elastic_config.yaml --vv
	// --confpath=clickhouse::$ContentRoot$\\app\\click_config.yaml --vv
	if *verbose {
		logrus.SetLevel(logrus.Level(4)) // info
	} else if *vverbose {
		logrus.SetLevel(logrus.Level(5)) // Debug
	} else {
		logrus.SetLevel(logrus.Level(3)) // Error/warning
	}

	if alert, err := new(app.Alert).Init(*confPath, *checkMode); err == nil {
		if err = alert.Run(context.Background()); err != nil {
			logrus.Fatal(err)
		}
	}
}
