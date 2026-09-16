package csv

import (
	"encoding/csv"
	"os"
	"reflect"
	"strconv"
)

func WriteCSV[T any](filename string, data []T) error {
	if len(data) == 0 {
		return nil
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// headers
	val := reflect.ValueOf(data[0])
	typ := val.Type()

	headers := make([]string, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("csv")
		if tag == "" {
			tag = typ.Field(i).Name
		}
		headers[i] = tag
	}
	_ = writer.Write(headers)

	// rows
	for _, item := range data {
		v := reflect.ValueOf(item)
		row := make([]string, v.NumField())

		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			switch field.Kind() {
			case reflect.Int, reflect.Int64:
				row[i] = strconv.FormatInt(field.Int(), 10)
			case reflect.String:
				row[i] = field.String()
			default:
				row[i] = ""
			}
		}
		_ = writer.Write(row)
	}

	return nil
}
