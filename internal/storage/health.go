package storage

type HealthChecker interface{
	HealthCheck() error
}