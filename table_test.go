package gotestify

import (
	"testing"
)

func TestTableEqual(t *testing.T) {
	type Row struct {
		Name string
		Age  int
	}

	rows := []Row{
		{"Alice", 25},
		{"Bob", 30},
	}

	expected := `
		Alice |25 |
		Bob   |30 |
	`

	TableEqual(t, expected, rows, []string{"Name", "Age"})
}
