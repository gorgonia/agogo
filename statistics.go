package agogo

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

type Statistics struct {
	Creation []string
	Wins     map[string][]float32
	Losses   map[string][]float32
	Draws    map[string][]float32
}

func makeStatistics() Statistics {
	return Statistics{
		Creation: make([]string, 0, 64),
		Wins:     make(map[string][]float32),
		Losses:   make(map[string][]float32),
		Draws:    make(map[string][]float32),
	}
}

func (s *Statistics) update(A *Agent) {
	aname := fmt.Sprintf("%p", A.NN)

	if _, ok := s.Wins[aname]; !ok {
		s.Creation = append(s.Creation, aname)
	}

	s.Wins[aname] = append(s.Wins[aname], A.Wins)
	s.Losses[aname] = append(s.Losses[aname], A.Loss)
	s.Draws[aname] = append(s.Draws[aname], A.Draw)
}

func (s *Statistics) Dump(filename string) error {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(s.Creation); err != nil {
		return err
	}
	var records [][]string
	for i, agent := range s.Creation {
		for j, win := range s.Wins[agent] {
			record := make([]string, len(s.Creation))
			winRate := win / (win + s.Losses[agent][j] + s.Draws[agent][j])

			record[i] = strconv.FormatFloat(float64(winRate), 'f', 3, 32)
			records = append(records, record)
		}
	}
	if err := w.WriteAll(records); err != nil {
		return err
	}
	w.Flush()
	return nil
}
