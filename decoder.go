package ecfformat

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	modeRoot = iota
	modeList
	modeNamedList
	modeExtendList
)

var (
	wsRegex = regexp.MustCompile(`^(\s+)`)
)

type Parser struct {
	mode         int
	currentSigWs string
	rv           reflect.Value
	currentLine  int
	listField    string
	listName     string
}

func NewParser() *Parser {
	p := &Parser{}
	p.clear()
	return p
}

func (p *Parser) clear() {
	p.mode = modeRoot
	p.currentSigWs = ""
	p.currentLine = 0
}

// ParseFile will load the file filename and put it into a TaskFile struct or return an error if something goes wrong
func (p *Parser) ParseFile(v interface{}, filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("Task file does not exist: %s", filename)
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return p.parse(v, file)
}

func (p *Parser) ParseString(v interface{}, data string) error {
	return p.parse(v, strings.NewReader(data))
}

func (p *Parser) parse(v interface{}, reader io.Reader) error {
	p.clear()

	// Create scanner
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	p.currentLine = 0
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("Invalid struct for decoding")
	}
	p.rv = rv.Elem()

	for scanner.Scan() {
		// Get next line
		lineRaw := scanner.Text()
		lineTrimmed := strings.TrimSpace(lineRaw)
		p.currentLine++

		// Dev special line means stop parsing
		if len(lineTrimmed) >= 3 && lineTrimmed[:3] == "###" {
			break
		}

		// Check for blank lines and comments
		if len(lineTrimmed) < 1 || lineTrimmed[0] == '#' {
			continue
		}

		if p.mode == modeList {
			if err := p.parseListLine(lineRaw); err != nil {
				return err
			}
		} else if p.mode == modeExtendList {
			if err := p.parseExtendedListLine(lineRaw); err != nil {
				return err
			}
			// } else if p.mode == modeDevices {
			// 	if err := p.parseDeviceLine(lineRaw, lineNum); err != nil {
			// 		return err
			// 	}
			// } else {
		} else {
			if err := p.parseLine(lineRaw); err != nil {
				return err
			}
		}
		//}
	}

	return nil
}

func (p *Parser) parseLine(line string) error {
	// Split only on the first colon
	p.mode = modeRoot
	parts := strings.SplitN(line, ":", 2)

	if len(parts) != 2 {
		return fmt.Errorf("Error on line %d of file", p.currentLine)
	}
	setting := strings.ToLower(parts[0])
	setting = strings.Title(setting)
	setting = strings.Replace(setting, " ", "", -1)
	setting = strings.TrimSpace(setting)
	settingVal := strings.TrimSpace(parts[1])

	// exported field
	f := p.rv.FieldByName(setting)
	if f.IsValid() {
		// A Value can be changed only if it is
		// addressable and was not obtained by
		// the use of unexported struct fields.
		if f.CanSet() {
			// change value of N
			switch f.Kind() {
			case reflect.String:
				if f.String() != "" {
					return fmt.Errorf("Cannot redeclare setting '%s'. Line %d", setting, p.currentLine)
				}
				f.SetString(settingVal)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				i, err := strconv.ParseInt(settingVal, 10, 64)
				if err != nil {
					return fmt.Errorf("Expected integer on line %d", p.currentLine)
				}
				if f.OverflowInt(i) {
					return fmt.Errorf("Integer is too big on line %d", p.currentLine)
				}
				f.SetInt(i)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				i, err := strconv.ParseUint(settingVal, 10, 64)
				if err != nil {
					return fmt.Errorf("Expected unsigned integer on line %d", p.currentLine)
				}
				if f.OverflowUint(i) {
					return fmt.Errorf("Unsigned integer is too big on line %d", p.currentLine)
				}
				f.SetUint(i)
			case reflect.Float32, reflect.Float64:
				i, err := strconv.ParseFloat(settingVal, 64)
				if err != nil {
					return fmt.Errorf("Expected float on line %d", p.currentLine)
				}
				if f.OverflowFloat(i) {
					return fmt.Errorf("Float is too big on line %d", p.currentLine)
				}
				f.SetFloat(i)
			case reflect.Bool:
				var b bool
				switch strings.ToLower(settingVal) {
				case "true", "yes", "t", "1":
					b = true
					break
				case "false", "no", "f", "0":
					b = false
					break
				default:
					return fmt.Errorf("Expected boolean on line %d", p.currentLine)
				}
				f.SetBool(b)
			case reflect.Slice:
				t := f.Type().Elem()
				// []string is a simple list
				if t.Kind() == reflect.String {
					p.mode = modeList
					p.listField = setting
				} else {
					// A [] of anything else is invalid
					return fmt.Errorf("Invalid slice type \"%s\"", setting)
				}
			case reflect.Map:
				t := f.Type()
				// All maps must have string keys
				if t.Key().Kind() != reflect.String {
					return fmt.Errorf("Maps must have a key type of string \"%s\"", setting)
				}
				// Create map if it's not initialized
				if f.IsNil() {
					f.Set(reflect.MakeMap(t))
				}

				t = t.Elem()
				// If the map element is a []string, it's a named simple list
				if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.String {
					p.mode = modeList
					p.listField = setting
					p.listName = settingVal
				} else if t.Kind() == reflect.Struct || t.Kind() == reflect.Ptr {
					p.mode = modeExtendList
					p.listField = setting
					return p.parseExtendListDeclaration(settingVal)
				} else {
					// A [] of anything else is invalid
					return fmt.Errorf("Invalid map type \"%s\"", setting)
				}
			case reflect.Struct:
				fmt.Println("Extended list")
			default:
				return fmt.Errorf("Invalid type %s:%s\n", setting, f.Kind().String())
			}
		} else {
			return fmt.Errorf("Cannot set field \"%s\". Line %d", setting, p.currentLine)
		}
	} else {
		return fmt.Errorf("Invalid field \"%s\". Line %d", setting, p.currentLine)
	}

	return nil
}

