package importer

import (
	"context"
	"io"
	"testing"
)

func TestXlsx(t *testing.T) {
	ctx := context.Background()
	reader, closer, err := ReadXlsx(ctx, "test.xlsx", "", nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer closer.Close()

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Error(err)
			return
		}

		t.Log(record)
	}
}
