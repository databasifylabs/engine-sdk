package enums

type LogDestinationKind string

const (
	LogDestinationRsyslog       LogDestinationKind = "rsyslog"
	LogDestinationElasticsearch LogDestinationKind = "elasticsearch"
	LogDestinationOpensearch    LogDestinationKind = "opensearch"
)

func (k LogDestinationKind) Valid() bool {
	switch k {
	case LogDestinationRsyslog, LogDestinationElasticsearch, LogDestinationOpensearch:
		return true
	}

	return false
}
