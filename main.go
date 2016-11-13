package main



import (
//	"flag"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/binding"
	"text/template"
	"mime/multipart"
	"github.com/gocarina/gocsv"
	"bytes"
	"io/ioutil"
	"github.com/blvp/devfest_email_sender/email_sender"
	"gopkg.in/yaml.v2"
	"flag"
)

type UploadForm struct {
	Template        string                `form:"template"`
	CsvTemplateData *multipart.FileHeader `form:"data"`
}

type EmailsPage struct {
	Emails []*email_sender.UserEmail
}

func main() {
	smtpConfigPath := flag.String("smtp-config-path", "", "define config for smtp")
	flag.Parse()
	fileContent, err := ioutil.ReadFile(*smtpConfigPath)

	if err != nil {
		panic(err)
	}
	sc := email_sender.SenderConfig{}
	yaml.Unmarshal(fileContent, &sc)
	m := martini.Classic()
	m.Use(render.Renderer(render.Options{
		Directory: "templates",
		Extensions: []string{".html", },
	}))

	m.Map(email_sender.NewEmailSender(sc))
	m.Get("/", func(r render.Render) {
		r.HTML(200, "index", nil)
	})
	m.Post("/createEmails", binding.MultipartForm(UploadForm{}), func(uf UploadForm, r render.Render, es *email_sender.EmailSender) {
		file, err := uf.CsvTemplateData.Open()

		if err != nil {
			panic("failed to upload file")
		}
		defer file.Close()
		users := []*email_sender.User{}
		csvContent, _ := ioutil.ReadAll(file)
		if err := gocsv.UnmarshalBytes(csvContent, &users); err != nil {
			panic(err)
		}

		t, err := template.New("messageTemplate").Parse(uf.Template)
		userEmails := []*email_sender.UserEmail{}
		for id, user := range users {
			sw := bytes.NewBuffer([]byte{})
			t.Execute(sw, user)
			userEmails = append(userEmails, &email_sender.UserEmail{
				Id: id,
				User: user,
				Message:sw.String(),
				Status: "created",
			})
		}
		es.UserEmails = userEmails
		r.Redirect("/emails")
	})

	m.Get("/emails", func(es *email_sender.EmailSender, r render.Render) {
		r.HTML(200, "emails", EmailsPage{
			Emails:es.UserEmails,
		})
	})

	m.Post("/sendEmails", func(es *email_sender.EmailSender, r render.Render) {
		es.SendFailedOrCreated()
		r.Status(200)
	})

	m.Post("/clearAll", func(es *email_sender.EmailSender, r render.Render) {
		es.CleanQueue()
		r.Status(200)
	})

	m.RunOnAddr(":7000")

}

