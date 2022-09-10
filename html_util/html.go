package html_util

import (
	"errors"
	"fmt"
	"github.com/rbnbr/go-utility/pkg/function"
	"github.com/rbnbr/go-utility/pkg/slices"
	"golang.org/x/net/html"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var TextRegex = regexp.MustCompile("[^!-~]") // without space

// WalkHtmlTree
// Calls f on node.
// If it returns true, call WalkHtmlTree on all of its children.
func WalkHtmlTree(node *html.Node, f func(n *html.Node) bool) {
	if node == nil {
		return
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if f(c) {
			WalkHtmlTree(c, f)
		}
	}
}

// GetChildren
// Same as below, return slice of pointers, even though considered bad practice, to be able to directly modify
// substructures of a bigger tree.
func GetChildren(node *html.Node) []*html.Node {
	var children []*html.Node
	if node == nil {
		return children
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		children = append(children, c)
	}
	return children
}

// GetNodeByCondition
// Returns the first node for which the provided condition yields true, including the start node
func GetNodeByCondition(startNode *html.Node, cond func(node *html.Node) bool) *html.Node {
	if startNode == nil {
		return nil
	}

	if cond(startNode) {
		return startNode
	} else {
		return GetNextNodeByCondition(startNode, cond)
	}
}

// GetNextNodeByCondition
// Returns the first node for which the provided condition yields true, excluding the start node
func GetNextNodeByCondition(startNode *html.Node, cond func(node *html.Node) bool) *html.Node {
	found := false
	var foundNode *html.Node

	if startNode == nil {
		return foundNode
	}

	WalkHtmlTree(startNode, func(n *html.Node) bool {
		if found {
			return false
		}

		if n == nil {
			return false
		}

		if cond(n) {
			found = true
			foundNode = n
			return false
		}

		return true
	})

	return foundNode
}

// GetNodesByCondition
// Return all nodes in the tree of startNode for which the provided condition yields true, including startNode.
// Note that this returns a slice with pointers to structs which is considered bad practice
// However, we do not want copies to the nodes but the actual pointers in case we want to modify nodes in part of
// a bigger tree structure.
func GetNodesByCondition(startNode *html.Node, cond func(node *html.Node) bool) []*html.Node {
	var foundNodes []*html.Node

	if startNode == nil {
		return foundNodes
	}

	if cond(startNode) {
		foundNodes = append(foundNodes, startNode)
	}

	return append(foundNodes, GetNextNodesByCondition(startNode, cond)...)
}

// GetNextNodesByCondition
// Return all nodes in the tree of startNode for which the provided condition yields true, excluding startNode.
// Note that this returns a slice with pointers to structs which is considered bad practice
// However, we do not want copies to the nodes but the actual pointers in case we want to modify nodes in part of
// a bigger tree structure.
func GetNextNodesByCondition(startNode *html.Node, cond func(node *html.Node) bool) []*html.Node {
	var foundNodes []*html.Node

	if startNode == nil {
		return foundNodes
	}

	WalkHtmlTree(startNode, func(n *html.Node) bool {
		if n == nil {
			return false
		}

		if cond(n) {
			foundNodes = append(foundNodes, n)
		}
		return true
	})

	return foundNodes
}

// GetElementNodeByTagName
// Returns the first node with the given tag name provided a starting node
// Returns nil if none found
func GetElementNodeByTagName(name string, startNode *html.Node) *html.Node {
	return GetNodeByCondition(startNode, MakeByTagNameCondition(name))
}

func MakeByTagNameCondition(name string) func(node *html.Node) bool {
	return func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == name
	}
}

func MakeByClassNameCondition(className string) func(node *html.Node) bool {
	return func(node *html.Node) bool {
		attr, err := GetAttributeByKey(node, "class")
		if err == nil {
			classNames := strings.Split(attr.Val, " ")
			for _, name := range classNames {
				if strings.EqualFold(name, className) {
					return true
				}
			}
		}
		return false
	}
}

func MakeByIdCondition(id string) func(node *html.Node) bool {
	return MakeByAttributeNameAndValueCondition("id", id)
}

func MakeByAttributeNameAndValueCondition(attributeName, attributeValue string) func(node *html.Node) bool {
	return func(node *html.Node) bool {
		attr, err := GetAttributeByKey(node, attributeName)
		if err == nil {
			if strings.EqualFold(attr.Val, attributeValue) {
				return true
			}
		}
		return false
	}
}

