package importer

import (
	"context"
	"io"
	"testing"
)

func TestXls(t *testing.T) {
	ctx := context.Background()
	reader, closer, err := ReadXls(ctx, "test.xls", "utf8", 0, nil)
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
