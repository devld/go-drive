package logging

import (
	"context"
	"errors"
	"fmt"
	"time"

	gormLogger "gorm.io/gorm/logger"
	gormUtils "gorm.io/gorm/utils"
)

// NewGormLogger maps GORM SQL traces to the application log levels: regular
// SQL is debug, slow SQL is warn, and failed SQL is error.
func NewGormLogger(component string, config gormLogger.Config) gormLogger.Interface {
	return newGormLogger(For(component), config)
}

type gormLoggerAdapter struct {
	logger *Logger
	config gormLogger.Config
}

func newGormLogger(log *Logger, config gormLogger.Config) gormLogger.Interface {
	return &gormLoggerAdapter{logger: log, config: config}
}

func (l *gormLoggerAdapter) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	copy := *l
	copy.config.LogLevel = level
	return &copy
}

func (l *gormLoggerAdapter) Info(_ context.Context, message string, args ...interface{}) {
	if l.config.LogLevel >= gormLogger.Info {
		l.logger.Infof(message, args...)
	}
}

func (l *gormLoggerAdapter) Warn(_ context.Context, message string, args ...interface{}) {
	if l.config.LogLevel >= gormLogger.Warn {
		l.logger.Warnf(message, args...)
	}
}

func (l *gormLoggerAdapter) Error(_ context.Context, message string, args ...interface{}) {
	if l.config.LogLevel >= gormLogger.Error {
		l.logger.Errorf(message, args...)
	}
}

func (l *gormLoggerAdapter) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.config.LogLevel <= gormLogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.config.LogLevel >= gormLogger.Error &&
		(!errors.Is(err, gormLogger.ErrRecordNotFound) || !l.config.IgnoreRecordNotFoundError):
		l.logSQL(ErrorLevel, elapsed, err.Error(), fc)
	case elapsed > l.config.SlowThreshold && l.config.SlowThreshold != 0 && l.config.LogLevel >= gormLogger.Warn:
		l.logSQL(WarnLevel, elapsed, fmt.Sprintf("SLOW SQL >= %v", l.config.SlowThreshold), fc)
	case l.config.LogLevel == gormLogger.Info:
		l.logSQL(DebugLevel, elapsed, "", fc)
	}
}

func (l *gormLoggerAdapter) ParamsFilter(_ context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.config.ParameterizedQueries {
		return sql, nil
	}
	return sql, params
}

func (l *gormLoggerAdapter) logSQL(level Level, elapsed time.Duration, detail string, fc func() (string, int64)) {
	sql, rows := fc()
	rowValue := any(rows)
	if rows == -1 {
		rowValue = "-"
	}

	location := "at " + gormUtils.FileWithLineNum()
	if detail != "" {
		location += " " + detail
	}
	message := fmt.Sprintf("[%.3fms] [rows:%v] %s\n%s",
		float64(elapsed.Nanoseconds())/1e6, rowValue, sql, location)
	l.logger.log(level, "%s", message)
}
