package handler_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/handler"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
	"github.com/squaredbusinessman/go-musthave-metrics/internal/service"
)

func newExampleServer() *httptest.Server {
	store := repository.NewMemStorage()
	metricsService := service.NewMetricsService(store)
	h := handler.New(metricsService, nil, nil)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.AcceptMetricsToStorage)
	r.Post("/update", h.UpdateMetricJSON)
	r.Post("/updates", h.UpdateMetricsBatch)
	r.Get("/value/{type}/{name}", h.GetMetric)
	r.Post("/value", h.GetMetricJSON)

	return httptest.NewServer(r)
}

func mustDo(client *http.Client, req *http.Request) *http.Response {
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	return resp
}

func mustRead(resp *http.Response) string {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(body))
}

func ExampleHandler_AcceptMetricsToStorage() {
	server := newExampleServer()
	defer server.Close()

	updateReq, err := http.NewRequest(http.MethodPost, server.URL+"/update/gauge/Alloc/42.5", nil)
	if err != nil {
		panic(err)
	}
	updateReq.Header.Set("Content-Type", "text/plain")
	updateResp := mustDo(server.Client(), updateReq)
	defer updateResp.Body.Close()

	valueResp, err := server.Client().Get(server.URL + "/value/gauge/Alloc")
	if err != nil {
		panic(err)
	}

	fmt.Println(updateResp.StatusCode)
	fmt.Println(valueResp.StatusCode)
	fmt.Println(mustRead(valueResp))

	// Output:
	// 200
	// 200
	// 42.5
}

func ExampleHandler_UpdateMetricJSON() {
	server := newExampleServer()
	defer server.Close()

	updateReq, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/update",
		bytes.NewBufferString(`{"id":"PollCount","type":"counter","delta":5}`),
	)
	if err != nil {
		panic(err)
	}
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := mustDo(server.Client(), updateReq)

	valueReq, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/value",
		bytes.NewBufferString(`{"id":"PollCount","type":"counter"}`),
	)
	if err != nil {
		panic(err)
	}
	valueReq.Header.Set("Content-Type", "application/json")
	valueResp := mustDo(server.Client(), valueReq)

	fmt.Println(updateResp.StatusCode)
	fmt.Println(mustRead(updateResp))
	fmt.Println(valueResp.StatusCode)
	fmt.Println(mustRead(valueResp))

	// Output:
	// 200
	// {"id":"PollCount","type":"counter","delta":5}
	// 200
	// {"id":"PollCount","type":"counter","delta":5}
}

func ExampleHandler_UpdateMetricsBatch() {
	server := newExampleServer()
	defer server.Close()

	batchReq, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/updates",
		bytes.NewBufferString(`[{"id":"Alloc","type":"gauge","value":21.5},{"id":"PollCount","type":"counter","delta":3}]`),
	)
	if err != nil {
		panic(err)
	}
	batchReq.Header.Set("Content-Type", "application/json")
	batchResp := mustDo(server.Client(), batchReq)
	defer batchResp.Body.Close()

	gaugeResp, err := server.Client().Get(server.URL + "/value/gauge/Alloc")
	if err != nil {
		panic(err)
	}

	counterResp, err := server.Client().Get(server.URL + "/value/counter/PollCount")
	if err != nil {
		panic(err)
	}

	fmt.Println(batchResp.StatusCode)
	fmt.Println(mustRead(gaugeResp))
	fmt.Println(mustRead(counterResp))

	// Output:
	// 200
	// 21.5
	// 3
}
