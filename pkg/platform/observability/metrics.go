package observability

// Count records an increment of the named counter metric.
func Count(name string, value int64, tags ...string) error {
	if statsdClient == nil {
		return nil
	}
	return statsdClient.Count(name, value, tags, 1)
}

// Histogram records a sample for the named histogram metric.
func Histogram(name string, value float64, tags ...string) error {
	if statsdClient == nil {
		return nil
	}
	return statsdClient.Histogram(name, value, tags, 1)
}

// Gauge records the current value of the named gauge metric.
func Gauge(name string, value float64, tags ...string) error {
	if statsdClient == nil {
		return nil
	}
	return statsdClient.Gauge(name, value, tags, 1)
}
