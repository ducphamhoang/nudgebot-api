package mocks

//go:generate mockgen -source=../events/bus.go -destination=./event_bus_mock.go -package=mocks
//go:generate mockgen -source=../chatbot/service.go -destination=./chatbot_service_mock.go -package=mocks
//go:generate mockgen -source=../llm/service.go -destination=./llm_service_mock.go -package=mocks
//go:generate mockgen -source=../llm/provider.go -destination=./llm_provider_mock.go -package=mocks
//go:generate mockgen -source=../nudge/service.go -destination=./nudge_service_mock.go -package=mocks
//go:generate mockgen -source=../nudge/repository.go -destination=./nudge_repository_mock.go -package=mocks
//go:generate mockgen -source=../scheduler/scheduler.go -destination=./scheduler_mock.go -package=mocks

// This file contains go:generate directives for creating mocks
// The actual mock implementations are in separate files to avoid import cycles
