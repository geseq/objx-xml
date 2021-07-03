package xml

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/beevik/etree"
	"github.com/stretchr/objx"
)

/**
 * Schema definition (in order of preference)
 * Start with ? in key for optional (element discarded when value absent or empty)
 * Start with @ in key for attribute
 * Start with : in key to run an array of sub elements. Key should be of the form :mainRoot:subElementRoot
 * Keep value empty for no check
 * Start with / in value to do a regex match
 * | in value to separate possible values for validation
 */

// XmlBuilder builds an xml output to objx definition with objx data
type XmlBuilder struct {
	BatchRootNode string
	RootNode      string
	Schema        objx.Map
	ProcessData   bool
}

// XmlDocument represents an xml document
type XmlDocument struct {
	*etree.Document
}

// Process builds an xml output for the schema
func (x *XmlBuilder) Process(data objx.Map) (string, error) {
	var doc = &XmlDocument{etree.NewDocument()}
	doc.Indent(4)

	var rootElement = doc.CreateElement(x.RootNode)
	var err = doc.processSchema(x.Schema, data, rootElement)
	if err != nil {
		return "", err
	}
	var result, _ = doc.WriteToString()
	return result, nil
}

func (d *XmlDocument) processSchema(schema objx.Map, data objx.Map, root *etree.Element) error {
	var err error = nil
	for key, valueSchema := range schema {
		var fc = string(key[0])
		var sKey = string(key[1:])

		// Optional
		if fc == "?" {
			if !data.Has(sKey) || data.Get(sKey).String() == "" {
				continue
			}

			key = sKey
			fc = string(key[0])
			sKey = string(key[1:])
		}

		var mapSchema, isMap = valueSchema.(objx.Map)
		if isMap {
			// Map Slice
			if fc == ":" {
				var parts = strings.Split(sKey, ":")
				if len(parts) != 2 {
					err = fmt.Errorf("Key of map slice  %s must contain 2 parts", key)
					break
				}

				var rootElement = root.CreateElement(parts[0])
				var dataSlice = data.Get(parts[0]).ObjxMapSlice()

				for _, slice := range dataSlice {
					var childRoot = rootElement.CreateElement(parts[1])
					d.processSchema(mapSchema, slice, childRoot)
				}
				continue
			}

			// Map
			var rootElement = root.CreateElement(key)
			d.processSchema(mapSchema, data.Get(key).ObjxMap(), rootElement)
			continue
		}

		var strSchema, isString = valueSchema.(string)
		if !isString {
			err = fmt.Errorf("Expects string schema for %s", key)
			break
		}

		// Attribute
		if fc == "@" {
			var value, err = d.getValidatedValue(strSchema, key, data.Get(key).String())
			if err != nil {
				break
			}
			root.CreateAttr(sKey, value)
			continue
		}

		var value, err = d.getValidatedValue(strSchema, key, data.Get(key).String())
		if err != nil {
			break
		}

		var el = root.CreateElement(key)
		el.SetText(value)
	}

	return err
}

func (d *XmlDocument) getValidatedValue(rule string, key string, value string) (string, error) {
	if rule == "" {
		return value, nil
	}

	var fc = string(rule[0])

	if fc == "/" {
		rule = strings.Trim(rule, "/")
		var r = regexp.MustCompile(rule)
		if r.ReplaceAllString(value, "") != "" {
			var err = fmt.Errorf("Invalid value %s for key %s", value, key)
			return "", err
		}

		return value, nil
	}

	var parts = strings.Split(rule, "|")
	if !d.stringInSlice(parts, value) {
		return parts[0], nil
	}

	return value, nil
}

func (d *XmlDocument) stringInSlice(list []string, a string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
