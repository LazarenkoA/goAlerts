package rules

import (
	"errors"
	"fmt"
	nfy "github.com/LazarenkoA/goAlerts/app/notify"
	"github.com/knetic/govaluate"
	"github.com/sirupsen/logrus"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Number interface {
	int64 | float64 | int | float32
}

type Сondition struct {
	Expression string `json:"expression"`

	logger *logrus.Entry
	buff   sync.Map
}

const buffSize = 10

func (c *Сondition) Init(logger *logrus.Entry) {
	c.logger = logger
}

func (c *Сondition) filterByCondition(matches []interface{}) (resp []interface{}) {
	c.logger.Debug("фильтрация по условию")

	appendBuff := func(arg interface{}, key string) {
		switch v := arg.(type) {
		case float64:
			buff, _ := c.buff.Load(key)
			if buff == nil {
				buff = []int64{}
			}

			buff = append(buff.([]int64), int64(v))
			if l := len(buff.([]int64)); l > buffSize {
				buff = buff.([]int64)[l-buffSize:]
			}

			c.buff.Store(key, buff)
		}
	}

	functions := map[string]govaluate.ExpressionFunction{
		"spike": func(args ...interface{}) (interface{}, error) {
			if len(args) >= 1 && len(args) <= 2 {
				key := ""
				if len(args) == 2 {
					if _, ok := args[1].(string); !ok {
						return nil, errors.New("spike: второй параметр должен быть строкой")
					}
					key = args[1].(string)
				}

				appendBuff(args[0], key)
				buff, _ := c.buff.Load(key)
				sp := spike(buff.([]int64))
				c.logger.WithField("spike", sp).WithField("key", key).WithField("buff", buff.([]int64)).Debug()

				return float64(sp), nil
			} else {
				return nil, errors.New(fmt.Sprintf("spike: bad params. Неверное количество параметров (параметры: %v)", args))
			}
		},
		"average": func(args ...interface{}) (interface{}, error) {
			if len(args) == 1 {
				appendBuff(args[0], "")
				buff, _ := c.buff.Load("")
				av := average(buff.([]int64))

				c.logger.WithField("average", av).WithField("buff", buff.([]int64)).Debug()

				return float64(av), nil
			} else {
				return nil, errors.New("average: bad params")
			}
		},
		"mediana": func(args ...interface{}) (interface{}, error) {
			if len(args) == 1 {
				appendBuff(args[0], "")
				buff, _ := c.buff.Load("")
				m := mediana(buff.([]int64))

				c.logger.WithField("mediana", m).WithField("buff", buff.([]int64)).Debug()

				return float64(m), nil
			} else {
				return nil, errors.New("mediana: bad params")
			}
		},
	}

	var rexp = regexp.MustCompile(`(?m)\[(.+?)\]`)
	compsField := rexp.FindAllStringSubmatch(c.Expression, -1)

	for _, m := range matches {
		if m == nil {
			continue
		}
		if _, ok := m.(map[string]interface{}); !ok {
			continue
		}
		for _, matchRg := range compsField {
			if v := nfy.GetValue(m.(map[string]interface{}), matchRg[1]); len(v) == 1 {
				valueString := strconv.FormatFloat(v[0].(float64), 'f', -1, 64)
				c.Expression = strings.Replace(c.Expression, matchRg[0], valueString, -1)
			}
		}

		if expression, err := govaluate.NewEvaluableExpressionWithFunctions(c.Expression, functions); err == nil {
			if result, err := expression.Evaluate(m.(map[string]interface{})); err != nil {
				c.logger.WithError(err).Errorf("ошибка выпонения выражения (%q)", c.Expression)
			} else {
				if r, ok := result.(bool); !ok {
					c.logger.Errorf("ошибка выпонения выражения (%q). Результат выражения должен быть bool", c.Expression)
				} else if r {
					resp = append(resp, m)
				}
			}
		} else {
			c.logger.WithError(err).Errorf("ошибка разбора выражения (%q)", c.Expression)
		}
	}

	return resp
}

func spike[T Number](in []T) T {
	if len(in) == 0 {
		return 0
	}

	// Если текущее значение меньше чем предыдущее, значит произошло падение, на такое мы не реагируем.
	// такое может быть при таких данных buffer=[130, 100, 329, 216, 90]
	downturn := len(in) > 1 && in[len(in)-2] > in[len(in)-1]
	av := average(in[:len(in)-1]) // среднюю считаем без учета текущего значения (оно последним будет)
	if av == 0 {
		return 0
	}

	if !downturn && len(in) > 3 {
		return in[len(in)-1] / av
	} else {
		return 0
	}
}

func average[T Number](in []T) (result T) {
	for _, v := range in {
		result += v / T(len(in))
	}

	return result
}

func mediana[T Number](selection []T) float64 {
	tmp := make([]T, len(selection), len(selection)) // что б исходный массив не сортировался
	copy(tmp, selection)
	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i] < tmp[j]
	})

	if len(tmp)%2 != 0 {
		return float64(tmp[((len(tmp) - 1) / 2)])
	} else {
		return float64(tmp[(len(tmp)/2)-1]+tmp[(len(tmp)/2)]) / 2
	}
}
