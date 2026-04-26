package module

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	"github.com/xuri/excelize/v2"
)

type DataReader interface {
	Read(filePath string, opts parser.MigrationSourceOptions) ([]map[string]any, error)
}

func NewDataReader(sourceType parser.MigrationSourceType) (DataReader, error) {
	switch sourceType {
	case parser.SourceJSON:
		return &JSONReader{}, nil
	case parser.SourceCSV:
		return &CSVReader{}, nil
	case parser.SourceXLSX:
		return &XLSXReader{}, nil
	case parser.SourceXML:
		return &XMLReader{}, nil
	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}
}

type JSONReader struct{}

func (r *JSONReader) Read(filePath string, opts parser.MigrationSourceOptions) ([]map[string]any, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read JSON file %s: %w", filePath, err)
	}

	var records []map[string]any
	if err := json.Unmarshal(data, &records); err != nil {
		var obj map[string]json.RawMessage
		if err2 := json.Unmarshal(data, &obj); err2 != nil {
			return nil, fmt.Errorf("invalid JSON in %s: %w", filePath, err)
		}

		if opts.RootElement != "" {
			return extractJSONPath(obj, opts.RootElement)
		}

		for _, v := range obj {
			var arr []map[string]any
			if err := json.Unmarshal(v, &arr); err == nil {
				records = append(records, arr...)
			}
		}
		if len(records) == 0 {
			return nil, fmt.Errorf("no records found in JSON file %s", filePath)
		}
		return records, nil
	}

	return records, nil
}

func extractJSONPath(obj map[string]json.RawMessage, path string) ([]map[string]any, error) {
	parts := strings.Split(path, ".")
	current := obj

	for i, part := range parts {
		raw, ok := current[part]
		if !ok {
			return nil, fmt.Errorf("path element %q not found", part)
		}

		if i == len(parts)-1 {
			var records []map[string]any
			if err := json.Unmarshal(raw, &records); err != nil {
				return nil, fmt.Errorf("path %q does not contain an array: %w", path, err)
			}
			return records, nil
		}

		var next map[string]json.RawMessage
		if err := json.Unmarshal(raw, &next); err != nil {
			return nil, fmt.Errorf("path element %q is not an object: %w", part, err)
		}
		current = next
	}

	return nil, fmt.Errorf("empty path")
}

type CSVReader struct{}

func (r *CSVReader) Read(filePath string, opts parser.MigrationSourceOptions) ([]map[string]any, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open CSV file %s: %w", filePath, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	if opts.Delimiter != "" {
		runes := []rune(opts.Delimiter)
		if len(runes) > 0 {
			reader.Comma = runes[0]
		}
	}
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	allRows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV %s: %w", filePath, err)
	}

	if len(allRows) == 0 {
		return nil, nil
	}

	headerIdx := 0
	if opts.HeaderRow > 0 {
		headerIdx = opts.HeaderRow - 1
	}
	if opts.SkipRows > 0 {
		headerIdx = opts.SkipRows
	}

	if headerIdx >= len(allRows) {
		return nil, fmt.Errorf("header row %d exceeds total rows %d", headerIdx+1, len(allRows))
	}

	headers := allRows[headerIdx]
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	var records []map[string]any
	for i := headerIdx + 1; i < len(allRows); i++ {
		row := allRows[i]
		record := make(map[string]any, len(headers))
		allEmpty := true
		for j, header := range headers {
			if header == "" {
				continue
			}
			val := ""
			if j < len(row) {
				val = strings.TrimSpace(row[j])
			}
			if val != "" {
				allEmpty = false
			}
			record[header] = inferType(val)
		}
		if !allEmpty {
			records = append(records, record)
		}
	}

	return records, nil
}

type XLSXReader struct{}