func GetFirstTextNode(startNode *html.Node) *html.Node {
	return GetNodeByCondition(startNode, func(node *html.Node) bool {
		return node.Type == html.TextNode
	})
}

func GetFirstTextNodeWithCondition(startNode *html.Node, cond func(s string) bool) *html.Node {
	return GetNodeByCondition(startNode, func(node *html.Node) bool {
		return node.Type == html.TextNode && cond(node.Data)
	})
}

func GetTextNodes(startNode *html.Node) []*html.Node {
	return GetNodesByCondition(startNode, func(node *html.Node) bool {
		return node.Type == html.TextNode
	})
}

func GetTextNodesByCondition(startNode *html.Node, cond func(s string) bool) []*html.Node {
	return GetNodesByCondition(startNode, func(node *html.Node) bool {
		return node.Type == html.TextNode && cond(node.Data)
	})
}

func MakeTextNodeCompositeWithNormalizerFunc(textNodes []*html.Node, compositeDelimiter string, normalizerFunc func(string) string) string {
	s := ""
	for i := 0; i < len(textNodes)-1; i++ {
		s += normalizerFunc(textNodes[i].Data) + compositeDelimiter
	}
	return s + normalizerFunc(textNodes[len(textNodes)-1].Data)
}

func MakeTextNodeComposite(textNodes []*html.Node, compositeRune string) string {
	return MakeTextNodeCompositeWithNormalizerFunc(textNodes, compositeRune, func(s string) string {
		return s
	})
}

// GetElementsInTableRowByConditionForOneOfTheElements
// Returns all children elements (with tag <td>) of the table row node with tag (<tr>), for which at least one children
// fulfills the provided condition cond
func GetElementsInTableRowByConditionForOneOfTheElements(tableNode *html.Node, cond func(n *html.Node) bool) []*html.Node {
	return GetNodesByCondition(tableNode, func(node *html.Node) bool {
		parent := node.Parent
		if parent != nil && parent.Type == html.ElementNode && parent.Data == "tr" {
			return node.Type == html.ElementNode && node.Data == "td" && GetNodeByCondition(parent, cond) != nil
		}
		return false
	})
}

// HtmlTable
// Represents an HTML table in a struct
// Contains only text content
type HtmlTable struct {
	Headers        []string              // Headers, equal to TableData[0, :] in numpy expression
	Index          []string              // Index, equal to TableData[:, 0] in numpy expression
	TableData      [][]string            // All data excluding headers and index
	normalizerFunc func(s string) string // used to normalize table content (additionally to the little regex)
	postfix        string                // postfix for recurring keys during parsing
}

// getRowByIndex
// Returns a reference of the table row with index i as well as the key of the corresponding index.
// panics if the row is out of bounds
// You can check the length of rows via the length of the index.
// getRowByIndex(0) returns the header row.
// getRowByIndex(1) returns the first row below the header row, and so on.
// Note: There is always a header row. Even if during parsing no header row was specified, the resulting table will have
// an artificial header row like (Index 1 2 3 4 ...)
func (ht HtmlTable) getRowByIndex(i int) ([]string, string) {
	if i == 0 {
		return ht.Headers, ht.Index[0]
	} else {
		return ht.TableData[i-1], ht.Index[i]
	}
}

// GetRowByIndex
// Returns a copy of the table row with index i as well as the key of the corresponding index.
// panics if the row is out of bounds
// You can check the length of rows via the length of the index.
// GetRowByIndex(0) returns the header row.
// GetRowByIndex(1) returns the first row below the header row, and so on.
// Note: There is always a header row. Even if during parsing no header row was specified, the resulting table will have
// an artificial header row like (Index 1 2 3 4 ...)
func (ht HtmlTable) GetRowByIndex(i int) ([]string, string) {
	foundRow, idx := ht.getRowByIndex(i)
	row := make([]string, len(foundRow))
	copy(row, foundRow)
	return row, idx
}

// GetRowByKey
// Returns the reference of the first row with the given key as index if it exists, else, returns (nil, false)
// The returned index would be the correct index to be used for getRowByIndex(idx)
func (ht HtmlTable) getRowByKey(key string) ([]string, int, bool) {
	for idx, idxKey := range ht.Index {
		if strings.EqualFold(idxKey, key) {
			return function.GetFirstReturnElement(ht.getRowByIndex(idx)).([]string), idx, true
		}
	}
	return nil, -1, false
}

