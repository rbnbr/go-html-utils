# Go-Html-Utils

Some utility functions I use for querying DOM elements which are represented as [*html.Node](https://pkg.go.dev/golang.org/x/net/html) objects.

Main Features:
- getting nodes by condition (with multiple variations)
- getting attributes of nodes
- parsing a html node representing a html \<table\> element into a golang struct
- parsing a html node representing a html \<select\> element into a golang map\[string\]string


## Examples:

**Retrieving the htmlNode**

````go
log.Printf("GET: '%s'\n", someUrl.String())
res, err := http.DefaultClient.Get(someUrl.String())
if err != nil {
	return err
}
defer res.Body.Close()

if res.StatusCode != http.StatusOK {
	return errors.New(fmt.Sprintf("Got status code '%v', expected '%v'.", res.StatusCode, http.StatusOK))
}

htmlNode, err = html.Parse(res.Body)
if err != nil {
	return err
}
````

**Running queries:**

***Get all \<div\> elements***

````go
divs := GetNodesByCondition(htmlNode, MakeByTagNameCondition("div"))
if len(divs) == 0 {
	return errors.New("no divs found")
}
````

***Get all elements containing the class name "button"***

````go
buttons := GetNodesByCondition(htmlNode, MakeByClassNameCondition("button"))
if len(divs) == 0 {
	return errors.New("no elements of class 'button' found")
}
````

***Get first element that has attribute "name" with value "Datum"***

````go
datum := GetNodeByCondition(htmlNode, MakeByAttributeNameAndValueCondition("name", "Datum"))
if datum == nil {
	return errors.New("no element with attribute:value that is 'name':'Datum' found")
}
````

## Create arbitrary conditions & Combining conditions

***Create arbitrary condition***
````go
element := GetNodeByCondition(htmlNode, func(node *html.Node) bool {
	boolean := true
	/*
	Some arbitrary logic.
	The @node parameter takes the value of all *html.Node elements in the tree of @htmlNode for which this function is evaluated.
	GetNodeByCondition will return the first node, for which this function evaluates true.
	 */
	return boolean
})
````

***Get first element with class name "clickable" that has as tag "button".***

Defining custom conditions also allows to combine existing conditions:

````go
element := GetNodeByCondition(htmlNode, func(node *html.Node) bool {
 return MakeByClassNameCondition("clickable")(node) && MakeByTagNameCondition("button")(node)
})
````

### Parsing html \<table\> elements

````go
table := GetNodeByCondition(htmlNode, MakeByTagNameCondition("table"))
htmlTable, err := ParseHtmlTable(table, true, true, "")

dates, columnIndex, found := htmlTable.GetColumnByKey("Datum")
````

### Parsing html \<select\> elements

````go
selectElement := GetNodeByCondition(htmlNode, MakeByTagNameCondition("select"))
availableOptions, currentlySelectedOption, err := ParseSelectHTMLNode(selectElement)

log.Printf("currently selected option has text content '%s' and option value '%s'\n", currentlySelectedOption, availableOptions[currentlySelectedOption])
````

