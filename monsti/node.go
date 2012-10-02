package main

import (
	"code.google.com/p/gorilla/sessions"
	"datenkarussell.de/monsti/rpc/client"
	"datenkarussell.de/monsti/template"
	"fmt"
	"github.com/chrneumann/g5t"
	"io/ioutil"
	"launchpad.net/goyaml"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var G func(string) string = g5t.String

// getFooter retrieves the footer.
//
// root is the path to the data directory
//
// Returns an empty string if there is no footer.
func getFooter(root string) string {
	path := filepath.Join(root, "footer.html")
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

// getBelowHeader retrieves the below header content for the given node.
//
// path is the node's path.
// root is the path to the data directory.
//
// Returns an empty string if there is no below header content.
func getBelowHeader(path, root string) string {
	file := filepath.Join(root, path, "below_header.html")
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return ""
	}
	return string(content)
}

// getSidebar retrieves the sidebar content for the given node.
//
// path is the node's path.
// root is the path to the data directory.
//
// It traverses up to the root until it finds a node with defined sidebar
// content.
//
// Returns an empty string if there is no sidebar content.
func getSidebar(path, root string) string {
	for {
		file := filepath.Join(root, path, "sidebar.html")
		content, err := ioutil.ReadFile(file)
		if err != nil {
			if path == filepath.Dir(path) {
				break
			}
			path = filepath.Dir(path)
			continue
		}
		return string(content)
	}
	return ""
}

// navLink represents a link in the navigation.
type navLink struct {
	Name, Target string
	Active       bool
}

// getNav returns the navigation for the given node.
// 
// node is the path of the node for which to get the navigation.
// active is the currently active node.
// root is the path of the data directory.
//
// The keys of the returned map are the link titles, the values are
// the link targets.
//
// If the node has no navigation defined (i.e. there exists no
// navigation.yaml), a navigation is searched recursively for the parent node up
// to the root.
func getNav(path, active, root string) []navLink {
	var content []byte
	for {
		file := filepath.Join(root, path, "navigation.yaml")
		var err error
		content, err = ioutil.ReadFile(file)
		if err != nil {
			if path == filepath.Dir(path) {
				break
			}
			path = filepath.Dir(path)
			continue
		}
		break
	}
	var navLinks []navLink
	goyaml.Unmarshal(content, &navLinks)
	for i, link := range navLinks {
		if link.Target == active {
			navLinks[i].Active = true
			break
		}
	}
	return navLinks
}

type addFormData struct {
	Type, Name, Title string
}

func (data *addFormData) Check(e *template.FormErrors) {
	e.Check("Type", data.Type, template.Required())
	e.Check("Name", data.Name, template.Required())
	e.Check("Name", data.Name, template.Regex(`^\w*$`,
		G("Contains	invalid characters.")))
	data.Name = strings.ToLower(data.Name)
	e.Check("Title", data.Title, template.Required())
}

// Add handles add requests.
func (h *nodeHandler) Add(w http.ResponseWriter, r *http.Request,
	nodePath string, session *sessions.Session) {
	var data addFormData
	var errors template.FormErrors
	switch r.Method {
	case "GET":
	case "POST":
		r.ParseForm()
		var err error
		errors, err = template.Validate(r.Form, &data)
		if err != nil {
			panic("Could not parse form data: " + err.Error())
		}
		if len(errors) == 0 {
			// TODO Allow other content types.
			if data.Type != "Document" {
				panic("Can't add this content type.")
			}
			newPath := filepath.Join(nodePath, data.Name)
			node := client.Node{
				Path:  newPath,
				Type:  data.Type,
				Title: data.Title}
			if err = writeNode(node, h.Settings.Directories.Data); err != nil {
				panic("Can't add node: " + err.Error())
			}
			http.Redirect(w, r, newPath+"/@@edit", http.StatusSeeOther)
		}
	default:
		panic("Request method not supported: " + r.Method)
	}
	body := h.Renderer.Render("actions/addform.html", errors, data)
	fmt.Fprint(w, renderInMaster(h.Renderer, []byte(body), new(masterTmplEnv),
		h.Settings))
}

// lookupNode look ups a node at the given path.
// If no such node exists, return nil.
func lookupNode(root, path string) (client.Node, error) {
	node_path := filepath.Join(root, path[1:], "node.yaml")
	content, err := ioutil.ReadFile(node_path)
	if err != nil {
		return client.Node{}, err
	}
	var node client.Node
	if err = goyaml.Unmarshal(content, &node); err != nil {
		return client.Node{}, err
	}
	node.Path = path
	return node, nil
}

// writeNode writes the given node to the data directory located at the given
// root.
func writeNode(node client.Node, root string) error {
	path := node.Path
	node.Path = ""
	content, err := goyaml.Marshal(&node)
	if err != nil {
		return err
	}
	node_path := filepath.Join(root, path[1:],
		"node.yaml")
	if err := os.Mkdir(filepath.Dir(node_path), 0700); err != nil {
		if !os.IsExist(err) {
			panic("Can't create directory for new node: " + err.Error())
		}
	}
	return ioutil.WriteFile(node_path, content, 0600)
}