// GetRowByKeyNum
// Returns the row with the original key (with possibly multiple occurrences) and the num occurrence
func (ht HtmlTable) GetRowByKeyNum(key string, occurrence int) ([]string, int, bool) {
	if occurrence == 0 {
		return ht.GetRowByKey(key)
	}
	return ht.GetRowByKey(fmt.Sprintf("%v%v%v", key, ht.postfix, occurrence))
}

// GetRowByKey
// Returns the copy of the row with the given key as index if it exists, else, returns (nil, false)
func (ht HtmlTable) GetRowByKey(key string) ([]string, int, bool) {
	for idx, idxKey := range ht.Index {
		if strings.EqualFold(idxKey, key) {
			return function.GetFirstReturnElement(ht.GetRowByIndex(idx)).([]string), idx, true
		}
	}
	return nil, -1, false
}

// GetColumnByIndex
// Analogous to GetRowByIndex but for columns.
// You can check the length of columns via the length of the Headers.
// GetColumnByIndex(0) returns the index column.
func (ht HtmlTable) GetColumnByIndex(j int) ([]string, string) {
	if j == 0 {
		cpy := make([]string, len(ht.Index))
		copy(cpy, ht.Index)
		return cpy, ht.Headers[0]
	} else {
		var column []string
		for i := 1; i < len(ht.Index); i++ {
			row, _ := ht.getRowByIndex(i) // can use this without copy since we copy when using append and row access
			column = append(column, row[j-1])
		}
		return column, ht.Headers[j]
	}
}

// GetColumnByKey
// Analogous to GetRowByKey but for columns.
func (ht HtmlTable) GetColumnByKey(key string) ([]string, int, bool) {
	for idx, header := range ht.Headers {
		if strings.EqualFold(header, key) {
			return function.GetFirstReturnElement(ht.GetColumnByIndex(idx)).([]string), idx, true
		}
	}
	return nil, -1, false
}

// GetColumnByKeyNum
// Returns the column with the original key (with possibly multiple occurrences) and the num occurrence
func (ht HtmlTable) GetColumnByKeyNum(key string, occurrence int) ([]string, int, bool) {
	if occurrence == 0 {
		return ht.GetColumnByKey(key)
	}
	return ht.GetColumnByKey(fmt.Sprintf("%v%v%v", key, ht.postfix, occurrence))
}

// GetElementByIndex
// Returns the element in table data for row i and column j.
// Panics if either is out of bounds.
func (ht HtmlTable) GetElementByIndex(i, j int) string {
	if j == 0 {
		// return index
		return function.GetFirstReturnElement(ht.GetColumnByIndex(0)).([]string)[i]
	} else {
		if i == 0 {
			// return header
			return function.GetFirstReturnElement(ht.getRowByIndex(i)).([]string)[j]
		} else {
			return function.GetFirstReturnElement(ht.getRowByIndex(i)).([]string)[j-1] // can use getRowByIndex without copy since we copy when using row access
		}
	}
}

// GetElementByKeys
// Returns the element in table data with the provided row key and column key.
// returns "", false if at least one key is missing.
func (ht HtmlTable) GetElementByKeys(rowKey, columnKey string) (string, int, int, bool) {
	row, i, ok := ht.getRowByKey(rowKey) // can use getRowByIndex without copy since we copy when using row access
	if ok {
		for idx, header := range ht.Headers {
			if strings.EqualFold(header, columnKey) {
				if idx == 0 {
					return row[idx], i, idx, true
				} else {
					return row[idx-1], i, idx, true
				}
			}
		}
	}
	return "", i, -1, false
}

// GetElementByKeysNum
// Returns the element in table data with the provided row key and column key and the corresponding occurrences.
// returns "", false if at least one key is missing.
func (ht HtmlTable) GetElementByKeysNum(rowKey, columnKey string, rowOccurrence, columnOccurrence int) (string, int, int, bool) {
	if rowOccurrence != 0 {
		rowKey = fmt.Sprintf("%v%v%v", rowKey, ht.postfix, rowOccurrence)
	}
	if columnOccurrence != 0 {
		columnKey = fmt.Sprintf("%v%v%v", columnKey, ht.postfix, columnOccurrence)
	}
	return ht.GetElementByKeys(rowKey, columnKey)
}

