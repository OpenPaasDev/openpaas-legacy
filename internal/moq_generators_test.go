package internal

//go:generate moq -pkg o11y -stub -out ./o11y/moq_consul_client_test.go ./hashistack Consul:MockConsul
//go:generate moq -pkg internal -stub -out ./moq_consul_client_test.go ./hashistack Consul:MockConsul
