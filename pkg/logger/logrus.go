package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// New инициализирует и настраивает новый экземпляр logrus.Logger.
// level — уровень логирования ("debug", "info", "warn", "error").
// isProd — если true, включает JSON-форматирование для систем сбора логов;
// если false — включает читаемый текстовый вывод с цветами для локальной разработки.
func New(level string, isProd bool) *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)

	parsedLevel, err := logrus.ParseLevel(strings.ToLower(strings.TrimSpace(level)))
	if err != nil {
		parsedLevel = logrus.InfoLevel
	}
	log.SetLevel(parsedLevel)

	if isProd {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	return log
}