// ParseHtmlTable
// Parses a given html.Node which should point to a <table> ElementNode in a html tree to an HtmlTable Struct which
// can be used to easily look up existing indices, headers, and values.
// Content is set after normalizing with identity normalizer func, normalizer(s) = s.
// we append '{postfix}_{keyCount}' to keys which appear multiple times to make them unique.
// the first occurrence does not have this.
func ParseHtmlTable(tableNode *html.Node, hasHeaderRow bool, hasIndexColumn bool, postfix string) (*HtmlTable, error) {
	return ParseHtmlTableWithNormalizer(tableNode, hasHeaderRow, hasIndexColumn, postfix, func(s string) string {
		return s
	}, false, "")
}

// ParseHtmlTableWithNormalizer
// Parses a given html.Node which should point to a <table> ElementNode in a html tree to an HtmlTable Struct which
// can be used to easily look up existing indices, headers, and values.
// Content is set after normalizing with normalizerFunc
// we append '{postfix}_{keyCount}' to keys which appear multiple times to make them unique.
// the first occurrence does not have this.
func ParseHtmlTableWithNormalizer(tableNode *html.Node, hasHeaderRow bool, hasIndexColumn bool, postfix string, normalizerFunc func(string) string, allowCompositeTexts bool, compositeDelimiter string) (*HtmlTable, error) {
	// first assert we are a tableNode
	if tableNode == nil {
		return nil, errors.New("node is nil")
	}
	if !(tableNode.Type == html.ElementNode && tableNode.Data == "table") {
		return nil, errors.New("node is not an table node")
	}

	// get all row and columns to get TableData size
	rows := GetNodesByCondition(tableNode, func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "tr" && GetNextNodeByCondition(node, MakeByTagNameCondition("tr")) == nil
	})
	if len(rows) == 0 {
		return &HtmlTable{}, nil
	}

	maxRows := len(rows)
	maxColumns := 0
	var rawTableData [][]*html.Node
	// get all columns
	for _, row := range rows {
		cols := GetNodesByCondition(row, func(node *html.Node) bool {
			return node.Type == html.ElementNode && ((node.Data == "td" && GetNextNodeByCondition(node, MakeByTagNameCondition("td")) == nil) ||
				(node.Data == "th" && GetNextNodeByCondition(node, MakeByTagNameCondition("th")) == nil))
		})
		rawTableData = append(rawTableData, cols)
		if len(cols) > maxColumns {
			maxColumns = len(cols)
		}
	}

	hasHeader := 1
	hasIndex := 1
	if !hasIndexColumn {
		hasIndex = 0
	}
	if !hasHeaderRow {
		hasHeader = 0
	}

	const topLeft = "Index\\Header"

	// set headers
	var headers []string
	if hasHeaderRow {
		headers = make([]string, maxColumns+1-hasIndex)

		if !hasIndexColumn {
			headers[0] = topLeft
		}

		// Single Texts
		if !allowCompositeTexts {
			// set header values
			for j, h := range rawTableData[0] {
				hText := GetFirstTextNodeWithCondition(h, func(s string) bool {
					return len(TextRegex.ReplaceAllString(s, "")) > 0
				})
				if hText != nil {
					headers[j+1-hasIndex] = normalizerFunc(hText.Data)
				} else {
					headers[j+1-hasIndex] = ""
				}
			}
		} else {
			// set header values for multiple texts
			for j, h := range rawTableData[0] {
				hTexts := GetTextNodesByCondition(h, func(s string) bool {
					return len(TextRegex.ReplaceAllString(s, "")) > 0
				})
				if hTexts != nil {
					headers[j+1-hasIndex] = MakeTextNodeCompositeWithNormalizerFunc(hTexts, compositeDelimiter, normalizerFunc)
				} else {
					headers[j+1-hasIndex] = ""
				}
			}
		}
	} else {
		hasHeader = 0
		headers = make([]string, maxColumns+1-hasIndex)

		// Add index column header
		headers[0] = topLeft

		for j := 1; j < len(headers); j++ {
			headers[j] = strconv.Itoa(j)
		}
	}

	// set index
	var index []string
	if hasIndexColumn {
		index = make([]string, maxRows+1-hasHeader)

		if !hasHeaderRow {
			index[0] = topLeft
		}

		if !allowCompositeTexts {
			// Single Texts
			// set index values
			for i, idxRow := range rawTableData {
				if len(idxRow) > 0 {
					iText := GetFirstTextNodeWithCondition(idxRow[0], func(s string) bool {
						return len(TextRegex.ReplaceAllString(s, "")) > 0
					})
					if iText != nil {
						index[i+1-hasHeader] = normalizerFunc(iText.Data)
					} else {
						index[i+1-hasHeader] = ""
					}
				}
			}
		} else {
			// set index values
			for i, idxRow := range rawTableData {
				if len(idxRow) > 0 {
					iTexts := GetTextNodesByCondition(idxRow[0], func(s string) bool {
						return len(TextRegex.ReplaceAllString(s, "")) > 0
					})
					if iTexts != nil {
						index[i+1-hasHeader] = MakeTextNodeCompositeWithNormalizerFunc(iTexts, compositeDelimiter, normalizerFunc)
					} else {
						index[i+1-hasHeader] = ""
					}
				}
			}
		}
	} else {
		hasIndex = 0
		index = make([]string, maxRows+1-hasHeader)
		index[0] = topLeft
		for i := 1; i < len(index); i++ {
			index[i] = strconv.Itoa(i)
		}
	}

	// make headers and index unique
	headers = slices.MakeUniqueStringSlice(headers, postfix)
	index = slices.MakeUniqueStringSlice(index, postfix)

	tableData := make([][]string, len(index)-1)
	for i := 0; i < len(tableData); i++ {
		tableData[i] = make([]string, len(headers)-1)
		for j := 0; j < len(rawTableData[i+hasHeader])-hasIndex; j++ {
			tdNode := rawTableData[i+hasHeader][j+hasIndex]

			if !allowCompositeTexts {
				// Single Texts
				tdText := GetFirstTextNodeWithCondition(tdNode, func(s string) bool {
					return len(TextRegex.ReplaceAllString(s, "")) > 0
				})
				if tdText != nil {
					tableData[i][j] = normalizerFunc(tdText.Data)
				}
			} else {
				tdTexts := GetTextNodesByCondition(tdNode, func(s string) bool {
					return len(TextRegex.ReplaceAllString(s, "")) > 0
				})
				if tdTexts != nil {
					tableData[i][j] = MakeTextNodeCompositeWithNormalizerFunc(tdTexts, compositeDelimiter, normalizerFunc)
				}
			}
		}
	}

	return &HtmlTable{
		Headers:   headers,
		Index:     index,
		TableData: tableData,
		postfix:   postfix,
	}, nil
}