func (p *Parser) parseListLine(line string) error {
	matches := wsRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return p.parseLine(line)
	}
	sigWs := matches[0]

	// At this point we already know the field is valid and can be set and is a slice
	f := p.rv.FieldByName(p.listField)

	if f.Len() == 0 {
		p.currentSigWs = sigWs
	} else {
		if sigWs != p.currentSigWs {
			return fmt.Errorf("Line not in block, check indention. Line %d\n", p.currentLine)
		}
	}

	switch f.Kind() {
	case reflect.Slice:
		line = strings.TrimSpace(line)
		f.Set(reflect.Append(f, reflect.ValueOf(line)))
	case reflect.Map:
		mapElem := f.MapIndex(reflect.ValueOf(p.listName))
		elemType := f.Type().Elem()
		if elemType.Elem().Kind() != reflect.String {
			return fmt.Errorf("Lists and named lists must be slices of strings")
		}

		if !mapElem.IsValid() {
			mapElem = reflect.New(elemType).Elem()
		}

		line = strings.TrimSpace(line)
		mapElem = reflect.Append(mapElem, reflect.ValueOf(line))
		kv := reflect.ValueOf(p.listName).Convert(f.Type().Key())
		f.SetMapIndex(kv, mapElem)
	}
	return nil
}

func (p *Parser) parseExtendListDeclaration(line string) error {
	pieces := strings.Split(line, " ")
	p.listName = pieces[0]

	// Map itself
	s := p.rv.FieldByName(p.listField)
	elemType := s.Type().Elem()
	mapElem := reflect.New(elemType).Elem()
	itemsField := mapElem.FieldByName("Items")
	itemsField.Set(reflect.MakeSlice(reflect.TypeOf([]string(nil)), 0, 0))

	nameField := mapElem.FieldByName("Name")
	if nameField.IsValid() && nameField.CanSet() {
		if nameField.Kind() != reflect.String {
			return fmt.Errorf("Name field must be a string")
		}
		nameField.SetString(p.listName)
	}

	if len(pieces) > 1 {
		for _, setting := range pieces[1:] {
			parts := strings.Split(setting, "=")
			if len(parts) < 2 {
				continue
			}
			setting := strings.ToLower(parts[0])
			setting = strings.Title(setting)
			setting = strings.TrimSpace(setting)

			// exported field
			f := mapElem.FieldByName(setting)
			if f.IsValid() {
				// A Value can be changed only if it is
				// addressable and was not obtained by
				// the use of unexported struct fields.
				if f.CanSet() {
					// change value of N
					if f.Kind() == reflect.String {
						f.SetString(parts[1])
					} else {
						return fmt.Errorf("Invalid list setting type")
					}
				} else {
					return fmt.Errorf("Cannot set field \"%s\". Line %d", setting, p.currentLine)
				}
			} else {
				return fmt.Errorf("Invalid block setting \"%s\". Line %d\n", parts[0], p.currentLine)
			}
		}
	}

	kv := reflect.ValueOf(p.listName).Convert(s.Type().Key())
	s.SetMapIndex(kv, mapElem)
	return nil
}

func (p *Parser) parseExtendedListLine(line string) error {
	matches := wsRegex.FindStringSubmatch(line)
	if len(matches) == 0 {
		return p.parseLine(line)
	}
	sigWs := matches[0]

	s := p.rv.FieldByName(p.listField)
	mapElem := s.MapIndex(reflect.ValueOf(p.listName))
	itemsField := mapElem.FieldByName("Items")

	if itemsField.Len() == 0 {
		p.currentSigWs = sigWs
	} else {
		if sigWs != p.currentSigWs {
			return fmt.Errorf("Line not in block, check indention. Line %d\n", p.currentLine)
		}
	}

	line = strings.TrimSpace(line)
	itemsField.Set(reflect.Append(itemsField, reflect.ValueOf(line)))
	return nil
}