func (r *XLSXReader) Read(filePath string, opts parser.MigrationSourceOptions) ([]map[string]any, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open XLSX file %s: %w", filePath, err)
	}
	defer f.Close()

	sheetName := opts.Sheet
	if sheetName == "" {
		sheetName = f.GetSheetName(0)
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet %q in %s: %w", sheetName, filePath, err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	headerIdx := 0
	if opts.HeaderRow > 0 {
		headerIdx = opts.HeaderRow - 1
	}
	if opts.SkipRows > 0 {
		headerIdx = opts.SkipRows
	}

	if headerIdx >= len(rows) {
		return nil, fmt.Errorf("header row %d exceeds total rows %d", headerIdx+1, len(rows))
	}

	headers := rows[headerIdx]
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	var records []map[string]any
	for i := headerIdx + 1; i < len(rows); i++ {
		row := rows[i]
		record := make(map[string]any, len(headers))
		allEmpty := true
		for j, header := range headers {
			if header == "" {
				continue
			}
			val := ""
			if j < len(row) {
				val = strings.TrimSpace(row[j])
			}
			if val != "" {
				allEmpty = false
			}
			record[header] = inferType(val)
		}
		if !allEmpty {
			records = append(records, record)
		}
	}

	return records, nil
}

type XMLReader struct{}

type xmlElement struct {
	XMLName  xml.Name
	Attrs    []xml.Attr    `xml:",any,attr"`
	Content  string        `xml:",chardata"`
	Children []xmlElement  `xml:",any"`
}

func (r *XMLReader) Read(filePath string, opts parser.MigrationSourceOptions) ([]map[string]any, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read XML file %s: %w", filePath, err)
	}

	var root xmlElement
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("invalid XML in %s: %w", filePath, err)
	}

	elements := root.Children
	if opts.RootElement != "" {
		elements = navigateXMLPath(root, opts.RootElement)
	}

	var records []map[string]any
	for _, elem := range elements {
		record := xmlElementToMap(elem)
		if len(record) > 0 {
			records = append(records, record)
		}
	}

	return records, nil
}

func navigateXMLPath(root xmlElement, path string) []xmlElement {
	parts := strings.Split(path, ".")
	current := []xmlElement{root}

	for _, part := range parts {
		var next []xmlElement
		for _, elem := range current {
			for _, child := range elem.Children {
				if child.XMLName.Local == part {
					next = append(next, child)
				}
			}
		}
		current = next
		if len(current) == 0 {
			return nil
		}
	}

	if len(current) == 1 && len(current[0].Children) > 0 {
		return current[0].Children
	}
	return current
}

func xmlElementToMap(elem xmlElement) map[string]any {
	record := make(map[string]any)

	for _, attr := range elem.Attrs {
		record[attr.Name.Local] = inferType(attr.Value)
	}

	if len(elem.Children) == 0 {
		content := strings.TrimSpace(elem.Content)
		if content != "" {
			record[elem.XMLName.Local] = inferType(content)
		}
		return record
	}

	childCounts := make(map[string]int)
	for _, child := range elem.Children {
		childCounts[child.XMLName.Local]++
	}

	childArrays := make(map[string][]any)
	for _, child := range elem.Children {
		name := child.XMLName.Local
		if len(child.Children) == 0 {
			val := inferType(strings.TrimSpace(child.Content))
			if childCounts[name] > 1 {
				childArrays[name] = append(childArrays[name], val)
			} else {
				record[name] = val
			}
		} else {
			childMap := xmlElementToMap(child)
			if childCounts[name] > 1 {
				childArrays[name] = append(childArrays[name], childMap)
			} else {
				record[name] = childMap
			}
		}
	}

	for name, arr := range childArrays {
		record[name] = arr
	}

	return record
}

func inferType(val string) any {
	if val == "" {
		return ""
	}

	if val == "true" {
		return true
	}
	if val == "false" {
		return false
	}

	if i, err := strconv.ParseInt(val, 10, 64); err == nil {
		if !strings.HasPrefix(val, "0") || val == "0" {
			return i
		}
	}

	if f, err := strconv.ParseFloat(val, 64); err == nil {
		if strings.Contains(val, ".") {
			return f
		}
	}

	return val
}

func ResolveSourcePath(basePath string, sourceFile string) string {
	if filepath.IsAbs(sourceFile) {
		return sourceFile
	}
	return filepath.Join(basePath, sourceFile)
}

func ReadSourceData(basePath string, source parser.MigrationSource) ([]map[string]any, error) {
	reader, err := NewDataReader(source.Type)
	if err != nil {
		return nil, err
	}

	filePath := ResolveSourcePath(basePath, source.File)
	return reader.Read(filePath, source.Options)
}
