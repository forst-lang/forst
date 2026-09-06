package transformergo

import logrus "github.com/sirupsen/logrus"

func (t *Transformer) stmtLog(fields logrus.Fields, msg string) {
	if t.log != nil {
		t.log.WithFields(fields).Debug(msg)
	}
}
