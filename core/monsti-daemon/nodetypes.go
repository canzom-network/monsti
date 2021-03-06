// This file is part of Monsti, a web content management system.
// Copyright 2012-2014 Christian Neumann <cneumann@datenkarussell.de>
//
// Monsti is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option) any
// later version.
//
// Monsti is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR
// A PARTICULAR PURPOSE.  See the GNU Affero General Public License for more
// details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Monsti.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"path"
	"github.com/chrneumann/htmlwidgets"
	gomail "gopkg.in/gomail.v1"
	"pkg.monsti.org/gettext"
	"pkg.monsti.org/monsti/api/util"
	"pkg.monsti.org/monsti/api/util/template"
)
import "pkg.monsti.org/monsti/api/service"

var availableLocales = []string{"en", "de"}

func initNodeTypes(settings *settings, session *service.Session, logger *log.Logger) error {
	G := func(in string) string { return in }
	pathType := service.NodeType{
		Id:   "core.Path",
		Hide: true,
		Name: util.GenLanguageMap(G("Path"), availableLocales),
	}
	if err := session.Monsti().RegisterNodeType(&pathType); err != nil {
		return fmt.Errorf("Could not register path node type: %v", err)
	}

	documentType := service.NodeType{
		Id:        "core.Document",
		AddableTo: []string{"."},
		Name:      util.GenLanguageMap(G("Document"), availableLocales),
		Fields: []*service.NodeField{
			{
				Id:       "core.Title",
				Required: true,
				Name:     util.GenLanguageMap(G("Title"), availableLocales),
				Type:     "Text",
			},
			{
				Id:       "core.Body",
				Required: true,
				Name:     util.GenLanguageMap(G("Body"), availableLocales),
				Type:     "HTMLArea",
			},
		},
	}
	if err := session.Monsti().RegisterNodeType(&documentType); err != nil {
		return fmt.Errorf("Could not register document node type: %v", err)
	}

	fileType := service.NodeType{
		Id:        "core.File",
		AddableTo: []string{"."},
		Name:      util.GenLanguageMap(G("File"), availableLocales),
		Fields: []*service.NodeField{
			{Id: "core.Title"},
			{Id: "core.Body"},
			{
				Id:       "core.File",
				Required: true,
				Name:     util.GenLanguageMap(G("File"), availableLocales),
				Type:     "File",
			},
		},
	}
	if err := session.Monsti().RegisterNodeType(&fileType); err != nil {
		return fmt.Errorf("Could not register file node type: %v", err)
	}

	imageType := service.NodeType{
		Id:        "core.Image",
		AddableTo: []string{"."},
		Name:      util.GenLanguageMap(G("Image"), availableLocales),
		Fields: []*service.NodeField{
			{Id: "core.Title"},
			{Id: "core.File"},
		},
	}
	if err := session.Monsti().RegisterNodeType(&imageType); err != nil {
		return fmt.Errorf("Could not register image node type: %v", err)
	}

	contactFormType := service.NodeType{
		Id:        "core.ContactForm",
		AddableTo: []string{"."},
		Name:      util.GenLanguageMap(G("Contact form"), availableLocales),
		Fields: []*service.NodeField{
			{Id: "core.Title"},
			{Id: "core.Body"},
		},
	}
	if err := session.Monsti().RegisterNodeType(&contactFormType); err != nil {
		return fmt.Errorf("Could not register contactform node type: %v", err)
	}
	return nil
}

type contactFormData struct {
	Name, Email, Subject, Message string
}

func renderContactForm(c *reqContext, context template.Context,
	formValues url.Values, h *nodeHandler) error {
	G, _, _, _ := gettext.DefaultLocales.Use("", c.Site.Locale)
	data := contactFormData{}
	form := htmlwidgets.NewForm(&data)
	form.AddWidget(&htmlwidgets.TextWidget{MinLength: 1,
		ValidationError: G("Required.")}, "Name", G("Name"), "")
	form.AddWidget(&htmlwidgets.TextWidget{MinLength: 1,
		ValidationError: G("Required.")}, "Email", G("Email"), "")
	form.AddWidget(&htmlwidgets.TextWidget{MinLength: 1,
		ValidationError: G("Required.")}, "Subject", G("Subject"), "")
	form.AddWidget(&htmlwidgets.TextAreaWidget{MinLength: 1,
		ValidationError: G("Required.")}, "Message", G("Message"), "")

	switch c.Req.Method {
	case "GET":
		if _, submitted := formValues["submitted"]; submitted {
			context["Submitted"] = 1
		}
	case "POST":
		if form.Fill(formValues) {
			mail := gomail.NewMessage()
			site := h.Settings.Monsti.Sites[c.Site.Name]
			mail.SetAddressHeader("From", site.EmailAddress, site.EmailName)
			mail.SetAddressHeader("To", site.Owner.Email, site.Owner.Name)
			mail.SetAddressHeader("Reply-To", data.Email, data.Name)
			mail.SetHeader("Subject", data.Subject)
			body := fmt.Sprintf("%v\n%v\n\n%v",
				fmt.Sprintf(G("Received from contact form at %v"), c.Site.Title),
				fmt.Sprintf(G("Name: %v | Email: %v"), data.Name, data.Email),
				data.Message)
			mail.SetBody("text/plain", body)
			mailer := gomail.NewCustomMailer("", nil, gomail.SetSendMail(
				c.Serv.Monsti().SendMailFunc()))
			err := mailer.Send(mail)
			if err != nil {
				return fmt.Errorf("Could not send mail: %v", err)
			}
			http.Redirect(c.Res, c.Req, path.Dir(c.Node.Path)+"/?submitted", http.StatusSeeOther)
			return nil
		}
	default:
		return fmt.Errorf("Request method not supported: %v", c.Req.Method)
	}
	context["Form"] = form.RenderData()
	return nil
}
