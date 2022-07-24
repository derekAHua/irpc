package log

type dummyLogger struct{}

func (l *dummyLogger) Debug(_ ...interface{}) {
}

func (l *dummyLogger) Debugf(_ string, _ ...interface{}) {
}

func (l *dummyLogger) Info(_ ...interface{}) {
}

func (l *dummyLogger) Infof(_ string, _ ...interface{}) {
}

func (l *dummyLogger) Warn(_ ...interface{}) {
}

func (l *dummyLogger) Warnf(_ string, _ ...interface{}) {
}

func (l *dummyLogger) Error(_ ...interface{}) {
}

func (l *dummyLogger) Errorf(_ string, _ ...interface{}) {
}

func (l *dummyLogger) Fatal(_ ...interface{}) {
}

func (l *dummyLogger) Fatalf(_ string, _ ...interface{}) {
}

func (l *dummyLogger) Panic(_ ...interface{}) {
}

func (l *dummyLogger) Panicf(_ string, _ ...interface{}) {
}
