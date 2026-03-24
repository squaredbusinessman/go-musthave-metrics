package agent

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	models "github.com/squaredbusinessman/go-musthave-metrics/internal/model"
	storage "github.com/squaredbusinessman/go-musthave-metrics/internal/repository"
)

// ColleсtGopsutilMetrics - собирает системные метрики через gopsutil.
func ColleсtGopsutilMetrics(store *storage.MemStorage) {
	ctx := context.Background()

	vm, err := mem.VirtualMemory()
	if err == nil {
		_ = store.SetGauge(ctx, "TotalMemory", models.Gauge{
			Value: float64(vm.Total),
		})

		_ = store.SetGauge(ctx, "FreeMemory", models.Gauge{
			Value: float64(vm.Free),
		})
	}

	cpu, err := cpu.Percent(0, true)
	if err == nil {
		for i, p := range cpu {
			name := fmt.Sprintf("CPUutilization%d", i+1)
			_ = store.SetGauge(ctx, name, models.Gauge{
				Value: p,
			})
		}
	}
}
