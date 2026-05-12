package domain

import "fmt"

type PillarWeight struct {
	TenantID  int    `json:"tenant_id"`
	EngineID  int    `json:"engine_id"`
	Pillar    string `json:"pillar"`
	Weight    int    `json:"weight"`
	UpdatedBy string `json:"updated_by"`
}

type SetWeightsRequest struct {
	EngineID  int            `json:"engine_id"`
	Weights   map[string]int `json:"weights"`
	UpdatedBy string         `json:"updated_by"`
}

func (r SetWeightsRequest) Validate(minWeight, maxWeight int) error {
	if r.EngineID == 0 {
		return validationError("engine_id é obrigatório")
	}
	if r.UpdatedBy == "" {
		return validationError("updated_by é obrigatório")
	}
	if len(r.Weights) == 0 {
		return validationError("weights não pode ser vazio")
	}

	total := 0
	for pillar, w := range r.Weights {
		if w < minWeight {
			return validationError(fmt.Sprintf("pilar '%s': peso %d abaixo do mínimo permitido (%d)", pillar, w, minWeight))
		}
		if w > maxWeight {
			return validationError(fmt.Sprintf("pilar '%s': peso %d acima do máximo permitido (%d)", pillar, w, maxWeight))
		}
		total += w
	}
	if total != 100 {
		return validationError(fmt.Sprintf("soma dos pesos deve ser 100, mas é %d", total))
	}
	return nil
}