func GetAttributeByKey(node *html.Node, key string) (html.Attribute, error) {
	if node == nil {
		return html.Attribute{}, errors.New("node is nil")
	} else {
		for _, attr := range node.Attr {
			if attr.Key == key {
				return attr, nil
			}
		}
		return html.Attribute{}, errors.New(fmt.Sprintf("node has no attribute with key: '%v%", key))
	}
}

// ParseSelectHTMLNode
// Parses the html node with tag 'select' into its different options.
// Returns a map containing key: value as strings, in which key is the content text content of the option and value is the content of the 'value' attribute of this option.
//
// If multiple options have the same content text, they will be overridden and only the last one is kept.
// Returns the currently selected option, which is the option with attribute 'selected' if it exists, otherwise the first occurring option.
//
// If multiple options have the "selected" attribute, returns the last option that has it as "selectedOption"
// Returns nil map and nil error if no options were found.
func ParseSelectHTMLNode(selectNode *html.Node) (map[string]string, string, error) {
	if selectNode == nil {
		return nil, "", errors.New("cannot parse nil node")
	}

	options := GetNodesByCondition(selectNode, MakeByTagNameCondition("option"))
	if len(options) == 0 {
		log.Println("failed to get any available options")
		return nil, "", nil
	}

	availableOptions := make(map[string]string)

	firstAvailableOptionKey := ""

	selectedOption := ""
	foundSelected := false
	for i, optionNode := range options {
		optionValueAttr, err := GetAttributeByKey(optionNode, "value")
		if err != nil {
			return nil, "", err
		}

		optionTextNode := GetFirstTextNode(optionNode)
		if optionTextNode == nil {
			return nil, "", errors.New("failed to get text of option node")
		}

		optionText := optionTextNode.Data

		availableOptions[optionText] = optionValueAttr.Val

		if i == 0 {
			firstAvailableOptionKey = optionText
		}

		_, err = GetAttributeByKey(optionNode, "selected")
		if err == nil {
			// attribute has key 'selected'
			selectedOption = optionText
			foundSelected = true
		}
	}

	if foundSelected == false {
		// set selected to first entry
		selectedOption = firstAvailableOptionKey
	}

	return availableOptions, selectedOption, nil
}
