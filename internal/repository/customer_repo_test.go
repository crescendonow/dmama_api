package repository

import "testing"

func TestValidateColumnAllowsStatsColumns(t *testing.T) {
	columns := []string{"prswtusg", "lstwtusg1", "lstwtusg11", "lstwtusg12"}

	for _, column := range columns {
		t.Run(column, func(t *testing.T) {
			if err := ValidateColumn(column); err != nil {
				t.Fatalf("expected %s to be allowed: %v", column, err)
			}
		})
	}
}

func TestValidateColumnRejectsUnsafeColumn(t *testing.T) {
	if err := ValidateColumn("lstwtusg13"); err == nil {
		t.Fatal("expected lstwtusg13 to be rejected")
	}
	if err := ValidateColumn("prswtusg;drop table"); err == nil {
		t.Fatal("expected SQL-like column to be rejected")
	}
}
